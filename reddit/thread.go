package reddit

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

// fetchThreadBase fetches and parses a thread without expansion.
func (c *Client) fetchThreadBase(permalink string, noCache bool) (*Thread, error) {
	permalink = extractPermalink(permalink)

	if !strings.Contains(permalink, "/comments/") {
		return nil, fmt.Errorf("not a valid thread URL — expected a link like reddit.com/r/sub/comments/id/title")
	}

	if !strings.HasSuffix(permalink, "/") {
		permalink += "/"
	}

	path := permalink + ".json?limit=500"

	data, err := c.Fetch(path, noCache)
	if err != nil {
		return nil, fmt.Errorf("fetching thread: %w", err)
	}

	var listings []Listing
	if err := json.Unmarshal(data, &listings); err != nil {
		return nil, fmt.Errorf("not a valid thread URL — expected a link like reddit.com/r/sub/comments/id/title")
	}

	if len(listings) < 2 {
		return nil, fmt.Errorf("not a valid thread URL — expected a link like reddit.com/r/sub/comments/id/title")
	}

	if len(listings[0].Data.Children) == 0 {
		return nil, fmt.Errorf("thread has no post — it may have been deleted or removed")
	}

	post := ParsePost(listings[0].Data.Children[0].Data)

	var comments []*Comment
	for _, thing := range listings[1].Data.Children {
		comment := parseComment(thing.Kind, thing.Data)
		if comment != nil {
			comments = append(comments, comment)
		}
	}

	return &Thread{Post: post, Comments: comments}, nil
}

// GetThread fetches a Reddit thread (post + comments) by permalink.
// Accepts full URLs like https://www.reddit.com/r/sub/comments/id/title/ or just the path.
func (c *Client) GetThread(permalink string, noCache bool) (*Thread, error) {
	thread, err := c.fetchThreadBase(permalink, noCache)
	if err != nil {
		return nil, err
	}

	// Recursively expand "more" placeholders until none remain (max 10 passes)
	for i := 0; i < 10; i++ {
		if expandMoreComments(c, thread, noCache) == 0 {
			break
		}
	}

	return thread, nil
}

// GetThreadShallow fetches a thread without expanding collapsed comments.
func (c *Client) GetThreadShallow(permalink string, noCache bool) (*Thread, error) {
	return c.fetchThreadBase(permalink, noCache)
}

// ExpandThread recursively expands "more" placeholders (up to 10 passes).
func (c *Client) ExpandThread(thread *Thread, noCache bool) {
	for i := 0; i < 10; i++ {
		if expandMoreComments(c, thread, noCache) == 0 {
			break
		}
	}
}

// walkComments traverses a comment tree, calling fn for each non-nil, non-"more" comment.
func walkComments(comments []*Comment, fn func(*Comment)) {
	for _, c := range comments {
		if c == nil || c.IsMore {
			continue
		}
		fn(c)
		if len(c.Replies) > 0 {
			walkComments(c.Replies, fn)
		}
	}
}

// TopCommentsByScore flattens the comment tree and returns the top N by score.
func TopCommentsByScore(comments []*Comment, limit int) []*Comment {
	var flat []*Comment
	walkComments(comments, func(c *Comment) {
		flat = append(flat, &Comment{
			ID:       c.ID,
			ParentID: c.ParentID,
			Author:   c.Author,
			Body:     c.Body,
			Score:    c.Score,
			Depth:    c.Depth,
			Created:  c.Created,
			Stickied: c.Stickied,
		})
	})

	sort.Slice(flat, func(i, j int) bool {
		return flat[i].Score > flat[j].Score
	})

	if limit > 0 && limit < len(flat) {
		flat = flat[:limit]
	}
	return flat
}

// CountComments counts all non-"more" comments in a tree.
func CountComments(comments []*Comment) int {
	count := 0
	walkComments(comments, func(_ *Comment) {
		count++
	})
	return count
}

// EstimateTokens estimates token count for a comment tree (~4 chars per token).
func EstimateTokens(comments []*Comment) int {
	chars := 0
	walkComments(comments, func(c *Comment) {
		chars += len(c.Body) + len(c.Author) + 20
	})
	return chars / 4
}

// expandMoreComments walks the comment tree, collects all "more" placeholders,
// fetches their children in batches of 100, and inserts them back into the tree.
// Returns the number of comments expanded (0 means no more placeholders found).
func expandMoreComments(client *Client, thread *Thread, noCache bool) int {
	if thread.Post == nil {
		return 0
	}

	// Collect all "more" placeholders
	type moreRef struct {
		parent  *[]*Comment
		index   int
		moreIDs []string
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
	var fetchedMore []*Comment // new "more" placeholders from expansion
	for i := 0; i < len(allIDs); i += batchSize {
		end := i + batchSize
		if end > len(allIDs) {
			end = len(allIDs)
		}
		batch, err := client.FetchMoreChildren(thread.Post.ID, allIDs[i:end], noCache)
		if err != nil {
			continue
		}
		for _, c := range batch {
			if c != nil {
				if c.IsMore {
					fetchedMore = append(fetchedMore, c)
				} else {
					fetched = append(fetched, c)
				}
			}
		}
	}

	if len(fetched) == 0 {
		return 0
	}

	// Build a map of all existing comments by ID for parent lookup
	existingByID := make(map[string]*Comment)
	var indexExisting func(comments []*Comment)
	indexExisting = func(comments []*Comment) {
		for _, c := range comments {
			if c != nil && !c.IsMore {
				existingByID[c.ID] = c
				if len(c.Replies) > 0 {
					indexExisting(c.Replies)
				}
			}
		}
	}
	indexExisting(thread.Comments)

	// Insert fetched comments into the tree by ParentID
	inserted := 0
	var orphans []*Comment
	for _, c := range fetched {
		if parent, ok := existingByID[c.ParentID]; ok {
			parent.Replies = append(parent.Replies, c)
			existingByID[c.ID] = c // make it findable for subsequent inserts
			inserted++
		} else if c.ParentID == thread.Post.ID {
			// Top-level comment (parent is the post)
			thread.Comments = append(thread.Comments, c)
			existingByID[c.ID] = c
			inserted++
		} else {
			orphans = append(orphans, c)
		}
	}

	// Second pass: try orphans again (their parent may have been inserted in the first pass)
	for _, c := range orphans {
		if parent, ok := existingByID[c.ParentID]; ok {
			parent.Replies = append(parent.Replies, c)
			existingByID[c.ID] = c
			inserted++
		}
	}

	// Remove "more" placeholders that were expanded
	// Process in reverse order to maintain stable indices
	for i := len(mores) - 1; i >= 0; i-- {
		m := mores[i]
		parent := *m.parent
		if m.index < len(parent) {
			newSlice := make([]*Comment, 0, len(parent)-1)
			newSlice = append(newSlice, parent[:m.index]...)
			newSlice = append(newSlice, parent[m.index+1:]...)
			*m.parent = newSlice
		}
	}

	// Add any new "more" placeholders from the expansion to appropriate parents
	for _, m := range fetchedMore {
		if parent, ok := existingByID[m.ParentID]; ok {
			parent.Replies = append(parent.Replies, m)
		}
	}

	return inserted
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

	parentID := getString(data, "parent_id")
	parentID = strings.TrimPrefix(parentID, "t1_")
	parentID = strings.TrimPrefix(parentID, "t3_")

	c := &Comment{
		ID:       getString(data, "id"),
		ParentID: parentID,
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
