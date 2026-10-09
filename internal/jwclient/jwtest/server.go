// Package jwtest 提供一个模拟学校教务系统的 HTTP 服务器，供各层测试使用。
package jwtest

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"jww/internal/model"
)

// 测试账号
const (
	Username = "2024001"
	Password = "pw"
	Captcha  = "1234"
)

const loginPage = `<html><form action="/studentportal.php/Index/checkLogin"><img src="/studentportal.php/Public/verify/"></form></html>`

var reDQZ = regexp.MustCompile(`/dqz/(\d+)/`)

// Server 模拟教务系统。cookie PHPSESSID=good 且未被踢下线时视为已登录。
type Server struct {
	*httptest.Server

	mu       sync.Mutex
	loggedIn bool
	start    time.Time
	courses  func(week int) []model.Course
	scores   []map[string]any
	failWeek int // 该周课表返回 502
	failPage int // 成绩该页返回 502
	requests int
}

// New 启动服务器并在测试结束时关闭；学期第 1 周从本周一开始，默认每周一第 1 节有高等数学
func New(t testing.TB) *Server {
	s := NewHandler()
	s.Server = httptest.NewServer(s)
	t.Cleanup(s.Close)
	return s
}

// NewHandler 不启动服务器，只返回 http.Handler（本地开发用，见 cmd/fakejw）
func NewHandler() *Server {
	s := &Server{start: ThisMonday()}
	s.courses = func(int) []model.Course {
		return []model.Course{{Name: "高等数学", Teacher: "王老师", Room: "A101", DayOfWeek: 1, PeriodStart: 1, Periods: 1}}
	}
	return s
}

// ServeHTTP 实现 http.Handler
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.serve(w, r) }

// ThisMonday 本周一 0 点（本地时区）
func ThisMonday() time.Time {
	now := time.Now()
	wd := int(now.Weekday())
	if wd == 0 {
		wd = 7
	}
	d := now.AddDate(0, 0, -(wd - 1))
	return time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.Local)
}

// SetCourses 设置每周的课（只支持每天第 1 节一门课）
func (s *Server) SetCourses(f func(week int) []model.Course) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.courses = f
}

// SetScores 设置成绩行（字段同教务系统 JSON，可用 ScoreRow 构造）
func (s *Server) SetScores(rows []map[string]any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.scores = rows
}

// FailWeek 让某一周的课表请求失败（0 取消）
func (s *Server) FailWeek(w int) { s.mu.Lock(); s.failWeek = w; s.mu.Unlock() }

// FailScorePage 让成绩的某一页请求失败（0 取消）
func (s *Server) FailScorePage(p int) { s.mu.Lock(); s.failPage = p; s.mu.Unlock() }

// KickAll 让所有会话失效（模拟学校服务器踢下线）
func (s *Server) KickAll() { s.mu.Lock(); s.loggedIn = false; s.mu.Unlock() }

// LogInDirectly 不经过登录流程直接视为已登录（配合 cookie PHPSESSID=good 使用）
func (s *Server) LogInDirectly() { s.mu.Lock(); s.loggedIn = true; s.mu.Unlock() }

// Requests 收到的请求总数
func (s *Server) Requests() int { s.mu.Lock(); defer s.mu.Unlock(); return s.requests }

// ScoreRow 构造一行成绩
func ScoreRow(course, grade, gpa, credit string) map[string]any {
	return map[string]any{
		"xn": "2025-2026", "xq": "1", "kcmc": course, "cj": grade, "cjjd": gpa, "kcxf": credit,
		"kcxz": "必修", "zdjsxm": "王老师", "ssbjmc": "软件1班", "cjsx": "正常",
	}
}

