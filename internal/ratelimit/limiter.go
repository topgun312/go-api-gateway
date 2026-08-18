// Package ratelimit реализует rate limiting по алгоритму fixed window
// поверх Redis (INCR + EXPIRE атомарным Lua-скриптом).
package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Limiter — интерфейс rate limiter'а. Реализация должна быть потокобезопасной.
type Limiter interface {
	// Allow увеличивает счётчик запросов для apiKey и возвращает
	// false, если лимит на текущее окно исчерпан. Вторым значением
	// возвращается текущее число запросов в окне.
	Allow(ctx context.Context, apiKey string) (bool, int, error)
	// Limit возвращает максимальное число запросов в одно окно.
	Limit() int
}

// RedisLimiter — реализация Limiter на Redis по алгоритму fixed window:
// атомарный Lua-скрипт INCR + EXPIRE. Окно — календарная минута.
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

// NewRedisLimiter создаёт RedisLimiter с заданным лимитом запросов в минуту.
func NewRedisLimiter(client *redis.Client, limit int) *RedisLimiter {
	return &RedisLimiter{client: client, limit: limit}
}

// Allow реализует Limiter.Allow поверх Redis.
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

// Limit возвращает лимит запросов в минуту.
func (l *RedisLimiter) Limit() int {
	return l.limit
}
