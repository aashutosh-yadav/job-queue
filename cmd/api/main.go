package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/redis/go-redis/v9"

	"job-queue/internal/producer"
	"job-queue/internal/queue"
)

var ctx = context.Background()

func main() {
	rdb := connectRedis()
	defer rdb.Close()

	q := queue.New(rdb)

	http.HandleFunc("/jobs", func(w http.ResponseWriter, r *http.Request) {
		producer.Enqueue(w, r, q)
	})

	port := os.Getenv("PORT_API")
	if port == "" {
		port = "8080"
	}

	log.Println("API running on http://localhost:" + port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
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
