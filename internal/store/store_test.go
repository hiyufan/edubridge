package store_test

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"jww/internal/model"
	"jww/internal/store"
)

var ctx = context.Background()

func newStore(t *testing.T) (*store.Store, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	s := store.New(redis.NewClient(&redis.Options{Addr: mr.Addr()}))
	t.Cleanup(func() { s.Close() })
	return s, mr
}

func TestICalTokens(t *testing.T) {
	s, mr := newStore(t)
	s.SaveICalToken(ctx, "u", "old", time.Hour)
	s.SaveICalToken(ctx, "u", "new", time.Hour)
	if _, err := s.ICalTokenUser(ctx, "old"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("old token: %v", err)
	}
	if uid, err := s.ICalTokenUser(ctx, "new"); err != nil || uid != "u" {
		t.Fatalf("new token: %q %v", uid, err)
	}
	if tok, exp, err := s.UserICalToken(ctx, "u"); err != nil || tok != "new" || time.Until(exp) < 59*time.Minute {
		t.Fatalf("user token: %q %v %v", tok, exp, err)
	}
	mr.FastForward(2 * time.Hour)
	if _, err := s.ICalTokenUser(ctx, "new"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("expired: %v", err)
	}
}

func TestSessionAndSnapshot(t *testing.T) {
	s, _ := newStore(t)
	if _, err := s.Cookies(ctx, "u"); !errors.Is(err, store.ErrNotFound) {
		t.Fatal(err)
	}
	s.SaveCookies(ctx, "u", map[string][]*http.Cookie{"jw": {{Name: "PHPSESSID", Value: "x"}}})
	if c, err := s.Cookies(ctx, "u"); err != nil || c["jw"][0].Value != "x" {
		t.Fatalf("cookies: %v %v", c, err)
	}

	s.AddMonitored(ctx, "u")
	if ok, _ := s.IsMonitored(ctx, "u"); !ok {
		t.Fatal("should be monitored")
	}
	if was, _ := s.RemoveMonitored(ctx, "u"); !was {
		t.Fatal("was monitored")
	}
	if was, _ := s.RemoveMonitored(ctx, "u"); was {
		t.Fatal("already removed")
	}

	full := &model.FullSchedule{StudentName: "张三", Courses: []model.Course{{Name: "数学", Weeks: []int{1, 2}}}}
	s.SaveSnapshot(ctx, "u", full, true)
	got, complete, err := s.Snapshot(ctx, "u")
	if err != nil || !complete || got.Courses[0].Name != "数学" || len(got.Courses[0].Weeks) != 2 {
		t.Fatalf("snapshot: %+v %v %v", got, complete, err)
	}

	k, _ := s.KnownScores(ctx, "u")
	if k.Grades == nil || k.Complete {
		t.Fatalf("empty known: %+v", k)
	}
}

func TestHistoryTrimmed(t *testing.T) {
	s, _ := newStore(t)
	for i := 0; i < 35; i++ {
		s.PushHistory(ctx, "u", &model.HistoryEntry{Event: "e", Text: string(rune('a' + i%26))})
	}
	h, err := s.History(ctx, "u", 100)
	if err != nil || len(h) != 30 || h[0].Text != string(rune('a'+34%26)) {
		t.Fatalf("history: %d %v", len(h), err)
	}
}

func TestReminderUsersSet(t *testing.T) {
	s, _ := newStore(t)
	s.SaveReminder(ctx, "u", &model.ReminderSettings{Daily: true})
	if users, _ := s.ReminderUsers(ctx); len(users) != 1 {
		t.Fatalf("users = %v", users)
	}
	s.SaveReminder(ctx, "u", &model.ReminderSettings{})
	if users, _ := s.ReminderUsers(ctx); len(users) != 0 {
		t.Fatalf("users = %v", users)
	}
	if first, _ := s.MarkReminderSent(ctx, "u", "x"); !first {
		t.Fatal("first mark")
	}
	if again, _ := s.MarkReminderSent(ctx, "u", "x"); again {
		t.Fatal("second mark")
	}
}

func TestRefreshTokens(t *testing.T) {
	s, mr := newStore(t)
	s.SaveRefreshToken(ctx, "t1", "u", time.Now().Add(time.Hour))
	if uid, err := s.RefreshTokenUser(ctx, "t1"); err != nil || uid != "u" {
		t.Fatalf("lookup: %q %v", uid, err)
	}
	s.DeleteRefreshToken(ctx, "t1", "u")
	if _, err := s.RefreshTokenUser(ctx, "t1"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("revoked: %v", err)
	}
	s.SaveRefreshToken(ctx, "t2", "u", time.Now().Add(time.Hour))
	mr.FastForward(2 * time.Hour)
	if _, err := s.RefreshTokenUser(ctx, "t2"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("expired: %v", err)
	}
}
