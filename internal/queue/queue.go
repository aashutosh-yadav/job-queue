package queue

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// Three Redis lists: waiting, in-flight, and failed jobs.
const queueKey = "job"
const processingKey = "job:processing"
const deadKey = "job:dead"

type Queue struct {
	rdb *redis.Client
}

// New creates a Queue that uses the given Redis client.
func New(rdb *redis.Client) *Queue {
	return &Queue{rdb: rdb}
}

// Enqueue adds a job to the left side of the list.
func Enqueue(ctx context.Context, q *Queue, job Job) error {
	data, err := json.Marshal(job)
	if err != nil {
		return err
	}
	return q.rdb.LPush(ctx, queueKey, data).Err()
}

// Dequeue waits for a job and MOVES it to the processing list
// (it is not deleted yet). It returns the job and its raw text,
// which we need later for Ack and Dead.
func Dequeue(ctx context.Context, q *Queue) (Job, string, error) {
	raw, err := q.rdb.BLMove(ctx, queueKey, processingKey, "RIGHT", "LEFT", 0).Result()
	if err != nil {
		return Job{}, "", err
	}

	var job Job
	err = json.Unmarshal([]byte(raw), &job)
	if err != nil {
		// Broken job: put it in the dead list so it doesn't get stuck.
		Dead(ctx, q, raw)
		Ack(ctx, q, raw)
		return Job{}, "", err
	}

	return job, raw, nil
}

// Ack removes a finished job from the processing list.
func Ack(ctx context.Context, q *Queue, raw string) error {
	return q.rdb.LRem(ctx, processingKey, 1, raw).Err()
}

// Dead stores a job that failed every attempt.
func Dead(ctx context.Context, q *Queue, raw string) error {
	return q.rdb.LPush(ctx, deadKey, raw).Err()
}

func PrintQueue(ctx context.Context, q *Queue) error {
	jobs, err := q.rdb.LRange(ctx, queueKey, 0, -1).Result()
	if err != nil {
		return err
	}

	fmt.Println("-----Current Redis Job Queue--------")
	for i, data := range jobs {
		var job Job
		err := json.Unmarshal([]byte(data), &job)
		if err != nil {
			fmt.Printf("Index %d: invalid job: %v\n", i, err)
			continue
		}
		fmt.Printf("Index : %d | ID : %d | Type: %s | Payload: %s\n",
			i, job.ID, job.Type, job.Payload)
	}
	fmt.Println("------------------------------------")

	return nil
}
