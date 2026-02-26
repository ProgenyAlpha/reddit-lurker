package reddit

import (
	"fmt"
	"net/url"
)

// Search performs a Reddit search, optionally scoped to a subreddit.
// Returns matching posts, the "after" pagination token, and any error.
func (c *Client) Search(query string, sub string, sort SortOrder, limit int, timeFilter TimeFilter, after string, noCache bool) ([]*Post, string, error) {
	if sort == "" {
		sort = SortRelevance
	}

	params := url.Values{}
	params.Set("q", query)
	params.Set("sort", string(sort))
	params.Set("limit", fmt.Sprintf("%d", limit))

	if timeFilter != "" {
		params.Set("t", string(timeFilter))
	}
	if after != "" {
		params.Set("after", after)
	}

	var path string
	if sub != "" {
		params.Set("restrict_sr", "on")
		path = fmt.Sprintf("/r/%s/search.json?%s", sub, params.Encode())
	} else {
		path = fmt.Sprintf("/search.json?%s", params.Encode())
	}

	data, err := c.Fetch(path, noCache)
	if err != nil {
		return nil, "", fmt.Errorf("searching reddit: %w", err)
	}

	listing, err := ParseListing(data)
	if err != nil {
		return nil, "", fmt.Errorf("parsing search results: %w", err)
	}

	var posts []*Post
	for _, thing := range listing.Data.Children {
		if thing.Kind == "t3" {
			posts = append(posts, ParsePost(thing.Data))
		}
	}

	return posts, listing.Data.After, nil
}
