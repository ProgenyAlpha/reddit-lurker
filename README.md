```
  ██████╗ ███████╗██████╗ ██████╗ ██╗████████╗
  ██╔══██╗██╔════╝██╔══██╗██╔══██╗██║╚══██╔══╝
  ██████╔╝█████╗  ██║  ██║██║  ██║██║   ██║
  ██╔══██╗██╔══╝  ██║  ██║██║  ██║██║   ██║
  ██║  ██║███████╗██████╔╝██████╔╝██║   ██║
  ╚═╝  ╚═╝╚══════╝╚═════╝ ╚═════╝ ╚═╝   ╚═╝
  ██╗     ██╗   ██╗██████╗ ██╗  ██╗███████╗██████╗
  ██║     ██║   ██║██╔══██╗██║ ██╔╝██╔════╝██╔══██╗
  ██║     ██║   ██║██████╔╝█████╔╝ █████╗  ██████╔╝
  ██║     ██║   ██║██╔══██╗██╔═██╗ ██╔══╝  ██╔══██╗
  ███████╗╚██████╔╝██║  ██║██║  ██╗███████╗██║  ██║
  ╚══════╝ ╚═════╝ ╚═╝  ╚═╝╚═╝  ╚═╝╚══════╝╚═╝  ╚═╝
```

<p align="center">
  <a href="https://github.com/ProgenyAlpha/reddit-lurker/releases"><img src="https://img.shields.io/github/v/release/ProgenyAlpha/reddit-lurker" alt="Release"></a>
  <a href="https://go.dev"><img src="https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go&logoColor=white" alt="Go"></a>
  <a href="https://github.com/ProgenyAlpha/reddit-lurker/blob/master/LICENSE"><img src="https://img.shields.io/github/license/ProgenyAlpha/reddit-lurker" alt="License: MIT"></a>
  <a href="https://github.com/ProgenyAlpha/reddit-lurker"><img src="https://img.shields.io/badge/platform-linux%20%7C%20macOS%20%7C%20windows-lightgrey" alt="Platform"></a>
</p>

> Every comment. Every reply. 77% fewer tokens.

An 800-comment Reddit thread costs ~120K tokens as raw JSON. Lurk delivers the same thread — full depth, every expanded reply — in ~31K tokens.

Most Reddit tools fetch top-level comments and stop. The useful stuff is buried 4-5 replies deep. Lurk expands every collapsed branch, resolves every `+N more replies` placeholder, and reconstructs the full comment tree. Then compresses it into compact tab-delimited notation before it reaches your model.

```text
Post: "Finally We have the best agentic AI at home"
 +-- Comment (180 pts)
 |   +-- Reply (46 pts)              <-- most tools stop here
 |   |   +-- Reply (34 pts)
 |   |       +-- Reply (29 pts)
 |   |           +-- Reply (8 pts)
 |   |               +-- Reply (20 pts)
 |   |                   +-- Reply (2 pts)
 |   |                       +-- Reply (4 pts)
 |   |                           +-- Reply (1 pt)
 |   |                               +-- Reply (2 pts)  <-- lurk gets all of it
 +-- Comment (82 pts)
 |   +-- Reply (45 pts)              <-- lurk gets all of this too
 |       +-- Reply ...
 +-- Comment (60 pts)
     +-- +47 more replies (expanded)
```

**104 of 109 comments. 10 levels deep. Fully automatic.**

## How Token Savings Work

The Go binary preprocesses everything before tokens reach your model:

1. **Fetch** — Hits Reddit's JSON endpoints, recursively expands every collapsed `more` placeholder
2. **Extract** — Strips the 50+ unused fields per comment (gildings, awards, flair, metadata) down to 5-6 that matter
3. **Compress** — Formats into compact tab-delimited notation: `d0 180 Recent-Success-1520 If you can host Kimi 2.5...`

The result:

| Format | ~Tokens for 109-comment thread |
|--------|-------------------------------|
| Raw Reddit JSON | ~12,000 |
| Markdown | ~5,200 |
| **Lurk compact** | **~3,050** |

**77% fewer tokens than JSON. 42% fewer than Markdown.** Same data, same structure, same depth.

### Smart Comment Limiting

Threads with 200+ comments get a preview first instead of dumping everything:

```text
#post   r/ClaudeAI   u/poster   422pts   93%   805cmt   2026-01-28
Finally We have the best agentic AI at home

#comments   461
d0  180  Recent-Success-1520  If you can host Kimi 2.5...
...

#warning   805 total comments, showing 461. Use limit=N for top N by score, or limit=0 for all (~31K tokens).
```

