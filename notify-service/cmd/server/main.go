package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/smtp"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Config struct {
	Port        string
	SMTPEnabled bool
	SMTPHost    string
	SMTPPort    string
	SMTPUser    string
	SMTPPass    string
	FromEmail   string
}

func loadConfig() *Config {
	smtpEnabled := getEnv("SMTP_ENABLED", "false") == "true"
	return &Config{
		Port:        getEnv("PORT", "8084"),
		SMTPEnabled: smtpEnabled,
		SMTPHost:    getEnv("SMTP_HOST", "smtp.example.com"),
		SMTPPort:    getEnv("SMTP_PORT", "587"),
		SMTPUser:    getEnv("SMTP_USER", "noreply@example.com"),
		SMTPPass:    getEnv("SMTP_PASS", ""),
		FromEmail:   getEnv("FROM_EMAIL", "noreply@inkwell.app"),
	}
}

type NotifyHandler struct {
	cfg *Config
}

type SendOTPRequest struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

func (h *NotifyHandler) SendOTP(w http.ResponseWriter, r *http.Request) {
	var req SendOTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	if h.cfg.SMTPEnabled {
		subject := "Your Inkwell login code"
		body := fmt.Sprintf(
			"Your one-time login code is:\n\n  %s\n\nExpires in 10 minutes.\n\n— Inkwell",
			req.Code,
		)
		if err := sendEmail(h.cfg, req.Email, subject, body); err != nil {
			log.Printf("SMTP error: %v", err)
		}
	}

	// Always log — dev mode lifeline
	log.Printf("OTP for %s: %s", req.Email, req.Code)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"sent":true}`))
}

func sendEmail(cfg *Config, to, subject, body string) error {
	addr := cfg.SMTPHost + ":" + cfg.SMTPPort
	msg := fmt.Sprintf(
		"From: Inkwell <%s>\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		cfg.FromEmail, to, subject, body,
	)

	dialer := &net.Dialer{Timeout: 5 * time.Second}
	conn, err := dialer.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}

	client, err := smtp.NewClient(conn, cfg.SMTPHost)
	if err != nil {
		conn.Close()
		return fmt.Errorf("smtp client: %w", err)
	}
	defer client.Close()

	tlsCfg := &tls.Config{ServerName: cfg.SMTPHost}
	if err := client.StartTLS(tlsCfg); err != nil {
		return fmt.Errorf("starttls: %w", err)
	}

	auth := smtp.PlainAuth("", cfg.SMTPUser, cfg.SMTPPass, cfg.SMTPHost)
	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("auth: %w", err)
	}
	if err := client.Mail(cfg.FromEmail); err != nil {
		return err
	}
	if err := client.Rcpt(to); err != nil {
		return err
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	fmt.Fprint(w, msg)
	return w.Close()
}

func main() {
	cfg := loadConfig()
	if cfg.SMTPEnabled {
		log.Printf("SMTP enabled — sending via %s", cfg.SMTPHost)
	} else {
		log.Println("SMTP disabled — OTP codes will be printed to logs only (set SMTP_ENABLED=true to send email)")
	}

	h := &NotifyHandler{cfg: cfg}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status":"ok","service":"notify"}`))
	})
	r.Post("/internal/send-otp", h.SendOTP)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("notify-service listening on :%s", cfg.Port)
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
