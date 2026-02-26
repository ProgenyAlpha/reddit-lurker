package cmd

import (
	"context"
	"fmt"
	"log"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/ProgenyAlpha/reddit-lurker/format"
	"github.com/ProgenyAlpha/reddit-lurker/reddit"
)

// Serve starts the MCP stdio server.
func Serve(version string) {
	s := server.NewMCPServer(
		"lurk",
		version,
	)

	client := reddit.NewClient()

	// Register tools
	s.AddTool(lurkTool(), lurkHandler(client))
	s.AddTool(lurkInfoTool(), lurkInfoHandler(client))

	if err := server.ServeStdio(s); err != nil {
		log.Fatalf("MCP server error: %s", err)
	}
}

func lurkTool() mcp.Tool {
	return mcp.NewTool("lurk",
		mcp.WithDescription(`Read Reddit content. Compact tab-delimited output.

Commands:
  thread    - Full post + comment tree. target = Reddit URL
  subreddit - List posts. target = subreddit name (no r/ prefix)
  search    - Search posts. target = query string
  user      - User profile + activity. target = username

Compact notation:
  #post header: sub, author, score, ratio, comments, date
  #comments: d0/d1/d2 = depth, score before author
  #sub/#search: numbered list with scores and permalinks
  +N = collapsed comments not loaded`),
		mcp.WithString("command",
			mcp.Required(),
			mcp.Description("Command: thread, subreddit, search, user"),
			mcp.Enum("thread", "subreddit", "search", "user"),
		),
		mcp.WithString("target",
			mcp.Required(),
			mcp.Description("URL (thread), subreddit name, search query, or username"),
		),
		mcp.WithString("sort",
			mcp.Description("Sort: hot, new, top, rising, controversial, relevance, comments"),
		),
		mcp.WithNumber("limit",
			mcp.Description("Max results (default 25). For threads: limits comments sorted by score (0=all, N=top N). Threads with 200+ comments show a preview unless limit is set."),
		),
		mcp.WithString("time",
			mcp.Description("Time filter: hour, day, week, month, year, all"),
		),
		mcp.WithString("sub",
			mcp.Description("Restrict search to subreddit (search command only)"),
		),
	)
}

func lurkHandler(client *reddit.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		command := req.GetString("command", "")
		target := req.GetString("target", "")
		sort := req.GetString("sort", "")
		limit := req.GetInt("limit", 25)
		timeFilter := req.GetString("time", "")
		sub := req.GetString("sub", "")

		if target == "" {
			return mcp.NewToolResultError("target is required"), nil
		}

		// Check if limit was explicitly provided (distinguish "not set" from "set to 0")
		args := req.GetArguments()
		_, limitSet := args["limit"]

		switch command {
		case "thread":
			thread, err := client.GetThreadShallow(target, false)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			numComments := thread.Post.NumComments
			shallowCount := reddit.CountComments(thread.Comments)

			if numComments > 200 && !limitSet {
				// Large thread, no explicit limit — return preview + warning
				output := format.CompactThread(thread)
				estTokensK := (reddit.EstimateTokens(thread.Comments) * numComments / max(shallowCount, 1)) / 1024
				if estTokensK < 1 {
					estTokensK = 1
				}
				warning := fmt.Sprintf(
					"\n#warning\t%d total comments, showing %d. Use limit=N for top N by score, or limit=0 for all (~%dK tokens).\n",
					numComments, shallowCount, estTokensK)
				return mcp.NewToolResultText(output + warning), nil
			}

			// Full expansion
			client.ExpandThread(thread, false)

			if limitSet && limit > 0 {
				thread.Comments = reddit.TopCommentsByScore(thread.Comments, limit)
			}

			return mcp.NewToolResultText(format.CompactThread(thread)), nil

		case "subreddit":
			if sort == "" {
				sort = "hot"
			}
			posts, after, err := client.GetSubreddit(target, reddit.SortOrder(sort), limit, reddit.TimeFilter(timeFilter), "", false)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return mcp.NewToolResultText(format.CompactPostList(posts, target, sort, after)), nil

		case "search":
			if sort == "" {
				sort = "relevance"
			}
			posts, after, err := client.Search(target, sub, reddit.SortOrder(sort), limit, reddit.TimeFilter(timeFilter), "", false)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return mcp.NewToolResultText(format.CompactSearchResults(posts, target, sub, after)), nil

		case "user":
			info, posts, comments, err := client.GetUser(target, limit, false)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return mcp.NewToolResultText(format.CompactUser(info, posts, comments)), nil

		default:
			return mcp.NewToolResultError(fmt.Sprintf("unknown command: %s", command)), nil
		}
	}
}

func lurkInfoTool() mcp.Tool {
	return mcp.NewTool("lurk_info",
		mcp.WithDescription("Get subreddit metadata: subscribers, active users, description, type."),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Subreddit name (no r/ prefix)"),
		),
	)
}

func lurkInfoHandler(client *reddit.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name := req.GetString("name", "")
		if name == "" {
			return mcp.NewToolResultError("name is required"), nil
		}

		info, err := client.GetSubredditInfo(name, false)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(format.CompactSubredditInfo(info)), nil
	}
}
