package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	_ "github.com/lib/pq"

	"github.com/sharanch/inkwell/blog-service/internal/model"
)

func NewPostgres(dsn string) (*sqlx.DB, error) {
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("sqlx connect: %w", err)
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	return db, nil
}

func Migrate(db *sqlx.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS posts (
			id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			author_id    UUID NOT NULL,
			title        TEXT NOT NULL,
			slug         TEXT NOT NULL,
			body         TEXT NOT NULL DEFAULT '',
			excerpt      TEXT NOT NULL DEFAULT '',
			cover_url    TEXT NOT NULL DEFAULT '',
			visibility   TEXT NOT NULL DEFAULT 'private' CHECK (visibility IN ('public','private')),
			tags         TEXT[] NOT NULL DEFAULT '{}',
			likes_count  INT NOT NULL DEFAULT 0,
			views_count  INT NOT NULL DEFAULT 0,
			reading_mins INT NOT NULL DEFAULT 1,
			published_at TIMESTAMPTZ,
			created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
		CREATE INDEX IF NOT EXISTS posts_author_idx ON posts(author_id);
		CREATE INDEX IF NOT EXISTS posts_visibility_idx ON posts(visibility);
		CREATE INDEX IF NOT EXISTS posts_tags_idx ON posts USING GIN(tags);

		CREATE TABLE IF NOT EXISTS post_likes (
			post_id   UUID NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
			user_id   UUID NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			PRIMARY KEY (post_id, user_id)
		);
	`)
	return err
}

type PostRepository struct {
	db *sqlx.DB
}

func NewPostRepository(db *sqlx.DB) *PostRepository {
	return &PostRepository{db: db}
}

func (r *PostRepository) Create(ctx context.Context, authorID string, req *model.CreatePostRequest) (*model.Post, error) {
	excerpt := req.Body
	if len(excerpt) > 200 {
		excerpt = excerpt[:200] + "…"
	}
	readingMins := max(1, len(strings.Fields(req.Body))/200)
	visibility := req.Visibility
	if visibility == "" {
		visibility = model.VisibilityPrivate
	}

	var publishedAt *time.Time
	if visibility == model.VisibilityPublic {
		now := time.Now()
		publishedAt = &now
	}

	post := &model.Post{}
	err := r.db.QueryRowxContext(ctx, `
		INSERT INTO posts (id, author_id, title, slug, body, excerpt, cover_url, visibility, tags, reading_mins, published_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING *`,
		uuid.New().String(), authorID, req.Title, slugify(req.Title),
		req.Body, excerpt, req.CoverURL, string(visibility),
		pq.Array(req.Tags), readingMins, publishedAt,
	).StructScan(post)
	return post, err
}

func (r *PostRepository) Update(ctx context.Context, id, authorID string, req *model.UpdatePostRequest) (*model.Post, error) {
	excerpt := req.Body
	if len(excerpt) > 200 {
		excerpt = excerpt[:200] + "…"
	}
	readingMins := max(1, len(strings.Fields(req.Body))/200)

	var publishedAt *time.Time
	if req.Visibility == model.VisibilityPublic {
		now := time.Now()
		publishedAt = &now
	}

	post := &model.Post{}
	err := r.db.QueryRowxContext(ctx, `
		UPDATE posts
		SET title=$1, body=$2, excerpt=$3, cover_url=$4, visibility=$5, tags=$6,
		    reading_mins=$7, published_at=COALESCE(published_at, $8), updated_at=NOW()
		WHERE id=$9 AND author_id=$10
		RETURNING *`,
		req.Title, req.Body, excerpt, req.CoverURL, string(req.Visibility),
		pq.Array(req.Tags), readingMins, publishedAt, id, authorID,
	).StructScan(post)
	return post, err
}

func (r *PostRepository) Delete(ctx context.Context, id, authorID string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM posts WHERE id=$1 AND author_id=$2`, id, authorID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("post not found or not owned by user")
	}
	return nil
}

func (r *PostRepository) GetByID(ctx context.Context, id string) (*model.Post, error) {
	post := &model.Post{}
	err := r.db.QueryRowxContext(ctx, `SELECT * FROM posts WHERE id=$1`, id).StructScan(post)
	return post, err
}

func (r *PostRepository) List(ctx context.Context, q *model.ListPostsQuery) (*model.PostsPage, error) {
	where := []string{"1=1"}
	args := []interface{}{}
	i := 1

	if q.AuthorID != "" {
		where = append(where, fmt.Sprintf("author_id=$%d", i))
		args = append(args, q.AuthorID)
		i++
	}
	if q.Visibility != "" {
		where = append(where, fmt.Sprintf("visibility=$%d", i))
		args = append(args, string(q.Visibility))
		i++
	}
	if q.Tag != "" {
		where = append(where, fmt.Sprintf("$%d=ANY(tags)", i))
		args = append(args, q.Tag)
		i++
	}

	clause := strings.Join(where, " AND ")
	offset := (q.Page - 1) * q.PageSize

	var total int
	r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM posts WHERE "+clause, args...).Scan(&total)

	rows, err := r.db.QueryxContext(ctx,
		fmt.Sprintf("SELECT * FROM posts WHERE %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d", clause, i, i+1),
		append(args, q.PageSize, offset)...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	posts := []*model.Post{}
	for rows.Next() {
		p := &model.Post{}
		if err := rows.StructScan(p); err != nil {
			return nil, err
		}
		posts = append(posts, p)
	}

	return &model.PostsPage{Posts: posts, Total: total, Page: q.Page, PageSize: q.PageSize}, nil
}

func (r *PostRepository) ToggleLike(ctx context.Context, postID, userID string) (bool, error) {
	var exists bool
	r.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM post_likes WHERE post_id=$1 AND user_id=$2)`, postID, userID).Scan(&exists)

	if exists {
		r.db.ExecContext(ctx, `DELETE FROM post_likes WHERE post_id=$1 AND user_id=$2`, postID, userID)
		r.db.ExecContext(ctx, `UPDATE posts SET likes_count = likes_count - 1 WHERE id=$1`, postID)
		return false, nil
	}
	r.db.ExecContext(ctx, `INSERT INTO post_likes(post_id, user_id) VALUES($1,$2) ON CONFLICT DO NOTHING`, postID, userID)
	r.db.ExecContext(ctx, `UPDATE posts SET likes_count = likes_count + 1 WHERE id=$1`, postID)
	return true, nil
}

func (r *PostRepository) IncrementView(ctx context.Context, postID string) {
	r.db.ExecContext(ctx, `UPDATE posts SET views_count = views_count + 1 WHERE id=$1`, postID)
}

// GetByIDs fetches multiple posts for the feed service.
func (r *PostRepository) GetByIDs(ctx context.Context, ids []string) ([]*model.Post, error) {
	query, args, err := sqlx.In(`SELECT * FROM posts WHERE id IN (?) AND visibility='public'`, ids)
	if err != nil {
		return nil, err
	}
	query = r.db.Rebind(query)
	rows, err := r.db.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	posts := []*model.Post{}
	for rows.Next() {
		p := &model.Post{}
		rows.StructScan(p)
		posts = append(posts, p)
	}
	return posts, nil
}

func slugify(title string) string {
	s := strings.ToLower(title)
	s = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			return r
		}
		return '-'
	}, s)
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	return strings.Trim(s, "-")
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
