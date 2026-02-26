package reddit

import (
	"encoding/json"
	"fmt"
	"time"
)

// GetUser fetches a user's profile info and recent activity (posts + comments).
func (c *Client) GetUser(username string, limit int, noCache bool) (*UserInfo, []*Post, []*Comment, error) {
	// Fetch user about info
	aboutPath := fmt.Sprintf("/user/%s/about.json", username)
	aboutData, err := c.Fetch(aboutPath, noCache)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("fetching user %s: %w", username, err)
	}

	var aboutThing struct {
		Kind string         `json:"kind"`
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(aboutData, &aboutThing); err != nil {
		return nil, nil, nil, fmt.Errorf("unexpected response for u/%s — user may not exist or account is suspended", username)
	}

	d := aboutThing.Data
	info := &UserInfo{
		Name:         getString(d, "name"),
		LinkKarma:    getInt(d, "link_karma"),
		CommentKarma: getInt(d, "comment_karma"),
		TotalKarma:   getInt(d, "total_karma"),
		IsSuspended:  getBool(d, "is_suspended"),
	}

	if created, ok := d["created_utc"].(float64); ok {
		info.Created = time.Unix(int64(created), 0)
	}

	// Fetch user overview (mixed posts and comments)
	overviewPath := fmt.Sprintf("/user/%s/overview.json?limit=%d", username, limit)
	overviewData, err := c.Fetch(overviewPath, noCache)
	if err != nil {
		return info, nil, nil, fmt.Errorf("fetching overview for %s: %w", username, err)
	}

	listing, err := ParseListing(overviewData)
	if err != nil {
		return info, nil, nil, fmt.Errorf("parsing overview for %s: %w", username, err)
	}

	var posts []*Post
	var comments []*Comment
	for _, thing := range listing.Data.Children {
		switch thing.Kind {
		case "t3": // post
			posts = append(posts, ParsePost(thing.Data))
		case "t1": // comment
			comments = append(comments, parseUserComment(thing.Data))
		}
	}

	return info, posts, comments, nil
}

// parseUserComment parses a standalone comment from user overview (no nested replies).
func parseUserComment(data map[string]any) *Comment {
	c := &Comment{
		ID:     getString(data, "id"),
		Author: getString(data, "author"),
		Body:   getString(data, "body"),
		Score:  getInt(data, "score"),
	}

	if created, ok := data["created_utc"].(float64); ok {
		c.Created = time.Unix(int64(created), 0)
	}

	return c
}
