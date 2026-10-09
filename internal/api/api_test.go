package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"jww/internal/api"
	"jww/internal/auth"
	"jww/internal/jwclient/jwtest"
	"jww/internal/monitor"
	"jww/internal/reminder"
	"jww/internal/testenv"
)

type client struct {
	t     *testing.T
	base  string
	http  *http.Client
	token string
}

type resp struct {
	code int
	body map[string]any
	raw  string
}

func (c *client) do(method, path string, body any) resp {
	c.t.Helper()
	var r *strings.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		r = strings.NewReader(string(b))
	} else {
		r = strings.NewReader("")
	}
	req, _ := http.NewRequest(method, c.base+path, r)
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	res, err := c.http.Do(req)
	if err != nil {
		c.t.Fatal(err)
	}
	defer res.Body.Close()
	var out resp
	out.code = res.StatusCode
	var buf strings.Builder
	b := make([]byte, 4096)
	for {
		n, err := res.Body.Read(b)
		buf.Write(b[:n])
		if err != nil {
			break
		}
	}
	out.raw = buf.String()
	json.Unmarshal([]byte(out.raw), &out.body)
	return out
}

func setup(t *testing.T) (*testenv.Env, *client) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	e := testenv.New(t)
	h := api.New(api.Deps{
		Academic: e.Academic,
		Auth:     auth.New("a-secret", "r-secret", e.Store),
		Notifier: e.Notifier,
		Reminder: reminder.New(e.Store, e.Notifier),
		Monitor:  monitor.New(e.Academic, e.Store, 10*time.Minute, time.Hour),
	}).Handler()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	jar, _ := cookiejar.New(nil)
	return e, &client{t: t, base: srv.URL, http: &http.Client{Jar: jar}}
}

func (c *client) login() {
	c.t.Helper()
	cap := c.do("GET", "/api/captcha", nil)
	if cap.code != 200 || !strings.HasPrefix(cap.body["data"].(string), "data:image/png;base64,") {
		c.t.Fatalf("captcha: %d %s", cap.code, cap.raw)
	}
	sid := cap.body["sessionId"].(string)

	bad := c.do("POST", "/api/auth/login", map[string]string{"sessionId": sid, "username": jwtest.Username, "password": jwtest.Password, "captcha": "0000"})
	if bad.code != 400 || bad.body["info"] != "验证码错误" {
		c.t.Fatalf("wrong captcha should be 400 with message: %d %s", bad.code, bad.raw)
	}

	c.do("GET", "/api/captcha", nil)
	r := c.do("POST", "/api/auth/login", map[string]string{"sessionId": sid, "username": jwtest.Username, "password": jwtest.Password, "captcha": jwtest.Captcha})
	if r.code != 200 || r.body["uid"] != jwtest.Username {
		c.t.Fatalf("login: %d %s", r.code, r.raw)
	}
	c.token = r.body["token"].(string)
}

