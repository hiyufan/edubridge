package service

import (
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"jww/internal/model"
	"jww/pkg/database"
)

func setupRedis(t *testing.T) *miniredis.Miniredis {
	t.Helper()
	mr := miniredis.RunT(t)
	database.RedisClient = redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { database.RedisClient.Close() })
	return mr
}

func TestICalTokenReplacesOldToken(t *testing.T) {
	setupRedis(t)

	if err := SaveICalToken("2024001", "old"); err != nil {
		t.Fatal(err)
	}
	if err := SaveICalToken("2024001", "new"); err != nil {
		t.Fatal(err)
	}

	if _, err := LookupICalToken("old"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("old token should be revoked, got err=%v", err)
	}
	uid, err := LookupICalToken("new")
	if err != nil || uid != "2024001" {
		t.Fatalf("LookupICalToken(new) = %q, %v", uid, err)
	}

	token, expireAt, err := GetUserICalToken("2024001")
	if err != nil || token != "new" {
		t.Fatalf("GetUserICalToken = %q, %v", token, err)
	}
	if d := time.Until(expireAt); d < ICalTokenTTL-time.Minute || d > ICalTokenTTL {
		t.Fatalf("unexpected expireAt %v", expireAt)
	}
}

func TestICalTokenExpires(t *testing.T) {
	mr := setupRedis(t)

	if err := SaveICalToken("2024001", "tok"); err != nil {
		t.Fatal(err)
	}
	mr.FastForward(ICalTokenTTL + time.Second)

	if _, err := LookupICalToken("tok"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound after TTL, got %v", err)
	}
	if _, _, err := GetUserICalToken("2024001"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound after TTL, got %v", err)
	}
}

func TestScheduleSnapshotRoundTrip(t *testing.T) {
	setupRedis(t)

	if _, err := GetScheduleSnapshot("2024001"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	in := &model.FullSchedule{
		StudentName:   "张三",
		SemesterStart: "2026-09-01",
		Courses: []model.Course{
			{Name: "高等数学", DayOfWeek: 1, PeriodStart: 1, Periods: 2, Weeks: []int{1, 2, 3}},
		},
	}
	if err := saveScheduleSnapshot("2024001", in); err != nil {
		t.Fatal(err)
	}
	out, err := GetScheduleSnapshot("2024001")
	if err != nil {
		t.Fatal(err)
	}
	if out.StudentName != in.StudentName || len(out.Courses) != 1 || out.Courses[0].Name != "高等数学" ||
		len(out.Courses[0].Weeks) != 3 {
		t.Fatalf("snapshot mismatch: %+v", out)
	}
}

func TestWebhookIsPerUser(t *testing.T) {
	setupRedis(t)

	if err := SaveWebhook("alice", &WebhookEntry{URL: "https://a.example/hook", Secret: "s1"}); err != nil {
		t.Fatal(err)
	}

	got, err := GetWebhook("alice")
	if err != nil || got.URL != "https://a.example/hook" || got.Secret != "s1" {
		t.Fatalf("GetWebhook(alice) = %+v, %v", got, err)
	}
	if _, err := GetWebhook("bob"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("bob should have no webhook, got %v", err)
	}
}
