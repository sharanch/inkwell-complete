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
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/golang-jwt/jwt/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// ─── Metrics ─────────────────────────────────────────────────────────────────

var (
	httpRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "inkwell_gateway_requests_total",
		Help: "Total HTTP requests handled by the API gateway.",
	}, []string{"method", "path", "status"})

	httpRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "inkwell_gateway_request_duration_seconds",
		Help:    "HTTP request latency at the API gateway.",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "path"})

	httpRequestsInFlight = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "inkwell_gateway_requests_in_flight",
		Help: "Number of HTTP requests currently being processed.",
	})
)

// ─── Config ───────────────────────────────────────────────────────────────────

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

// ─── Main ────────────────────────────────────────────────────────────────────

func main() {
	cfg := loadConfig()

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(prometheusMiddleware) // RED metrics on every request
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000", "http://127.0.0.1:3000", "http://inkwell.local"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
	}))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status":"ok","service":"gateway"}`))
	})

	// Prometheus scrape endpoint — matches the annotation on the K8s Deployment
	r.Handle("/metrics", promhttp.Handler())

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

// ─── Middleware ───────────────────────────────────────────────────────────────

// prometheusMiddleware records RED metrics (Rate, Errors, Duration) per request.
// It uses a responseWriter wrapper to capture the status code after the handler runs.
func prometheusMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip the /metrics endpoint itself to avoid self-referential noise
		if r.URL.Path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}

		start := time.Now()
		httpRequestsInFlight.Inc()
		defer httpRequestsInFlight.Dec()

		ww := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(ww, r)

		// Normalise path: replace chi URL params with their pattern so cardinality
		// stays bounded (e.g. /api/v1/posts/abc123 → /api/v1/posts/{id})
		path := normalisePath(r)
		status := strconv.Itoa(ww.status)

		httpRequestsTotal.WithLabelValues(r.Method, path, status).Inc()
		httpRequestDuration.WithLabelValues(r.Method, path).Observe(time.Since(start).Seconds())
	})
}

// normalisePath returns the matched chi route pattern when available, falling
// back to the raw URL path. This prevents unbounded label cardinality from
// unique resource IDs (a classic Prometheus footgun).
func normalisePath(r *http.Request) string {
	if rctx := chi.RouteContext(r.Context()); rctx != nil && rctx.RoutePattern() != "" {
		return rctx.RoutePattern()
	}
	return r.URL.Path
}

// statusWriter wraps http.ResponseWriter to capture the status code.
type statusWriter struct {
	http.ResponseWriter
	status int
}

func (sw *statusWriter) WriteHeader(code int) {
	sw.status = code
	sw.ResponseWriter.WriteHeader(code)
}

// ─── JWT middleware ───────────────────────────────────────────────────────────

// jwtMiddleware validates the Bearer token and injects X-User-ID / X-User-Email headers.
func jwtMiddleware(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			bearer := r.Header.Get("Authorization")
			tokenStr := strings.TrimPrefix(bearer, "Bearer ")
			if tokenStr == ""  {
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

// ─── Proxy helpers ────────────────────────────────────────────────────────────

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

// ─── Helpers ─────────────────────────────────────────────────────────────────

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
