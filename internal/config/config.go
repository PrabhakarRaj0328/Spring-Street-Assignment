package config

import (
	"os"
)

type Config struct {
	DatabaseURL string
	APIPort     string
}

func LoadConfig() Config {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://springstreet:secretpassword@localhost:5432/prisma?sslmode=disable"
	}

	apiPort := os.Getenv("API_PORT")
	if apiPort == "" {
		apiPort = "8080"
	}

	return Config{
		DatabaseURL: dbURL,
		APIPort:     apiPort,
	}
}