func (s *Server) serve(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	s.requests++
	loggedIn := s.loggedIn
	s.mu.Unlock()

	switch r.URL.Path {
	case "/studentportal.php/Public/verify/":
		http.SetCookie(w, &http.Cookie{Name: "PHPSESSID", Value: "pending", Path: "/"})
		w.Header().Set("Content-Type", "image/png")
		w.Write([]byte("\x89PNG fake"))
		return
	case "/studentportal.php/Index/checkLogin":
		s.checkLogin(w, r)
		return
	}

	if c, err := r.Cookie("PHPSESSID"); !loggedIn || err != nil || c.Value != "good" {
		fmt.Fprint(w, loginPage)
		return
	}

	switch {
	case r.URL.Path == "/studentportal.php/Main/":
		fmt.Fprint(w, "<html>欢迎</html>")
	case r.URL.Path == "/studentportal.php/Jxxx/xskbxx/optype/1":
		fmt.Fprint(w, `<html><a href="/studentportal.php/Jxxx/xskbxx/optype/2/xn/2026-2027/xq/1/dqz/1/sybmdmstr/X/bjmc/软件1班">课表</a></html>`)
	case strings.HasPrefix(r.URL.Path, "/studentportal.php/Jxxx/xskbxx/optype/2/"):
		week, _ := strconv.Atoi(reDQZ.FindStringSubmatch(r.URL.Path)[1])
		s.mu.Lock()
		fail := s.failWeek == week
		s.mu.Unlock()
		if fail {
			http.Error(w, "boom", http.StatusBadGateway)
			return
		}
		if week > 20 {
			fmt.Fprint(w, "<html>无数据</html>")
			return
		}
		fmt.Fprint(w, s.weekPage(week))
	case r.URL.Path == "/studentportal.php/Jxxx/cjxxlb":
		s.scoresPage(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) checkLogin(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	hash := md5.Sum([]byte(Password))
	if c, err := r.Cookie("PHPSESSID"); err != nil || c.Value != "pending" {
		json.NewEncoder(w).Encode(map[string]any{"status": 0, "info": "请先获取验证码"})
		return
	}
	if r.Form.Get("yzm") != Captcha {
		json.NewEncoder(w).Encode(map[string]any{"status": 0, "info": "验证码错误"})
		return
	}
	if r.Form.Get("xsxh") != Username || r.Form.Get("dlmm") != hex.EncodeToString(hash[:]) {
		json.NewEncoder(w).Encode(map[string]any{"status": 0, "info": "用户名或密码错误"})
		return
	}
	s.LogInDirectly()
	http.SetCookie(w, &http.Cookie{Name: "PHPSESSID", Value: "good", Path: "/"})
	json.NewEncoder(w).Encode(map[string]any{"status": 1, "info": "登录成功"})
}

func (s *Server) weekPage(week int) string {
	s.mu.Lock()
	courses := s.courses(week)
	monday := s.start.AddDate(0, 0, (week-1)*7)
	s.mu.Unlock()

	var b strings.Builder
	fmt.Fprintf(&b, `<html><div class="f2 b">软件1班 张三同学 第%d周 课程表(2026-2027第1学期)</div><table><tr><td></td>`, week)
	for i, d := range []string{"星期一", "星期二", "星期三", "星期四", "星期五", "星期六", "星期日"} {
		fmt.Fprintf(&b, "<td>%s<br/>%s</td>", d, monday.AddDate(0, 0, i).Format("2006-01-02"))
	}
	b.WriteString("</tr><tr><td>第1节</td>")
	byDay := make(map[int]model.Course)
	for _, c := range courses {
		byDay[c.DayOfWeek] = c
	}
	for d := 1; d <= 7; d++ {
		if c, ok := byDay[d]; ok {
			fmt.Fprintf(&b, "<td><div title=\"%s\n%s\n%s\">%s</div></td>", c.Name, c.Teacher, c.Room, c.Name)
		} else {
			b.WriteString("<td></td>")
		}
	}
	b.WriteString("</tr></table></html>")
	return b.String()
}

func (s *Server) scoresPage(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	page, _ := strconv.Atoi(r.Form.Get("page"))
	s.mu.Lock()
	rows, failPage := s.scores, s.failPage
	s.mu.Unlock()
	if page == failPage {
		http.Error(w, "boom", http.StatusBadGateway)
		return
	}
	start, end := min((page-1)*9, len(rows)), min(page*9, len(rows))
	json.NewEncoder(w).Encode(map[string]any{"total": strconv.Itoa(len(rows)), "rows": rows[start:end]})
}
