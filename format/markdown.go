package format

import (
	"fmt"
	"strings"
	"time"

	"github.com/progenyalpha/lurk/reddit"
)

// FormatThread renders a full post with its comment tree in readable markdown.
func FormatThread(t *reddit.Thread) string {
	if t == nil || t.Post == nil {
		return "No data."
	}

	var b strings.Builder
	p := t.Post

	// Header
	fmt.Fprintf(&b, "# %s\n", p.Title)
	fmt.Fprintf(&b, "**r/%s** | u/%s | %d pts (%.0f%% upvoted) | %d comments | %s\n\n",
		p.Subreddit, p.Author, p.Score, p.UpvoteRatio*100,
		p.NumComments, formatTime(p.Created))

	// Body
	if p.IsSelf {
		if p.SelfText != "" {
			b.WriteString(p.SelfText)
			b.WriteString("\n\n")
		}
	} else {
		url := p.MediaURL
		if url == "" {
			url = p.URL
		}
		if url != "" {
			fmt.Fprintf(&b, "%s\n\n", url)
		}
	}

	// Comments
	if len(t.Comments) > 0 {
		fmt.Fprintf(&b, "## Comments (%d loaded)\n\n", countComments(t.Comments))
		b.WriteString(formatComments(t.Comments, 0))
	}

	return b.String()
}

// FormatPostList renders a numbered list of posts for subreddit listings.
func FormatPostList(posts []*reddit.Post, header string) string {
	if len(posts) == 0 {
		return "No posts found."
	}

	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", header)

	for i, p := range posts {
		if p == nil {
			continue
		}
		fmt.Fprintf(&b, "%d. **%s** — %d pts | %d comments | u/%s | %s\n",
			i+1, p.Title, p.Score, p.NumComments, p.Author, formatTime(p.Created))
		fmt.Fprintf(&b, "   %s\n\n", p.Permalink)
	}

	return b.String()
}

// FormatSearchResults renders search results with subreddit names.
func FormatSearchResults(posts []*reddit.Post, query string) string {
	if len(posts) == 0 {
		return "No results found."
	}

	var b strings.Builder
	fmt.Fprintf(&b, "# Search: \"%s\" (%d results)\n\n", query, len(posts))

	for i, p := range posts {
		if p == nil {
			continue
		}
		fmt.Fprintf(&b, "%d. **%s** — r/%s | %d pts | %d comments | u/%s | %s\n",
			i+1, p.Title, p.Subreddit, p.Score, p.NumComments, p.Author, formatTime(p.Created))
		fmt.Fprintf(&b, "   %s\n\n", p.Permalink)
	}

	return b.String()
}

// FormatUser renders a user profile with recent activity.
func FormatUser(info *reddit.UserInfo, posts []*reddit.Post, comments []*reddit.Comment) string {
	if info == nil {
		return "No data."
	}

	var b strings.Builder

	fmt.Fprintf(&b, "# u/%s\n", info.Name)
	fmt.Fprintf(&b, "**Karma:** %d total (%d link, %d comment) | **Joined:** %s\n\n",
		info.TotalKarma, info.LinkKarma, info.CommentKarma, formatTime(info.Created))

	if info.IsSuspended {
		b.WriteString("*Account suspended.*\n\n")
	}

	if len(posts) > 0 {
		fmt.Fprintf(&b, "## Recent Posts (%d)\n\n", len(posts))
		for i, p := range posts {
			if p == nil {
				continue
			}
			fmt.Fprintf(&b, "%d. **%s** — r/%s | %d pts | %s\n",
				i+1, p.Title, p.Subreddit, p.Score, formatTime(p.Created))
		}
		b.WriteString("\n")
	}

	if len(comments) > 0 {
		fmt.Fprintf(&b, "## Recent Comments (%d)\n\n", len(comments))
		for i, c := range comments {
			if c == nil {
				continue
			}
			body := c.Body
			if len(body) > 200 {
				body = body[:200] + "..."
			}
			fmt.Fprintf(&b, "%d. **u/%s** (%d pts) — %s\n   %s\n\n",
				i+1, c.Author, c.Score, formatTime(c.Created), body)
		}
	}

	return b.String()
}

// FormatSubredditInfo renders subreddit metadata.
func FormatSubredditInfo(info *reddit.SubredditInfo) string {
	if info == nil {
		return "No data."
	}

	var b strings.Builder

	fmt.Fprintf(&b, "# r/%s\n", info.Name)
	if info.Title != "" {
		fmt.Fprintf(&b, "**%s**\n\n", info.Title)
	}
	fmt.Fprintf(&b, "**Subscribers:** %d | **Active:** %d | **Created:** %s | **Type:** %s\n\n",
		info.Subscribers, info.ActiveUsers, formatTime(info.Created), info.SubredditType)

	if info.Over18 {
		b.WriteString("*NSFW*\n\n")
	}

	if info.Description != "" {
		b.WriteString(info.Description)
		b.WriteString("\n")
	}

	return b.String()
}

// formatTime returns a date in YYYY-MM-DD format.
func formatTime(t time.Time) string {
	if t.IsZero() {
		return "unknown"
	}
	return t.Format("2006-01-02")
}

// formatComments recursively renders a comment tree with indentation.
func formatComments(comments []*reddit.Comment, depth int) string {
	var b strings.Builder
	indent := strings.Repeat("  ", depth)

	for _, c := range comments {
		if c == nil {
			continue
		}

		if c.IsMore {
			fmt.Fprintf(&b, "%s*%d more replies...*\n\n", indent, c.MoreCount)
			continue
		}

		fmt.Fprintf(&b, "%s**u/%s** (%d pts) — %s\n", indent, c.Author, c.Score, formatTime(c.Created))
		// Indent each line of the body
		for _, line := range strings.Split(c.Body, "\n") {
			fmt.Fprintf(&b, "%s%s\n", indent, line)
		}
		b.WriteString("\n")

		if len(c.Replies) > 0 {
			b.WriteString(formatComments(c.Replies, depth+1))
		}
	}

	return b.String()
}

// countComments counts all comments in a tree (excluding "more" placeholders).
func countComments(comments []*reddit.Comment) int {
	n := 0
	for _, c := range comments {
		if c == nil || c.IsMore {
			continue
		}
		n++
		n += countComments(c.Replies)
	}
	return n
}
