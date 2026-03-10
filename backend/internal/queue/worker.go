package queue

import (
	"context"
	"errors"
	"log"
	"time"
)

type Handler func(ctx context.Context, job *Job) error

type Worker struct {
	Q        *RedisQueue
	Handlers map[string]Handler
}

func (w *Worker) Run(ctx context.Context) error {
	if w.Q == nil {
		return errors.New("nil queue")
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if _, err := w.Q.DrainDueRetries(ctx); err != nil {
			log.Printf("drain retries failed: %v", err)
		}
		job, err := w.Q.Pop(ctx, 2*time.Second)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			log.Printf("queue pop failed: %v", err)
			continue
		}
		h, ok := w.Handlers[job.Type]
		if !ok {
			log.Printf("unknown job type: %s", job.Type)
			if derr := w.Q.RetryOrDeadLetter(ctx, job, "unknown_job_type"); derr != nil {
				log.Printf("failed to move unknown job %s to retry/dlq: %v", job.ID, derr)
			}
			continue
		}
		if err := h(ctx, job); err != nil {
			log.Printf("job failed %s: %v", job.ID, err)
			if derr := w.Q.RetryOrDeadLetter(ctx, job, err.Error()); derr != nil {
				log.Printf("failed to move failed job %s to retry/dlq: %v", job.ID, derr)
			}
		}
	}
}
