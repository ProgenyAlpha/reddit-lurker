package reddit

import (
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	baseURL        = "https://www.reddit.com"
	userAgent      = "lurk/1.0 (github.com/ProgenyAlpha/reddit-lurker)"
	maxRetries     = 3
	retryDelay     = 2 * time.Second
	maxCacheSize   = 50 * 1024 * 1024 // 50MB
	// Reddit's unauthenticated limit is 10 req/min averaged over 10 minutes,
	// which allows bursting. We use a 10-minute window with 100 tokens.
	rateLimitWindow = 10 * time.Minute
	rateLimitTokens = 100
)

// Client handles all Reddit API communication.
type Client struct {
	http      *http.Client
	cache     map[string]*cacheEntry
	cacheMu   sync.RWMutex
	cacheSize int64
	limiter   *rateLimiter
	oauth     *oauthToken // nil when unauthenticated
}

type cacheEntry struct {
	data      []byte
	expiresAt time.Time
	size      int
	lastUsed  time.Time
	key       string
}

// cacheTTLFor returns an adaptive TTL based on URL pattern.
func cacheTTLFor(key string) time.Duration {
	switch {
	case strings.Contains(key, "/new.json"), strings.Contains(key, "/new/"):
		return 2 * time.Minute
	case strings.Contains(key, "/hot.json"), strings.Contains(key, "/hot/"):
		return 5 * time.Minute
	case strings.Contains(key, "/top.json"), strings.Contains(key, "/top/"):
		return 30 * time.Minute
	case strings.Contains(key, "/comments/"), strings.Contains(key, "/api/morechildren"):
		return 10 * time.Minute
	case strings.Contains(key, "/user/"), strings.Contains(key, "/about.json"):
		return 15 * time.Minute
	case strings.Contains(key, "/search.json"):
		return 10 * time.Minute
	default:
		return 15 * time.Minute
	}
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
	for {
		rl.mu.Lock()
		now := time.Now()
		elapsed := now.Sub(rl.lastFill)
		if elapsed >= rateLimitWindow {
			rl.tokens = rl.max
			rl.lastFill = now
		}

		if rl.tokens > 0 {
			rl.tokens--
			rl.mu.Unlock()
			return
		}

		// Out of tokens — compute wait, release lock, then sleep.
		// Other goroutines can proceed if tokens refill while we wait.
		wait := rateLimitWindow - elapsed
		rl.mu.Unlock()
		time.Sleep(wait)
	}
}

// NewClient creates a new Reddit API client.
// Automatically loads OAuth credentials if available (~/.config/lurk/credentials.json).
func NewClient() *Client {
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	c := &Client{
		http:    httpClient,
		cache:   make(map[string]*cacheEntry),
		limiter: newRateLimiter(rateLimitTokens),
	}

	// Try loading OAuth credentials — silent fallback to anonymous
	if creds, err := LoadCredentials(); err == nil && creds.ClientID != "" && creds.ClientSecret != "" {
		c.oauth = newOAuthToken(httpClient, creds)
		// Authenticated: 60 req/min = 600 over 10-min window
		c.limiter = newRateLimiter(600)
	}

	return c
}

// IsAuthenticated returns true if OAuth credentials are configured.
func (c *Client) IsAuthenticated() bool {
	return c.oauth != nil
}

// Fetch makes a GET request to the Reddit JSON API with retry, rate limiting, and caching.
func (c *Client) Fetch(path string, noCache bool) ([]byte, error) {
	if !noCache {
		if data := c.getCache(path); data != nil {
			return data, nil
		}
	}

	if path == "" {
		return nil, fmt.Errorf("fetch: path must not be empty")
	}

	c.limiter.wait()

	apiBase := baseURL
	if c.oauth != nil {
		apiBase = oauthBaseURL
	}

	url := apiBase + path
	if path[0] != '/' {
		url = apiBase + "/" + path
	}

	var lastErr error
	oauth401Retried := false
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(retryDelay * time.Duration(attempt))
		}

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("creating request: %w", err)
		}
		req.Header.Set("User-Agent", userAgent)
		if c.oauth != nil {
			token, tokenErr := c.oauth.get()
			if tokenErr != nil {
				return nil, fmt.Errorf("OAuth token refresh failed: %w — run 'lurk auth --status' to check credentials", tokenErr)
			}
			req.Header.Set("Authorization", "Bearer "+token)
		}

		resp, err := c.http.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("network error — check your internet connection")
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("failed to read response from Reddit")
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
		case resp.StatusCode == 429:
			lastErr = fmt.Errorf("rate limited — too many requests, try again shortly")
			continue
		case resp.StatusCode >= 500:
			lastErr = fmt.Errorf("Reddit server error (HTTP %d) — Reddit may be down", resp.StatusCode)
			continue
		case resp.StatusCode == 301 || resp.StatusCode == 302:
			return nil, fmt.Errorf("redirect — URL may be malformed or pointing to a non-API page")
		case resp.StatusCode == 401:
			if c.oauth != nil && !oauth401Retried {
				oauth401Retried = true
				c.oauth.invalidate()
				lastErr = fmt.Errorf("OAuth token expired, refreshing")
				continue
			}
			if c.oauth != nil {
				return nil, fmt.Errorf("OAuth credentials rejected — run 'lurk auth --status' to check")
			}
			return nil, fmt.Errorf("unauthorized — this content may require authentication")
		default:
			return nil, fmt.Errorf("unexpected error (HTTP %d)", resp.StatusCode)
		}
	}

	return nil, fmt.Errorf("failed after %d retries: %w", maxRetries, lastErr)
}

