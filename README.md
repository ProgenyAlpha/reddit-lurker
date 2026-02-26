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

> Full Reddit threads for Claude. No keys. No OAuth. No missing replies.

Reddit killed self-serve API keys in November 2025. Now if you want an LLM to analyze a Reddit thread, your options are:

- Copy/paste (destroys structure)
- Screenshot (loses half the replies)
- Print to PDF (good luck)
- Use an existing Reddit MCP that silently ignores "+47 more replies"

The best parts of Reddit are buried 4-5 replies deep. The correction. The real answer. The "actually you're wrong and here's why" that saves you hours. Most tools never fetch them.

**Reddit Lurker does.**

It expands every collapsed reply. Resolves every `kind: more`. Reconstructs the full comment tree. Paste a Reddit URL and Claude gets the entire conversation.

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
- ~42% fewer tokens than markdown output
- Single Go binary, zero dependencies
- Read-only by design

## Why Not Use Reddit's Official API?

- Requires approval under the Responsible Builder Policy (self-serve keys no longer available)
- Returns `"kind": "more"` stubs for deep replies that most clients never expand
- Adds OAuth complexity and token management
- Keys can expire mid-workflow

Reddit still serves full JSON on every public page. Lurker uses that. No signup, no approval wait, no tokens to rotate. Respects Reddit's unauthenticated rate limits (10 req/min) with automatic retry and backoff.

**Note:** Lurker only works with public subreddits and posts. Private and restricted subreddits require authentication, which Lurker deliberately does not support.

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

## The Difference

**Without Reddit Lurker:**
- 12 top-level comments
- No nested replies
- Missing half the conversation
- "kind: more" objects silently dropped

**With Reddit Lurker:**
- 137 comments, full depth
- Every collapsed thread expanded
- Complete conversational context
- Structure preserved for LLM analysis

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

Both Skill and MCP use compact tab-delimited output designed for LLMs. Same data as markdown, ~42% fewer tokens. Here's what Claude sees:

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
