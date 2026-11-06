package main

import (
	"context"
	"log"
	"os"

	"github.com/redis/go-redis/v9"

	"job-queue/internal/consumer"
	"job-queue/internal/queue"
)

var ctx = context.Background()

func main() {
	rdb := connectRedis()
	defer rdb.Close()

	q := queue.New(rdb)

	const workers = 5

	for i := 1; i <= workers; i++ {
		go consumer.Run(ctx, q, i)
	}

	log.Printf("Worker started with %d consumers", workers)

	select {}
}

func connectRedis() *redis.Client {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379/0"
	}

	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Fatal("Invalid REDIS_URL: ", err)
	}

	rdb := redis.NewClient(opt)

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatal("Cannot connect to Redis: ", err)
	}

	return rdb
}
