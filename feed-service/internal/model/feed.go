package model

import "time"

type FeedPost struct {
	ID          string     `json:"id"`
	AuthorID    string     `json:"author_id"`
	AuthorName  string     `json:"author_name"`
	Title       string     `json:"title"`
	Slug        string     `json:"slug"`
	Excerpt     string     `json:"excerpt"`
	CoverURL    string     `json:"cover_url"`
	Tags        []string   `json:"tags"`
	LikesCount  int        `json:"likes_count"`
	ViewsCount  int        `json:"views_count"`
	ReadingMins int        `json:"reading_mins"`
	PublishedAt *time.Time `json:"published_at"`
}

type FeedPage struct {
	Posts    []*FeedPost `json:"posts"`
	Page     int         `json:"page"`
	PageSize int         `json:"page_size"`
	Cached   bool        `json:"cached"`
}

type SetInterestsRequest struct {
	Tags []string `json:"tags"`
}
