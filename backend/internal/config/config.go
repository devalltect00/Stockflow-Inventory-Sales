package config

import "os"

type Config struct {
	Address     string
	DatabaseURL string
}

// Load supplies local-development defaults while Compose injects its own database hostname.
func Load() Config {
	return Config{Address: value("HTTP_ADDR", ":8080"), DatabaseURL: value("DATABASE_URL", "postgres://stockflow:stockflow_local@localhost:5433/stockflow?sslmode=disable")}
}

// value returns a non-empty environment override or the documented default.
func value(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
