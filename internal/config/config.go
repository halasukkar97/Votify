// Package config centralizes environment loading for the server.
// Keeping this here stops startup code from spreading os.Getenv calls around main.
package config

import (
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// Config contains the environment-backed settings needed to start the API.
type Config struct {
	Port           string
	DatabaseURL    string
	TMDBAPIKey     string
	AllowedOrigins map[string]bool
}

// Load reads .env when present and returns the settings used by cmd/server.
func Load() Config {
	// godotenv.Load reads key/value pairs from .env so os.Getenv can use them.
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	return Config{
		Port:        port,
		DatabaseURL: os.Getenv("DATABASE_URL"),
		TMDBAPIKey:  os.Getenv("TMDB_API_KEY"),
		AllowedOrigins: allowedOriginsFromEnv(
			os.Getenv("ALLOWED_ORIGINS"),
			[]string{"http://localhost:5173", "https://votify-six.vercel.app"},
		),
	}
}

func allowedOriginsFromEnv(rawOrigins string, fallbackOrigins []string) map[string]bool {
	origins := fallbackOrigins
	if rawOrigins != "" {
		origins = strings.Split(rawOrigins, ",")
	}

	allowedOrigins := make(map[string]bool)
	for _, origin := range origins {
		origin = strings.TrimSpace(origin)
		if origin != "" {
			allowedOrigins[origin] = true
		}
	}

	return allowedOrigins
}
