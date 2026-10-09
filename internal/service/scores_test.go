package service

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"jww/internal/model"
)

func scoreRow(course, grade, gpa, credit string) map[string]interface{} {
	return map[string]interface{}{
		"xn": "2025-2026", "xq": "1", "kcmc": course, "cj": grade, "cjjd": gpa, "kcxf": credit,
		"kcxz": "必修", "zdjsxm": "王老师", "ssbjmc": "软件1班", "cjsx": "正常",
	}
}

func setupScoreFixture(t *testing.T, uid string) (*fakeJw, *JwService, *webhookRecorder) {
	t.Helper()
	setupRedis(t)
	fake := &fakeJw{loggedIn: true, start: thisMonday()}
	fake.courses = func(int) []model.Course { return nil }
	srv := httptest.NewServer(fake)
	t.Cleanup(srv.Close)
	oldBase := baseURL
	baseURL = srv.URL
	t.Cleanup(func() { baseURL = oldBase })

	hook := newWebhookRecorder(t)
	u, _ := url.Parse(srv.URL)
	saveJwCookies(uid, map[string][]*http.Cookie{u.Host: {{Name: "PHPSESSID", Value: "good"}}})
	addMonitorUser(uid)
	SaveWebhook(uid, &WebhookEntry{URL: hook.srv.URL})

	svc := NewJwService()
	t.Cleanup(svc.Close)
	return fake, svc, hook
}

func TestNewScoresNotified(t *testing.T) {
	const uid = "2024010"
	fake, svc, hook := setupScoreFixture(t, uid)

	// 12 门已有成绩（2 页），第一次只建立基线
	for i := 0; i < 12; i++ {
		fake.scores = append(fake.scores, scoreRow(fmt.Sprintf("课程%d", i), "80", "3.0", "2"))
	}
	fake.scores = append(fake.scores, scoreRow("体育", "", "", "1")) // 未录入
	if !svc.checkUser(uid) {
		t.Fatal("baseline check failed")
	}
	hook.none(t)

	// 出了新成绩、体育录入成绩、一门成绩被修改
	fake.mu.Lock()
	fake.scores = append(fake.scores[:12], scoreRow("体育", "优秀", "4.0", "1"), scoreRow("高等数学", "92", "4.2", "4"))
	fake.scores[3] = scoreRow("课程3", "85", "3.5", "2")
	fake.mu.Unlock()
	if !svc.checkUser(uid) {
		t.Fatal("check failed")
	}
	p := hook.next(t)
	if p.Event != EventScoreNew {
		t.Fatalf("event = %q", p.Event)
	}
	for _, want := range []string{"《高等数学》92（绩点 4.2，学分 4）", "《体育》优秀", "《课程3》85（绩点 3.5，学分 2） [由 80 修改]"} {
		if !strings.Contains(p.Text, want) {
			t.Errorf("text missing %q:\n%s", want, p.Text)
		}
	}

	// 没有变化不再通知
	if !svc.checkUser(uid) {
		t.Fatal("check failed")
	}
	hook.none(t)
}

func TestIncompleteScoreBaselineDoesNotNotify(t *testing.T) {
	const uid = "2024011"
	fake, svc, hook := setupScoreFixture(t, uid)

	for i := 0; i < 20; i++ {
		fake.scores = append(fake.scores, scoreRow(fmt.Sprintf("课程%d", i), "80", "3.0", "2"))
	}
	// 第一次第 2 页失败：基线不完整
	fake.failPage = 2
	svc.checkUser(uid)
	hook.none(t)

	// 第二次拿全：第 2 页的老成绩不能当成新成绩
	fake.mu.Lock()
	fake.failPage = 0
	fake.mu.Unlock()
	if !svc.checkUser(uid) {
		t.Fatal("check failed")
	}
	hook.none(t)

	// 之后真正的新成绩才通知
	fake.mu.Lock()
	fake.scores = append(fake.scores, scoreRow("新课", "90", "4.0", "3"))
	fake.mu.Unlock()
	svc.checkUser(uid)
	if p := hook.next(t); !strings.Contains(p.Text, "《新课》90") {
		t.Fatalf("text = %s", p.Text)
	}
}
