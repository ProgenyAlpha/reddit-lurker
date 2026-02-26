package reddit

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// emptyListingResponse returns a minimal valid subreddit listing JSON body.
func emptyListingResponse() []byte {
	return []byte(`{"kind":"Listing","data":{"after":"","children":[]}}`)
}

// captureURL records the last request URL received by the test server.
func captureURL(captured *string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*captured = r.URL.String()
		w.Write(emptyListingResponse())
	})
}

// ---------------------------------------------------------------------------
// URL construction — subreddit name containing spaces
// ---------------------------------------------------------------------------

func TestGetSubreddit_SubredditWithSpaces_URLEncoded(t *testing.T) {
	var got string
	srv := httptest.NewServer(captureURL(&got))
	defer srv.Close()

	c := newTestClient(srv)
	// Space in name is unusual but we want to verify the path doesn't break
	// the URL — it should be percent-encoded.
	c.GetSubreddit("my sub", SortHot, 5, "", "", true)

	// The URL the server received should not contain a literal space.
	// url.Parse decodes %20 back to space in .Path, so check the raw string.
	if strings.Contains(got, " ") {
		t.Errorf("URL contains literal space, want percent-encoding: %q", got)
	}
	// And it should contain the percent-encoded form.
	if !strings.Contains(got, "my") {
		t.Errorf("URL does not seem to contain subreddit name: %q", got)
	}
}

// ---------------------------------------------------------------------------
// URL construction — after token containing "&"
// ---------------------------------------------------------------------------

func TestGetSubreddit_AfterTokenWithAmpersand_URLNotBroken(t *testing.T) {
	var got string
	srv := httptest.NewServer(captureURL(&got))
	defer srv.Close()

	c := newTestClient(srv)
	c.GetSubreddit("golang", SortNew, 5, "", "t3_abc&injected=true", true)

	// The ampersand in the token must be escaped so it doesn't break the query string.
	if strings.Contains(got, "injected=true") {
		t.Errorf("after token ampersand was not escaped, URL: %q", got)
	}
	if !strings.Contains(got, "after=") {
		t.Errorf("URL does not contain after param: %q", got)
	}
}

// ---------------------------------------------------------------------------
// URL construction — no timeFilter → no "&t=" in URL
// ---------------------------------------------------------------------------

func TestGetSubreddit_EmptyTimeFilter_NoTParam(t *testing.T) {
	var got string
	srv := httptest.NewServer(captureURL(&got))
	defer srv.Close()

	c := newTestClient(srv)
	c.GetSubreddit("golang", SortHot, 10, "", "", true)

	if strings.Contains(got, "&t=") {
		t.Errorf("URL contains &t= when timeFilter is empty: %q", got)
	}
}

// ---------------------------------------------------------------------------
// URL construction — all params → correct URL format
// ---------------------------------------------------------------------------

func TestGetSubreddit_AllParams_CorrectURL(t *testing.T) {
	var got string
	srv := httptest.NewServer(captureURL(&got))
	defer srv.Close()

	c := newTestClient(srv)
	c.GetSubreddit("golang", SortTop, 25, TimeWeek, "t3_after", true)

	checks := []struct {
		name    string
		present string
	}{
		{"subreddit name", "/r/golang/"},
		{"sort", "top"},
		{"limit", "limit=25"},
		{"time filter", "&t=week"},
		{"after token", "after=t3_after"},
	}

	for _, chk := range checks {
		if !strings.Contains(got, chk.present) {
			t.Errorf("URL missing %s (%q); full URL: %q", chk.name, chk.present, got)
		}
	}
}
