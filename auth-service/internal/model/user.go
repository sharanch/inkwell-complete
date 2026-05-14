package model

import "time"

type User struct {
	ID        string    `db:"id" json:"id"`
	Email     string    `db:"email" json:"email"`
	Name      string    `db:"name" json:"name"`
	Bio       string    `db:"bio" json:"bio"`
	AvatarURL string    `db:"avatar_url" json:"avatar_url"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

// OTPRequest is the payload to request a one-time password via email.
type OTPRequest struct {
	Email string `json:"email"`
}

// OTPVerifyRequest is the payload to verify the OTP code.
type OTPVerifyRequest struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

// TokenPair is the access + refresh token pair returned after successful login.
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	User         *User  `json:"user"`
}

// RefreshRequest is the payload to exchange a refresh token for a new pair.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// Claims is the JWT payload stored in the access token.
type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
}
