package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Address  string
	DBHost   string
	DBPort   string
	DBUser   string
	DBPass   string
	DBName   string
	DBParams string
}

func Load() (Config, error) {
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		return Config{}, fmt.Errorf("load .env: %w", err)
	}

	port := env("DB_PORT", "3306")
	if n, err := strconv.Atoi(port); err != nil || n < 1 || n > 65535 {
		return Config{}, fmt.Errorf("DB_PORT must be a valid TCP port")
	}

	cfg := Config{
		Address:  env("APP_ADDRESS", ":8080"),
		DBHost:   env("DB_HOST", "127.0.0.1"),
		DBPort:   port,
		DBUser:   env("DB_USER", "root"),
		DBPass:   os.Getenv("DB_PASSWORD"),
		DBName:   env("DB_NAME", "quotation"),
		DBParams: env("DB_PARAMS", "charset=utf8mb4&parseTime=True&loc=Local"),
	}
	return cfg, nil
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
