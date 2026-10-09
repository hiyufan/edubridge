// jww 教务系统中间件：组装各组件并启动 HTTP 服务与后台任务。
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"jww/internal/academic"
	"jww/internal/api"
	"jww/internal/auth"
	"jww/internal/config"
	"jww/internal/monitor"
	"jww/internal/notify"
	"jww/internal/reminder"
	"jww/internal/session"
	"jww/internal/store"
)

func main() {
	if err := run(); err != nil {
		slog.Error("server exited", "err", err)
		os.Exit(1)
	}
}

func run() error {
	_ = godotenv.Load()
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	st, err := store.Open(ctx, cfg.RedisAddr, cfg.RedisPassword, 0)
	if err != nil {
		return err
	}
	defer st.Close()

	notifier := notify.New(st, notify.Options{SMTP: notify.SMTPConfig(cfg.SMTP)})
	sessions := session.New(cfg.JWBaseURL, st)
	academicSvc := academic.New(sessions, st, notifier)
	mon := monitor.New(academicSvc, st, cfg.MonitorKeepalive, cfg.MonitorCheck)
	reminders := reminder.New(st, notifier)

	gin.SetMode(gin.ReleaseMode)
	srv := &http.Server{
		Addr: ":" + cfg.Port,
		Handler: api.New(api.Deps{
			Academic:      academicSvc,
			Auth:          auth.New(cfg.JWTSecret, cfg.JWTRefreshSecret, st),
			Notifier:      notifier,
			Reminder:      reminders,
			Monitor:       mon,
			AllowedOrigin: cfg.AllowedOrigin,
			SecureCookie:  cfg.SecureCookie,
		}).Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	var wg sync.WaitGroup
	for _, task := range []func(context.Context){sessions.Run, mon.Run, reminders.Run} {
		wg.Add(1)
		go func(run func(context.Context)) {
			defer wg.Done()
			run(ctx)
		}(task)
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("server started", "addr", srv.Addr)
		if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		stop()
		wg.Wait()
		return err
	case <-ctx.Done():
	}

	slog.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = srv.Shutdown(shutdownCtx)
	wg.Wait()
	notifier.Wait()
	return err
}
