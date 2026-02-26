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

> Full Reddit threads for Claude. No keys. No OAuth. No missing replies.

Reddit killed self-serve API keys in November 2025. Now if you want an LLM to analyze a Reddit thread, your options are:

- Copy/paste (destroys structure)
- Screenshot (loses half the replies)
- Print to PDF (good luck)
- Use an existing Reddit MCP that silently ignores "+47 more replies"

The best parts of Reddit are buried 4-5 replies deep. The correction. The real answer. The "actually you're wrong and here's why" that saves you hours. Most tools never fetch them.

**Reddit Lurker does.**

It expands every collapsed reply. Resolves every `kind: more`. Reconstructs the full comment tree. Paste a Reddit URL and Claude gets the entire conversation. Works with any LLM that can consume structured text.

```
Post: "I gave Claude the one thing it was missing: memory"
  164 comments fetched
  Max depth: 5
  Collapsed branches expanded: all
  Authentication required: none
```

```
Post
 +-- Comment (74 pts)
 |   +-- Reply (26 pts)
 |   |   +-- Reply (2 pts)
 |   +-- Reply (6 pts)
 |       +-- Reply (13 pts)        <-- most tools stop here
 |           +-- Reply (21 pts)
 |               +-- Reply (7 pts)
 |                   +-- Reply (1 pt)   <-- Lurker gets this
 +-- Comment (16 pts)
     +-- Reply (20 pts)
         +-- Reply (8 pts)
             +-- Reply (1 pt)
```

Most tools stop at depth 1 or 2. Lurker reconstructs the entire tree.

## What You Get

- Full comment trees at any depth
- Collapsed threads automatically expanded
- Cross-posts traced back to the original
- Galleries, video, media URLs extracted
- 15-minute cache (same thread twice = instant)
- 74-79% fewer tokens than raw JSON output
- Single Go binary, zero dependencies
- Read-only by design

## Why Not Use Reddit's Official API?

- Requires approval under the Responsible Builder Policy (self-serve keys no longer available)
- Adds OAuth complexity and token management for what should be a read-only operation
- Keys can expire mid-workflow

Reddit still serves full JSON on every public page. Lurker uses that. No signup, no approval wait, no tokens to rotate. Respects Reddit's unauthenticated rate limits (100 req/10min burst window) with automatic retry and backoff.

**Note:** Lurker only works with public subreddits and posts. Private and restricted subreddits require authentication, which Lurker does not support at this time.

## Install

### Zero-Dependency Install

```bash
curl -fsSL https://raw.githubusercontent.com/ProgenyAlpha/reddit-lurker/master/install.sh | bash
```

No Node. No Go. No nothing. Downloads the binary for your platform and walks you through editor setup. Supports Claude Code, Cursor, Windsurf, VS Code, Cline, and Zed. Works on Linux and macOS.

### Windows Install

```powershell
irm https://raw.githubusercontent.com/ProgenyAlpha/reddit-lurker/master/install.ps1 | iex
```

Same interactive setup, installs to `%LOCALAPPDATA%\lurk` and adds it to your PATH.

### Node Install

```bash
npx reddit-lurker
```

If you already have Node/npm. Same result, different delivery truck.

### Go Install

```bash
go install github.com/ProgenyAlpha/reddit-lurker@latest
```

