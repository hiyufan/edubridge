package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"jww/internal/model"
)

// fakeJw 模拟教务系统：cookie PHPSESSID=good 视为已登录，否则返回登录页
type fakeJw struct {
	mu       sync.Mutex
	loggedIn bool
	start    time.Time // 第 1 周周一
	// week -> 该周的课（DayOfWeek/PeriodStart/Periods/Name/Teacher/Room）
	courses func(week int) []model.Course
}

var reDQZ = regexp.MustCompile(`/dqz/(\d+)/`)

const loginPage = `<html><form action="/studentportal.php/Index/checkLogin"><img src="/studentportal.php/Public/verify/"></form></html>`

func (f *fakeJw) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	loggedIn := f.loggedIn
	f.mu.Unlock()
	c, err := r.Cookie("PHPSESSID")
	if !loggedIn || err != nil || c.Value != "good" {
		fmt.Fprint(w, loginPage)
		return
	}

	switch {
	case r.URL.Path == "/studentportal.php/Main/":
		fmt.Fprint(w, "<html>欢迎</html>")
	case r.URL.Path == "/studentportal.php/Jxxx/xskbxx/optype/1":
		fmt.Fprint(w, `<html><a href="/studentportal.php/Jxxx/xskbxx/optype/2/xn/2026-2027/xq/1/dqz/1/sybmdmstr/X/bjmc/软件1班">课表</a></html>`)
	case strings.HasPrefix(r.URL.Path, "/studentportal.php/Jxxx/xskbxx/optype/2/"):
		m := reDQZ.FindStringSubmatch(r.URL.Path)
		week, _ := strconv.Atoi(m[1])
		fmt.Fprint(w, f.weekPage(week))
	default:
		http.NotFound(w, r)
	}
}

func (f *fakeJw) weekPage(week int) string {
	monday := f.start.AddDate(0, 0, (week-1)*7)
	var b strings.Builder
	fmt.Fprintf(&b, `<html><div class="f2 b">软件1班 张三同学 第%d周 课程表(2026-2027第1学期)</div><table><tr><td></td>`, week)
	days := []string{"星期一", "星期二", "星期三", "星期四", "星期五", "星期六", "星期日"}
	for i, d := range days {
		fmt.Fprintf(&b, "<td>%s<br/>%s</td>", d, monday.AddDate(0, 0, i).Format("2006-01-02"))
	}
	b.WriteString("</tr>")
	// 只模拟第 1 节这一行，课程都放在第 1 节
	b.WriteString("<tr><td>第1节</td>")
	byDay := make(map[int]model.Course)
	for _, c := range f.courses(week) {
		byDay[c.DayOfWeek] = c
	}
	for d := 1; d <= 7; d++ {
		if c, ok := byDay[d]; ok {
			fmt.Fprintf(&b, `<td><div title="%s
%s
%s">%s</div></td>`, c.Name, c.Teacher, c.Room, c.Name)
		} else {
			b.WriteString("<td></td>")
		}
	}
	b.WriteString("</tr></table></html>")
	return b.String()
}

type webhookRecorder struct {
	srv *httptest.Server
	ch  chan NotifyPayload
}

func newWebhookRecorder(t *testing.T) *webhookRecorder {
	rec := &webhookRecorder{ch: make(chan NotifyPayload, 10)}
	rec.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var p NotifyPayload
		json.NewDecoder(r.Body).Decode(&p)
		rec.ch <- p
	}))
	t.Cleanup(rec.srv.Close)
	return rec
}

func (r *webhookRecorder) next(t *testing.T) NotifyPayload {
	t.Helper()
	select {
	case p := <-r.ch:
		return p
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for notification")
		return NotifyPayload{}
	}
}

func (r *webhookRecorder) none(t *testing.T) {
	t.Helper()
	select {
	case p := <-r.ch:
		t.Fatalf("unexpected notification: %+v", p)
	case <-time.After(200 * time.Millisecond):
	}
}

func thisMonday() time.Time {
	now := time.Now()
	wd := int(now.Weekday())
	if wd == 0 {
		wd = 7
	}
	d := now.AddDate(0, 0, -(wd - 1))
	return time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.Local)
}

