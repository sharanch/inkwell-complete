package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"

	"github.com/sharanch/inkwell/auth-service/internal/model"
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
		CREATE TABLE IF NOT EXISTS users (
			id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			email      TEXT NOT NULL UNIQUE,
			name       TEXT NOT NULL DEFAULT '',
			bio        TEXT NOT NULL DEFAULT '',
			avatar_url TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
	`)
	return err
}

type UserRepository struct {
	db *sqlx.DB
}

func NewUserRepository(db *sqlx.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) FindOrCreate(ctx context.Context, email string) (*model.User, error) {
	user := &model.User{}
	err := r.db.GetContext(ctx, user, `SELECT * FROM users WHERE email = $1`, email)
	if err == nil {
		return user, nil
	}
	if err != sql.ErrNoRows {
		return nil, fmt.Errorf("find user: %w", err)
	}

	// First login — create account
	user.ID = uuid.New().String()
	user.Email = email
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO users (id, email) VALUES ($1, $2) ON CONFLICT (email) DO NOTHING`,
		user.ID, user.Email)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	// Re-fetch to get db defaults
	return r.GetByEmail(ctx, email)
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (*model.User, error) {
	user := &model.User{}
	if err := r.db.GetContext(ctx, user, `SELECT * FROM users WHERE id = $1`, id); err != nil {
		return nil, err
	}
	return user, nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	user := &model.User{}
	if err := r.db.GetContext(ctx, user, `SELECT * FROM users WHERE email = $1`, email); err != nil {
		return nil, err
	}
	return user, nil
}

func (r *UserRepository) UpdateProfile(ctx context.Context, id, name, bio, avatarURL string) (*model.User, error) {
	_, err := r.db.ExecContext(ctx, `
		UPDATE users SET name=$1, bio=$2, avatar_url=$3 WHERE id=$4`,
		name, bio, avatarURL, id)
	if err != nil {
		return nil, err
	}
	return r.GetByID(ctx, id)
}
