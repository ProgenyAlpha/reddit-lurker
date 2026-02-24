package reddit

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

const (
	baseURL        = "https://www.reddit.com"
	oauthBaseURL   = "https://oauth.reddit.com"
	userAgent      = "lurk/1.0 (github.com/progenyalpha/lurk)"
	maxRetries     = 3
	retryDelay     = 2 * time.Second
	cacheTTL       = 5 * time.Minute
	unauthRateLimit = 10 // requests per minute
)

// Client handles all Reddit API communication.
type Client struct {
	http      *http.Client
	cache     map[string]*cacheEntry
	cacheMu   sync.RWMutex
	limiter   *rateLimiter
}

type cacheEntry struct {
	data      []byte
	expiresAt time.Time
}

type rateLimiter struct {
	mu       sync.Mutex
	tokens   int
	max      int
	lastFill time.Time
}

func newRateLimiter(maxPerMin int) *rateLimiter {
	return &rateLimiter{
		tokens:   maxPerMin,
		max:      maxPerMin,
		lastFill: time.Now(),
	}
}

func (rl *rateLimiter) wait() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(rl.lastFill)
	if elapsed >= time.Minute {
		rl.tokens = rl.max
		rl.lastFill = now
	}

	if rl.tokens <= 0 {
		wait := time.Minute - elapsed
		time.Sleep(wait)
		rl.tokens = rl.max
		rl.lastFill = time.Now()
	}
	rl.tokens--
}

// NewClient creates a new Reddit API client.
func NewClient() *Client {
	return &Client{
		http: &http.Client{
			Timeout: 30 * time.Second,
		},
		cache:   make(map[string]*cacheEntry),
		limiter: newRateLimiter(unauthRateLimit),
	}
}

// Fetch makes a GET request to the Reddit JSON API with retry, rate limiting, and caching.
func (c *Client) Fetch(path string, noCache bool) ([]byte, error) {
	if !noCache {
		if data := c.getCache(path); data != nil {
			return data, nil
		}
	}

	c.limiter.wait()

	url := baseURL + path
	if path[0] != '/' {
		url = baseURL + "/" + path
	}

	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(retryDelay * time.Duration(attempt))
		}

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("creating request: %w", err)
		}
		req.Header.Set("User-Agent", userAgent)

		resp, err := c.http.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("request failed: %w", err)
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("reading body: %w", err)
			continue
		}

		switch {
		case resp.StatusCode == 200:
			c.setCache(path, body)
			return body, nil
		case resp.StatusCode == 403:
			return nil, fmt.Errorf("access denied — subreddit may be private (requires auth) or quarantined")
		case resp.StatusCode == 404:
			return nil, fmt.Errorf("not found — check the URL or subreddit name")
		case resp.StatusCode == 429 || resp.StatusCode >= 500:
			lastErr = fmt.Errorf("HTTP %d", resp.StatusCode)
			continue
		default:
			return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body[:min(len(body), 200)]))
		}
	}

	return nil, fmt.Errorf("failed after %d retries: %w", maxRetries, lastErr)
}

func (c *Client) getCache(key string) []byte {
	c.cacheMu.RLock()
	defer c.cacheMu.RUnlock()
	entry, ok := c.cache[key]
	if !ok || time.Now().After(entry.expiresAt) {
		return nil
	}
	return entry.data
}

func (c *Client) setCache(key string, data []byte) {
	c.cacheMu.Lock()
	defer c.cacheMu.Unlock()
	c.cache[key] = &cacheEntry{
		data:      data,
		expiresAt: time.Now().Add(cacheTTL),
	}
}

// ParseListing decodes a JSON response into a Listing.
func ParseListing(data []byte) (*Listing, error) {
	var listing Listing
	if err := json.Unmarshal(data, &listing); err != nil {
		return nil, fmt.Errorf("parsing listing: %w", err)
	}
	return &listing, nil
}

// ParsePost extracts a Post from a Thing's data map.
func ParsePost(data map[string]any) *Post {
	p := &Post{
		ID:        getString(data, "id"),
		Subreddit: getString(data, "subreddit"),
		Author:    getString(data, "author"),
		Title:     getString(data, "title"),
		SelfText:  getString(data, "selftext"),
		Score:     getInt(data, "score"),
		NumComments: getInt(data, "num_comments"),
		URL:       getString(data, "url"),
		Permalink: getString(data, "permalink"),
		Domain:    getString(data, "domain"),
		IsVideo:   getBool(data, "is_video"),
		IsSelf:    getBool(data, "is_self"),
		Over18:    getBool(data, "over_18"),
		Stickied:  getBool(data, "stickied"),
		LinkFlairText: getString(data, "link_flair_text"),
	}

	if ratio, ok := data["upvote_ratio"].(float64); ok {
		p.UpvoteRatio = ratio
	}

	if created, ok := data["created_utc"].(float64); ok {
		p.Created = time.Unix(int64(created), 0)
	}

	// Cross-post detection
	if crosspostList, ok := data["crosspost_parent_list"].([]any); ok && len(crosspostList) > 0 {
		if cp, ok := crosspostList[0].(map[string]any); ok {
			p.CrosspostParent = getString(cp, "permalink")
		}
	}

	// Media URL extraction
	if !p.IsSelf {
		p.MediaURL = p.URL
	}
	if preview, ok := data["preview"].(map[string]any); ok {
		if images, ok := preview["images"].([]any); ok && len(images) > 0 {
			if img, ok := images[0].(map[string]any); ok {
				if source, ok := img["source"].(map[string]any); ok {
					p.MediaURL = getString(source, "url")
				}
			}
		}
	}

	return p
}

// Helper functions for safe type assertion from map[string]any.
func getString(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getInt(m map[string]any, key string) int {
	if v, ok := m[key].(float64); ok {
		return int(v)
	}
	return 0
}

func getBool(m map[string]any, key string) bool {
	if v, ok := m[key].(bool); ok {
		return v
	}
	return false
}
