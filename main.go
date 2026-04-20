package main

import (
	"log"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"jww/config"
	"jww/internal/handler"
	"jww/internal/middleware"
)

func main() {
	cfg := config.Load()

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// CORS
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "http://localhost:5173")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Header("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// 初始化处理器
	authHandler := handler.NewAuthHandler(cfg.JWTSecret, cfg.JWTRefreshSecret)
	captchaHandler := handler.NewCaptchaHandler()
	scheduleHandler := handler.NewScheduleHandler()
	scoreHandler := handler.NewScoreHandler()
	authMiddleware := middleware.NewAuthMiddleware(cfg.JWTSecret, cfg.JWTRefreshSecret)

	// 限速中间件（并发安全实现）
	var loginAttemptsMu sync.Mutex
	loginAttempts := make(map[string][]time.Time)
	loginLimiter := func(c *gin.Context) {
		ip := c.ClientIP()
		now := time.Now()
		window := now.Add(-5 * time.Minute)

		loginAttemptsMu.Lock()
		var valid []time.Time
		for _, t := range loginAttempts[ip] {
			if t.After(window) {
				valid = append(valid, t)
			}
		}
		loginAttempts[ip] = valid

		if len(valid) >= 10 {
			loginAttemptsMu.Unlock()
			c.JSON(429, gin.H{"status": 0, "info": "登录尝试次数过多，请稍后再试"})
			c.Abort()
			return
		}

		loginAttempts[ip] = append(valid, now)
		loginAttemptsMu.Unlock()
		c.Next()
	}

	var captchaAttemptsMu sync.Mutex
	captchaAttempts := make(map[string][]time.Time)
	captchaLimiter := func(c *gin.Context) {
		ip := c.ClientIP()
		now := time.Now()
		window := now.Add(-5 * time.Minute)

		captchaAttemptsMu.Lock()
		var valid []time.Time
		for _, t := range captchaAttempts[ip] {
			if t.After(window) {
				valid = append(valid, t)
			}
		}
		captchaAttempts[ip] = valid

		if len(valid) >= 100 {
			captchaAttemptsMu.Unlock()
			c.JSON(429, gin.H{"status": 0, "info": "请求过于频繁，请稍后再试"})
			c.Abort()
			return
		}

		captchaAttempts[ip] = append(valid, now)
		captchaAttemptsMu.Unlock()
		c.Next()
	}

	// 路由
	api := r.Group("/api")
	{
		// 公开接口
		api.GET("/captcha", captchaLimiter, captchaHandler.GetCaptcha)
		api.POST("/auth/login", loginLimiter, authHandler.Login)
		api.POST("/auth/refresh", authHandler.Refresh) // 公开，依赖 refresh token 自己验证

		// 需要认证的接口
		protected := api.Group("")
		protected.Use(authMiddleware.AuthRequired())
		{
			// 认证
			protected.POST("/auth/logout", authHandler.Logout)
			protected.GET("/auth/me", authHandler.Me)

			// 课表
			protected.GET("/schedule", scheduleHandler.GetSchedule)
			protected.GET("/schedule/full", scheduleHandler.GetFullSchedule)

			// 成绩
			protected.GET("/score", scoreHandler.GetScore)
			protected.GET("/score/semesters", scoreHandler.GetSemesters)
		}
	}

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": 1})
	})

	// 根路径
	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{"name": "jw-server-go", "version": "1.0.0"})
	})

	log.Printf("Server starting on http://localhost:%s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
