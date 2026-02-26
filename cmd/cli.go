package cmd

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/ProgenyAlpha/reddit-lurker/format"
	"github.com/ProgenyAlpha/reddit-lurker/reddit"
)

// extractPositionalAndFlags separates positional args from flags so flags
// can appear before or after the positional argument.
// Go's flag package stops at the first non-flag arg, which breaks
// "lurk search 'query' --sub ClaudeAI".
func extractPositionalAndFlags(args []string) (positional string, flags []string) {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			flags = append(flags, arg)
			// If this flag takes a value (not a boolean), grab the next arg too
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") && !strings.Contains(arg, "=") {
				// Peek: if it's a known bool flag, don't consume next
				// For simplicity, always consume next as value if it exists
				// Exception: flags like --json, --compact, --no-cache are boolean
				flagName := strings.TrimLeft(arg, "-")
				if flagName == "json" || flagName == "compact" || flagName == "no-cache" || flagName == "info" {
					continue
				}
				i++
				flags = append(flags, args[i])
			}
		} else if positional == "" {
			positional = arg
		}
	}
	return
}

// Thread handles the "lurk thread <url>" command.
func Thread(args []string) {
	url, flags := extractPositionalAndFlags(args)

	fs := flag.NewFlagSet("thread", flag.ExitOnError)
	jsonOut := fs.Bool("json", false, "Output raw JSON")
	compact := fs.Bool("compact", false, "Output compact notation")
	noCache := fs.Bool("no-cache", false, "Skip cache")
	fs.Parse(flags)

	if url == "" {
		fmt.Fprintln(os.Stderr, "Usage: lurk thread <url>")
		os.Exit(1)
	}
	client := reddit.NewClient()

	thread, err := client.GetThread(url, *noCache)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}

	switch {
	case *jsonOut:
		fmt.Println(format.ToJSON(thread))
	case *compact:
		fmt.Print(format.CompactThread(thread))
	default:
		fmt.Print(format.FormatThread(thread))
	}
}

// Subreddit handles the "lurk subreddit <name>" command.
func Subreddit(args []string) {
	name, flags := extractPositionalAndFlags(args)

	fs := flag.NewFlagSet("subreddit", flag.ExitOnError)
	sort := fs.String("sort", "hot", "Sort order (hot, new, top, rising, controversial)")
	limit := fs.Int("limit", 25, "Number of posts")
	timeFilter := fs.String("time", "", "Time filter (hour, day, week, month, year, all)")
	after := fs.String("after", "", "Pagination token for next page")
	jsonOut := fs.Bool("json", false, "Output raw JSON")
	compact := fs.Bool("compact", false, "Output compact notation")
	noCache := fs.Bool("no-cache", false, "Skip cache")
	info := fs.Bool("info", false, "Show subreddit info instead of posts")
	fs.Parse(flags)

	if name == "" {
		fmt.Fprintln(os.Stderr, "Usage: lurk subreddit <name> [flags]")
		os.Exit(1)
	}
	client := reddit.NewClient()

	if *info {
		subInfo, err := client.GetSubredditInfo(name, *noCache)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s\n", err)
			os.Exit(1)
		}
		switch {
		case *jsonOut:
			fmt.Println(format.ToJSON(subInfo))
		case *compact:
			fmt.Print(format.CompactSubredditInfo(subInfo))
		default:
			fmt.Print(format.FormatSubredditInfo(subInfo))
		}
		return
	}

	posts, nextAfter, err := client.GetSubreddit(name, reddit.SortOrder(*sort), *limit, reddit.TimeFilter(*timeFilter), *after, *noCache)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}

	switch {
	case *jsonOut:
		fmt.Println(format.ToJSON(posts))
	case *compact:
		fmt.Print(format.CompactPostList(posts, name, *sort, nextAfter))
	default:
		header := fmt.Sprintf("r/%s — %s", name, *sort)
		fmt.Print(format.FormatPostList(posts, header, nextAfter))
	}
}

// Search handles the "lurk search <query>" command.
func Search(args []string) {
	query, flags := extractPositionalAndFlags(args)

	fs := flag.NewFlagSet("search", flag.ExitOnError)
	sub := fs.String("sub", "", "Restrict to subreddit(s) (comma-separated)")
	sort := fs.String("sort", "relevance", "Sort order (relevance, hot, top, new, comments)")
	limit := fs.Int("limit", 25, "Number of results")
	timeFilter := fs.String("time", "", "Time filter (hour, day, week, month, year, all)")
	after := fs.String("after", "", "Pagination token for next page")
	jsonOut := fs.Bool("json", false, "Output raw JSON")
	compact := fs.Bool("compact", false, "Output compact notation")
	noCache := fs.Bool("no-cache", false, "Skip cache")
	fs.Parse(flags)

	if query == "" {
		fmt.Fprintln(os.Stderr, "Usage: lurk search <query> [flags]")
		os.Exit(1)
	}
	client := reddit.NewClient()

	posts, nextAfter, err := client.Search(query, *sub, reddit.SortOrder(*sort), *limit, reddit.TimeFilter(*timeFilter), *after, *noCache)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}

	switch {
	case *jsonOut:
		fmt.Println(format.ToJSON(posts))
	case *compact:
		fmt.Print(format.CompactSearchResults(posts, query, *sub, nextAfter))
	default:
		fmt.Print(format.FormatSearchResults(posts, query, nextAfter))
	}
}

// User handles the "lurk user <username>" command.
func User(args []string) {
	username, flags := extractPositionalAndFlags(args)

	fs := flag.NewFlagSet("user", flag.ExitOnError)
	limit := fs.Int("limit", 10, "Number of activity items")
	jsonOut := fs.Bool("json", false, "Output raw JSON")
	compact := fs.Bool("compact", false, "Output compact notation")
	noCache := fs.Bool("no-cache", false, "Skip cache")
	fs.Parse(flags)

	if username == "" {
		fmt.Fprintln(os.Stderr, "Usage: lurk user <username> [flags]")
		os.Exit(1)
	}
	client := reddit.NewClient()

	info, posts, comments, err := client.GetUser(username, *limit, *noCache)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}

	switch {
	case *jsonOut:
		fmt.Println(format.ToJSON(map[string]any{
			"info":     info,
			"posts":    posts,
			"comments": comments,
		}))
	case *compact:
		fmt.Print(format.CompactUser(info, posts, comments))
	default:
		fmt.Print(format.FormatUser(info, posts, comments))
	}
}
