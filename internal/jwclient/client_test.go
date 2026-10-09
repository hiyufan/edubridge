package jwclient_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"jww/internal/jwclient"
	"jww/internal/jwclient/jwtest"
	"jww/internal/model"
)

var ctx = context.Background()

func loggedIn(t *testing.T, srv *jwtest.Server) *jwclient.Client {
	t.Helper()
	c := jwclient.New(srv.URL)
	if _, _, err := c.Captcha(ctx); err != nil {
		t.Fatal(err)
	}
	if err := c.Login(ctx, jwtest.Username, jwtest.Password, jwtest.Captcha, ""); err != nil {
		t.Fatal(err)
	}
	return c
}

func TestLogin(t *testing.T) {
	srv := jwtest.New(t)
	c := jwclient.New(srv.URL)
	img, ct, err := c.Captcha(ctx)
	if err != nil || len(img) == 0 || ct != "image/png" {
		t.Fatalf("captcha = %d bytes, %q, %v", len(img), ct, err)
	}

	err = c.Login(ctx, jwtest.Username, jwtest.Password, "0000", "")
	var le *jwclient.LoginError
	if !errors.As(err, &le) || le.Info != "验证码错误" {
		t.Fatalf("wrong captcha: %v", err)
	}
	if err := c.Login(ctx, jwtest.Username, jwtest.Password, jwtest.Captcha, ""); err != nil {
		t.Fatal(err)
	}
	if err := c.Ping(ctx); err != nil {
		t.Fatalf("ping after login: %v", err)
	}
	if !c.Jar().TakeDirty() || len(c.Jar().Export()) == 0 {
		t.Fatal("cookies should be recorded")
	}
}

func TestSessionExpiredDetection(t *testing.T) {
	srv := jwtest.New(t)
	c := loggedIn(t, srv)
	srv.KickAll()

	if err := c.Ping(ctx); !errors.Is(err, jwclient.ErrSessionExpired) {
		t.Errorf("Ping: %v", err)
	}
	if _, _, err := c.FullSchedule(ctx, 20); !errors.Is(err, jwclient.ErrSessionExpired) {
		t.Errorf("FullSchedule: %v", err)
	}
	if _, err := c.WeekSchedule(ctx, 0); !errors.Is(err, jwclient.ErrSessionExpired) {
		t.Errorf("WeekSchedule: %v", err)
	}
	if _, _, err := c.Scores(ctx); !errors.Is(err, jwclient.ErrSessionExpired) {
		t.Errorf("Scores: %v", err)
	}
}

func TestRestoredCookiesWork(t *testing.T) {
	srv := jwtest.New(t)
	saved := loggedIn(t, srv).Jar().Export()

	c := jwclient.New(srv.URL)
	c.Jar().Import(saved)
	if err := c.Ping(ctx); err != nil {
		t.Fatalf("restored session: %v", err)
	}
}

func TestFullSchedule(t *testing.T) {
	srv := jwtest.New(t)
	srv.SetCourses(func(week int) []model.Course {
		list := []model.Course{{Name: "高等数学", Teacher: "王老师", Room: "A101", DayOfWeek: 1, PeriodStart: 1, Periods: 1}}
		if week%2 == 0 {
			list = append(list, model.Course{Name: "体育", DayOfWeek: 3, PeriodStart: 1, Periods: 1})
		}
		return list
	})
	c := loggedIn(t, srv)

	full, complete, err := c.FullSchedule(ctx, 6)
	if err != nil || !complete {
		t.Fatalf("complete=%v err=%v", complete, err)
	}
	if full.Semester != "2026-2027第1学期" || full.StudentName != "张三" || full.FetchedWeeks != 6 ||
		full.SemesterStart != jwtest.ThisMonday().Format("2006-01-02") {
		t.Fatalf("info = %+v", full)
	}
	if len(full.Courses) != 2 || full.Courses[0].Name != "高等数学" || fmt.Sprint(full.Courses[0].Weeks) != "[1 2 3 4 5 6]" ||
		fmt.Sprint(full.Courses[1].Weeks) != "[2 4 6]" {
		t.Fatalf("courses = %+v", full.Courses)
	}

	// 某周失败 => 不完整
	srv.FailWeek(3)
	if _, complete, err := c.FullSchedule(ctx, 6); err != nil || complete {
		t.Fatalf("with failed week: complete=%v err=%v", complete, err)
	}

	// 超出学期的周没有课表页：仍算完整
	srv.FailWeek(0)
	if full, complete, err := c.FullSchedule(ctx, 22); err != nil || !complete || full.FetchedWeeks != 20 {
		t.Fatalf("beyond semester: complete=%v fetched=%d err=%v", complete, full.FetchedWeeks, err)
	}
}

