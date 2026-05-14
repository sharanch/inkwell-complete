package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/golang-jwt/jwt/v5"
)

type Config struct {
	Port             string
	AuthServiceURL   string
	BlogServiceURL   string
	FeedServiceURL   string
	NotifyServiceURL string
	JWTSecret        string
}

func loadConfig() *Config {
	return &Config{
		Port:             getEnv("PORT", "8080"),
		AuthServiceURL:   getEnv("AUTH_SERVICE_URL", "http://localhost:8081"),
		BlogServiceURL:   getEnv("BLOG_SERVICE_URL", "http://localhost:8082"),
		FeedServiceURL:   getEnv("FEED_SERVICE_URL", "http://localhost:8083"),
		NotifyServiceURL: getEnv("NOTIFY_SERVICE_URL", "http://localhost:8084"),
		JWTSecret:        getEnv("JWT_SECRET", "dev-secret"),
	}
}

type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
}

func main() {
	cfg := loadConfig()

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000", "http://127.0.0.1:3000", "http://inkwell.local"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
	}))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status":"ok","service":"gateway"}`))
	})

	// Auth routes — no JWT required
	r.Mount("/api/v1/auth", proxyTo(cfg.AuthServiceURL))

	// Blog public routes — no JWT required
	r.Get("/api/v1/posts/public", proxyHandler(cfg.BlogServiceURL))
	r.Get("/api/v1/posts/{id}", proxyHandler(cfg.BlogServiceURL))

	// Protected routes — JWT required
	r.Group(func(r chi.Router) {
		r.Use(jwtMiddleware(cfg.JWTSecret))
		r.Mount("/api/v1/posts", proxyTo(cfg.BlogServiceURL))
		r.Mount("/api/v1/feed", proxyTo(cfg.FeedServiceURL))
	})

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	go func() {
		log.Printf("api-gateway listening on :%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
}

// jwtMiddleware validates the Bearer token and injects X-User-ID / X-User-Email headers.
func jwtMiddleware(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			bearer := r.Header.Get("Authorization")
			tokenStr := strings.TrimPrefix(bearer, "Bearer ")
			if tokenStr == "" {
				writeError(w, http.StatusUnauthorized, "authorization required")
				return
			}

			token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method")
				}
				return []byte(secret), nil
			})
			if err != nil || !token.Valid {
				writeError(w, http.StatusUnauthorized, "invalid token")
				return
			}

			mc, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				writeError(w, http.StatusUnauthorized, "invalid claims")
				return
			}

			userID, _ := mc["user_id"].(string)
			email, _ := mc["email"].(string)
			if userID == "" {
				writeError(w, http.StatusUnauthorized, "invalid claims")
				return
			}

			// Inject user identity for downstream services
			r.Header.Set("X-User-ID", userID)
			r.Header.Set("X-User-Email", email)
			next.ServeHTTP(w, r)
		})
	}
}

func proxyTo(target string) http.Handler {
	return http.HandlerFunc(proxyHandler(target))
}

func proxyHandler(target string) http.HandlerFunc {
	u, _ := url.Parse(target)
	proxy := httputil.NewSingleHostReverseProxy(u)
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("proxy error: %v", err)
		writeError(w, http.StatusBadGateway, "upstream unavailable")
	}
	return func(w http.ResponseWriter, r *http.Request) {
		proxy.ServeHTTP(w, r)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// drain discards a response body.
func drain(r io.ReadCloser) {
	io.Copy(io.Discard, r)
	r.Close()
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}
