package storage

import "github.com/redis/go-redis/v9"

// NewRedisClient создаёт клиент Redis. Соединение устанавливается лениво,
// при первом обращении к серверу.
func NewRedisClient(addr string) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr: addr,
	})
}