func TestWeekSchedule(t *testing.T) {
	srv := jwtest.New(t)
	c := loggedIn(t, srv)
	s, err := c.WeekSchedule(ctx, 5)
	if err != nil || s.Week != 5 || len(s.Courses) != 1 || s.Courses[0].Room != "A101" {
		t.Fatalf("week 5 = %+v, %v", s, err)
	}
	if s, err := c.WeekSchedule(ctx, 0); err != nil || s.Week != 1 {
		t.Fatalf("default week = %+v, %v", s, err)
	}
}

func TestScores(t *testing.T) {
	srv := jwtest.New(t)
	var rows []map[string]any
	for i := 0; i < 20; i++ {
		rows = append(rows, jwtest.ScoreRow(fmt.Sprintf("课程%d", i), "80", "3.0", "2"))
	}
	rows = append(rows, jwtest.ScoreRow("课程0", "60", "1.0", "2")) // 同名重复行只保留第一条
	srv.SetScores(rows)
	c := loggedIn(t, srv)

	scores, complete, err := c.Scores(ctx)
	if err != nil || !complete || len(scores) != 20 {
		t.Fatalf("scores=%d complete=%v err=%v", len(scores), complete, err)
	}
	if scores[0].Course != "课程0" || scores[0].Grade != "80" || scores[0].GPA != 3 || scores[0].Credit != 2 {
		t.Fatalf("first = %+v", scores[0])
	}

	srv.FailScorePage(2) // 共 3 页：第 1 页 9 条 + 第 3 页 2 条（另 1 条重复）
	if scores, complete, err := c.Scores(ctx); err != nil || complete || len(scores) != 11 {
		t.Fatalf("failed page: scores=%d complete=%v err=%v", len(scores), complete, err)
	}

	srv.SetScores(nil)
	srv.FailScorePage(0)
	if scores, complete, err := c.Scores(ctx); err != nil || !complete || len(scores) != 0 {
		t.Fatalf("empty: scores=%d complete=%v err=%v", len(scores), complete, err)
	}
}

func TestUnavailable(t *testing.T) {
	c := jwclient.New("http://127.0.0.1:1")
	if err := c.Ping(ctx); !errors.Is(err, jwclient.ErrUnavailable) {
		t.Fatalf("err = %v", err)
	}
}

// 跨行（rowspan）格子、中午行
const rowspanHTML = `<html><div class="f2 b">软件1班 张三同学 第3周 课程表(2026-2027第1学期)</div><table>
<tr><td></td><td>星期一<br/>2026-09-21</td><td>星期二<br/>2026-09-22</td><td>星期三<br/>2026-09-23</td></tr>
<tr><td>第1节</td><td rowspan="2"><div title="高等数学
王老师
A101">高等数学</div></td><td></td><td><div title="英语
赵老师
C301">英语</div></td></tr>
<tr><td>第2节</td><td></td><td><div title="物理
李老师
B202">物理</div></td></tr>
<tr><td>中午</td><td></td><td></td><td></td></tr>
</table></html>`

func TestParseScheduleRowspan(t *testing.T) {
	doc, _ := goquery.NewDocumentFromReader(strings.NewReader(rowspanHTML))
	s := jwclient.ParseScheduleForTest(doc)
	if s.Week != 3 || s.SemesterStart != "2026-09-07" || s.ClassName != "软件1班" {
		t.Fatalf("info = %+v", s)
	}
	want := []model.Course{
		{Name: "高等数学", Teacher: "王老师", Room: "A101", DayOfWeek: 1, PeriodStart: 1, Periods: 2},
		{Name: "英语", Teacher: "赵老师", Room: "C301", DayOfWeek: 3, PeriodStart: 1, Periods: 1},
		{Name: "物理", Teacher: "李老师", Room: "B202", DayOfWeek: 3, PeriodStart: 2, Periods: 1},
	}
	if fmt.Sprint(s.Courses) != fmt.Sprint(want) {
		t.Fatalf("courses =\n%+v\nwant\n%+v", s.Courses, want)
	}
}
