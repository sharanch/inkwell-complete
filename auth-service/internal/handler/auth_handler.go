package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/sharanch/inkwell/auth-service/internal/model"
	"github.com/sharanch/inkwell/auth-service/internal/service"
)

type AuthHandler struct {
	svc *service.AuthService
}

func NewAuthHandler(svc *service.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

func (h *AuthHandler) RequestOTP(w http.ResponseWriter, r *http.Request) {
	var req model.OTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Email == "" {
		writeError(w, http.StatusBadRequest, "valid email required")
		return
	}

	if err := h.svc.RequestOTP(r.Context(), req.Email); err != nil {
		if errors.Is(err, service.ErrRateLimited) {
			writeError(w, http.StatusTooManyRequests, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to send OTP")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "OTP sent to " + req.Email})
}

func (h *AuthHandler) VerifyOTP(w http.ResponseWriter, r *http.Request) {
	var req model.OTPVerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	pair, err := h.svc.VerifyOTP(r.Context(), req.Email, req.Code)
	if err != nil {
		if errors.Is(err, service.ErrInvalidOTP) {
			writeError(w, http.StatusUnauthorized, "invalid or expired OTP")
			return
		}
		writeError(w, http.StatusInternalServerError, "verification failed")
		return
	}

	writeJSON(w, http.StatusOK, pair)
}

func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var req model.RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	pair, err := h.svc.RefreshToken(r.Context(), req.RefreshToken)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid refresh token")
		return
	}

	writeJSON(w, http.StatusOK, pair)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	// Stateless JWT logout: client drops tokens.
	// For full invalidation, store refresh token in Redis and check here.
	writeJSON(w, http.StatusOK, map[string]string{"message": "logged out"})
}

// ValidateToken is an internal endpoint used by the API gateway.
func (h *AuthHandler) ValidateToken(w http.ResponseWriter, r *http.Request) {
	bearer := r.Header.Get("Authorization")
	tokenStr := strings.TrimPrefix(bearer, "Bearer ")
	if tokenStr == "" {
		writeError(w, http.StatusUnauthorized, "missing token")
		return
	}

	claims, err := h.svc.ValidateToken(tokenStr)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid token")
		return
	}

	writeJSON(w, http.StatusOK, claims)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
