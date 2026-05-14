package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/redis/go-redis/v9"

	"github.com/sharanch/inkwell/feed-service/internal/repository"
	"github.com/sharanch/inkwell/feed-service/internal/service"
)

func main() {
	port := getEnv("PORT", "8083")
	dbURL := getEnv("DB_URL", "postgres://feed_user:feed_pass@localhost:5432/feed_db?sslmode=disable")
	redisURL := getEnv("REDIS_URL", "localhost:6379")
	blogURL := getEnv("BLOG_SERVICE_URL", "http://localhost:8082")

	db, err := repository.NewPostgres(dbURL)
	if err != nil {
		log.Fatalf("postgres: %v", err)
	}
	defer db.Close()

	if err := repository.Migrate(db); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	rdb := redis.NewClient(&redis.Options{Addr: redisURL})
	defer rdb.Close()

	repo := repository.NewFeedRepository(db)
	svc := service.NewFeedService(repo, rdb, blogURL)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Accept", "Authorization", "Content-Type", "X-User-ID"},
	}))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status":"ok","service":"feed"}`))
	})

	r.Route("/api/v1/feed", func(r chi.Router) {
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			userID := r.Header.Get("X-User-ID")
			page, _ := strconv.Atoi(r.URL.Query().Get("page"))
			if page < 1 {
				page = 1
			}
			fp, err := svc.GetFeed(r.Context(), userID, page)
			if err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			writeJSON(w, http.StatusOK, fp)
		})

		r.Get("/interests", func(w http.ResponseWriter, r *http.Request) {
			userID := r.Header.Get("X-User-ID")
			tags, err := svc.GetInterests(r.Context(), userID)
			if err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			writeJSON(w, http.StatusOK, map[string][]string{"tags": tags})
		})

		r.Put("/interests", func(w http.ResponseWriter, r *http.Request) {
			userID := r.Header.Get("X-User-ID")
			var body struct {
				Tags []string `json:"tags"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				writeError(w, http.StatusBadRequest, "invalid body")
				return
			}
			if err := svc.SetInterests(r.Context(), userID, body.Tags); err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
		})
	})

	// Internal: blog-service calls this to index a newly published post
	r.Post("/internal/feed/index", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			ID       string   `json:"id"`
			AuthorID string   `json:"author_id"`
			Tags     []string `json:"tags"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid body")
			return
		}
		repo := repository.NewFeedRepository(db)
		if err := repo.IndexPost(r.Context(), body.ID, body.AuthorID, body.Tags); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"indexed": true})
	})

	srv := &http.Server{Addr: ":" + port, Handler: r, ReadTimeout: 10 * time.Second, WriteTimeout: 10 * time.Second}

	go func() {
		log.Printf("feed-service listening on :%s", port)
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

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}
