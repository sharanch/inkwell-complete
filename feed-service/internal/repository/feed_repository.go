package repository

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	_ "github.com/lib/pq"
)

func NewPostgres(dsn string) (*sqlx.DB, error) {
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("sqlx connect: %w", err)
	}
	return db, nil
}

func Migrate(db *sqlx.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS user_interests (
			user_id    UUID NOT NULL,
			tag        TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			PRIMARY KEY (user_id, tag)
		);

		-- Denormalised public post index for fast feed queries
		-- Populated via API call from blog-service on publish event (future: event bus)
		CREATE TABLE IF NOT EXISTS feed_posts (
			id           UUID PRIMARY KEY,
			author_id    UUID NOT NULL,
			tags         TEXT[] NOT NULL DEFAULT '{}',
			published_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			score        FLOAT NOT NULL DEFAULT 0
		);
		CREATE INDEX IF NOT EXISTS feed_posts_tags_idx ON feed_posts USING GIN(tags);
		CREATE INDEX IF NOT EXISTS feed_posts_score_idx ON feed_posts(score DESC);
	`)
	return err
}

type FeedRepository struct {
	db *sqlx.DB
}

func NewFeedRepository(db *sqlx.DB) *FeedRepository {
	return &FeedRepository{db: db}
}

func (r *FeedRepository) SetInterests(ctx context.Context, userID string, tags []string) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	tx.ExecContext(ctx, `DELETE FROM user_interests WHERE user_id=$1`, userID)
	for _, tag := range tags {
		tx.ExecContext(ctx, `INSERT INTO user_interests(user_id, tag) VALUES($1,$2) ON CONFLICT DO NOTHING`, userID, tag)
	}
	return tx.Commit()
}

func (r *FeedRepository) GetInterests(ctx context.Context, userID string) ([]string, error) {
	var tags []string
	rows, err := r.db.QueryContext(ctx, `SELECT tag FROM user_interests WHERE user_id=$1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var t string
		rows.Scan(&t)
		tags = append(tags, t)
	}
	return tags, nil
}

// GetFeedPostIDs returns ranked post IDs for the user.
// If interests is empty, returns the global trending feed.
func (r *FeedRepository) GetFeedPostIDs(ctx context.Context, userID string, interests []string, page, pageSize int) ([]string, error) {
	offset := (page - 1) * pageSize
	var rows *sqlx.Rows
	var err error

	if len(interests) == 0 {
		rows, err = r.db.QueryxContext(ctx,
			`SELECT id FROM feed_posts ORDER BY score DESC, published_at DESC LIMIT $1 OFFSET $2`,
			pageSize, offset,
		)
	} else {
		rows, err = r.db.QueryxContext(ctx,
			`SELECT id FROM feed_posts WHERE tags && $1 ORDER BY score DESC, published_at DESC LIMIT $2 OFFSET $3`,
			pq.Array(interests), pageSize, offset,
		)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		rows.Scan(&id)
		ids = append(ids, id)
	}
	return ids, nil
}

// IndexPost adds or updates a post in the feed index (called when a post is published).
func (r *FeedRepository) IndexPost(ctx context.Context, id, authorID string, tags []string) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO feed_posts(id, author_id, tags, score)
		VALUES($1, $2, $3, 0)
		ON CONFLICT(id) DO UPDATE SET tags=$3`,
		id, authorID, pq.Array(tags),
	)
	return err
}

// UpdateScore bumps score when a post gets a like or view (denormalised from blog-service).
func (r *FeedRepository) UpdateScore(ctx context.Context, postID string, delta float64) error {
	_, err := r.db.ExecContext(ctx, `UPDATE feed_posts SET score = score + $1 WHERE id=$2`, delta, postID)
	return err
}