func TestFullFlow(t *testing.T) {
	e, c := setup(t)

	if r := c.do("GET", "/api/schedule", nil); r.code != 401 {
		t.Fatalf("unauthenticated: %d", r.code)
	}
	c.login()

	r := c.do("GET", "/api/schedule", nil)
	data, _ := r.body["data"].(map[string]any)
	if r.code != 200 || data["currentWeek"] != float64(1) || len(data["courses"].([]any)) != 1 {
		t.Fatalf("week schedule: %d %s", r.code, r.raw)
	}
	if r := c.do("GET", "/api/schedule/full", nil); r.code != 200 || r.body["data"].(map[string]any)["fetchedWeeks"] != float64(20) {
		t.Fatalf("full schedule: %d %s", r.code, r.raw)
	}
	if r := c.do("GET", "/api/score/stats", nil); r.code != 200 || r.body["data"].(map[string]any)["semesterStats"] == nil {
		t.Fatalf("score stats: %d %s", r.code, r.raw)
	}
	if r := c.do("GET", "/api/monitor/status", nil); r.code != 200 || r.body["data"].(map[string]any)["monitoring"] != true {
		t.Fatalf("monitor status: %d %s", r.code, r.raw)
	}

	// access token 过期后用 refresh cookie 换新的
	c.token = ""
	r = c.do("POST", "/api/auth/refresh", nil)
	if r.code != 200 || r.body["token"] == nil {
		t.Fatalf("refresh: %d %s", r.code, r.raw)
	}
	c.token = r.body["token"].(string)

	// 学校踢下线：返回 401 + JW_SESSION_EXPIRED，前端据此回到登录页
	e.JW.KickAll()
	r = c.do("GET", "/api/schedule/full?maxWeek=19", nil)
	if r.code != 401 || r.body["code"] != "JW_SESSION_EXPIRED" {
		t.Fatalf("expired: %d %s", r.code, r.raw)
	}

	// 重新登录后恢复
	c.login()
	if r := c.do("GET", "/api/schedule?week=2", nil); r.code != 200 {
		t.Fatalf("after relogin: %d %s", r.code, r.raw)
	}

	// 退出：作废 refresh token、停止监控
	if r := c.do("POST", "/api/auth/logout", nil); r.code != 200 {
		t.Fatalf("logout: %d", r.code)
	}
	if ok, _ := e.Store.IsMonitored(context.Background(), jwtest.Username); ok {
		t.Fatal("logout should stop monitoring")
	}
	if r := c.do("POST", "/api/auth/refresh", nil); r.code != 401 {
		t.Fatalf("refresh after logout: %d", r.code)
	}
}

func TestICalSubscribe(t *testing.T) {
	_, c := setup(t)
	c.login()

	r := c.do("POST", "/api/schedule/ical/token", nil)
	link, _ := r.body["data"].(map[string]any)["url"].(string)
	if r.code != 200 || link == "" {
		t.Fatalf("token: %d %s", r.code, r.raw)
	}
	u, _ := url.Parse(link)

	anon := &client{t: t, base: c.base, http: http.DefaultClient}
	ics := anon.do("GET", u.RequestURI(), nil)
	if ics.code != 200 || !strings.Contains(ics.raw, "BEGIN:VCALENDAR") || strings.Count(ics.raw, "BEGIN:VEVENT") != 20 {
		t.Fatalf("subscribe: %d %.200s", ics.code, ics.raw)
	}
	if r := anon.do("GET", "/api/schedule/ical/subscribe?token=bad", nil); r.code != 401 {
		t.Fatalf("bad token: %d", r.code)
	}
	if r := c.do("GET", "/api/schedule/ical/token-info", nil); r.body["data"].(map[string]any)["token"] == nil {
		t.Fatalf("token info: %s", r.raw)
	}
}

func TestSettingsValidation(t *testing.T) {
	_, c := setup(t)
	c.login()

	r := c.do("PUT", "/api/notify/channels", map[string]any{"bot": map[string]string{"type": "wecom", "url": "http://127.0.0.1/hook"}})
	if r.code != 400 || !strings.Contains(r.body["info"].(string), "qyapi.weixin.qq.com") {
		t.Fatalf("bad bot url: %d %s", r.code, r.raw)
	}
	if r := c.do("PUT", "/api/notify/reminder", map[string]any{"daily": true, "dailyTime": "7点"}); r.code != 400 {
		t.Fatalf("bad reminder: %d %s", r.code, r.raw)
	}
	if r := c.do("PUT", "/api/notify/reminder", map[string]any{"beforeClass": true}); r.code != 200 || r.body["data"].(map[string]any)["beforeMinutes"] != float64(15) {
		t.Fatalf("reminder: %d %s", r.code, r.raw)
	}
	if r := c.do("POST", "/api/notify/test", nil); r.code != 400 {
		t.Fatalf("test without channel: %d %s", r.code, r.raw)
	}
	if r := c.do("POST", "/api/webhook/register", map[string]string{"url": "https://example.com/hook", "secret": "0123456789"}); r.code != 200 {
		t.Fatalf("webhook: %d %s", r.code, r.raw)
	}
	if r := c.do("GET", "/api/webhook/info", nil); r.body["data"].(map[string]any)["secret"] != "01234567***" {
		t.Fatalf("webhook info: %s", r.raw)
	}
}
