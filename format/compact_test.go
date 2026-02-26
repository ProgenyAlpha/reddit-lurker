package format

import (
	"strings"
	"testing"

	"github.com/ProgenyAlpha/reddit-lurker/reddit"
)

// ---------------------------------------------------------------------------
// truncate
// ---------------------------------------------------------------------------

func TestTruncate(t *testing.T) {
	tests := []struct {
		name  string
		input string
		max   int
		want  string
	}{
		{
			name:  "truncates long string",
			input: "hello world",
			max:   5,
			want:  "he...",
		},
		{
			name:  "no truncation when short",
			input: "hi",
			max:   10,
			want:  "hi",
		},
		{
			name:  "exactly at boundary — no truncation",
			input: "hi",
			max:   3,
			want:  "hi",
		},
		{
			name:  "max=0 does not panic",
			input: "a",
			max:   0,
			// max <= 3 branch: returns s[:0] = "". That's the current behaviour.
			// What we care about is NO PANIC.
			want: "",
		},
		{
			name:  "exactly 3 chars — no truncation marker",
			input: "abc",
			max:   3,
			want:  "abc",
		},
		{
			name:  "newlines replaced with spaces",
			input: "hello\nworld",
			max:   20,
			want:  "hello world",
		},
		{
			name:  "empty string",
			input: "",
			max:   10,
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncate(tt.input, tt.max)
			if got != tt.want {
				t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.max, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// CompactThread — nil comments must not panic
// ---------------------------------------------------------------------------

func TestCompactThread_NilComments_NoPanic(t *testing.T) {
	thread := &reddit.Thread{
		Post: &reddit.Post{
			ID:    "abc",
			Title: "test post",
		},
		Comments: nil,
	}
	// Must not panic.
	out := CompactThread(thread)
	if out == "" {
		t.Error("expected non-empty output for valid post")
	}
}

// ---------------------------------------------------------------------------
// CompactThread — "more" comment placeholder outputs "+N" line
// ---------------------------------------------------------------------------

func TestCompactThread_MoreComment_OutputsPlusN(t *testing.T) {
	thread := &reddit.Thread{
		Post: &reddit.Post{
			ID:    "abc",
			Title: "test post",
		},
		Comments: []*reddit.Comment{
			{
				ID:        "more1",
				IsMore:    true,
				MoreCount: 42,
			},
		},
	}

	out := CompactThread(thread)
	if !strings.Contains(out, "+42") {
		t.Errorf("expected '+42' in compact output for more placeholder, got:\n%s", out)
	}
}

// ---------------------------------------------------------------------------
// flattenComments — depth prefixes
// ---------------------------------------------------------------------------

func TestFlattenComments_DepthPrefixes(t *testing.T) {
	// Build a simple two-level tree: one root comment (depth 0) with one reply (depth 1).
	root := &reddit.Comment{
		ID:     "r1",
		Author: "alice",
		Body:   "root comment",
		Score:  10,
		Depth:  0,
		Replies: []*reddit.Comment{
			{
				ID:     "r2",
				Author: "bob",
				Body:   "reply",
				Score:  5,
				Depth:  1,
			},
		},
	}

	var b strings.Builder
	flattenComments(&b, []*reddit.Comment{root})
	out := b.String()

	if !strings.Contains(out, "d0\t") {
		t.Errorf("expected 'd0\\t' prefix for depth-0 comment, got:\n%s", out)
	}
	if !strings.Contains(out, "d1\t") {
		t.Errorf("expected 'd1\\t' prefix for depth-1 reply, got:\n%s", out)
	}
}

// ---------------------------------------------------------------------------
// CompactThread — nil thread and nil post
// ---------------------------------------------------------------------------

func TestCompactThread_NilThread_ReturnsEmpty(t *testing.T) {
	if out := CompactThread(nil); out != "" {
		t.Errorf("expected empty string for nil thread, got %q", out)
	}
}

func TestCompactThread_NilPost_ReturnsEmpty(t *testing.T) {
	thread := &reddit.Thread{Post: nil}
	if out := CompactThread(thread); out != "" {
		t.Errorf("expected empty string for thread with nil post, got %q", out)
	}
}
