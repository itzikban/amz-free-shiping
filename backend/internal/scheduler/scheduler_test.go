package scheduler

import (
	"context"
	"testing"
	"time"
)

type fakeSource struct {
	items  []DueItem
	marked []int64
}

func (f *fakeSource) ListDueItems(ctx context.Context, now time.Time, limit int) ([]DueItem, error) {
	return f.items, nil
}
func (f *fakeSource) MarkNextCheck(ctx context.Context, trackedItemID int64, next time.Time) error {
	f.marked = append(f.marked, trackedItemID)
	return nil
}

type fakeQueue struct{ queued []int64 }

func (f *fakeQueue) EnqueueCheck(ctx context.Context, item DueItem) error {
	f.queued = append(f.queued, item.TrackedItemID)
	return nil
}

func TestTickEnqueuesAndMarks(t *testing.T) {
	src := &fakeSource{items: []DueItem{{TrackedItemID: 1}, {TrackedItemID: 2}}}
	q := &fakeQueue{}
	svc := &Service{Source: src, Queue: q, DefaultInterval: time.Hour}

	n, err := svc.Tick(context.Background(), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("expected 2, got %d", n)
	}
	if len(src.marked) != 2 || len(q.queued) != 2 {
		t.Fatalf("expected both queues and marks to have 2 entries")
	}
}
