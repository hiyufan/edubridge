// Package testenv 为集成测试组装整套服务：模拟教务系统 + 内存 Redis + 记录通知的 Webhook。
package testenv

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"jww/internal/academic"
	"jww/internal/jwclient/jwtest"
	"jww/internal/model"
	"jww/internal/notify"
	"jww/internal/session"
	"jww/internal/store"
)

// Env 一套完整的服务
type Env struct {
	JW       *jwtest.Server
	Redis    *miniredis.Miniredis
	Store    *store.Store
	Notifier *notify.Notifier
	Sessions *session.Manager
	Academic *academic.Service
	Hook     *Webhook
}

// New 组装服务；用户的通知会发到 Env.Hook（需先调用 Login 或 SubscribeHook）
func New(t *testing.T) *Env {
	t.Helper()
	jw := jwtest.New(t)
	mr := miniredis.RunT(t)
	st := store.New(redis.NewClient(&redis.Options{Addr: mr.Addr()}))
	t.Cleanup(func() { st.Close() })
	n := notify.New(st, notify.Options{})
	sessions := session.New(jw.URL, st)
	return &Env{
		JW: jw, Redis: mr, Store: st, Notifier: n, Sessions: sessions,
		Academic: academic.New(sessions, st, n),
		Hook:     newWebhook(t),
	}
}

// Login 以测试账号走一遍验证码+登录，并把通知订阅到 Env.Hook
func (e *Env) Login(t *testing.T) string {
	t.Helper()
	ctx := context.Background()
	if _, _, err := e.Academic.Captcha(ctx, "sid"); err != nil {
		t.Fatal(err)
	}
	if err := e.Academic.Login(ctx, "sid", jwtest.Username, jwtest.Password, jwtest.Captcha, ""); err != nil {
		t.Fatal(err)
	}
	e.SubscribeHook(t, jwtest.Username)
	return jwtest.Username
}

// SubscribeHook 把用户的 Webhook 指向 Env.Hook
func (e *Env) SubscribeHook(t *testing.T, uid string) {
	t.Helper()
	if err := e.Store.SaveWebhook(context.Background(), uid, &model.WebhookConfig{URL: e.Hook.URL}); err != nil {
		t.Fatal(err)
	}
}

// Webhook 记录收到的通知
type Webhook struct {
	*httptest.Server
	ch chan notify.Payload
}

func newWebhook(t *testing.T) *Webhook {
	w := &Webhook{ch: make(chan notify.Payload, 20)}
	w.Server = httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		var p notify.Payload
		json.NewDecoder(r.Body).Decode(&p)
		w.ch <- p
	}))
	t.Cleanup(w.Close)
	return w
}

// Next 等待下一条通知
func (w *Webhook) Next(t *testing.T) notify.Payload {
	t.Helper()
	select {
	case p := <-w.ch:
		return p
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for notification")
		return notify.Payload{}
	}
}

// None 确认短时间内没有通知
func (w *Webhook) None(t *testing.T) {
	t.Helper()
	select {
	case p := <-w.ch:
		t.Fatalf("unexpected notification: %+v", p)
	case <-time.After(200 * time.Millisecond):
	}
}
