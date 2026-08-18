package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"go-api-gateway/internal/config"
	"go-api-gateway/internal/handlers"
	"go-api-gateway/internal/proxy"
	"go-api-gateway/internal/ratelimit"
	"go-api-gateway/internal/storage"
)

func main() {
	cfg := config.Load()

	store, err := storage.NewStore(cfg.DBDSN)
	if err != nil {
		log.Fatal("db: ", err)
	}
	defer store.Close()

	rdb := storage.NewRedisClient(cfg.RedisAddr)
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Fatal("redis: ", err)
	}
	defer rdb.Close()

	limiter := ratelimit.NewRedisLimiter(rdb, cfg.MaxRequests)
	p := proxy.NewOllamaProxy(cfg.OllamaURL + "/api/generate")

	srv := &http.Server{
		Addr:         cfg.ServerPort,
		Handler:      handlers.NewGateway(store, limiter, p),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
	}

	log.Println("gateway listening on", cfg.ServerPort)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}

}
