package config

import (
	"os"
)

type Config struct {
	Port            string
	JWTSecret       string
	JWTRefreshSecret string
}

func Load() *Config {
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		panic("JWT_SECRET environment variable is required")
	}

	jwtRefreshSecret := os.Getenv("JWT_REFRESH_SECRET")
	if jwtRefreshSecret == "" {
		panic("JWT_REFRESH_SECRET environment variable is required")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	return &Config{
		Port:            port,
		JWTSecret:       jwtSecret,
		JWTRefreshSecret: jwtRefreshSecret,
	}
}
