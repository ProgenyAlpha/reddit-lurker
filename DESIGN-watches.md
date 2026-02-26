# Lurk Watch System — Design Document

## Overview

MCP-native watch system that tracks Reddit threads, subreddits, and searches over time. Returns only deltas (new comments, new posts, state changes) instead of full re-fetches. Differentiator: reddit-mcp-buddy is stateless per-request. Lurk gives awareness over time.

## MCP Tools

### `lurk_watch` — Manage watches

| Param | Type | Description |
|-------|------|-------------|
| `action` | string | `create`, `list`, `remove`, `clear` |
| `type` | string | `thread`, `sub`, `search` (required for create) |
| `target` | string | URL, subreddit name, or search query |
| `label` | string | Human-readable description (Claude sets this) |
| `keywords` | string | Comma-separated keyword filter (sub/search only) |
| `id` | string | Watch ID (for remove) |

### `lurk_check` — Poll all watches for changes

| Param | Type | Description |
|-------|------|-------------|
| `id` | string | Optional — check single watch. Omit for all. |

Returns compact diff output showing only what changed since last check.

## Watch Types

### Thread Watch
- Tracks: new comments, score delta, flair changes, locked/removed state
- Storage: set of seen comment IDs + last known post state (score, flair, locked)
- On check: fetch thread shallow, diff comment IDs, report new ones
- Auto-complete: when thread is locked or archived

### Sub Watch
- Tracks: new posts, optionally filtered by keywords
- Storage: last seen post ID (newest at time of watch creation)
- On check: fetch sub /new, return posts newer than stored ID
- Noise control: warn if sub has >50K subscribers and no keyword filter

### Search Watch
- Tracks: new results matching a saved query
- Storage: set of seen post IDs
- On check: re-run search, return posts not in seen set
- Useful for tracking topics across all of Reddit

## Storage

File: `~/.config/lurk/watches.json`

```json
{
  "watches": [
    {
      "id": "w_abc123",
      "type": "thread",
      "target": "/r/ClaudeAI/comments/abc/title/",
      "label": "Claude 4 announcement thread",
      "created": "2026-02-25T12:00:00Z",
      "last_checked": "2026-02-25T14:30:00Z",
      "state": {
        "seen_comment_ids": ["id1", "id2", "id3"],
        "post_score": 1234,
        "post_flair": "Official",
        "post_locked": false,
        "comment_count": 342
      }
    },
    {
      "id": "w_def456",
      "type": "sub",
      "target": "LocalLLaMA",
      "label": "quantization posts",
      "keywords": ["quantization", "GGUF", "Q4_K"],
      "created": "2026-02-25T12:00:00Z",
      "last_checked": "2026-02-25T14:30:00Z",
      "state": {
        "last_post_id": "t3_xyz789"
      }
    }
  ]
}
```

## Diff Output Format

```
#watch thread "Claude 4 announcement" — 3 new comments, score 1234→1567
d0	42	newuser123	This is a new reply to the top comment
d1	8	anotheruser	Responding to the new reply
d0	15	someone_else	Late to the party but...

#watch sub "LocalLLaMA quantization" — 2 new posts
1	156	12cmt	2h	u/researcher	"GGUF Q4_K_M vs Q5_K_S benchmarks"	/r/LocalLLaMA/comments/...
2	43	3cmt	45m	u/hobbyist	"Quantizing Llama 4 with llama.cpp"	/r/LocalLLaMA/comments/...

#watch search "rust async" — 1 new result
1	89	5cmt	1h	u/dev	"Async Rust patterns for MCP servers"	/r/rust/comments/...
```

## Smart Behaviors

### Noise Control
- Sub watches on subs with >50K subscribers: warn and suggest keyword filter
- Default keyword matching: case-insensitive title + selftext substring
- Max 20 active watches (prevent abuse)

### Auto-cleanup
- Watches expire after 7 days with no checks (configurable)
- Thread watches auto-complete when locked/archived
- Expired watches reported once then removed

### Natural Language Flow
```
User: "Keep an eye on this thread"
→ lurk_watch(action=create, type=thread, target=url, label="...")

User: "Watch r/LocalLLaMA for posts about quantization"
→ lurk_watch(action=create, type=sub, target=LocalLLaMA, keywords="quantization", label="...")

User: "Anything new on Reddit?"
→ lurk_check()

User: "Stop watching that thread"
→ lurk_watch(action=remove, id="w_abc123")

User: "What am I watching?"
→ lurk_watch(action=list)
```

## Implementation Plan

### Files to create
- `reddit/watch.go` — Watch types, storage, diff logic
- `cmd/watch.go` — MCP tool handlers for lurk_watch and lurk_check
- `format/watch.go` — Compact diff formatting

### Files to modify
- `cmd/serve.go` — Register lurk_watch and lurk_check tools
- `main.go` — No CLI command needed (MCP-only feature)

### Estimated scope
- ~400-500 lines new code
- 3 new files, 1 modified file
- No new dependencies

## Edge Cases

| Case | Behavior |
|------|----------|
| Thread deleted between checks | Report "thread deleted/removed" and auto-complete watch |
| Sub goes private | Report error, keep watch active (may come back) |
| 1000+ new comments since last check | Cap at newest 50, note "and ~950 more" |
| Watch target is invalid URL | Validate on create, return error immediately |
| Concurrent lurk_check calls | File lock on watches.json, or in-memory mutex |
| MCP server restart | Reload watches from disk on startup |
| No changes found | Return "#watches checked — nothing new" |
| All watches expired | Return "no active watches" |
| Keyword match in URL but not content | Match title + selftext only, not metadata |
