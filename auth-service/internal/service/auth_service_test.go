package service

import (
	"context"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/sharanch/inkwell/auth-service/config"
	"github.com/sharanch/inkwell/auth-service/internal/model"
)

// ─── Fakes ───────────────────────────────────────────────────────────────────

// fakeUserRepo implements userRepository without a real database.
type fakeUserRepo struct {
	users map[string]*model.User
}

func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{users: make(map[string]*model.User)}
}

func (f *fakeUserRepo) FindOrCreate(_ context.Context, email string) (*model.User, error) {
	if u, ok := f.users[email]; ok {
		return u, nil
	}
	u := &model.User{ID: "user-" + email, Email: email}
	f.users[email] = u
	return u, nil
}

func (f *fakeUserRepo) GetByID(_ context.Context, id string) (*model.User, error) {
	for _, u := range f.users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, ErrInvalidToken
}

// fakeOTPRepo is an in-memory OTP store — no Redis required.
type fakeOTPRepo struct {
	codes     map[string]string
	callCount map[string]int
}

func newFakeOTPRepo() *fakeOTPRepo {
	return &fakeOTPRepo{
		codes:     make(map[string]string),
		callCount: make(map[string]int),
	}
}

func (f *fakeOTPRepo) Store(_ context.Context, email, code string, _ time.Duration) error {
	f.codes[email] = code
	return nil
}

func (f *fakeOTPRepo) Verify(_ context.Context, email, code string) (bool, error) {
	stored, ok := f.codes[email]
	if !ok || stored != code {
		return false, nil
	}
	delete(f.codes, email) // one-time use
	return true, nil
}

func (f *fakeOTPRepo) RateLimit(_ context.Context, email string) (bool, error) {
	f.callCount[email]++
	return f.callCount[email] <= 3, nil
}

// fakeNotifyClient swallows SendOTP calls — no HTTP required.
type fakeNotifyClient struct{}

func (f *fakeNotifyClient) SendOTP(_ context.Context, _, _ string) error { return nil }

// ─── Helpers ─────────────────────────────────────────────────────────────────

func testConfig() *config.Config {
	return &config.Config{
		JWTSecret:        "test-jwt-secret-at-least-32-bytes!",
		JWTRefreshSecret: "test-refresh-secret-at-least-32!!",
		OTPTTLMinutes:    10,
	}
}

func newTestService() (*AuthService, *fakeOTPRepo, *fakeUserRepo) {
	userRepo := newFakeUserRepo()
	otpRepo := newFakeOTPRepo()
	svc := NewAuthService(userRepo, otpRepo, &fakeNotifyClient{}, testConfig())
	return svc, otpRepo, userRepo
}

// ─── generateOTP ─────────────────────────────────────────────────────────────

func TestGenerateOTP_Length(t *testing.T) {
	for i := 0; i < 20; i++ {
		code, err := generateOTP(6)
		if err != nil {
			t.Fatalf("generateOTP returned error: %v", err)
		}
		if len(code) != 6 {
			t.Errorf("expected 6-digit code, got %q (len=%d)", code, len(code))
		}
	}
}

func TestGenerateOTP_OnlyDigits(t *testing.T) {
	for i := 0; i < 50; i++ {
		code, err := generateOTP(6)
		if err != nil {
			t.Fatalf("generateOTP returned error: %v", err)
		}
		for _, c := range code {
			if c < '0' || c > '9' {
				t.Errorf("non-digit character %q in OTP %q", c, code)
			}
		}
	}
}

func TestGenerateOTP_UsesSecureRandom(t *testing.T) {
	// Generate a batch and verify there is more than one unique value.
	// A fixed-seed math/rand would produce the same value every time.
	seen := make(map[string]struct{})
	for i := 0; i < 10; i++ {
		code, err := generateOTP(6)
		if err != nil {
			t.Fatalf("generateOTP returned error: %v", err)
		}
		seen[code] = struct{}{}
	}
	if len(seen) == 1 {
		t.Error("all 10 OTPs were identical — likely not using crypto/rand")
	}
}

// ─── RequestOTP ──────────────────────────────────────────────────────────────

func TestRequestOTP_StoresCode(t *testing.T) {
	svc, otpRepo, _ := newTestService()
	if err := svc.RequestOTP(context.Background(), "user@example.com"); err != nil {
		t.Fatalf("RequestOTP error: %v", err)
	}
	if _, ok := otpRepo.codes["user@example.com"]; !ok {
		t.Error("expected OTP to be stored for user@example.com")
	}
}

