package main

import (
	"fmt"
	"os"

	"github.com/ProgenyAlpha/reddit-lurker/cmd"
)

const version = "1.0.0"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "thread":
		cmd.Thread(os.Args[2:])
	case "subreddit", "sub":
		cmd.Subreddit(os.Args[2:])
	case "search":
		cmd.Search(os.Args[2:])
	case "user":
		cmd.User(os.Args[2:])
	case "serve":
		cmd.Serve(version)
	case "version", "--version", "-v":
		fmt.Printf("lurk v%s\n", version)
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Print(`lurk — Reddit reader for Claude Code

Usage:
  lurk thread <url>                          Read a thread with all comments
  lurk subreddit <name> [flags]              Browse a subreddit
  lurk search <query> [flags]                Search Reddit
  lurk user <username> [flags]               View user activity
  lurk serve                                 Start MCP stdio server

Flags:
  --sort <value>     Sort order (hot, new, top, rising, controversial, relevance, comments)
  --limit <n>        Number of results (default 25, max 500)
  --time <value>     Time filter (hour, day, week, month, year, all)
  --sub <name>       Restrict search to subreddit
  --json             Output raw JSON
  --compact          Output compact notation (default in MCP mode)
  --no-cache         Skip cache

Examples:
  lurk thread "https://www.reddit.com/r/ClaudeAI/comments/abc123/post_title/"
  lurk subreddit ClaudeAI --sort top --limit 10
  lurk search "MCP server" --sub ClaudeAI
  lurk user spez --limit 5
`)
}
