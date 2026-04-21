package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"jww/config"
	"jww/internal/handler"
	"jww/internal/middleware"
	"jww/internal/service"
)

// C5 修复：魔法数字集中管理
const (
	loginMaxAttempts   = 10
	captchaMaxAttempts = 100
	rateLimitWindow   = 5 * time.Minute
	sessionTTL        = 30 * time.Minute
	scoreCacheTTL     = 15 * time.Minute
	cleanupInterval   = 10 * time.Minute
	defaultMaxWeek    = 20
)

// C1 修复：限速中间件工厂函数，消除 loginLimiter 和 captchaLimiter 的重复代码
func newRateLimiter(limit int) gin.HandlerFunc {
	var mu sync.Mutex
	attempts := make(map[string][]time.Time)

	// C5 修复：后台 goroutine 定时清理，使用统一的 cleanupInterval
	go func() {
		ticker := time.NewTicker(cleanupInterval)
		defer ticker.Stop()
		for range ticker.C {
			mu.Lock()
			window := time.Now().Add(-rateLimitWindow)
			for ip, times := range attempts {
				valid := make([]time.Time, 0)
				for _, t := range times {
					if t.After(window) {
						valid = append(valid, t)
					}
				}
				if len(valid) == 0 {
					delete(attempts, ip)
				} else {
					attempts[ip] = valid
				}
			}
			mu.Unlock()
		}
	}()

	return func(c *gin.Context) {
		ip := c.ClientIP()
		now := time.Now()
		window := now.Add(-rateLimitWindow)

		mu.Lock()
		var valid []time.Time
		for _, t := range attempts[ip] {
			if t.After(window) {
				valid = append(valid, t)
			}
		}
		if len(valid) == 0 {
			delete(attempts, ip)
		} else {
			attempts[ip] = valid
		}

		if len(valid) >= limit {
			mu.Unlock()
			c.JSON(429, gin.H{"status": 0, "info": "请求过于频繁，请稍后再试"})
			c.Abort()
			return
		}

		attempts[ip] = append(valid, now)
		mu.Unlock()
		c.Next()
	}
}

func main() {
	cfg := config.Load()

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// CORS（SEC-3 修复：从配置读取 AllowedOrigin）
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", cfg.AllowedOrigin)
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
	authHandler := handler.NewAuthHandler(cfg.JWTSecret, cfg.JWTRefreshSecret, cfg.SecureCookie)
	captchaHandler := handler.NewCaptchaHandler()
	scheduleHandler := handler.NewScheduleHandler()
	scoreHandler := handler.NewScoreHandler()
	authMiddleware := middleware.NewAuthMiddleware(cfg.JWTSecret, cfg.JWTRefreshSecret)

	// 限速中间件：C1 修复，改用工厂函数
	loginLimiter   := newRateLimiter(loginMaxAttempts)
	captchaLimiter := newRateLimiter(captchaMaxAttempts)

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

	srv := &http.Server{Addr: ":" + cfg.Port, Handler: r}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	// 停止 JwService 后台 goroutine
	service.GetJwService().Close()

	log.Println("Server exited gracefully")
}
