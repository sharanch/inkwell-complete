package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/sharanch/inkwell/blog-service/internal/handler"
	"github.com/sharanch/inkwell/blog-service/internal/repository"
	"github.com/sharanch/inkwell/blog-service/internal/service"
)

func main() {
	port := getEnv("PORT", "8082")
	dbURL := getEnv("DB_URL", "postgres://blog_user:blog_pass@localhost:5432/blog_db?sslmode=disable")
	feedURL := getEnv("FEED_SERVICE_URL", "http://localhost:8083")

	db, err := repository.NewPostgres(dbURL)
	if err != nil {
		log.Fatalf("postgres: %v", err)
	}
	defer db.Close()

	if err := repository.Migrate(db); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	repo := repository.NewPostRepository(db)
	feedClient := service.NewFeedClient(feedURL)
	svc := service.NewBlogService(repo, feedClient)
	h := handler.NewBlogHandler(svc)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Accept", "Authorization", "Content-Type", "X-User-ID"},
	}))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status":"ok","service":"blog"}`))
	})

	r.Route("/api/v1/posts", func(r chi.Router) {
		r.Get("/public", h.ListPublicPosts)
		r.Get("/{id}", h.GetPost)
		r.Post("/", h.CreatePost)
		r.Put("/{id}", h.UpdatePost)
		r.Delete("/{id}", h.DeletePost)
		r.Post("/{id}/like", h.ToggleLike)
		r.Get("/my/all", h.ListMyPosts)
	})

	// Internal endpoint for feed-service batch fetch
	r.Post("/internal/posts/batch", h.GetByIDs)

	srv := &http.Server{Addr: ":" + port, Handler: r, ReadTimeout: 10 * time.Second, WriteTimeout: 10 * time.Second}

	go func() {
		log.Printf("blog-service listening on :%s", port)
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

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}