// FetchPost makes a POST request to the Reddit JSON API with retry, rate limiting, and caching.
func (c *Client) FetchPost(path string, formData url.Values, noCache bool) ([]byte, error) {
	cacheKey := "POST:" + path + "?" + formData.Encode()
	if !noCache {
		if data := c.getCache(cacheKey); data != nil {
			return data, nil
		}
	}

	if path == "" {
		return nil, fmt.Errorf("fetch: path must not be empty")
	}

	c.limiter.wait()

	apiBase := baseURL
	if c.oauth != nil {
		apiBase = oauthBaseURL
	}

	reqURL := apiBase + path
	if path[0] != '/' {
		reqURL = apiBase + "/" + path
	}

	var lastErr error
	oauth401Retried := false
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(retryDelay * time.Duration(attempt))
		}

		req, err := http.NewRequest("POST", reqURL, strings.NewReader(formData.Encode()))
		if err != nil {
			return nil, fmt.Errorf("creating request: %w", err)
		}
		req.Header.Set("User-Agent", userAgent)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		if c.oauth != nil {
			token, tokenErr := c.oauth.get()
			if tokenErr != nil {
				return nil, fmt.Errorf("OAuth token refresh failed: %w — run 'lurk auth --status' to check credentials", tokenErr)
			}
			req.Header.Set("Authorization", "Bearer "+token)
		}

		resp, err := c.http.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("network error — check your internet connection")
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("failed to read response from Reddit")
			continue
		}

		switch {
		case resp.StatusCode == 200:
			c.setCache(cacheKey, body)
			return body, nil
		case resp.StatusCode == 403:
			return nil, fmt.Errorf("access denied — subreddit may be private (requires auth) or quarantined")
		case resp.StatusCode == 404:
			return nil, fmt.Errorf("not found — check the URL or subreddit name")
		case resp.StatusCode == 429:
			lastErr = fmt.Errorf("rate limited — too many requests, try again shortly")
			continue
		case resp.StatusCode >= 500:
			lastErr = fmt.Errorf("Reddit server error (HTTP %d) — Reddit may be down", resp.StatusCode)
			continue
		case resp.StatusCode == 301 || resp.StatusCode == 302:
			return nil, fmt.Errorf("redirect — URL may be malformed or pointing to a non-API page")
		case resp.StatusCode == 401:
			if c.oauth != nil && !oauth401Retried {
				oauth401Retried = true
				c.oauth.invalidate()
				lastErr = fmt.Errorf("OAuth token expired, refreshing")
				continue
			}
			if c.oauth != nil {
				return nil, fmt.Errorf("OAuth credentials rejected — run 'lurk auth --status' to check")
			}
			return nil, fmt.Errorf("unauthorized — this content may require authentication")
		default:
			return nil, fmt.Errorf("unexpected error (HTTP %d)", resp.StatusCode)
		}
	}

	return nil, fmt.Errorf("failed after %d retries: %w", maxRetries, lastErr)
}

// FetchMoreChildren fetches collapsed comment threads using Reddit's /api/morechildren endpoint.
// The response format differs from normal comments: it returns contentText, parent, and
// author/score embedded in HTML content rather than structured JSON fields.
func (c *Client) FetchMoreChildren(linkID string, childrenIDs []string, noCache bool) ([]*Comment, error) {
	formData := url.Values{}
	formData.Set("api_type", "json")
	formData.Set("link_id", "t3_"+linkID)
	formData.Set("children", strings.Join(childrenIDs, ","))
	formData.Set("sort", "confidence")

	data, err := c.FetchPost("/api/morechildren", formData, noCache)
	if err != nil {
		return nil, fmt.Errorf("fetching more children: %w", err)
	}

	var resp struct {
		JSON struct {
			Data struct {
				Things []Thing `json:"things"`
			} `json:"data"`
		} `json:"json"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("failed to expand collapsed comments — Reddit returned an unexpected response")
	}

	var comments []*Comment
	for _, thing := range resp.JSON.Data.Things {
		if thing.Kind == "more" {
			comment := parseComment(thing.Kind, thing.Data)
			if comment != nil {
				comments = append(comments, comment)
			}
			continue
		}
		if thing.Kind != "t1" {
			continue
		}

		// morechildren returns a different format: contentText for body,
		// parent for tree position, and author/score in HTML content
		d := thing.Data
		comment := parseMoreChildrenComment(d)
		if comment != nil {
			comments = append(comments, comment)
		}
	}

	return comments, nil
}

// Regex patterns for parsing morechildren HTML content
var (
	reAuthor = regexp.MustCompile(`data-author="([^"]+)"`)
	reScore  = regexp.MustCompile(`class="score unvoted" title="(-?\d+)"`)
)

