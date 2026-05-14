package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

func NewRedis(addr string) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{Addr: addr})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("redis ping: %w", err)
	}
	return rdb, nil
}

type OTPRepository struct {
	rdb *redis.Client
}

func NewOTPRepository(rdb *redis.Client) *OTPRepository {
	return &OTPRepository{rdb: rdb}
}

func otpKey(email string) string  { return "otp:" + email }
func rlKey(email string) string   { return "otp_rl:" + email }

// Store saves an OTP code for an email with a TTL.
func (r *OTPRepository) Store(ctx context.Context, email, code string, ttl time.Duration) error {
	return r.rdb.Set(ctx, otpKey(email), code, ttl).Err()
}

// Verify checks the OTP and deletes it if valid (one-time use).
func (r *OTPRepository) Verify(ctx context.Context, email, code string) (bool, error) {
	stored, err := r.rdb.Get(ctx, otpKey(email)).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if stored != code {
		return false, nil
	}
	r.rdb.Del(ctx, otpKey(email))
	return true, nil
}

// RateLimit returns true if the email is allowed to request another OTP.
// Allows 3 requests per 15 minutes.
func (r *OTPRepository) RateLimit(ctx context.Context, email string) (bool, error) {
	key := rlKey(email)
	count, err := r.rdb.Incr(ctx, key).Result()
	if err != nil {
		return false, err
	}
	if count == 1 {
		r.rdb.Expire(ctx, key, 15*time.Minute)
	}
	return count <= 3, nil
}
