package reddit

import "time"

// Listing is the Reddit API envelope for paginated results.
type Listing struct {
	Kind string      `json:"kind"`
	Data ListingData `json:"data"`
}

type ListingData struct {
	After    string  `json:"after"`
	Before   string  `json:"before"`
	Children []Thing `json:"children"`
}

// Thing wraps every Reddit object (post, comment, more, etc.)
type Thing struct {
	Kind string          `json:"kind"`
	Data map[string]any  `json:"data"`
}

// Post represents a parsed Reddit post.
type Post struct {
	ID            string
	Subreddit     string
	Author        string
	Title         string
	SelfText      string
	Score         int
	UpvoteRatio   float64
	NumComments   int
	URL           string
	Permalink     string
	Domain        string
	Created       time.Time
	IsVideo       bool
	IsSelf        bool
	Over18        bool
	Stickied      bool
	LinkFlairText string
	CrosspostParent string
	MediaURL      string
}

// Comment represents a parsed Reddit comment.
type Comment struct {
	ID       string
	ParentID string // parent comment/post ID (used by morechildren insertion)
	Author   string
	Body     string
	Score    int
	Created  time.Time
	Depth    int
	Stickied bool
	Replies  []*Comment
	IsMore   bool   // true if this represents a "load more" placeholder
	MoreIDs  []string // IDs of collapsed comments
	MoreCount int
}

// Thread is a post with its full comment tree.
type Thread struct {
	Post     *Post
	Comments []*Comment
}

// SubredditInfo holds metadata about a subreddit.
type SubredditInfo struct {
	Name           string
	Title          string
	Description    string
	Subscribers    int
	ActiveUsers    int
	Created        time.Time
	Over18         bool
	SubredditType  string // public, private, restricted
}

// UserInfo holds metadata about a user.
type UserInfo struct {
	Name         string
	Created      time.Time
	LinkKarma    int
	CommentKarma int
	TotalKarma   int
	IsSuspended  bool
}

// SearchResult wraps a post found via search with its source subreddit.
type SearchResult struct {
	Post *Post
}

// SortOrder for subreddit/search listings.
type SortOrder string

const (
	SortHot           SortOrder = "hot"
	SortNew           SortOrder = "new"
	SortTop           SortOrder = "top"
	SortRising        SortOrder = "rising"
	SortControversial SortOrder = "controversial"
	SortRelevance     SortOrder = "relevance"
	SortComments      SortOrder = "comments"
)

// TimeFilter for top/controversial sorting.
type TimeFilter string

const (
	TimeHour  TimeFilter = "hour"
	TimeDay   TimeFilter = "day"
	TimeWeek  TimeFilter = "week"
	TimeMonth TimeFilter = "month"
	TimeYear  TimeFilter = "year"
	TimeAll   TimeFilter = "all"
)
