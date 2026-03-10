package queue

import (
	"context"
	"testing"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestRetryOrDeadLetter_AllowsMaxRetriesBeforeDLQ(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	defer mr.Close()

	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	q := &RedisQueue{client: client}
	job := &Job{ID: "j1", Type: JobTypeCheckItem, MaxRetries: 2}

	ctx := context.Background()
	if err := q.RetryOrDeadLetter(ctx, job, "e1"); err != nil {
		t.Fatal(err)
	}
	if got, _ := mr.DB(0).List(QueueDeadLetter); len(got) != 0 {
		t.Fatalf("expected no dlq on first failure")
	}
	if err := q.RetryOrDeadLetter(ctx, job, "e2"); err != nil {
		t.Fatal(err)
	}
	if got, _ := mr.DB(0).List(QueueDeadLetter); len(got) != 0 {
		t.Fatalf("expected no dlq on second failure (max retry reached but not exceeded)")
	}
	if err := q.RetryOrDeadLetter(ctx, job, "e3"); err != nil {
		t.Fatal(err)
	}
	if got, _ := mr.DB(0).List(QueueDeadLetter); len(got) != 1 {
		t.Fatalf("expected dlq on third failure, got %d", len(got))
	}
}
