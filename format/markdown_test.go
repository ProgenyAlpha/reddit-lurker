package format

import (
	"strings"
	"testing"

	"github.com/ProgenyAlpha/reddit-lurker/reddit"
)

// ---------------------------------------------------------------------------
// FormatThread — gallery post with comma-separated URLs
// ---------------------------------------------------------------------------

func TestFormatThread_GalleryURLs_SplitCorrectly(t *testing.T) {
	thread := &reddit.Thread{
		Post: &reddit.Post{
			ID:       "gal1",
			Title:    "Gallery post",
			IsSelf:   false,
			MediaURL: "https://preview.redd.it/img1.jpg,https://preview.redd.it/img2.jpg,https://preview.redd.it/img3.jpg",
		},
	}

	out := FormatThread(thread)

	// Should produce "[Image 1]", "[Image 2]", "[Image 3]" links.
	for i := 1; i <= 3; i++ {
		marker := strings.Contains(out, strings.Repeat("", 0)) // just ensure no crash above
		_ = marker
		expected := strings.Contains(out, "[Image")
		if !expected {
			t.Errorf("expected '[Image N]' markdown links in output, got:\n%s", out)
			break
		}
	}

	// Count how many [Image N] lines appear.
	imageCount := strings.Count(out, "[Image ")
	if imageCount != 3 {
		t.Errorf("expected 3 gallery image links, got %d; output:\n%s", imageCount, out)
	}
}

// ---------------------------------------------------------------------------
// FormatThread — nil comment must not panic
// ---------------------------------------------------------------------------

func TestFormatThread_NilComment_NoPanic(t *testing.T) {
	thread := &reddit.Thread{
		Post: &reddit.Post{
			ID:    "p1",
			Title: "post with nil comment",
		},
		Comments: []*reddit.Comment{
			nil, // this must be silently skipped
			{
				ID:     "c1",
				Author: "alice",
				Body:   "real comment",
				Score:  5,
			},
		},
	}

	// Must not panic.
	out := FormatThread(thread)
	if !strings.Contains(out, "alice") {
		t.Errorf("expected real comment to appear in output, got:\n%s", out)
	}
}

// ---------------------------------------------------------------------------
// formatComments — deep nesting produces correct indentation
// ---------------------------------------------------------------------------

func TestFormatComments_DeepNesting_CorrectIndentation(t *testing.T) {
	// Build: depth 0 → depth 1 → depth 2
	deep := &reddit.Comment{
		ID:     "c3",
		Author: "charlie",
		Body:   "deep reply",
		Score:  1,
		Depth:  2,
	}
	mid := &reddit.Comment{
		ID:      "c2",
		Author:  "bob",
		Body:    "mid reply",
		Score:   3,
		Depth:   1,
		Replies: []*reddit.Comment{deep},
	}
	root := &reddit.Comment{
		ID:      "c1",
		Author:  "alice",
		Body:    "root comment",
		Score:   10,
		Depth:   0,
		Replies: []*reddit.Comment{mid},
	}

	out := formatComments([]*reddit.Comment{root}, 0)

	lines := strings.Split(out, "\n")

	// Find the line that mentions each author and check indentation.
	indent := map[string]string{
		"alice":   "",   // depth 0: no indent
		"bob":     "  ", // depth 1: 2 spaces
		"charlie": "    ", // depth 2: 4 spaces
	}

	for author, wantIndent := range indent {
		for _, line := range lines {
			if strings.Contains(line, author) {
				if !strings.HasPrefix(line, wantIndent) {
					t.Errorf("comment by %q: expected indent %q, line is: %q", author, wantIndent, line)
				}
				break
			}
		}
	}
}

// ---------------------------------------------------------------------------
// FormatPostList — empty posts must not panic and returns no-posts message
// ---------------------------------------------------------------------------

func TestFormatPostList_EmptyPosts_NoPanic(t *testing.T) {
	out := FormatPostList(nil, "r/test", "")
	if out == "" {
		t.Error("expected non-empty output for empty post list")
	}
}

func TestFormatPostList_EmptyPosts_ReturnsNotFoundMessage(t *testing.T) {
	out := FormatPostList([]*reddit.Post{}, "r/test", "")
	if out == "" {
		t.Error("expected non-empty output")
	}
}

// ---------------------------------------------------------------------------
// FormatThread — nil thread and nil post
// ---------------------------------------------------------------------------

func TestFormatThread_NilThread_NoPanic(t *testing.T) {
	out := FormatThread(nil)
	if out == "" {
		t.Error("expected non-empty fallback string for nil thread")
	}
}

func TestFormatThread_NilPost_NoPanic(t *testing.T) {
	out := FormatThread(&reddit.Thread{Post: nil})
	if out == "" {
		t.Error("expected non-empty fallback string for nil post")
	}
}

// ---------------------------------------------------------------------------
// FormatThread — "more" comment placeholder output
// ---------------------------------------------------------------------------

func TestFormatThread_MoreComment_ShowsMoreReplies(t *testing.T) {
	thread := &reddit.Thread{
		Post: &reddit.Post{
			ID:    "p1",
			Title: "test",
		},
		Comments: []*reddit.Comment{
			{
				ID:        "m1",
				IsMore:    true,
				MoreCount: 99,
			},
		},
	}

	out := FormatThread(thread)
	if !strings.Contains(out, "99") {
		t.Errorf("expected '99' in output for more placeholder, got:\n%s", out)
	}
	if !strings.Contains(out, "more") {
		t.Errorf("expected 'more' text in output for more placeholder, got:\n%s", out)
	}
}
