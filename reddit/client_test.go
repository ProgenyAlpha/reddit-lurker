package reddit

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newTestClient returns a Client whose HTTP transport points at the given server.
func newTestClient(server *httptest.Server) *Client {
	c := NewClient()
	c.http = server.Client()
	// Override the base URL by pointing directly at the test server.
	// We do this by replacing the transport so requests to baseURL are
	// redirected; easiest approach is just to swap the baseURL constant
	// substitute via a wrapper. Since baseURL is package-level, we do it
	// by setting up a RoundTripper that rewrites the host.
	c.http = &http.Client{
		Transport: &rewriteTransport{
			wrapped: server.Client().Transport,
			from:    "https://www.reddit.com",
			to:      server.URL,
		},
	}
	// Disable rate limiter so tests don't sleep.
	c.limiter = newRateLimiter(1_000_000)
	return c
}

// rewriteTransport replaces the host in outgoing requests.
type rewriteTransport struct {
	wrapped http.RoundTripper
	from    string
	to      string
}

func (rt *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	cloned := req.Clone(req.Context())
	cloned.URL.Scheme = "http"
	// Replace the full from-prefix with to; preserve path/query.
	raw := req.URL.String()
	raw = strings.Replace(raw, rt.from, rt.to, 1)
	// Strip scheme for the test server (it's http not https).
	cloned.URL, _ = cloned.URL.Parse(raw)
	return rt.wrapped.RoundTrip(cloned)
}

// ---------------------------------------------------------------------------
// Fetch edge cases
// ---------------------------------------------------------------------------

func TestFetch_EmptyPath_ReturnsError(t *testing.T) {
	// Before the fix, path[0] would panic with index out of range.
	// After the fix it must return a descriptive error.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Should never be reached for empty path.
		t.Error("server should not be called for empty path")
	}))
	defer srv.Close()

	c := newTestClient(srv)
	_, err := c.Fetch("", false)
	if err == nil {
		t.Fatal("expected error for empty path, got nil")
	}
}

// ---------------------------------------------------------------------------
// Cache behaviour
// ---------------------------------------------------------------------------

func TestFetch_Cache_SamePath_ReturnsCachedResult(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Write([]byte(`{"cached":true}`))
	}))
	defer srv.Close()

	c := newTestClient(srv)
	first, err := c.Fetch("/r/test.json", false)
	if err != nil {
		t.Fatalf("first fetch failed: %v", err)
	}

	second, err := c.Fetch("/r/test.json", false)
	if err != nil {
		t.Fatalf("second fetch failed: %v", err)
	}

	if calls != 1 {
		t.Errorf("expected 1 HTTP call for cached path, got %d", calls)
	}
	if string(first) != string(second) {
		t.Errorf("cached result mismatch: %s != %s", first, second)
	}
}

func TestFetch_Cache_DifferentPath_MissesCache(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		fmt.Fprintf(w, `{"path":%q}`, r.URL.Path)
	}))
	defer srv.Close()

	c := newTestClient(srv)
	_, err := c.Fetch("/r/a.json", false)
	if err != nil {
		t.Fatalf("fetch a failed: %v", err)
	}

	_, err = c.Fetch("/r/b.json", false)
	if err != nil {
		t.Fatalf("fetch b failed: %v", err)
	}

	if calls != 2 {
		t.Errorf("expected 2 HTTP calls for different paths, got %d", calls)
	}
}

// ---------------------------------------------------------------------------
// Retry on 429
// ---------------------------------------------------------------------------

func TestFetch_Retry_429ThenSuccess(t *testing.T) {
	// Server returns 429 twice, then 200 on the third attempt.
	attempt := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt++
		if attempt < 3 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c := newTestClient(srv)
	// Override retry delay to zero so the test is fast.
	data, err := c.Fetch("/test.json", true)
	if err != nil {
		t.Fatalf("expected success after retries, got error: %v", err)
	}
	if !strings.Contains(string(data), "ok") {
		t.Errorf("unexpected response body: %s", data)
	}
	if attempt != 3 {
		t.Errorf("expected 3 attempts, got %d", attempt)
	}
}

