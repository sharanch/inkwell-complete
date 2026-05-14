package model

import (
	"time"

	"github.com/lib/pq"
)

type Visibility string

const (
	VisibilityPublic  Visibility = "public"
	VisibilityPrivate Visibility = "private"
)

type Post struct {
	ID          string         `db:"id"           json:"id"`
	AuthorID    string         `db:"author_id"    json:"author_id"`
	AuthorName  string         `db:"author_name"  json:"author_name,omitempty"`
	Title       string         `db:"title"        json:"title"`
	Slug        string         `db:"slug"         json:"slug"`
	Body        string         `db:"body"         json:"body"`
	Excerpt     string         `db:"excerpt"      json:"excerpt"`
	CoverURL    string         `db:"cover_url"    json:"cover_url"`
	Visibility  Visibility     `db:"visibility"   json:"visibility"`
	Tags        pq.StringArray `db:"tags"         json:"tags"`
	LikesCount  int            `db:"likes_count"  json:"likes_count"`
	ViewsCount  int            `db:"views_count"  json:"views_count"`
	ReadingMins int            `db:"reading_mins" json:"reading_mins"`
	PublishedAt *time.Time     `db:"published_at" json:"published_at"`
	CreatedAt   time.Time      `db:"created_at"   json:"created_at"`
	UpdatedAt   time.Time      `db:"updated_at"   json:"updated_at"`
}

type CreatePostRequest struct {
	Title      string     `json:"title"`
	Body       string     `json:"body"`
	CoverURL   string     `json:"cover_url"`
	Visibility Visibility `json:"visibility"`
	Tags       []string   `json:"tags"`
}

type UpdatePostRequest struct {
	Title      string     `json:"title"`
	Body       string     `json:"body"`
	CoverURL   string     `json:"cover_url"`
	Visibility Visibility `json:"visibility"`
	Tags       []string   `json:"tags"`
}

type ListPostsQuery struct {
	AuthorID   string
	Visibility Visibility
	Tag        string
	Page       int
	PageSize   int
}

type PostsPage struct {
	Posts    []*Post `json:"posts"`
	Total    int     `json:"total"`
	Page     int     `json:"page"`
	PageSize int     `json:"page_size"`
}
