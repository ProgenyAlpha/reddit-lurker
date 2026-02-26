package reddit

import (
	"fmt"
	"net/url"
	"strings"

	gosort "sort"
)

// Search performs a Reddit search, optionally scoped to a subreddit.
// Supports comma-separated subreddit names for parallel multi-sub search.
func (c *Client) Search(query string, sub string, sort SortOrder, limit int, timeFilter TimeFilter, after string, noCache bool) ([]*Post, string, error) {
	if strings.Contains(sub, ",") {
		return c.searchMulti(query, sub, sort, limit, timeFilter, noCache)
	}

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

// searchMulti searches multiple subreddits in parallel and merges results by score.
// Partial failures are tolerated: if some subreddits fail but others succeed, the
// successful results are returned. Only returns an error if ALL subreddits fail.
// Pagination is not supported for multi-sub queries — the after token is always empty.
func (c *Client) searchMulti(query string, subs string, sortOrder SortOrder, limit int, timeFilter TimeFilter, noCache bool) ([]*Post, string, error) {
	parts := strings.Split(subs, ",")
	type result struct {
		posts []*Post
		err   error
	}
	ch := make(chan result, len(parts))

	for _, s := range parts {
		s = strings.TrimSpace(s)
		if s == "" {
			ch <- result{}
			continue
		}
		go func(sub string) {
			posts, _, err := c.Search(query, sub, sortOrder, limit, timeFilter, "", noCache)
			ch <- result{posts, err}
		}(s)
	}

	seen := make(map[string]bool)
	var merged []*Post
	var firstErr error
	for range parts {
		r := <-ch
		if r.err != nil {
			if firstErr == nil {
				firstErr = r.err
			}
			continue
		}
		for _, p := range r.posts {
			if !seen[p.ID] {
				seen[p.ID] = true
				merged = append(merged, p)
			}
		}
	}

	if len(merged) == 0 && firstErr != nil {
		return nil, "", firstErr
	}

	gosort.Slice(merged, func(i, j int) bool {
		return merged[i].Score > merged[j].Score
	})

	if limit > 0 && len(merged) > limit {
		merged = merged[:limit]
	}

	return merged, "", nil
}
