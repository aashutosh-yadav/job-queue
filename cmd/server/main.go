package main

import (
	"context"
	"log"
	"net/http"

	"github.com/redis/go-redis/v9"

	"message-queue/internal/consumer"
	"message-queue/internal/producer"
	"message-queue/internal/queue"
)

func main() {
	ctx := context.Background()

	// Connect to Redis.
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer rdb.Close()

	// Check the connection.
	err := rdb.Ping(ctx).Err()
	if err != nil {
		log.Fatalf("Cannot connect to Redis: %v", err)
	}

	q := queue.New(rdb)

	// Start 5 consumers.
	for i := 1; i <= 5; i++ {
		go consumer.Run(ctx, q, i)
	}

	// POST endpoint.
	http.HandleFunc("/jobs", func(w http.ResponseWriter, r *http.Request) {
		producer.Enqueue(w, r, q)
	})

	log.Println("Server running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
