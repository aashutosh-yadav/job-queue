package consumer

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"message-queue/internal/queue"
)

const maxAttempts = 2

// process simulates the work for a job. The ID-based failures keep the
// retry and dead-list paths easy to demo.
func process(job queue.Job, attempt int) error {
	if !job.Type.Valid() {
		return fmt.Errorf("unknown job type %q", job.Type)
	}

	if job.ID%3 == 0 {
		return errors.New("simulated permanent failure")
	}
	if job.ID%4 == 0 && attempt == 1 {
		return errors.New("simulated transient failure")
	}

	switch job.Type {
	case queue.SendEmail:
		time.Sleep(time.Second)
		log.Printf("simulated send_email: sent email to %s", job.Payload)
	case queue.ResizeImage:
		time.Sleep(2 * time.Second)
		log.Printf("simulated resize_image: resized %s to 300x300", job.Payload)
	case queue.GeneratePDF:
		time.Sleep(3 * time.Second)
		log.Printf("simulated generate_pdf: generated PDF for %q", job.Payload)
	}

	return nil
}

func Run(ctx context.Context, q *queue.Queue, id int) {
	for {
		job, raw, err := queue.Dequeue(ctx, q)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("worker %d: dequeue error: %v", id, err)
			time.Sleep(time.Second)
			continue
		}

		log.Printf("worker %d: processing job %d (%s)", id, job.ID, job.Type)

		// Try the job twice.
		success := false
		for attempt := 1; attempt <= maxAttempts; attempt++ {
			err := process(job, attempt)
			if err == nil {
				success = true
				break
			}
			log.Printf("worker %d: job %d failed (attempt %d of %d): %v",
				id, job.ID, attempt, maxAttempts, err)
			if attempt < maxAttempts {
				time.Sleep(time.Second)
			}
		}

		if success {
			log.Printf("worker %d: job %d done", id, job.ID)
		} else {
			log.Printf("worker %d: job %d gave up, moving to dead list", id, job.ID)
			if err := queue.Dead(ctx, q, raw); err != nil {
				log.Printf("worker %d: dead-list error: %v", id, err)
			}
		}

		// Either way we are finished with it, so remove it from processing.
		if err := queue.Ack(ctx, q, raw); err != nil {
			log.Printf("worker %d: ack error: %v", id, err)
		}
	}
}
