package reddit

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// makeThreadJSON builds a minimal 2-listing array for a thread response.
func makeThreadJSON(postData map[string]any, comments []map[string]any) []byte {
	children := make([]Thing, 0, len(comments))
	for _, c := range comments {
		kind := "t1"
		if k, ok := c["_kind"].(string); ok {
			kind = k
			delete(c, "_kind")
		}
		children = append(children, Thing{Kind: kind, Data: c})
	}

	listings := []Listing{
		{
			Kind: "Listing",
			Data: ListingData{
				Children: []Thing{{Kind: "t3", Data: postData}},
			},
		},
		{
			Kind: "Listing",
			Data: ListingData{Children: children},
		},
	}
	b, _ := json.Marshal(listings)
	return b
}

// ---------------------------------------------------------------------------
// GetThread: 0 comments — must not panic
// ---------------------------------------------------------------------------

func TestGetThread_ZeroComments_NoPanic(t *testing.T) {
	body := makeThreadJSON(
		map[string]any{"id": "abc", "title": "no comments post"},
		nil, // empty children in comments listing
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(body)
	}))
	defer srv.Close()

	c := newTestClient(srv)
	thread, err := c.GetThread("/r/test/comments/abc/title/", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(thread.Comments) != 0 {
		t.Errorf("expected 0 comments, got %d", len(thread.Comments))
	}
}

// ---------------------------------------------------------------------------
// GetThread: response with < 2 listings returns proper error
// ---------------------------------------------------------------------------

func TestGetThread_TooFewListings_ReturnsError(t *testing.T) {
	// Single listing instead of the expected pair.
	single := []Listing{
		{Kind: "Listing", Data: ListingData{
			Children: []Thing{{Kind: "t3", Data: map[string]any{"id": "x"}}},
		}},
	}
	b, _ := json.Marshal(single)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(b)
	}))
	defer srv.Close()

	c := newTestClient(srv)
	_, err := c.GetThread("/r/test/comments/x/title/", true)
	if err == nil {
		t.Fatal("expected error for <2 listings, got nil")
	}
}

// ---------------------------------------------------------------------------
// parseComment kind="t1"
// ---------------------------------------------------------------------------

func TestParseComment_T1_Normal(t *testing.T) {
	data := map[string]any{
		"id":     "c1",
		"author": "alice",
		"body":   "hello world",
		"score":  float64(42),
		"depth":  float64(0),
	}

	c := parseComment("t1", data)
	if c == nil {
		t.Fatal("expected non-nil comment for t1")
	}
	if c.ID != "c1" {
		t.Errorf("unexpected ID: %q", c.ID)
	}
	if c.Author != "alice" {
		t.Errorf("unexpected author: %q", c.Author)
	}
	if c.Body != "hello world" {
		t.Errorf("unexpected body: %q", c.Body)
	}
	if c.Score != 42 {
		t.Errorf("unexpected score: %d", c.Score)
	}
	if c.IsMore {
		t.Error("IsMore should be false for t1")
	}
}

// ---------------------------------------------------------------------------
// parseComment kind="more"
// ---------------------------------------------------------------------------

func TestParseComment_More_Placeholder(t *testing.T) {
	data := map[string]any{
		"id":       "moreid",
		"count":    float64(15),
		"depth":    float64(1),
		"children": []any{"aa", "bb", "cc"},
	}

	c := parseComment("more", data)
	if c == nil {
		t.Fatal("expected non-nil comment for kind=more")
	}
	if !c.IsMore {
		t.Error("IsMore should be true for kind=more")
	}
	if c.MoreCount != 15 {
		t.Errorf("unexpected MoreCount: %d", c.MoreCount)
	}
	if len(c.MoreIDs) != 3 {
		t.Errorf("expected 3 MoreIDs, got %d", len(c.MoreIDs))
	}
}

// ---------------------------------------------------------------------------
// parseComment replies="" must not panic
// ---------------------------------------------------------------------------

func TestParseComment_EmptyRepliesString_NoPanic(t *testing.T) {
	// Reddit sends replies: "" (empty string) when there are no replies.
	// The code guards with a type assertion to map[string]any, so a string
	// value must simply be ignored, not panic.
	data := map[string]any{
		"id":      "c2",
		"author":  "bob",
		"body":    "reply test",
		"score":   float64(1),
		"depth":   float64(0),
		"replies": "", // this is the edge case
	}

	c := parseComment("t1", data)
	if c == nil {
		t.Fatal("expected non-nil comment")
	}
	if len(c.Replies) != 0 {
		t.Errorf("expected 0 replies, got %d", len(c.Replies))
	}
}

// ---------------------------------------------------------------------------
// expandMoreComments calls API with correct IDs
// ---------------------------------------------------------------------------

func TestExpandMoreComments_CallsAPIWithCorrectIDs(t *testing.T) {
	// Build a thread that has a "more" placeholder with two IDs.
	moreData := map[string]any{
		"id":       "m1",
		"count":    float64(2),
		"depth":    float64(1),
		"children": []any{"x1", "x2"},
	}

	postData := map[string]any{
		"id":    "postid",
		"title": "test thread",
	}

	body := makeThreadJSON(postData, []map[string]any{
		{"_kind": "more", "id": "m1", "count": float64(2), "depth": float64(1), "children": []any{"x1", "x2"}},
	})
	_ = moreData

	var capturedChildren string

	// The morechildren response.
	moreChildrenResp, _ := json.Marshal(map[string]any{
		"json": map[string]any{
			"data": map[string]any{
				"things": []any{
					map[string]any{
						"kind": "t1",
						"data": map[string]any{
							"id":     "x1",
							"author": "user1",
							"body":   "expanded comment 1",
							"score":  float64(5),
							"depth":  float64(1),
						},
					},
				},
			},
		},
	})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/morechildren" {
			r.ParseForm()
			capturedChildren = r.FormValue("children")
			w.Write(moreChildrenResp)
			return
		}
		w.Write(body)
	}))
	defer srv.Close()

	c := newTestClient(srv)
	thread, err := c.GetThread("/r/test/comments/postid/title/", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The more placeholder should have triggered a morechildren call.
	if capturedChildren == "" {
		t.Fatal("morechildren API was never called")
	}
	// The IDs x1 and x2 should both appear in the request.
	if !contains(capturedChildren, "x1") || !contains(capturedChildren, "x2") {
		t.Errorf("morechildren call missing expected IDs; got children=%q", capturedChildren)
	}
	_ = thread
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsStr(s, sub))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
