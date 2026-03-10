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

		_, _ = w.Q.DrainDueRetries(ctx)
		job, err := w.Q.Pop(ctx, 2*time.Second)
		if err != nil {
			continue
		}
		h, ok := w.Handlers[job.Type]
		if !ok {
			log.Printf("unknown job type: %s", job.Type)
			_ = w.Q.RetryOrDeadLetter(ctx, job, "unknown_job_type")
			continue
		}
		if err := h(ctx, job); err != nil {
			log.Printf("job failed %s: %v", job.ID, err)
			_ = w.Q.RetryOrDeadLetter(ctx, job, err.Error())
		}
	}
}
