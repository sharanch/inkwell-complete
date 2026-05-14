package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/sharanch/inkwell/auth-service/config"
	"github.com/sharanch/inkwell/auth-service/internal/model"
)

var (
	ErrInvalidOTP   = errors.New("invalid or expired OTP")
	ErrRateLimited  = errors.New("too many OTP requests, please wait")
	ErrInvalidToken = errors.New("invalid or expired token")
)

// ─── Interfaces (enable unit testing without a real DB or Redis) ─────────────

type userRepository interface {
	FindOrCreate(ctx context.Context, email string) (*model.User, error)
	GetByID(ctx context.Context, id string) (*model.User, error)
}

type otpRepository interface {
	Store(ctx context.Context, email, code string, ttl time.Duration) error
	Verify(ctx context.Context, email, code string) (bool, error)
	RateLimit(ctx context.Context, email string) (bool, error)
}

type notifyClient interface {
	SendOTP(ctx context.Context, email, code string) error
}

// ─── Service ─────────────────────────────────────────────────────────────────

type AuthService struct {
	users  userRepository
	otps   otpRepository
	notify notifyClient
	cfg    *config.Config
}

func NewAuthService(users userRepository, otps otpRepository, notify notifyClient, cfg *config.Config) *AuthService {
	return &AuthService{users: users, otps: otps, notify: notify, cfg: cfg}
}

// RequestOTP generates a 6-digit code, stores it in Redis, and sends it via notify-service.
func (s *AuthService) RequestOTP(ctx context.Context, email string) error {
	allowed, err := s.otps.RateLimit(ctx, email)
	if err != nil {
		return fmt.Errorf("rate limit check: %w", err)
	}
	if !allowed {
		return ErrRateLimited
	}

	code, err := generateOTP(6)
	if err != nil {
		return fmt.Errorf("generate otp: %w", err)
	}

	ttl := time.Duration(s.cfg.OTPTTLMinutes) * time.Minute
	if err := s.otps.Store(ctx, email, code, ttl); err != nil {
		return fmt.Errorf("store otp: %w", err)
	}

	if err := s.notify.SendOTP(ctx, email, code); err != nil {
		// Non-fatal for development; log and continue
		fmt.Printf("WARN: notify failed: %v\n", err)
	}

	return nil
}

// VerifyOTP validates the OTP, upserts the user, and returns a token pair.
func (s *AuthService) VerifyOTP(ctx context.Context, email, code string) (*model.TokenPair, error) {
	valid, err := s.otps.Verify(ctx, email, code)
	if err != nil {
		return nil, fmt.Errorf("verify otp: %w", err)
	}
	if !valid {
		return nil, ErrInvalidOTP
	}

	user, err := s.users.FindOrCreate(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("find or create user: %w", err)
	}

	pair, err := s.issueTokens(user)
	if err != nil {
		return nil, err
	}
	pair.User = user
	return pair, nil
}

// RefreshToken validates a refresh token and issues a new pair.
func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*model.TokenPair, error) {
	token, err := jwt.Parse(refreshToken, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return []byte(s.cfg.JWTRefreshSecret), nil
	})
	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, ErrInvalidToken
	}

	userID, _ := claims["user_id"].(string)
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	pair, err := s.issueTokens(user)
	if err != nil {
		return nil, err
	}
	pair.User = user
	return pair, nil
}

// ValidateToken parses and validates an access token, returning its claims.
func (s *AuthService) ValidateToken(tokenStr string) (*model.Claims, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return []byte(s.cfg.JWTSecret), nil
	})
	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}

	mc, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, ErrInvalidToken
	}
	return &model.Claims{
		UserID: mc["user_id"].(string),
		Email:  mc["email"].(string),
	}, nil
}

func (s *AuthService) issueTokens(user *model.User) (*model.TokenPair, error) {
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
		"email":   user.Email,
		"exp":     time.Now().Add(15 * time.Minute).Unix(),
	})
	accessStr, err := accessToken.SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return nil, err
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
		"exp":     time.Now().Add(7 * 24 * time.Hour).Unix(),
	})
	refreshStr, err := refreshToken.SignedString([]byte(s.cfg.JWTRefreshSecret))
	if err != nil {
		return nil, err
	}

	return &model.TokenPair{
		AccessToken:  accessStr,
		RefreshToken: refreshStr,
	}, nil
}

func generateOTP(digits int) (string, error) {
	max := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(digits)), nil)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%0*d", digits, n), nil
}
