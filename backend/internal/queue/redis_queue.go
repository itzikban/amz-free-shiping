package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type RedisQueue struct {
	client *redis.Client
}

func NewRedisQueueFromEnv() (*RedisQueue, error) {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "127.0.0.1:6379"
	}
	pass := os.Getenv("REDIS_PASSWORD")
	db := 0
	c := redis.NewClient(&redis.Options{Addr: addr, Password: pass, DB: db})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := c.Ping(ctx).Err(); err != nil {
		return nil, err
	}
	return &RedisQueue{client: c}, nil
}

func (q *RedisQueue) Enqueue(ctx context.Context, jobType string, payload any) (string, error) {
	b, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	job := Job{
		ID:         uuid.NewString(),
		Type:       jobType,
		Payload:    b,
		Attempts:   0,
		MaxRetries: DefaultMaxRetries,
		CreatedAt:  time.Now().UTC(),
	}
	jb, _ := json.Marshal(job)
	if err := q.client.LPush(ctx, QueueMain, jb).Err(); err != nil {
		return "", err
	}
	return job.ID, nil
}

func (q *RedisQueue) Pop(ctx context.Context, timeout time.Duration) (*Job, error) {
	res, err := q.client.BRPop(ctx, timeout, QueueMain).Result()
	if err != nil {
		return nil, err
	}
	if len(res) != 2 {
		return nil, fmt.Errorf("unexpected BRPOP response")
	}
	var j Job
	if err := json.Unmarshal([]byte(res[1]), &j); err != nil {
		return nil, err
	}
	return &j, nil
}

func (q *RedisQueue) RetryOrDeadLetter(ctx context.Context, job *Job, reason string) error {
	job.Attempts++
	job.LastError = reason
	if job.Attempts > job.MaxRetries {
		jb, _ := json.Marshal(job)
		return q.client.LPush(ctx, QueueDeadLetter, jb).Err()
	}
	backoff := time.Duration(job.Attempts*job.Attempts) * time.Second
	jb, _ := json.Marshal(job)
	// delayed retry via sorted set + poller (simple)
	return q.client.ZAdd(ctx, "jobs:retry", redis.Z{Score: float64(time.Now().Add(backoff).Unix()), Member: string(jb)}).Err()
}

func (q *RedisQueue) DrainDueRetries(ctx context.Context) (int64, error) {
	now := float64(time.Now().Unix())
	items, err := q.client.ZRangeByScore(ctx, "jobs:retry", &redis.ZRangeBy{Min: "-inf", Max: fmt.Sprintf("%f", now)}).Result()
	if err != nil {
		return 0, err
	}
	if len(items) == 0 {
		return 0, nil
	}
	pipe := q.client.TxPipeline()
	for _, it := range items {
		pipe.LPush(ctx, QueueMain, it)
		pipe.ZRem(ctx, "jobs:retry", it)
	}
	if _, err := pipe.Exec(ctx); err != nil {
		return 0, err
	}
	return int64(len(items)), nil
}
