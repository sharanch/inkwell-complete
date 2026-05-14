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

	"github.com/sharanch/inkwell/auth-service/config"
	"github.com/sharanch/inkwell/auth-service/internal/handler"
	"github.com/sharanch/inkwell/auth-service/internal/repository"
	"github.com/sharanch/inkwell/auth-service/internal/service"
)

func main() {
	cfg := config.Load()

	db, err := repository.NewPostgres(cfg.DBURL)
	if err != nil {
		log.Fatalf("postgres connect: %v", err)
	}
	defer db.Close()

	if err := repository.Migrate(db); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	rdb, err := repository.NewRedis(cfg.RedisURL)
	if err != nil {
		log.Fatalf("redis connect: %v", err)
	}
	defer rdb.Close()

	userRepo := repository.NewUserRepository(db)
	otpRepo := repository.NewOTPRepository(rdb)

	notifyClient := service.NewNotifyClient(cfg.NotifyServiceURL)
	authSvc := service.NewAuthService(userRepo, otpRepo, notifyClient, cfg)

	h := handler.NewAuthHandler(authSvc)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Accept", "Authorization", "Content-Type"},
	}))

	r.Route("/api/v1/auth", func(r chi.Router) {
		r.Post("/request-otp", h.RequestOTP)
		r.Post("/verify-otp", h.VerifyOTP)
		r.Post("/refresh", h.RefreshToken)
		r.Post("/logout", h.Logout)
		// Internal: used by API gateway to validate tokens
		r.Get("/validate", h.ValidateToken)
	})

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","service":"auth"}`))
	})

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("auth-service listening on :%s", cfg.Port)
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
	log.Println("auth-service stopped")
}