func TestMonitorDetectsChangesAndExpiry(t *testing.T) {
	setupRedis(t)

	var mu sync.Mutex
	changed := false
	fake := &fakeJw{loggedIn: true, start: thisMonday()}
	fake.courses = func(week int) []model.Course {
		mu.Lock()
		defer mu.Unlock()
		list := []model.Course{{Name: "高等数学", Teacher: "王老师", Room: "A101", DayOfWeek: 1, PeriodStart: 1, Periods: 1}}
		if changed {
			if week == 3 {
				list = nil // 第 3 周停课
			}
			if week == 4 {
				list = append(list, model.Course{Name: "大学物理", Teacher: "李老师", Room: "B202", DayOfWeek: 3, PeriodStart: 1, Periods: 1})
			}
		}
		return list
	}
	jwSrv := httptest.NewServer(fake)
	defer jwSrv.Close()
	oldBase := baseURL
	baseURL = jwSrv.URL
	defer func() { baseURL = oldBase }()

	hook := newWebhookRecorder(t)
	const uid = "2024001"

	// 模拟之前登录过：Redis 里有 cookie、sessionID 映射、监控登记
	u, _ := url.Parse(jwSrv.URL)
	if err := saveJwCookies(uid, map[string][]*http.Cookie{u.Host: {{Name: "PHPSESSID", Value: "good"}}}); err != nil {
		t.Fatal(err)
	}
	if err := bindSessionID("sid-1", uid); err != nil {
		t.Fatal(err)
	}
	if err := addMonitorUser(uid); err != nil {
		t.Fatal(err)
	}
	if err := SaveWebhook(uid, &WebhookEntry{URL: hook.srv.URL}); err != nil {
		t.Fatal(err)
	}

	// 新的服务实例（相当于服务重启）：凭 sessionID 从 Redis 恢复登录态
	svc := NewJwService()
	defer svc.Close()

	full, err := svc.GetFullSchedule("sid-1", 20)
	if err != nil {
		t.Fatalf("GetFullSchedule: %v", err)
	}
	if len(full.Courses) != 1 || len(full.Courses[0].Weeks) != 20 {
		t.Fatalf("unexpected schedule: %+v", full.Courses)
	}
	hook.none(t) // 第一次只建立基线，不通知

	// 课表无变化：不通知
	if !svc.checkUser(uid) {
		t.Fatal("checkUser failed")
	}
	hook.none(t)

	// 教务端加课/停课
	mu.Lock()
	changed = true
	mu.Unlock()
	if !svc.checkUser(uid) {
		t.Fatal("checkUser failed")
	}
	p := hook.next(t)
	if p.Event != EventScheduleDiff {
		t.Fatalf("event = %q", p.Event)
	}
	if !strings.Contains(p.Text, "+ 大学物理 周三 第1节 @B202（第4周）") ||
		!strings.Contains(p.Text, "- 高等数学 周一 第1节 @A101（第3周）") {
		t.Fatalf("unexpected text:\n%s", p.Text)
	}
	diff, err := GetLatestDiff(uid)
	if err != nil || len(diff.Added) != 1 || len(diff.Removed) != 1 {
		t.Fatalf("latest diff = %+v, %v", diff, err)
	}

	// 保活正常时不通知
	svc.keepaliveUser(uid)
	hook.none(t)

	// 学校端会话失效：保活发现后停止监控并通知重新登录
	fake.mu.Lock()
	fake.loggedIn = false
	fake.mu.Unlock()
	svc.keepaliveUser(uid)
	p = hook.next(t)
	if p.Event != EventSessionExpired {
		t.Fatalf("event = %q", p.Event)
	}
	if ok, _ := IsMonitored(uid); ok {
		t.Fatal("user should no longer be monitored")
	}
	if _, err := svc.GetFullSchedule("sid-1", 20); !errors.Is(err, ErrSessionExpired) {
		t.Fatalf("expected ErrSessionExpired, got %v", err)
	}
	// 再次检测不会重复通知
	svc.keepaliveUser(uid)
	hook.none(t)
}

func TestIncompleteFetchDoesNotNotify(t *testing.T) {
	setupRedis(t)

	var mu sync.Mutex
	failWeek := 0
	fake := &fakeJw{loggedIn: true, start: thisMonday()}
	fake.courses = func(week int) []model.Course {
		return []model.Course{{Name: "高等数学", Teacher: "王老师", Room: "A101", DayOfWeek: 1, PeriodStart: 1, Periods: 1}}
	}
	jwSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		fw := failWeek
		mu.Unlock()
		if fw > 0 && strings.Contains(r.URL.Path, fmt.Sprintf("/dqz/%d/", fw)) {
			http.Error(w, "boom", http.StatusBadGateway)
			return
		}
		fake.ServeHTTP(w, r)
	}))
	defer jwSrv.Close()
	oldBase := baseURL
	baseURL = jwSrv.URL
	defer func() { baseURL = oldBase }()

	hook := newWebhookRecorder(t)
	const uid = "2024002"
	u, _ := url.Parse(jwSrv.URL)
	saveJwCookies(uid, map[string][]*http.Cookie{u.Host: {{Name: "PHPSESSID", Value: "good"}}})
	addMonitorUser(uid)
	SaveWebhook(uid, &WebhookEntry{URL: hook.srv.URL})

	svc := NewJwService()
	defer svc.Close()
	if !svc.checkUser(uid) {
		t.Fatal("baseline check failed")
	}

	// 第 5 周请求失败：这次结果缺了第 5 周的课，不能当成“减课”通知
	mu.Lock()
	failWeek = 5
	mu.Unlock()
	svc.mu.Lock()
	delete(svc.scheduleCache, uid)
	svc.mu.Unlock()
	if !svc.checkUser(uid) {
		t.Fatal("check failed")
	}
	hook.none(t)

	// 恢复后再比对：与基线一致，也不通知
	mu.Lock()
	failWeek = 0
	mu.Unlock()
	if !svc.checkUser(uid) {
		t.Fatal("check failed")
	}
	hook.none(t)
}

func TestCheckDetectsLoggedOutSession(t *testing.T) {
	setupRedis(t)

	fake := &fakeJw{loggedIn: false, start: thisMonday()}
	fake.courses = func(int) []model.Course { return nil }
	jwSrv := httptest.NewServer(fake)
	defer jwSrv.Close()
	oldBase := baseURL
	baseURL = jwSrv.URL
	defer func() { baseURL = oldBase }()

	hook := newWebhookRecorder(t)
	const uid = "2024003"
	u, _ := url.Parse(jwSrv.URL)
	saveJwCookies(uid, map[string][]*http.Cookie{u.Host: {{Name: "PHPSESSID", Value: "good"}}})
	addMonitorUser(uid)
	SaveWebhook(uid, &WebhookEntry{URL: hook.srv.URL})

	svc := NewJwService()
	defer svc.Close()
	if svc.checkUser(uid) {
		t.Fatal("check should fail when logged out")
	}
	if p := hook.next(t); p.Event != EventSessionExpired {
		t.Fatalf("event = %q", p.Event)
	}
	if _, err := loadJwCookies(uid); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cookies should be deleted, got %v", err)
	}
}
