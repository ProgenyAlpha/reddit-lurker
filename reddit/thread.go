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

	// Validate that it looks like a thread URL (must contain /comments/)
	if !strings.Contains(permalink, "/comments/") {
		return nil, fmt.Errorf("not a valid thread URL — expected a link like reddit.com/r/sub/comments/id/title")
	}

	// Ensure trailing slash
	if !strings.HasSuffix(permalink, "/") {
		permalink += "/"
	}

	// Request max comments from Reddit (default is ~200, max is 500)
	path := permalink + ".json?limit=500"

	data, err := c.Fetch(path, noCache)
	if err != nil {
		return nil, fmt.Errorf("fetching thread: %w", err)
	}

	// Reddit returns an array of 2 listings: [post_listing, comments_listing]
	var listings []Listing
	if err := json.Unmarshal(data, &listings); err != nil {
		return nil, fmt.Errorf("not a valid thread URL — expected a link like reddit.com/r/sub/comments/id/title")
	}

	if len(listings) < 2 {
		return nil, fmt.Errorf("not a valid thread URL — expected a link like reddit.com/r/sub/comments/id/title")
	}

	// First listing contains the post
	if len(listings[0].Data.Children) == 0 {
		return nil, fmt.Errorf("thread has no post — it may have been deleted or removed")
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

	thread := &Thread{
		Post:     post,
		Comments: comments,
	}

	// Recursively expand "more" placeholders until none remain (max 10 passes)
	for i := 0; i < 10; i++ {
		expanded := expandMoreComments(c, thread, noCache)
		if expanded == 0 {
			break
		}
	}

	return thread, nil
}

// expandMoreComments walks the comment tree, collects all "more" placeholders,
// fetches their children in batches of 100, and inserts them back into the tree.
// Returns the number of comments expanded (0 means no more placeholders found).
func expandMoreComments(client *Client, thread *Thread, noCache bool) int {
	if thread.Post == nil {
		return 0
	}

	// Collect all "more" placeholders and their parent references
	type moreRef struct {
		parent   *[]*Comment // pointer to the slice containing the "more" placeholder
		index    int         // index within that slice
		moreIDs  []string
	}

	var mores []moreRef

	var walk func(comments *[]*Comment)
	walk = func(comments *[]*Comment) {
		for i, c := range *comments {
			if c == nil {
				continue
			}
			if c.IsMore && len(c.MoreIDs) > 0 {
				mores = append(mores, moreRef{
					parent:  comments,
					index:   i,
					moreIDs: c.MoreIDs,
				})
			} else if len(c.Replies) > 0 {
				walk(&c.Replies)
			}
		}
	}
	walk(&thread.Comments)

	if len(mores) == 0 {
		return 0
	}

	// Collect all IDs and batch them (max 100 per request)
	var allIDs []string
	for _, m := range mores {
		allIDs = append(allIDs, m.moreIDs...)
	}

	const batchSize = 100
	var fetched []*Comment
	for i := 0; i < len(allIDs); i += batchSize {
		end := i + batchSize
		if end > len(allIDs) {
			end = len(allIDs)
		}
		batch, err := client.FetchMoreChildren(thread.Post.ID, allIDs[i:end], noCache)
		if err != nil {
			// Non-fatal: just skip expansion on error
			continue
		}
		fetched = append(fetched, batch...)
	}

	if len(fetched) == 0 {
		return 0
	}

	// Build a map of fetched comments by ID for quick lookup
	fetchedByID := make(map[string]*Comment, len(fetched))
	for _, c := range fetched {
		if c != nil {
			fetchedByID[c.ID] = c
		}
	}

	// Replace "more" placeholders with fetched comments
	// Process in reverse order to maintain stable indices
	for i := len(mores) - 1; i >= 0; i-- {
		m := mores[i]
		var replacements []*Comment
		for _, id := range m.moreIDs {
			if c, ok := fetchedByID[id]; ok {
				replacements = append(replacements, c)
			}
		}
		if len(replacements) > 0 {
			// Replace the "more" placeholder with the fetched comments
			parent := *m.parent
			newSlice := make([]*Comment, 0, len(parent)-1+len(replacements))
			newSlice = append(newSlice, parent[:m.index]...)
			newSlice = append(newSlice, replacements...)
			newSlice = append(newSlice, parent[m.index+1:]...)
			*m.parent = newSlice
		}
	}

	return len(fetched)
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
