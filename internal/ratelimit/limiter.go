package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Limiter interface {
	Allow(ctx context.Context, apiKey string) (bool, int, error)
	Limit() int
}

type RedisLimiter struct {
	client *redis.Client
	limit  int
}

const incrScript = `
local current = redis.call('INCR', KEYS[1])
if current == 1 then
    redis.call('EXPIRE', KEYS[1], ARGV[1])
end
return current
`

func NewRedisLimiter(client *redis.Client, limit int) *RedisLimiter {
	return &RedisLimiter{client: client, limit: limit}
}

func (l *RedisLimiter) Allow(ctx context.Context, apiKey string) (bool, int, error) {
	key := fmt.Sprintf("ratelimit:%s:%s", apiKey, time.Now().UTC().Format("2006-01-02T15:04"))

	count, err := l.client.Eval(ctx, incrScript, []string{key}, 60).Int()
	if err != nil {
		return false, 0, err
	}

	if count > l.limit {
		return false, l.limit, nil
	}
	return true, count, nil
}

func (l *RedisLimiter) Limit() int {
	return l.limit
}