func TestRequestOTP_RateLimitEnforced(t *testing.T) {
	svc, _, _ := newTestService()
	ctx := context.Background()

	// First 3 requests must succeed
	for i := 0; i < 3; i++ {
		if err := svc.RequestOTP(ctx, "ratelimited@example.com"); err != nil {
			t.Fatalf("request %d failed unexpectedly: %v", i+1, err)
		}
	}
	// 4th request must be rejected
	if err := svc.RequestOTP(ctx, "ratelimited@example.com"); err != ErrRateLimited {
		t.Errorf("expected ErrRateLimited on 4th request, got %v", err)
	}
}

// ─── VerifyOTP ───────────────────────────────────────────────────────────────

func TestVerifyOTP_ValidCode_ReturnsTokenPair(t *testing.T) {
	svc, otpRepo, _ := newTestService()
	email := "verify@example.com"
	otpRepo.codes[email] = "123456"

	pair, err := svc.VerifyOTP(context.Background(), email, "123456")
	if err != nil {
		t.Fatalf("VerifyOTP error: %v", err)
	}
	if pair.AccessToken == "" {
		t.Error("expected non-empty access token")
	}
	if pair.RefreshToken == "" {
		t.Error("expected non-empty refresh token")
	}
	if pair.User == nil || pair.User.Email != email {
		t.Errorf("expected user email %q, got %v", email, pair.User)
	}
}

func TestVerifyOTP_InvalidCode_ReturnsError(t *testing.T) {
	svc, otpRepo, _ := newTestService()
	otpRepo.codes["bad@example.com"] = "999999"

	_, err := svc.VerifyOTP(context.Background(), "bad@example.com", "000000")
	if err != ErrInvalidOTP {
		t.Errorf("expected ErrInvalidOTP, got %v", err)
	}
}

func TestVerifyOTP_OneTimeUse(t *testing.T) {
	svc, otpRepo, _ := newTestService()
	email := "once@example.com"
	otpRepo.codes[email] = "654321"

	// First verify must succeed
	if _, err := svc.VerifyOTP(context.Background(), email, "654321"); err != nil {
		t.Fatalf("first verify failed: %v", err)
	}
	// Second verify with the same code must fail
	if _, err := svc.VerifyOTP(context.Background(), email, "654321"); err != ErrInvalidOTP {
		t.Errorf("expected ErrInvalidOTP on second use, got %v", err)
	}
}

// ─── RefreshToken ────────────────────────────────────────────────────────────

func TestRefreshToken_ValidToken_ReturnsNewPair(t *testing.T) {
	svc, otpRepo, _ := newTestService()
	ctx := context.Background()

	otpRepo.codes["refresh@example.com"] = "111111"
	pair, err := svc.VerifyOTP(ctx, "refresh@example.com", "111111")
	if err != nil {
		t.Fatalf("setup VerifyOTP: %v", err)
	}

	newPair, err := svc.RefreshToken(ctx, pair.RefreshToken)
	if err != nil {
		t.Fatalf("RefreshToken error: %v", err)
	}
	if newPair.AccessToken == "" {
		t.Error("expected non-empty access token from refresh")
	}
}

func TestRefreshToken_InvalidToken_ReturnsError(t *testing.T) {
	svc, _, _ := newTestService()
	_, err := svc.RefreshToken(context.Background(), "not.a.real.token")
	if err != ErrInvalidToken {
		t.Errorf("expected ErrInvalidToken, got %v", err)
	}
}

func TestRefreshToken_ExpiredToken_ReturnsError(t *testing.T) {
	svc, _, _ := newTestService()
	cfg := testConfig()

	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": "ghost-user",
		"exp":     time.Now().Add(-1 * time.Hour).Unix(), // already expired
	})
	signed, _ := tok.SignedString([]byte(cfg.JWTRefreshSecret))

	_, err := svc.RefreshToken(context.Background(), signed)
	if err != ErrInvalidToken {
		t.Errorf("expected ErrInvalidToken for expired token, got %v", err)
	}
}

// ─── ValidateToken ────────────────────────────────────────────────────────────

func TestValidateToken_ValidToken_ReturnsClaims(t *testing.T) {
	svc, otpRepo, _ := newTestService()
	otpRepo.codes["validate@example.com"] = "777777"

	pair, err := svc.VerifyOTP(context.Background(), "validate@example.com", "777777")
	if err != nil {
		t.Fatalf("setup: %v", err)
	}

	claims, err := svc.ValidateToken(pair.AccessToken)
	if err != nil {
		t.Fatalf("ValidateToken error: %v", err)
	}
	if claims.Email != "validate@example.com" {
		t.Errorf("expected email validate@example.com, got %q", claims.Email)
	}
}

func TestValidateToken_TamperedToken_ReturnsError(t *testing.T) {
	svc, _, _ := newTestService()
	_, err := svc.ValidateToken("eyJhbGciOiJIUzI1NiJ9.tampered.signature")
	if err != ErrInvalidToken {
		t.Errorf("expected ErrInvalidToken for tampered token, got %v", err)
	}
}
