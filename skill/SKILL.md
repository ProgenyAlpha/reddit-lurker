# Lurk — Reddit Reader

Read Reddit threads, browse subreddits, search posts, and check user profiles. Full comment trees, no auth needed.

## When to Use

- User pastes a Reddit URL
- User asks about a Reddit thread, subreddit, or user
- User wants to search Reddit for information
- User asks "what's on r/..." or "check Reddit for..."

## Commands

```bash
# Read a full thread with all comments
lurk thread "<reddit_url>"

# Browse a subreddit
lurk subreddit <name> --sort hot --limit 25

# Search Reddit (optionally within a subreddit)
lurk search "<query>" --sub <subreddit> --sort relevance --limit 25

# View user activity
lurk user <username> --limit 10
```

## Sort Options

- Subreddit: hot, new, top, rising, controversial
- Search: relevance, hot, top, new, comments
- Time filter (for top/controversial): hour, day, week, month, year, all

## Output

Default output is human-readable markdown. Add `--json` for raw JSON or `--compact` for AI-optimized compact notation.

## Examples

```bash
lurk thread "https://www.reddit.com/r/ClaudeAI/comments/abc123/post_title/"
lurk subreddit ClaudeAI --sort top --time week --limit 10
lurk search "MCP server setup" --sub ClaudeAI
lurk user spez --limit 5
```