Claude sees the warning and decides whether to fetch everything or grab the top 50 by score. No surprise 31K-token dumps.

## What You Get

- **Full comment trees** at any depth — every collapsed branch expanded
- **42% fewer tokens** than Markdown, 77% fewer than raw JSON
- **Smart limiting** — large threads preview first, expand on demand
- **Adaptive caching** — new feeds: 2min, hot: 5min, threads: 10min, top: 30min, 50MB LRU cap
- **Multi-subreddit search** — comma-separated subs, parallel fetch, deduped results
- **OAuth support** — optional `lurk auth` for 6x rate limits (60 req/min vs 10)
- **All URL formats** — reddit.com, old.reddit.com, new.reddit.com, np.reddit.com, redd.it short links, m.reddit.com, amp.reddit.com
- **Cross-posts** traced to the original
- **Galleries, video, media URLs** extracted as clickable links
- **Single Go binary**, zero runtime dependencies
- **Read-only by design** — no write operations, no account access

## Install

### One-Line Install (Recommended)

**Linux / macOS:**
```bash
curl -fsSL https://raw.githubusercontent.com/ProgenyAlpha/reddit-lurker/master/install.sh | bash
```

**Windows (PowerShell):**
```powershell
irm https://raw.githubusercontent.com/ProgenyAlpha/reddit-lurker/master/install.ps1 | iex
```

Downloads the binary for your platform and walks you through editor setup. Supports Claude Code, Cursor, Windsurf, VS Code (Copilot), Cline, and Zed.

### Homebrew

```bash
brew install ProgenyAlpha/tap/lurk
```

### npx

```bash
npx reddit-lurker
```