// parseMoreChildrenComment parses the morechildren response format,
// which uses contentText/parent/content(HTML) instead of the standard comment fields.
func parseMoreChildrenComment(d map[string]any) *Comment {
	id := getString(d, "id")
	// Strip t1_ prefix if present
	id = strings.TrimPrefix(id, "t1_")

	body := getString(d, "contentText")
	if body == "" {
		return nil
	}

	parentID := getString(d, "parent")
	parentID = strings.TrimPrefix(parentID, "t1_")
	parentID = strings.TrimPrefix(parentID, "t3_")

	c := &Comment{
		ID:       id,
		Body:     body,
		ParentID: parentID,
	}

	// Extract author and score from HTML content
	content := html.UnescapeString(getString(d, "content"))
	if m := reAuthor.FindStringSubmatch(content); len(m) > 1 {
		c.Author = m[1]
	}
	if m := reScore.FindStringSubmatch(content); len(m) > 1 {
		if score, err := strconv.Atoi(m[1]); err == nil {
			c.Score = score
		}
	}

	return c
}

func (c *Client) getCache(key string) []byte {
	c.cacheMu.Lock()
	defer c.cacheMu.Unlock()
	entry, ok := c.cache[key]
	if !ok {
		return nil
	}
	if time.Now().After(entry.expiresAt) {
		c.cacheSize -= int64(entry.size)
		delete(c.cache, key)
		return nil
	}
	entry.lastUsed = time.Now()
	return entry.data
}

func (c *Client) setCache(key string, data []byte) {
	c.cacheMu.Lock()
	defer c.cacheMu.Unlock()

	entrySize := len(data) + len(key) + 64
	if int64(entrySize) > maxCacheSize {
		return // don't cache entries larger than the limit
	}

	if old, ok := c.cache[key]; ok {
		c.cacheSize -= int64(old.size)
	}

	for c.cacheSize+int64(entrySize) > maxCacheSize && len(c.cache) > 0 {
		c.evictLRU()
	}

	now := time.Now()
	c.cache[key] = &cacheEntry{
		data:      data,
		expiresAt: now.Add(cacheTTLFor(key)),
		size:      entrySize,
		lastUsed:  now,
		key:       key,
	}
	c.cacheSize += int64(entrySize)
}

func (c *Client) evictLRU() {
	var oldest *cacheEntry
	for _, entry := range c.cache {
		if oldest == nil || entry.lastUsed.Before(oldest.lastUsed) {
			oldest = entry
		}
	}
	if oldest != nil {
		c.cacheSize -= int64(oldest.size)
		delete(c.cache, oldest.key)
	}
}

// ParseListing decodes a JSON response into a Listing.
func ParseListing(data []byte) (*Listing, error) {
	var listing Listing
	if err := json.Unmarshal(data, &listing); err != nil {
		return nil, fmt.Errorf("unexpected response format — Reddit may have returned an error page or changed its API")
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

	// Reddit galleries: extract image URLs from media_metadata
	if _, hasGallery := data["gallery_data"]; hasGallery {
		if metadata, ok := data["media_metadata"].(map[string]any); ok {
			var urls []string
			for _, item := range metadata {
				if m, ok := item.(map[string]any); ok {
					if s, ok := m["s"].(map[string]any); ok {
						if u := getString(s, "u"); u != "" {
							urls = append(urls, u)
						} else if gif := getString(s, "gif"); gif != "" {
							urls = append(urls, gif)
						} else if mp4 := getString(s, "mp4"); mp4 != "" {
							urls = append(urls, mp4)
						}
					}
				}
			}
			if len(urls) > 0 {
				p.MediaURL = strings.Join(urls, ",")
			}
		}
	} else if media, ok := data["media"].(map[string]any); ok {
		// Reddit-hosted video
		if rv, ok := media["reddit_video"].(map[string]any); ok {
			if fallback := getString(rv, "fallback_url"); fallback != "" {
				p.MediaURL = fallback
			}
		}
	} else if preview, ok := data["preview"].(map[string]any); ok {
		// Preview image fallback
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
