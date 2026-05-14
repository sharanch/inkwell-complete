package config

import (
	"log"
	"os"
	"strconv"
)

type Config struct {
	Port             string
	DBURL            string
	RedisURL         string
	JWTSecret        string
	JWTRefreshSecret string
	OTPTTLMinutes    int
	NotifyServiceURL string
}

func Load() *Config {
	otpTTL, err := strconv.Atoi(getEnv("OTP_TTL_MINUTES", "10"))
	if err != nil {
		log.Fatalf("invalid OTP_TTL_MINUTES: %v", err)
	}
	return &Config{
		Port:             getEnv("PORT", "8081"),
		DBURL:            getEnv("DB_URL", "postgres://auth_user:auth_pass@localhost:5432/auth_db?sslmode=disable"),
		RedisURL:         getEnv("REDIS_URL", "localhost:6379"),
		JWTSecret:        getEnv("JWT_SECRET", "dev-secret"),
		JWTRefreshSecret: getEnv("JWT_REFRESH_SECRET", "dev-refresh-secret"),
		OTPTTLMinutes:    otpTTL,
		NotifyServiceURL: getEnv("NOTIFY_SERVICE_URL", "http://localhost:8084"),
	}
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}