If you already have Node/npm. Also available on [Smithery](https://smithery.ai).

### Go Install

```bash
go install github.com/ProgenyAlpha/reddit-lurker@latest
```

Builds from source. Requires [Go 1.24+](https://go.dev/dl/). Run `./install.sh` afterward for editor configuration.

### From Source

```bash
git clone https://github.com/ProgenyAlpha/reddit-lurker.git
cd reddit-lurker
./install.sh
```

---

The installer walks you through editor selection and integration mode.

### Supported Editors

| Editor | Config location | MCP key |
|--------|----------------|---------|
| Claude Code | `~/.claude.json` or `~/.claude/skills/reddit/` | `mcpServers` |
| Cursor | `~/.cursor/mcp.json` | `mcpServers` |
| Windsurf | `~/.codeium/windsurf/mcp_config.json` | `mcpServers` |
| VS Code (Copilot) | `~/.config/Code/User/mcp.json` (Linux) / `~/Library/.../Code/User/mcp.json` (macOS) | `servers` |
| Cline | VS Code globalStorage (auto-detected) | `mcpServers` |
| Zed | `~/.config/zed/settings.json` | `context_servers` |

Claude Code also supports a **Skill** mode (~20 tokens overhead vs ~438 for MCP). The installer will ask which you prefer.

### Manual Configuration

Add lurk to your editor's MCP config:

**Claude Code, Cursor, Windsurf, Cline:**
```json
{
  "mcpServers": {
    "lurk": {
      "command": "lurk",
      "args": ["serve"]
    }
  }
}
```

**GitHub Copilot (VS Code):**
```json
{
  "servers": {
    "lurk": {
      "command": "lurk",
      "args": ["serve"]
    }
  }
}
```

**Zed:**
```json
{
  "context_servers": {
    "lurk": {
      "command": "lurk",
      "args": ["serve"]
    }
  }
}
```

## OAuth (Optional)

Lurk works without any authentication. But if you want 6x the rate limit (60 req/min instead of 10):

```bash
lurk auth
```

This opens Reddit's app creation page, walks you through the 5-minute setup, tests your credentials, and saves them. One-time process. Lurk handles token refresh automatically.

```bash
lurk auth --status   # Check if credentials are configured
lurk auth --clear    # Remove saved credentials
```

You can also set credentials via environment variables:
```bash
export LURK_CLIENT_ID=your_client_id
export LURK_CLIENT_SECRET=your_client_secret
```

## Usage

Just talk to Claude naturally:

- *"Read this thread"* + paste a Reddit URL
- *"What's trending on r/ClaudeAI?"*
- *"Search r/selfhosted,r/homelab for 'ZFS backup'"*
- *"What has u/spez been up to?"*

Claude handles the rest. No commands to memorize.

## Real Example

Here's what lurk actually outputs for a [109-comment r/LocalLLM thread](https://www.reddit.com/r/LocalLLM/comments/1qp880l/finally_we_have_the_best_agentic_ai_at_home/) about running Kimi K2.5 at home.

**What most tools give your LLM:**
```
u/Recent-Success-1520 (180 pts)
  If you can host Kimi 2.5 1T+ model at home then it tells
  me you have a really big home

u/No_Conversation9561 (82 pts)
  not in my home

u/rookan (60 pts)
  yeah, my 16GB VRAM card can easily handle it /s

... 12 top-level comments, no replies
```

**What lurk gives your LLM:**
```
#post	r/LocalLLM	u/moks4tda	422pts	93%	109cmt	2026-01-28
Finally We have the best agentic AI at home

#comments	104
d0	180	Recent-Success-1520	If you can host Kimi 2.5 1T+ model at home...
d1	46	HenkPoley	Apparently it's a native 4 bit weights. So "only" 640 GB needed...
d2	34	TechnicalGeologist99	Sorry...you're going to run that model on RAM?
d3	29	HenkPoley	24 tokens per second on 2x 512GB Max Studio M3 Ultra
d4	8	doradus_novae	See you tomorrow when it answers your question
d5	20	Scrubbingbubblz	You are over exaggerating. 24 tokens per second...
d6	2	Infinite100p	But what is the prompt processing speed?
d7	4	Miserable-Dare5090	It's GPU inference, on two m3 ultras over TB5...
d8	1	Infinite100p	How?
d9	2	Eastern-Group-1993	Via usb-c networking, RDMA.
d0	82	No_Conversation9561	not in my home
d1	45	gonxot	[image] Maybe it's the same guy lol
d0	60	rookan	yeah, my 16GB VRAM card can easily handle it /s
d0	27	keypa_	"at home" we probably don't have the same home...
...
```

**104 of 109 comments. 10 levels deep. ~3,050 tokens.** The 5 missing are deleted posts Reddit still counts but no longer serves.

## Benchmarks

Real numbers from live Reddit threads:

| Thread | Comments fetched | Compact tokens | Savings vs JSON | Fetch time |
|--------|-----------------|---------------|-----------------|------------|
| 25-comment announcement | 25 / 25 (100%) | ~786 | **74%** | ~0.3s |
| 83-comment discussion | 80 / 83 (96%) | ~1,905 | **79%** | ~1.4s |
| 109-comment deep thread | 104 / 109 (95%) | ~3,050 | **79%** | ~0.6s |
| 348-comment mega thread | 340 / 348 (98%) | ~8,100 | **77%** | ~3.2s |
| 1,092-comment mega thread | 805 / 1,092 (74%) | ~34,500 | **77%** | ~4.8s |

Most threads return 95%+ of comments. Mega threads (1,000+) hit diminishing returns because Reddit counts deleted/removed comments in the total but no longer serves them — the ~287 "missing" in the 1,092 thread are ghosts.

## Updates

Lurk checks for new versions once every 24 hours (background, non-blocking, 3-second timeout). If a newer release exists, you'll see a one-line notice after your command finishes.

```bash
lurk update              # Download and install latest
lurk update --check      # Check only, don't install
```

If you installed via npm, brew, or `go install`, `lurk update` will detect that and tell you to use your package manager instead.

### Disable Update Checks

```bash
# Option 1: environment variable
export LURK_NO_UPDATE_CHECK=1

# Option 2: config file
mkdir -p ~/.config/lurk && echo "disabled" > ~/.config/lurk/no-update-check
```

## Error Handling

Lurk fails cleanly with a reason, never with raw HTTP dumps or stack traces:

| Scenario | Error message |
|----------|---------------|
| Deleted/nonexistent thread | `not found — check the URL or subreddit name` |
| Private/quarantined subreddit | `access denied — subreddit may be private or quarantined` |
| Malformed URL | `not a valid thread URL — expected reddit.com/r/sub/comments/id/title` |
| Reddit is down | `Reddit server error (HTTP 5xx) — Reddit may be down` |
| Rate limited | `rate limited — too many requests, try again shortly` |

## Limitations

- **Public content only** (without OAuth). Private and quarantined subreddits require `lurk auth`.
- **Deleted comments are ghosts.** Reddit counts them in the total but no longer serves content. A "1,092-comment" thread may only have ~805 live comments.
- **No media download.** Media URLs (images, video, galleries) are extracted as clickable links — not downloaded or embedded.

---

## Reference

Details for the curious.

### CLI Commands

```bash
lurk thread "https://reddit.com/r/ClaudeAI/comments/..."   # Full thread + comments
lurk subreddit ClaudeAI --sort top --time week --limit 10   # Browse
lurk search "prompt engineering" --sub ClaudeAI --limit 5   # Search
lurk search "ZFS" --sub selfhosted,homelab,datahoarder      # Multi-sub search
lurk user spez --limit 5                                    # User activity
lurk subreddit ClaudeAI --info                              # Subreddit metadata
lurk auth                                                   # OAuth setup
lurk update                                                 # Self-update
```

### Flags

| Flag | What it does | Works with |
|------|-------------|------------|
| `--sort` | hot, new, top, rising, controversial, relevance, comments | subreddit, search |
| `--limit` | Max results (default 25) | subreddit, search, user |
| `--time` | hour, day, week, month, year, all | subreddit, search |
| `--sub` | Restrict search to subreddit(s) — comma-separated for multi-sub | search |
| `--after` | Pagination token for next page | subreddit, search (single-sub only) |
| `--info` | Subreddit metadata instead of posts | subreddit |
| `--json` | Raw JSON output | all |
| `--compact` | Compact notation (default in MCP mode) | all |
| `--no-cache` | Skip cache | all |

### MCP Tools

| Tool | Purpose |
|------|---------|
| `lurk` | Read threads, browse subreddits, search posts, view user activity |
| `lurk_info` | Get subreddit metadata (subscribers, active users, description) |

### Understanding Skill vs MCP

Both modes use the same compact notation, so per-call token cost is identical. The differences:

**Context overhead.** Every message you send, Claude also receives hidden tool definitions. Skill adds ~20 tokens. MCP adds ~438 tokens. On subscription plans this is cached and free. On the API, you pay for it every message.

**Caching.** MCP runs as a background server. Its adaptive in-memory cache means hitting the same thread or subreddit twice is instant. Skill starts a fresh process each call — no cross-call cache.

**Permissions.** Skill works through Bash, so Claude needs shell permission. MCP is a native tool call. If you run with Bash restricted, MCP works without it.

### Compact Notation

Tab-delimited output designed for LLMs. `d0/d1/d2` = comment depth. Score before author. `+N` = collapsed comments not loaded. `#next` = pagination token. `#warning` = smart limit triggered.

```text
#post   r/ClaudeAI   u/BusyBea2   1pts   57%   9cmt   2026-02-23
Email and Claude
Have you figured out how to use Claude to manage your inbox?

#comments   3
d0   6   Ok-Version-8996   I'm surprised gmail hasn't done this already
d1   3   BusyBea2   i hear you, that's one of my first clean up things
d0   2   turtle-toaster   Claude Settings lets you connect your Gmail
```

### Adaptive Cache

| Content | TTL | Rationale |
|---------|-----|-----------|
| `/new` feeds | 2 min | Fresh content, stale quickly |
| `/hot` feeds | 5 min | Changes moderately |
| Threads & comments | 10 min | Stable once posted |
| Search results | 10 min | Results shift slowly |
| User profiles | 15 min | Rarely changes |
| `/top` feeds | 30 min | Rankings are stable |

50MB LRU cap with automatic eviction. OAuth-authenticated requests use `oauth.reddit.com` automatically.

### Under the Hood

- Appends `.json` to any Reddit URL — no API keys needed for public content
- Recursively walks comment trees to arbitrary depth
- Fetches `/api/morechildren` to expand collapsed threads (batched, max 100 IDs per request)
- Unauthenticated: 10 req/min with burst allowance (100 tokens / 10-min window)
- Authenticated: 60 req/min (OAuth client_credentials grant)
- Retries 429/5xx with exponential backoff (3 attempts)
- Resolves redd.it short links via HTTP redirect
- Single static Go binary, cross-compiled for linux/darwin/windows amd64/arm64

### Build from Source

```bash
make build                # Local binary
make all                  # Cross-compile all platforms
make install-skill        # Install to ~/.claude/skills/reddit/
```

### Uninstall

```bash
rm -rf ~/.claude/skills/reddit                    # Remove skill
lurk auth --clear                                 # Remove saved credentials
# For MCP: edit ~/.claude.json, delete "lurk" from mcpServers
```

## License

MIT
