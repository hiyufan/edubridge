package config

import (
	"strings"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	t.Setenv("JWT_SECRET", "a")
	t.Setenv("JWT_REFRESH_SECRET", "b")
	c, err := Load()
	if err != nil || c.Port != "3000" || c.RedisAddr != "localhost:6379" || c.MonitorKeepalive != 10*time.Minute || c.SMTP.Port != 465 {
		t.Fatalf("defaults: %+v %v", c, err)
	}

	t.Setenv("JWT_REFRESH_SECRET", "a")
	t.Setenv("MONITOR_CHECK_INTERVAL", "5s")
	t.Setenv("SMTP_PORT", "abc")
	_, err = Load()
	for _, want := range []string{"不能相同", "MONITOR_CHECK_INTERVAL", "SMTP_PORT"} {
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("error should mention %q: %v", want, err)
		}
	}
}
