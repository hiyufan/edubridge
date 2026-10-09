// Package config 从环境变量读取配置
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config 服务配置
type Config struct {
	Port             string
	JWTSecret        string
	JWTRefreshSecret string
	AllowedOrigin    string
	SecureCookie     bool
	JWBaseURL        string

	RedisAddr     string
	RedisPassword string

	// MonitorKeepalive 访问教务系统保持登录的间隔（需小于学校会话超时时间）
	MonitorKeepalive time.Duration
	// MonitorCheck 完整拉取课表和成绩比对变动的间隔
	MonitorCheck time.Duration

	SMTP SMTPConfig
}

// SMTPConfig 发信邮箱（用于邮件通知，可不配置）
type SMTPConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	From     string
}

// Load 读取并校验配置
func Load() (*Config, error) {
	c := &Config{
		Port:             env("PORT", "3000"),
		JWTSecret:        os.Getenv("JWT_SECRET"),
		JWTRefreshSecret: os.Getenv("JWT_REFRESH_SECRET"),
		AllowedOrigin:    env("ALLOWED_ORIGIN", "http://localhost:5173"),
		SecureCookie:     os.Getenv("SECURE_COOKIE") == "true",
		JWBaseURL:        env("JW_URL", "https://jw.fzrjxy.com"),
		RedisAddr:        env("REDIS_HOST", "localhost") + ":" + env("REDIS_PORT", "6379"),
		RedisPassword:    os.Getenv("REDIS_PASSWORD"),
		SMTP: SMTPConfig{
			Host:     os.Getenv("SMTP_HOST"),
			User:     os.Getenv("SMTP_USER"),
			Password: os.Getenv("SMTP_PASSWORD"),
			From:     os.Getenv("SMTP_FROM"),
		},
	}

	var errs []error
	if c.JWTSecret == "" || c.JWTRefreshSecret == "" {
		errs = append(errs, errors.New("JWT_SECRET 和 JWT_REFRESH_SECRET 必须设置"))
	} else if c.JWTSecret == c.JWTRefreshSecret {
		errs = append(errs, errors.New("JWT_SECRET 与 JWT_REFRESH_SECRET 不能相同"))
	}
	var err error
	if c.MonitorKeepalive, err = duration("MONITOR_KEEPALIVE_INTERVAL", 10*time.Minute); err != nil {
		errs = append(errs, err)
	}
	if c.MonitorCheck, err = duration("MONITOR_CHECK_INTERVAL", time.Hour); err != nil {
		errs = append(errs, err)
	}
	if c.SMTP.Port, err = strconv.Atoi(env("SMTP_PORT", "465")); err != nil {
		errs = append(errs, fmt.Errorf("SMTP_PORT 不是数字: %q", os.Getenv("SMTP_PORT")))
	}
	return c, errors.Join(errs...)
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func duration(key string, def time.Duration) (time.Duration, error) {
	v := os.Getenv(key)
	if v == "" {
		return def, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil || d < time.Minute {
		return 0, fmt.Errorf("%s 应为不小于 1m 的时长（如 10m、1h），当前为 %q", key, v)
	}
	return d, nil
}