Builds from source. Requires [Go 1.24+](https://go.dev/dl/). Run `./install.sh` afterward for skill/MCP configuration.

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
| VS Code | `~/.config/Code/User/mcp.json` (Linux) / `~/Library/.../Code/User/mcp.json` (macOS) | `servers` |
| Cline | VS Code globalStorage (auto-detected) | `mcpServers` |
| Zed | `~/.config/zed/settings.json` | `context_servers` |

Claude Code also supports a **Skill** mode (~20 tokens overhead vs ~438 for MCP). The installer will ask which you prefer.

### Manual Configuration

If you installed the binary yourself, add lurk to your editor's MCP config:

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

## Updates

Lurk checks for new versions once every 24 hours (background, non-blocking, 3-second timeout). If a newer release exists, you'll see a one-line notice after your command finishes.

```bash
lurk update              # Download and install latest
lurk update --check      # Check only, don't install
```

If you installed via npm, brew, or `go install`, `lurk update` will detect that and tell you to use your package manager instead.

### Disable Update Checks

If you don't want any phone-home behavior:

```bash
# Option 1: environment variable
export LURK_NO_UPDATE_CHECK=1

# Option 2: config file
mkdir -p ~/.config/lurk && echo "disabled" > ~/.config/lurk/no-update-check
```

This only disables the background check. `lurk update` still works manually.

## Usage

Just talk to Claude naturally:

- *"Read this thread"* + paste a Reddit URL
- *"What's trending on r/ClaudeAI?"*
- *"What's r/selfhosted arguing about today?"*
- *"What has u/spez been up to?"*

Claude handles the rest. No commands to memorize.

## Real Example

Here's what Lurker actually outputs for a [109-comment r/LocalLLM thread](https://www.reddit.com/r/LocalLLM/comments/1qp880l/finally_we_have_the_best_agentic_ai_at_home/) about running Kimi K2.5 at home.

**What most tools give Claude:**
```
# Finally We have the best agentic AI at home
u/moks4tda | 422 pts | r/LocalLLM

u/Recent-Success-1520 (180 pts)
  If you can host Kimi 2.5 1T+ model at home then it tells
  me you have a really big home

u/No_Conversation9561 (82 pts)
  not in my home

u/rookan (60 pts)
  yeah, my 16GB VRAM card can easily handle it /s

... 12 top-level comments, no replies
```

**What Lurker gives Claude (compact notation):**
```
#post	r/LocalLLM	u/moks4tda	422pts	93%	109cmt	2026-01-28
Finally We have the best agentic AI at home

#comments	104
d0	180	Recent-Success-1520	If you can host Kimi 2.5 1T+ model at home...
d1	46	HenkPoley	Apparently it's a native 4 bit weights. So "only" 640 GB needed...
d2	34	TechnicalGeologist99	Sorry...you're going to run that model on RAM?
d3	29	HenkPoley	24 tokens per second on 2x 512GB Max Studio M3 Ultra
d4	8	doradus_novae	See you tomorrow when it answers your question 😆
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

**104 of 109 comments loaded. 10 levels deep.** The 5 missing comments are deleted/removed posts that Reddit still counts but no longer serves content for. Lurker fetched everything Reddit was willing to return.

The best stuff — hardware specs, cost breakdowns, the debate about whether 24 tok/s is actually useful for agentic workflows — is all buried at depth 3-9. Most tools never see it.

**Search works too.** Here's `lurk search "reddit MCP" --sub ClaudeAI --limit 5`:

```
#search	"reddit MCP"	ClaudeAI	5
1	237pts	58cmt	r/ClaudeAI	u/karanb192	Reddit MCP just hit the Anthropic Directory
2	2228pts	311cmt	r/ClaudeAI	u/JokeGold5455	Claude Code is a Beast – Tips from 6 Months of Hardcore Use
3	67pts	23cmt	r/ClaudeAI	u/karanb192	Built an MCP server for Claude Desktop to browse Reddit in real-time
4	0pts	7cmt	r/ClaudeAI	u/New-Requirement-3742	Open Sourcing my Reddit MCP Server (TypeScript + Apify)
5	1pts	1cmt	r/ClaudeAI	u/hurrah-dev	I built an MCP server for the Reddit Ads API
```

Result #1 is the most popular Reddit MCP on the Anthropic Directory. In its comments, a user asks:

> *"I see that it only extracts a few top level comments right? ... when I need summarization of comments I need all of them, not just a few top level."*

The developer's response: *"You can instruct the LLM to fetch all comments. It'll then go up to 100 comments."*

Lurker pulled 104 comments from that thread on the first call — no "fetch more" instruction needed, no second request. That's already past their tool's ceiling, with full depth to boot.

## Benchmarks

Real numbers from live Reddit threads (February 2025):

| Thread | Comments fetched | Compact | Savings vs JSON | Fetch time |
|--------|-----------------|---------|-----------------|------------|
| 25-comment announcement | 25 / 25 (100%) | ~786 tokens | **74%** | ~0.3s |
| 83-comment discussion | 80 / 83 (96%) | ~1,905 tokens | **79%** | ~1.4s |
| 109-comment deep thread | 104 / 109 (95%) | ~3,050 tokens | **79%** | ~0.6s |
| 348-comment mega thread | 333 / 348 (95%) | ~7,889 tokens | **77%** | ~2.9s |
| 1,092-comment mega thread | 461 / 1,092 (42%) | ~19,709 tokens | **77%** | ~28s |

Compact notation saves 74-79% of tokens compared to raw JSON. Most threads return 95%+ of comments. Threads over ~500 comments hit Reddit's unauthenticated response cap — Lurker still gets ~460 comments with full depth, which is more than any other unauthenticated tool manages. Reddit allows burst requests averaged over a 10-minute window, so even large threads with multiple expansion passes stay within limits.

## Error Handling

Lurker fails cleanly with a reason, never with raw HTTP dumps or Go stack traces:

| Scenario | Error message |
|----------|---------------|
| Deleted/nonexistent thread | `not found — check the URL or subreddit name` |
| Private/quarantined subreddit | `access denied — subreddit may be private (requires auth) or quarantined` |
| Malformed URL | `not a valid thread URL — expected a link like reddit.com/r/sub/comments/id/title` |
| Reddit is down | `Reddit server error (HTTP 5xx) — Reddit may be down` |
| No internet | `network error — check your internet connection` |
| Rate limited | `rate limited — too many requests, try again shortly` |

Deleted comments show as `[deleted]` in the output — Reddit still counts them in the comment total but no longer serves content for them.

## Limitations

- **Public content only.** Private, restricted, and quarantined subreddits require OAuth, which Lurker doesn't support.
- **Unauthenticated rate limit.** 100 requests per 10 minutes (Reddit allows bursting). Large threads with many expansion batches may take 20-30 seconds. Normal threads return in under 3 seconds.
- **~500 comment cap on mega threads.** Reddit limits initial responses to 500 comments. Lurker then expands collapsed branches recursively, reaching 95%+ on threads under 500 comments. Threads over 1,000 cap around ~460 — still far more than any unauthenticated tool, but not the complete set.
- **No NSFW age gates.** Some NSFW subreddits gate content behind login even for public posts.

---

## Reference

Everything below is for people who want the details. You don't need any of this to use Reddit Lurker.

### Commands

Claude runs these automatically, but you can also run them directly:

```bash
lurk thread "https://reddit.com/r/ClaudeAI/comments/..."   # Full thread + comments
lurk subreddit ClaudeAI --sort top --time week --limit 10   # Browse
lurk search "prompt engineering" --sub ClaudeAI --limit 5   # Search
lurk user spez --limit 5                                    # User activity
lurk subreddit ClaudeAI --info                              # Subreddit metadata
```

### Flags

| Flag | What it does | Works with |
|------|-------------|------------|
| `--sort` | hot, new, top, rising, controversial, relevance, comments | subreddit, search |
| `--limit` | Max results (default 25) | subreddit, search, user |
| `--time` | hour, day, week, month, year, all | subreddit, search |
| `--sub` | Restrict search to one subreddit | search |
| `--after` | Pagination token for next page | subreddit, search |
| `--info` | Subreddit metadata instead of posts | subreddit |
| `--json` | Raw JSON output | all |
| `--compact` | Compact notation (default in MCP mode) | all |
| `--no-cache` | Skip the 15-minute cache | all |

### Understanding Skill vs MCP

Both modes use the same compact notation for output, so per-call token cost is identical. The real differences:

**Context overhead.** Every message you send, Claude also receives hidden instructions you don't see: tool definitions, context, rules. Skill adds ~20 tokens to this. MCP adds ~438 tokens. On subscription plans this is cached and free. On the API, you pay for it every message.

**Caching.** MCP runs as a background server that stays alive between calls. Its 15-minute in-memory cache means hitting the same thread or subreddit twice is instant, no Reddit request. Skill starts a fresh process each call, so there's no cache between calls. You can adjust the TTL in `reddit/client.go` if you want it longer or shorter.

**Permissions.** Skill works through Bash, so Claude needs permission to run shell commands. MCP is a native tool call that doesn't touch the shell. If you run Claude Code with Bash restricted or prefer fewer permission prompts, MCP works without it.

### Compact Notation

Both Skill and MCP use compact tab-delimited output designed for LLMs. Same data as markdown, ~42% fewer tokens (and ~77% fewer than raw JSON). Here's what Claude sees:

**Standard markdown (for comparison):**
```markdown
# Email and Claude
**r/ClaudeAI** | u/BusyBea2 | 1 pts (57% upvoted) | 9 comments

Have you figured out how to use Claude to manage your inbox?

## Comments (3 loaded)

**u/Ok-Version-8996** (6 pts)
  I'm surprised gmail hasn't done this already

  **u/BusyBea2** (3 pts)
    i hear you, that's one of my first clean up things

**u/turtle-toaster** (2 pts)
  Claude Settings lets you connect your Gmail
```
~180 tokens

**Compact (what Claude actually gets):**
```
#post	r/ClaudeAI	u/BusyBea2	1pts	57%	9cmt	2026-02-23
Email and Claude
Have you figured out how to use Claude to manage your inbox?

#comments	3
d0	6	Ok-Version-8996	I'm surprised gmail hasn't done this already
d1	3	BusyBea2	i hear you, that's one of my first clean up things
d0	2	turtle-toaster	Claude Settings lets you connect your Gmail
```
~105 tokens

Same thread, same structure, **42% fewer tokens than markdown, 77% fewer than JSON.** `d0/d1/d2` = comment depth. Score before author. `+N` = collapsed comments. `#next` = pagination token.

### Under the Hood

- Appends `.json` to any Reddit URL
- Recursively walks comment trees to arbitrary depth
- Fetches `/api/morechildren` to expand collapsed threads (batched, max 100 IDs)
- Rate limited to 100 req/10min (Reddit's unauthenticated burst window)
- Retries 429/5xx with exponential backoff (3 attempts)
- In-memory cache with 15-minute TTL

### Build from Source

```bash
make build                # Local binary
make all                  # Cross-compile linux/darwin amd64/arm64
make install-skill        # Install to ~/.claude/skills/reddit/
```

### Uninstall

```bash
rm -rf ~/.claude/skills/reddit                    # Remove skill
# For MCP: edit ~/.claude.json, delete "lurk" from mcpServers
```

## Project Status

Actively maintained. Stable JSON parsing approach with minimal external dependencies. Read-only by design.

## License

MIT
