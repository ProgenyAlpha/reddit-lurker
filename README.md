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

> *Because sometimes you just need to read the comments without anyone knowing you were there.*

A Reddit reader for [Claude Code](https://docs.anthropic.com/en/docs/claude-code). Paste a URL, ask about a subreddit, search for posts. Full comment trees, collapsed threads expanded, no auth required.

## Install

### Zero-Dependency Install

```bash
curl -fsSL https://raw.githubusercontent.com/ProgenyAlpha/reddit-lurker/master/install.sh | bash
```

No Node. No Go. No nothing. Downloads the binary for your platform and sets up Claude Code.

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

The installer asks one question: **Skill or MCP?**

| | Skill | MCP |
|---|---|---|
| **How it works** | Claude runs a shell command when needed | A background server gives Claude a native tool |
| **Context overhead** | ~20 tokens/message (lightweight) | ~438 tokens/message (heavier) |
| **Caching between calls** | None (fresh process each call) | Yes (server stays warm, 15-min cache) |
| **Requires Bash permission** | Yes | No |

**Pick Skill** if you want minimal overhead and don't mind Bash prompts. **Pick MCP** if you use Reddit often (cache helps) or run Claude Code with Bash restricted. You can always switch later.

## Usage

Just talk to Claude naturally:

- *"Read this thread"* + paste a Reddit URL
- *"What's trending on r/ClaudeAI?"*
- *"What's r/selfhosted arguing about today?"*
- *"What has u/spez been up to?"*

Claude handles the rest. No commands to memorize.

## Why This Exists

Reddit's official API requires OAuth credentials, and as of November 2025 self-service API key creation is no longer available. New developers need to apply through Reddit's Responsible Builder Policy - a process many find difficult to navigate.

Even with credentials, Reddit's API returns stub objects for collapsed comment threads (`"kind": "more"`). Expanding these requires extra API calls that most tools don't make, so you get a shallow slice of the conversation without realizing what's missing.

Lurk takes a different approach. Reddit serves full JSON for every public page without authentication - just append `.json` to any URL. No API keys, no approval process, no OAuth tokens to expire. Lurk does that:

- **Full comment trees** - every reply, every depth, properly nested
- **Collapsed threads expanded** - "+47 more replies"? Fetched automatically
- **Cross-posts** traced back to the original
- **Galleries, video, media** - URLs extracted
- **15-minute cache** - same thread twice? Instant
- **Retry + backoff** - rate limits handled silently

One Go binary. Zero authentication. Zero tracking.

---

## Reference

Everything below is for people who want the details. You don't need any of this to use lurk.

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

**Context overhead.** Every message you send, Claude also receives hidden instructions you don't see - tool definitions, context, rules. Skill adds ~20 tokens to this. MCP adds ~438 tokens. On subscription plans this is cached and free. On the API, you pay for it every message.

**Caching.** MCP runs as a background server that stays alive between calls. Its 15-minute in-memory cache means hitting the same thread or subreddit twice is instant - no Reddit request. Skill starts a fresh process each call, so there's no cache between calls. You can adjust the TTL in `reddit/client.go` if you want it longer or shorter.

**Permissions.** Skill works through Bash - Claude needs permission to run shell commands. MCP is a native tool call that doesn't touch the shell. If you run Claude Code with Bash restricted or prefer fewer permission prompts, MCP works without it.

### Compact Notation

Both Skill and MCP use compact tab-delimited output designed for LLMs - same data as markdown, ~42% fewer tokens. Here's what Claude sees:

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

Same thread, same structure, **42% fewer tokens.** `d0/d1/d2` = comment depth. Score before author. `+N` = collapsed comments. `#next` = pagination token.

### Under the Hood

- Appends `.json` to any Reddit URL
- Recursively walks comment trees to arbitrary depth
- Fetches `/api/morechildren` to expand collapsed threads (batched, max 100 IDs)
- Rate limited to 10 req/min (Reddit's unauthenticated limit)
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

## License

MIT

