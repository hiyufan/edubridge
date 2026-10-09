package session

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"jww/internal/jwclient"
	"jww/internal/jwclient/jwtest"
	"jww/internal/store"
)

func TestLoginBindRestoreDrop(t *testing.T) {
	ctx := context.Background()
	srv := jwtest.New(t)
	mr := miniredis.RunT(t)
	st := store.New(redis.NewClient(&redis.Options{Addr: mr.Addr()}))

	m := New(srv.URL, st)
	if _, err := m.Get(ctx, jwtest.Username); !errors.Is(err, jwclient.ErrSessionExpired) {
		t.Fatalf("before login: %v", err)
	}

	c := m.Pending("sid")
	if c != m.Pending("sid") {
		t.Fatal("same sessionID should reuse client")
	}
	c.Captcha(ctx)
	if err := c.Login(ctx, jwtest.Username, jwtest.Password, jwtest.Captcha, ""); err != nil {
		t.Fatal(err)
	}
	if err := m.Bind(ctx, "sid", jwtest.Username, c); err != nil {
		t.Fatal(err)
	}
	if ok, _ := st.IsMonitored(ctx, jwtest.Username); !ok {
		t.Fatal("should be monitored after login")
	}

	// 新的 Manager（相当于服务重启）从 Redis 恢复
	m2 := New(srv.URL, st)
	restored, err := m2.Get(ctx, jwtest.Username)
	if err != nil {
		t.Fatal(err)
	}
	if err := restored.Ping(ctx); err != nil {
		t.Fatalf("restored session should work: %v", err)
	}

	// 内存淘汰后仍能恢复
	m2.evict(ctx, time.Now().Add(time.Minute))
	if again, err := m2.Get(ctx, jwtest.Username); err != nil || again == restored {
		t.Fatalf("after evict: %v (same=%v)", err, again == restored)
	}

	if was, err := m2.Drop(ctx, jwtest.Username); err != nil || !was {
		t.Fatalf("drop: %v %v", was, err)
	}
	if _, err := m2.Get(ctx, jwtest.Username); !errors.Is(err, jwclient.ErrSessionExpired) {
		t.Fatalf("after drop: %v", err)
	}
}
