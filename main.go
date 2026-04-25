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
	"github.com/joho/godotenv"
	"jww/config"
	"jww/internal/handler"
	"jww/internal/middleware"
	"jww/internal/model"
	"jww/internal/service"
	"jww/pkg/database"
)

const (
	loginMaxAttempts   = 10
	captchaMaxAttempts = 100
	rateLimitWindow    = 5 * time.Minute
	sessionTTL         = 30 * time.Minute
	scoreCacheTTL      = 15 * time.Minute
	cleanupInterval    = 10 * time.Minute
	defaultMaxWeek     = 20
)

func newRateLimiter(limit int) gin.HandlerFunc {
	var mu sync.Mutex
	attempts := make(map[string][]time.Time)

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
	godotenv.Load()

	cfg := config.Load()

	if err := database.InitMySQL(cfg.MySQL); err != nil {
		log.Fatalf("Failed to initialize MySQL: %v", err)
	}

	if err := model.AutoMigrateRefreshToken(database.GetDB()); err != nil {
		log.Fatalf("Failed to migrate RefreshToken model: %v", err)
	}

	if err := database.InitRedis(cfg.Redis); err != nil {
		log.Fatalf("Failed to initialize Redis: %v", err)
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

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

	authHandler := handler.NewAuthHandler(cfg.JWTSecret, cfg.JWTRefreshSecret, cfg.SecureCookie)
	captchaHandler := handler.NewCaptchaHandler()
	scheduleHandler := handler.NewScheduleHandler()
	scoreHandler := handler.NewScoreHandler()
	notifyHandler := handler.NewNotifyHandler()
	authMiddleware := middleware.NewAuthMiddleware(cfg.JWTSecret, cfg.JWTRefreshSecret)

	loginLimiter := newRateLimiter(loginMaxAttempts)
	captchaLimiter := newRateLimiter(captchaMaxAttempts)

	api := r.Group("/api")
	{
		api.GET("/captcha", captchaLimiter, captchaHandler.GetCaptcha)
		api.POST("/auth/login", loginLimiter, authHandler.Login)
		api.POST("/auth/refresh", authHandler.Refresh)
		api.GET("/schedule/ical/subscribe", scheduleHandler.SubscribeICal)

		protected := api.Group("")
		protected.Use(authMiddleware.AuthRequired())
		{
			protected.POST("/auth/logout", authHandler.Logout)
			protected.GET("/auth/me", authHandler.Me)

			protected.GET("/schedule", scheduleHandler.GetSchedule)
			protected.GET("/schedule/full", scheduleHandler.GetFullSchedule)
			protected.GET("/schedule/conflicts", scheduleHandler.GetConflicts)

			protected.GET("/score", scoreHandler.GetScore)
			protected.GET("/score/semesters", scoreHandler.GetSemesters)
			protected.GET("/score/stats", scoreHandler.GetScoreStats)

			protected.POST("/notify/register", notifyHandler.RegisterToken)
			protected.POST("/notify/test", notifyHandler.TestNotify)

			protected.GET("/schedule/ical", scheduleHandler.GetICal)
			protected.POST("/schedule/ical/token", scheduleHandler.GenerateICalToken)
			protected.GET("/schedule/ical/token-info", scheduleHandler.GetICalTokenInfo)

			protected.POST("/webhook/register", scheduleHandler.RegisterWebhook)
			protected.POST("/webhook/trigger", scheduleHandler.TriggerWebhook)
			protected.GET("/webhook/info", scheduleHandler.GetWebhookInfo)
			protected.GET("/schedule/diff", scheduleHandler.GetScheduleDiff)
		}
	}

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": 1})
	})

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

	service.GetJwService().Close()

	log.Println("Server exited gracefully")
}
