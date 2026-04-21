package config

import (
	"os"
)

type Config struct {
	Port             string
	JWTSecret        string
	JWTRefreshSecret string
	AllowedOrigin    string // CORS 允许的源（SEC-3 修复）
	SecureCookie     bool   // Secure Cookie 标志（SEC-2 修复）
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

	// SEC-3 修复：CORS 源从环境变量读取，支持配置多个（逗号分隔）
	allowedOrigin := os.Getenv("ALLOWED_ORIGIN")
	if allowedOrigin == "" {
		allowedOrigin = "http://localhost:5173" // 默认为本地开发地址
	}

	// SEC-2 修复：Secure Cookie 从环境变量读取，生产环境应设为 true
	secureCookie := os.Getenv("SECURE_COOKIE") == "true"

	return &Config{
		Port:             port,
		JWTSecret:        jwtSecret,
		JWTRefreshSecret: jwtRefreshSecret,
		AllowedOrigin:    allowedOrigin,
		SecureCookie:     secureCookie,
	}
}
