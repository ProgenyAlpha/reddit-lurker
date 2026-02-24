package format

import (
	"fmt"
	"strings"

	"github.com/progenyalpha/lurk/reddit"
)

// CompactThread renders a thread in tab-delimited compact notation.
func CompactThread(t *reddit.Thread) string {
	if t == nil || t.Post == nil {
		return ""
	}

	var b strings.Builder
	p := t.Post

	// Post header
	fmt.Fprintf(&b, "#post\tr/%s\tu/%s\t%dpts\t%.0f%%\t%dcmt\t%s\n",
		p.Subreddit, p.Author, p.Score, p.UpvoteRatio*100,
		p.NumComments, formatTime(p.Created))
	b.WriteString(p.Title)
	b.WriteByte('\n')

	// Body
	if p.IsSelf {
		b.WriteString(truncate(p.SelfText, 500))
	} else {
		url := p.MediaURL
		if url == "" {
			url = p.URL
		}
		b.WriteString(truncate(url, 500))
	}
	b.WriteByte('\n')

	// Comments
	if len(t.Comments) > 0 {
		fmt.Fprintf(&b, "\n#comments\t%d\n", countComments(t.Comments))
		b.WriteString(compactComments(t.Comments))
	}

	return b.String()
}

// CompactPostList renders a subreddit listing in compact notation.
func CompactPostList(posts []*reddit.Post, subName string, sort string) string {
	if len(posts) == 0 {
		return ""
	}

	var b strings.Builder
	fmt.Fprintf(&b, "#sub\tr/%s\t%s\t%d\n", subName, sort, len(posts))

	for i, p := range posts {
		if p == nil {
			continue
		}
		fmt.Fprintf(&b, "%d\t%dpts\t%dcmt\t%s\tu/%s\t%s\t%s\n",
			i+1, p.Score, p.NumComments, formatTime(p.Created),
			p.Author, truncate(p.Title, 80), p.Permalink)
	}

	return b.String()
}

// CompactSearchResults renders search results in compact notation.
func CompactSearchResults(posts []*reddit.Post, query string, sub string) string {
	if len(posts) == 0 {
		return ""
	}

	scope := sub
	if scope == "" {
		scope = "all"
	}

	var b strings.Builder
	fmt.Fprintf(&b, "#search\t\"%s\"\t%s\t%d\n", query, scope, len(posts))

	for i, p := range posts {
		if p == nil {
			continue
		}
		fmt.Fprintf(&b, "%d\t%dpts\t%dcmt\tr/%s\tu/%s\t%s\t%s\n",
			i+1, p.Score, p.NumComments, p.Subreddit,
			p.Author, truncate(p.Title, 80), p.Permalink)
	}

	return b.String()
}

// CompactUser renders user info and activity in compact notation.
func CompactUser(info *reddit.UserInfo, posts []*reddit.Post, comments []*reddit.Comment) string {
	if info == nil {
		return ""
	}

	var b strings.Builder
	fmt.Fprintf(&b, "#user\t%s\t%dkarma\t%s\n",
		info.Name, info.TotalKarma, formatTime(info.Created))

	if len(posts) > 0 {
		fmt.Fprintf(&b, "#posts\t%d\n", len(posts))
		for i, p := range posts {
			if p == nil {
				continue
			}
			fmt.Fprintf(&b, "%d\t%dpts\tr/%s\t%s\n",
				i+1, p.Score, p.Subreddit, truncate(p.Title, 80))
		}
	}

	if len(comments) > 0 {
		fmt.Fprintf(&b, "#comments\t%d\n", len(comments))
		for i, c := range comments {
			if c == nil {
				continue
			}
			fmt.Fprintf(&b, "%d\t%dpts\tr/%s\t%s\n",
				i+1, c.Score, "", truncate(c.Body, 100))
		}
	}

	return b.String()
}

// CompactSubredditInfo renders subreddit metadata in compact notation.
func CompactSubredditInfo(info *reddit.SubredditInfo) string {
	if info == nil {
		return ""
	}

	var b strings.Builder
	fmt.Fprintf(&b, "#subreddit\tr/%s\t%dsubs\t%dactive\t%s\t%s\n",
		info.Name, info.Subscribers, info.ActiveUsers,
		info.SubredditType, formatTime(info.Created))

	if info.Title != "" {
		b.WriteString(info.Title)
		b.WriteByte('\n')
	}

	if info.Description != "" {
		b.WriteString(truncate(info.Description, 500))
		b.WriteByte('\n')
	}

	if info.Over18 {
		b.WriteString("nsfw\n")
	}

	return b.String()
}

// truncate cuts a string to max length, appending "..." if truncated.
// Newlines are replaced with spaces for single-line output.
func truncate(s string, max int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.TrimSpace(s)
	if len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}

// compactComments renders a flat list of comments with depth prefixes.
func compactComments(comments []*reddit.Comment) string {
	var b strings.Builder
	flattenComments(&b, comments)
	return b.String()
}

func flattenComments(b *strings.Builder, comments []*reddit.Comment) {
	for _, c := range comments {
		if c == nil {
			continue
		}

		if c.IsMore {
			fmt.Fprintf(b, "+%d\tmore comments collapsed\n", c.MoreCount)
			continue
		}

		fmt.Fprintf(b, "d%d\t%d\t%s\t%s\n",
			c.Depth, c.Score, c.Author, truncate(c.Body, 200))

		if len(c.Replies) > 0 {
			flattenComments(b, c.Replies)
		}
	}
}
