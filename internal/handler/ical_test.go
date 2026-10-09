package handler

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"jww/internal/model"
	"jww/internal/service"
	"jww/pkg/database"
)

func setupRouter(t *testing.T, uid string) (*gin.Engine, *miniredis.Miniredis) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	mr := miniredis.RunT(t)
	database.RedisClient = redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { database.RedisClient.Close() })

	h := NewScheduleHandler()
	r := gin.New()
	r.GET("/api/schedule/ical/subscribe", h.SubscribeICal)
	protected := r.Group("/api", func(c *gin.Context) {
		c.Set("uid", uid)
		c.Set("sessionId", "sess-"+uid)
		c.Next()
	})
	protected.GET("/schedule/ical/token-info", h.GetICalTokenInfo)
	protected.POST("/webhook/register", h.RegisterWebhook)
	protected.GET("/webhook/info", h.GetWebhookInfo)
	return r, mr
}

func do(r *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// 订阅链接不依赖教务会话：只要 token 有效且有课表快照就能返回 iCal
func TestSubscribeICalUsesSnapshot(t *testing.T) {
	r, mr := setupRouter(t, "2024001")

	snap, _ := json.Marshal(&model.FullSchedule{
		StudentName:   "张三",
		SemesterStart: "2026-09-07",
		Courses: []model.Course{
			{Name: "高等数学", Room: "A101", DayOfWeek: 1, PeriodStart: 1, Periods: 2, Weeks: []int{1}},
		},
	})
	mr.Set("schedule:snapshot:2024001", string(snap))
	if err := service.SaveICalToken("2024001", "tok123"); err != nil {
		t.Fatal(err)
	}

	w := do(r, "GET", "/api/schedule/ical/subscribe?token=tok123", "")
	if w.Code != 200 {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body)
	}
	if !strings.Contains(w.Body.String(), "SUMMARY:高等数学") ||
		!strings.Contains(w.Body.String(), "DTSTART:20260907T080000") {
		t.Fatalf("unexpected ical body:\n%s", w.Body)
	}

	if w := do(r, "GET", "/api/schedule/ical/subscribe?token=bad", ""); w.Code != 401 {
		t.Fatalf("invalid token status = %d", w.Code)
	}

	w = do(r, "GET", "/api/schedule/ical/token-info", "")
	if !strings.Contains(w.Body.String(), `"token":"tok123"`) {
		t.Fatalf("token-info = %s", w.Body)
	}
}

func TestWebhookInfoIsScopedToUser(t *testing.T) {
	r, _ := setupRouter(t, "alice")

	w := do(r, "POST", "/api/webhook/register", `{"url":"https://a.example/hook","secret":"0123456789"}`)
	if w.Code != 200 {
		t.Fatalf("register status = %d, body = %s", w.Code, w.Body)
	}
	w = do(r, "GET", "/api/webhook/info", "")
	if !strings.Contains(w.Body.String(), `"registered":true`) ||
		!strings.Contains(w.Body.String(), `"secret":"01234567***"`) {
		t.Fatalf("alice info = %s", w.Body)
	}

	if _, err := service.GetWebhook("bob"); err == nil {
		t.Fatal("bob must not see alice's webhook")
	}
}