// ---------------------------------------------------------------------------
// ParseListing with malformed JSON
// ---------------------------------------------------------------------------

func TestParseListing_MalformedJSON_ReturnsError(t *testing.T) {
	bad := []byte(`{not valid json`)
	_, err := ParseListing(bad)
	if err == nil {
		t.Fatal("expected error for malformed JSON, got nil")
	}
}

// ---------------------------------------------------------------------------
// ParsePost media extraction
// ---------------------------------------------------------------------------

func TestParsePost_GalleryData_MultipleURLs(t *testing.T) {
	// gallery_data present; media_metadata has two items with "s.u" URLs.
	data := map[string]any{
		"id":    "abc123",
		"title": "gallery post",
		// gallery_data presence triggers the metadata path
		"gallery_data": map[string]any{"items": []any{}},
		"media_metadata": map[string]any{
			"img1": map[string]any{
				"s": map[string]any{"u": "https://preview.redd.it/img1.jpg"},
			},
			"img2": map[string]any{
				"s": map[string]any{"u": "https://preview.redd.it/img2.jpg"},
			},
		},
	}

	p := ParsePost(data)
	if p == nil {
		t.Fatal("ParsePost returned nil")
	}
	if p.MediaURL == "" {
		t.Fatal("expected MediaURL to be set for gallery post")
	}
	// MediaURL should be comma-separated.
	parts := strings.Split(p.MediaURL, ",")
	if len(parts) != 2 {
		t.Errorf("expected 2 gallery URLs, got %d: %s", len(parts), p.MediaURL)
	}
}

func TestParsePost_RedditVideo(t *testing.T) {
	data := map[string]any{
		"id":    "vid1",
		"title": "video post",
		"media": map[string]any{
			"reddit_video": map[string]any{
				"fallback_url": "https://v.redd.it/abc123/DASH_720.mp4",
			},
		},
	}

	p := ParsePost(data)
	if p.MediaURL != "https://v.redd.it/abc123/DASH_720.mp4" {
		t.Errorf("expected reddit_video fallback_url, got %q", p.MediaURL)
	}
}

func TestParsePost_PreviewImageOnly(t *testing.T) {
	data := map[string]any{
		"id":    "prev1",
		"title": "preview post",
		"preview": map[string]any{
			"images": []any{
				map[string]any{
					"source": map[string]any{
						"url": "https://external-preview.redd.it/img.jpg",
					},
				},
			},
		},
	}

	p := ParsePost(data)
	if p.MediaURL != "https://external-preview.redd.it/img.jpg" {
		t.Errorf("expected preview source URL, got %q", p.MediaURL)
	}
}

func TestParsePost_NoMedia_EmptyMediaURL(t *testing.T) {
	data := map[string]any{
		"id":      "self1",
		"title":   "text post",
		"is_self": true,
	}

	p := ParsePost(data)
	if p.MediaURL != "" {
		t.Errorf("expected empty MediaURL for self-post with no media, got %q", p.MediaURL)
	}
}

func TestParsePost_ExternalLink_SetsMediaURL(t *testing.T) {
	// Non-self posts with no gallery/video/preview should use URL directly.
	data := map[string]any{
		"id":      "link1",
		"title":   "link post",
		"is_self": false,
		"url":     "https://example.com/article",
	}

	p := ParsePost(data)
	if p.MediaURL != "https://example.com/article" {
		t.Errorf("expected MediaURL == URL for external link post, got %q", p.MediaURL)
	}
}

// ---------------------------------------------------------------------------
// ParseListing round-trip
// ---------------------------------------------------------------------------

func TestParseListing_ValidJSON(t *testing.T) {
	listing := Listing{
		Kind: "Listing",
		Data: ListingData{
			After: "t3_next",
			Children: []Thing{
				{Kind: "t3", Data: map[string]any{"id": "abc", "title": "hello"}},
			},
		},
	}
	b, _ := json.Marshal(listing)

	got, err := ParseListing(b)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Data.Children) != 1 {
		t.Errorf("expected 1 child, got %d", len(got.Data.Children))
	}
	if got.Data.After != "t3_next" {
		t.Errorf("unexpected after token: %q", got.Data.After)
	}
}
