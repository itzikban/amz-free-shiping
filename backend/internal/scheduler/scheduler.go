package scheduler

import (
	"context"
	"time"
)

type DueItem struct {
	TrackedItemID int64
	Country       string
	ZIP           string
	URL           string
}

type DueItemSource interface {
	ClaimDueItems(ctx context.Context, now time.Time, next time.Time, limit int) ([]DueItem, error)
	ReleaseClaim(ctx context.Context, trackedItemID int64, now time.Time) error
}

type Enqueuer interface {
	EnqueueCheck(ctx context.Context, item DueItem) error
}

type Service struct {
	Source          DueItemSource
	Queue           Enqueuer
	DefaultInterval time.Duration
	BatchSize       int
}

func (s *Service) Tick(ctx context.Context, now time.Time) (int, error) {
	if s.BatchSize <= 0 {
		s.BatchSize = 200
	}
	if s.DefaultInterval <= 0 {
		s.DefaultInterval = 6 * time.Hour
	}

	next := now.Add(s.DefaultInterval)
	items, err := s.Source.ClaimDueItems(ctx, now, next, s.BatchSize)
	if err != nil {
		return 0, err
	}

	enqueued := 0
	for _, item := range items {
		if err := s.Queue.EnqueueCheck(ctx, item); err != nil {
			_ = s.Source.ReleaseClaim(ctx, item.TrackedItemID, now)
			continue
		}
		enqueued++
	}
	return enqueued, nil
}
