package reddit

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// GetThread fetches a Reddit thread (post + comments) by permalink.
// Accepts full URLs like https://www.reddit.com/r/sub/comments/id/title/ or just the path.
func (c *Client) GetThread(permalink string, noCache bool) (*Thread, error) {
	// Strip full URL down to the path portion
	permalink = extractPermalink(permalink)

	// Ensure trailing slash
	if !strings.HasSuffix(permalink, "/") {
		permalink += "/"
	}

	path := permalink + ".json"

	data, err := c.Fetch(path, noCache)
	if err != nil {
		return nil, fmt.Errorf("fetching thread: %w", err)
	}

	// Reddit returns an array of 2 listings: [post_listing, comments_listing]
	var listings []Listing
	if err := json.Unmarshal(data, &listings); err != nil {
		return nil, fmt.Errorf("parsing thread listings: %w", err)
	}

	if len(listings) < 2 {
		return nil, fmt.Errorf("unexpected thread response: got %d listings, want 2", len(listings))
	}

	// First listing contains the post
	if len(listings[0].Data.Children) == 0 {
		return nil, fmt.Errorf("thread listing has no post")
	}

	post := ParsePost(listings[0].Data.Children[0].Data)

	// Second listing contains comments
	var comments []*Comment
	for _, thing := range listings[1].Data.Children {
		comment := parseComment(thing.Kind, thing.Data)
		if comment != nil {
			comments = append(comments, comment)
		}
	}

	return &Thread{
		Post:     post,
		Comments: comments,
	}, nil
}

// extractPermalink strips a full Reddit URL down to the path.
func extractPermalink(raw string) string {
	// Remove common Reddit domain prefixes
	for _, prefix := range []string{
		"https://www.reddit.com",
		"https://old.reddit.com",
		"https://reddit.com",
		"http://www.reddit.com",
		"http://old.reddit.com",
		"http://reddit.com",
	} {
		if strings.HasPrefix(raw, prefix) {
			return strings.TrimPrefix(raw, prefix)
		}
	}
	return raw
}

// parseComment recursively parses a comment tree from Reddit's API response.
func parseComment(kind string, data map[string]any) *Comment {
	if kind == "more" {
		return parseMoreComment(data)
	}

	if kind != "t1" {
		return nil
	}

	c := &Comment{
		ID:       getString(data, "id"),
		Author:   getString(data, "author"),
		Body:     getString(data, "body"),
		Score:    getInt(data, "score"),
		Depth:    getInt(data, "depth"),
		Stickied: getBool(data, "stickied"),
	}

	if created, ok := data["created_utc"].(float64); ok {
		c.Created = time.Unix(int64(created), 0)
	}

	// Parse nested replies. Reddit returns either "" (no replies) or a Listing object.
	if replies, ok := data["replies"].(map[string]any); ok {
		if repliesData, ok := replies["data"].(map[string]any); ok {
			if children, ok := repliesData["children"].([]any); ok {
				for _, child := range children {
					if childMap, ok := child.(map[string]any); ok {
						childKind, _ := childMap["kind"].(string)
						if childData, ok := childMap["data"].(map[string]any); ok {
							reply := parseComment(childKind, childData)
							if reply != nil {
								c.Replies = append(c.Replies, reply)
							}
						}
					}
				}
			}
		}
	}

	return c
}

// parseMoreComment creates a placeholder Comment for "load more" entries.
func parseMoreComment(data map[string]any) *Comment {
	c := &Comment{
		ID:        getString(data, "id"),
		IsMore:    true,
		MoreCount: getInt(data, "count"),
		Depth:     getInt(data, "depth"),
	}

	if ids, ok := data["children"].([]any); ok {
		for _, id := range ids {
			if s, ok := id.(string); ok {
				c.MoreIDs = append(c.MoreIDs, s)
			}
		}
	}

	return c
}
