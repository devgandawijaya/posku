package config

import (
	"os"

	"github.com/joho/godotenv"
)

var (
	AppPort   string
	DBHost    string
	DBPort    string
	DBUser    string
	DBPass    string
	DBName    string
	JWTSecret string
)

func LoadEnv() {
	godotenv.Load()

	AppPort = getEnv("APP_PORT", "2040")
	DBHost = getEnv("DB_HOST", "db")
	DBPort = getEnv("DB_PORT", "5432")
	DBUser = getEnv("DB_USER", "postgres")
	DBPass = getEnv("DB_PASS", "secret")
	DBName = getEnv("DB_NAME", "golangku_db")
	JWTSecret = getEnv("JWT_SECRET", "supersecretkey")
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
