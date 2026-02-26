# reddit-lurker

> Every comment. Every reply. 94% fewer tokens.

Reddit MCP server for AI coding tools. Fetches full comment trees at any depth, compresses them into compact notation, and delivers 94% fewer tokens than raw JSON.

Most Reddit tools fetch top-level comments and stop. Lurk expands every collapsed branch, resolves every `+N more replies` placeholder, and reconstructs the full tree.

## Quick Start

```bash
npx reddit-lurker
```

This downloads the platform binary and runs the MCP server. Add it to your editor config for persistent use:

**Claude Code, Cursor, Windsurf, Cline:**
```json
{
  "mcpServers": {
    "lurk": {
      "command": "npx",
      "args": ["-y", "reddit-lurker"]
    }
  }
}
```

**VS Code (Copilot):**
```json
{
  "servers": {
    "lurk": {
      "command": "npx",
      "args": ["-y", "reddit-lurker"]
    }
  }
}
```

## What It Does

- **Full comment trees** at any depth — every collapsed branch expanded
- **94% fewer tokens** than JSON, 44% fewer than Markdown
- **Smart limiting** — threads with 200+ comments preview first, expand on demand
- **Adaptive caching** — 2-30min TTLs by content type, 50MB LRU cap
- **Multi-subreddit search** — comma-separated subs, parallel fetch, deduped results
- **All URL formats** — reddit.com, old.reddit.com, redd.it short links, etc.
- **Single binary**, zero runtime dependencies

## Token Savings

Benchmarked across 12 threads (452 comments, 6 subreddits):

| Format | Total Tokens | vs JSON | vs Markdown |
|--------|-------------|---------|-------------|
| Raw Reddit JSON | 286,425 | — | — |
| Markdown | 28,993 | -90% | — |
| **Lurk (compact)** | **16,186** | **-94%** | **-44%** |

## OAuth (Optional)

For 6x rate limits (60 req/min instead of 10), create a Reddit app at [reddit.com/prefs/apps](https://www.reddit.com/prefs/apps) (type: "script"), then add `env` to your MCP config:

**Claude Code, Cursor, Windsurf, Cline:**
```json
{
  "mcpServers": {
    "lurk": {
      "command": "npx",
      "args": ["-y", "reddit-lurker"],
      "env": {
        "LURK_CLIENT_ID": "your_client_id",
        "LURK_CLIENT_SECRET": "your_client_secret"
      }
    }
  }
}
```

Or use a credentials file at `~/.config/lurk/credentials.json`:
```json
{"client_id": "your_client_id", "client_secret": "your_client_secret"}
```

Or use `lurk auth` from the standalone binary for interactive setup.

## Other Install Methods

- **Homebrew:** `brew install ProgenyAlpha/tap/lurk`
- **Go:** `go install github.com/ProgenyAlpha/reddit-lurker@latest`
- **Direct download:** [GitHub Releases](https://github.com/ProgenyAlpha/reddit-lurker/releases)
- **Install script:** `curl -fsSL https://raw.githubusercontent.com/ProgenyAlpha/reddit-lurker/master/install.sh | bash`

## Documentation

Full docs, benchmarks, and CLI reference: [github.com/ProgenyAlpha/reddit-lurker](https://github.com/ProgenyAlpha/reddit-lurker)

## License

MIT
