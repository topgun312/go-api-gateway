package ratelimit

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func newTestRedis(t *testing.T) *redis.Client {
	t.Helper()

	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	return rdb
}

func TestRedisLimiterAllowsUpToLimit(t *testing.T) {
	rdb := newTestRedis(t)
	l := NewRedisLimiter(rdb, 5)

	for i := 1; i <= 5; i++ {
		ok, count, err := l.Allow(context.Background(), "key")
		if err != nil {
			t.Fatalf("request %d: unexpected error: %v", i, err)
		}
		if !ok {
			t.Fatalf("request %d: want allowed, got blocked", i)
		}
		if count != i {
			t.Fatalf("request %d: count = %d, want %d", i, count, i)
		}
	}
}

func TestRedisLimiterBlocksSixthRequest(t *testing.T) {
	rdb := newTestRedis(t)
	l := NewRedisLimiter(rdb, 5)

	for i := 0; i < 5; i++ {
		if _, _, err := l.Allow(context.Background(), "key"); err != nil {
			t.Fatalf("request %d: unexpected error: %v", i+1, err)
		}
	}

	ok, count, err := l.Allow(context.Background(), "key")
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatalf("6th request must be blocked, got allowed (count=%d)", count)
	}
}

func TestRedisLimiterAllowsDifferentKeys(t *testing.T) {
	rdb := newTestRedis(t)
	l := NewRedisLimiter(rdb, 5)

	for i := 0; i < 5; i++ {
		if _, _, err := l.Allow(context.Background(), "user-a"); err != nil {
			t.Fatal(err)
		}
	}

	ok, _, err := l.Allow(context.Background(), "user-b")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("different key must be allowed independently")
	}
}

func TestRedisLimiterLimitGetter(t *testing.T) {
	if got := NewRedisLimiter(nil, 7).Limit(); got != 7 {
		t.Errorf("Limit() = %d, want 7", got)
	}
}
