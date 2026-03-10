package scheduler

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeSource struct {
	items    []DueItem
	claimed  []int64
	released []int64
}

func (f *fakeSource) ClaimDueItems(ctx context.Context, now time.Time, next time.Time, limit int) ([]DueItem, error) {
	for _, it := range f.items {
		f.claimed = append(f.claimed, it.TrackedItemID)
	}
	return f.items, nil
}
func (f *fakeSource) ReleaseClaim(ctx context.Context, trackedItemID int64, now time.Time) error {
	f.released = append(f.released, trackedItemID)
	return nil
}

type fakeQueue struct {
	queued []int64
	failID int64
}

func (f *fakeQueue) EnqueueCheck(ctx context.Context, item DueItem) error {
	if f.failID != 0 && item.TrackedItemID == f.failID {
		return errors.New("enqueue failed")
	}
	f.queued = append(f.queued, item.TrackedItemID)
	return nil
}

func TestTickClaimsThenEnqueues(t *testing.T) {
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
	if len(src.claimed) != 2 || len(q.queued) != 2 {
		t.Fatalf("expected claims and queued to have 2 entries")
	}
	if len(src.released) != 0 {
		t.Fatalf("expected no released claims, got %d", len(src.released))
	}
}

func TestTickReleasesClaimWhenEnqueueFails(t *testing.T) {
	src := &fakeSource{items: []DueItem{{TrackedItemID: 1}, {TrackedItemID: 2}}}
	q := &fakeQueue{failID: 2}
	svc := &Service{Source: src, Queue: q, DefaultInterval: time.Hour}

	n, err := svc.Tick(context.Background(), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("expected 1 successful enqueue, got %d", n)
	}
	if len(src.released) != 1 || src.released[0] != 2 {
		t.Fatalf("expected claim release for id=2, got %#v", src.released)
	}
}
