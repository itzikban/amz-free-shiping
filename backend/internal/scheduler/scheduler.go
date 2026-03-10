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
	ListDueItems(ctx context.Context, now time.Time, limit int) ([]DueItem, error)
	MarkNextCheck(ctx context.Context, trackedItemID int64, next time.Time) error
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

	items, err := s.Source.ListDueItems(ctx, now, s.BatchSize)
	if err != nil {
		return 0, err
	}

	enqueued := 0
	for _, item := range items {
		if err := s.Queue.EnqueueCheck(ctx, item); err != nil {
			continue
		}
		next := now.Add(s.DefaultInterval)
		if err := s.Source.MarkNextCheck(ctx, item.TrackedItemID, next); err != nil {
			continue
		}
		enqueued++
	}
	return enqueued, nil
}
