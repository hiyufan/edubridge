package config

import (
	"log"
	"os"
	"time"
)

type Config struct {
	Port             string
	JWTSecret        string
	JWTRefreshSecret string
	AllowedOrigin    string
	SecureCookie     bool
	// MonitorKeepalive 访问教务系统保持登录的间隔（需小于学校会话超时时间）
	MonitorKeepalive time.Duration
	// MonitorCheck 完整拉取课表比对变动的间隔
	MonitorCheck time.Duration
	MySQL        MySQLConfig
	Redis        RedisConfig
}

type MySQLConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Database string
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
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

	allowedOrigin := os.Getenv("ALLOWED_ORIGIN")
	if allowedOrigin == "" {
		allowedOrigin = "http://localhost:5173"
	}

	secureCookie := os.Getenv("SECURE_COOKIE") == "true"

	mysqlHost := os.Getenv("MYSQL_HOST")
	if mysqlHost == "" {
		mysqlHost = "localhost"
	}
	mysqlPort := os.Getenv("MYSQL_PORT")
	if mysqlPort == "" {
		mysqlPort = "3306"
	}
	mysqlUser := os.Getenv("MYSQL_USER")
	if mysqlUser == "" {
		mysqlUser = "root"
	}
	mysqlPassword := os.Getenv("MYSQL_PASSWORD")
	mysqlDatabase := os.Getenv("MYSQL_DATABASE")
	if mysqlDatabase == "" {
		mysqlDatabase = "jww"
	}

	redisHost := os.Getenv("REDIS_HOST")
	if redisHost == "" {
		redisHost = "localhost"
	}
	redisPort := os.Getenv("REDIS_PORT")
	if redisPort == "" {
		redisPort = "6379"
	}
	redisPassword := os.Getenv("REDIS_PASSWORD")
	redisDB := 0

	return &Config{
		MonitorKeepalive: durationEnv("MONITOR_KEEPALIVE_INTERVAL", 10*time.Minute),
		MonitorCheck:     durationEnv("MONITOR_CHECK_INTERVAL", time.Hour),
		Port:             port,
		JWTSecret:        jwtSecret,
		JWTRefreshSecret: jwtRefreshSecret,
		AllowedOrigin:    allowedOrigin,
		SecureCookie:     secureCookie,
		MySQL: MySQLConfig{
			Host:     mysqlHost,
			Port:     mysqlPort,
			User:     mysqlUser,
			Password: mysqlPassword,
			Database: mysqlDatabase,
		},
		Redis: RedisConfig{
			Host:     redisHost,
			Port:     redisPort,
			Password: redisPassword,
			DB:       redisDB,
		},
	}
}

func durationEnv(key string, def time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil || d < time.Minute {
		log.Printf("invalid %s=%q, using default %s", key, v, def)
		return def
	}
	return d
}
