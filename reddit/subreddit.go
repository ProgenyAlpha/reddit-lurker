package reddit

import (
	"encoding/json"
	"fmt"
	"time"
)

// GetSubreddit fetches posts from a subreddit with sorting and pagination.
// Returns the list of posts, the "after" token for the next page, and any error.
func (c *Client) GetSubreddit(name string, sort SortOrder, limit int, timeFilter TimeFilter, after string, noCache bool) ([]*Post, string, error) {
	if sort == "" {
		sort = SortHot
	}

	path := fmt.Sprintf("/r/%s/%s.json?limit=%d", name, sort, limit)

	if timeFilter != "" {
		path += "&t=" + string(timeFilter)
	}
	if after != "" {
		path += "&after=" + after
	}

	data, err := c.Fetch(path, noCache)
	if err != nil {
		return nil, "", fmt.Errorf("fetching subreddit %s: %w", name, err)
	}

	listing, err := ParseListing(data)
	if err != nil {
		return nil, "", fmt.Errorf("parsing subreddit %s: %w", name, err)
	}

	var posts []*Post
	for _, thing := range listing.Data.Children {
		if thing.Kind == "t3" {
			posts = append(posts, ParsePost(thing.Data))
		}
	}

	return posts, listing.Data.After, nil
}

// GetSubredditInfo fetches metadata about a subreddit.
func (c *Client) GetSubredditInfo(name string, noCache bool) (*SubredditInfo, error) {
	path := fmt.Sprintf("/r/%s/about.json", name)

	data, err := c.Fetch(path, noCache)
	if err != nil {
		return nil, fmt.Errorf("fetching subreddit info for %s: %w", name, err)
	}

	var thing struct {
		Kind string         `json:"kind"`
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(data, &thing); err != nil {
		return nil, fmt.Errorf("parsing subreddit info for %s: %w", name, err)
	}

	d := thing.Data
	info := &SubredditInfo{
		Name:          getString(d, "display_name"),
		Title:         getString(d, "title"),
		Description:   getString(d, "public_description"),
		Subscribers:   getInt(d, "subscribers"),
		ActiveUsers:   getInt(d, "accounts_active"),
		Over18:        getBool(d, "over18"),
		SubredditType: getString(d, "subreddit_type"),
	}

	if created, ok := d["created_utc"].(float64); ok {
		info.Created = time.Unix(int64(created), 0)
	}

	return info, nil
}
