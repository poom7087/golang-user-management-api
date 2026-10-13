package config

import (
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Port         string
	MongoURI     string
	MongoDBName  string
	JWTSecret    string
	JWTExpiresIn int
}

func Load() Config {
	// .env is optional; real environment variables take precedence.
	_ = godotenv.Load()

	return Config{
		Port:         getEnv("PORT", "3000"),
		MongoURI:     getEnv("MONGODB_URI", "mongodb://localhost:27017"),
		MongoDBName:  getEnv("MONGODB_DATABASE", "user_management"),
		JWTSecret:    strings.TrimSpace(os.Getenv("JWT_SECRET")),
		JWTExpiresIn: getEnvInt("JWT_EXPIRES_IN", 24),
	}
}

func getEnv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	n, err := strconv.Atoi(value)
	if err != nil || n <= 0 {
		return fallback
	}

	return n
}
