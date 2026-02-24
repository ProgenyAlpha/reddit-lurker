# Lurk — Reddit Reader

Read Reddit threads, browse subreddits, search posts, and check user profiles. Full comment trees with automatic expansion of collapsed threads. No auth needed.

## When to Use

- User pastes a Reddit URL
- User asks about a Reddit thread, subreddit, or user
- User wants to search Reddit for information
- User asks "what's on r/..." or "check Reddit for..."

## Commands

```bash
# Read a full thread with all comments
~/.claude/skills/reddit/lurk thread "<reddit_url>" --compact

# Browse a subreddit
~/.claude/skills/reddit/lurk subreddit <name> --sort hot --limit 25 --compact

# Search Reddit (optionally within a subreddit)
~/.claude/skills/reddit/lurk search "<query>" --sub <subreddit> --sort relevance --limit 25 --compact

# View user activity
~/.claude/skills/reddit/lurk user <username> --limit 10 --compact

# Subreddit info
~/.claude/skills/reddit/lurk subreddit <name> --info --compact
```

## Sort Options

- Subreddit: hot, new, top, rising, controversial
- Search: relevance, hot, top, new, comments
- Time filter (for top/controversial): hour, day, week, month, year, all

## Pagination

Results include a `--after` token when more pages are available.

## Output

Always use `--compact` for efficient token usage. Add `--json` for raw JSON if needed.
