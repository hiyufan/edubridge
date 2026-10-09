package auth

import (
	"context"
	"errors"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"jww/internal/store"
)

func TestIssueRefreshRevoke(t *testing.T) {
	ctx := context.Background()
	mr := miniredis.RunT(t)
	s := New("access-secret", "refresh-secret", store.New(redis.NewClient(&redis.Options{Addr: mr.Addr()})))

	access, refresh, err := s.Issue(ctx, "2024001")
	if err != nil {
		t.Fatal(err)
	}
	if uid, err := s.Verify(access); err != nil || uid != "2024001" {
		t.Fatalf("verify: %q %v", uid, err)
	}
	if _, err := s.Verify(refresh); !errors.Is(err, ErrInvalidToken) {
		t.Fatal("refresh token must not work as access token")
	}

	uid, access2, refresh2, err := s.Refresh(ctx, refresh)
	if err != nil || uid != "2024001" || access2 == "" {
		t.Fatalf("refresh: %v", err)
	}
	if _, _, _, err := s.Refresh(ctx, refresh); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("old refresh token should be rotated out: %v", err)
	}

	if uid, err := s.Revoke(ctx, refresh2); err != nil || uid != "2024001" {
		t.Fatalf("revoke: %q %v", uid, err)
	}
	if _, _, _, err := s.Refresh(ctx, refresh2); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("revoked: %v", err)
	}

	other := New("x", "y", store.New(redis.NewClient(&redis.Options{Addr: mr.Addr()})))
	if _, err := other.Verify(access); !errors.Is(err, ErrInvalidToken) {
		t.Fatal("wrong secret must fail")
	}
}
