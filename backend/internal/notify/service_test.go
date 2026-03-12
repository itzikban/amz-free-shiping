package notify

import (
	"context"
	"errors"
	"testing"
	"time"
)

type flakySender struct{ calls int }

func (f *flakySender) Send(_ context.Context, _ Entry) error {
	f.calls++
	if f.calls == 1 {
		return errors.New("temporary outage")
	}
	return nil
}

func TestDispatchDue_RetryAndIdempotency(t *testing.T) {
	s := New(&flakySender{})
	now := time.Now().UTC()
	key := BuildIdempotencyKey("a1", "in_app", "u1")

	s.Enqueue("a1", "in_app", "u1", key, now)
	s.Enqueue("a1", "in_app", "u1", key, now)
	if got := len(s.Entries()); got != 1 {
		t.Fatalf("expected single deduped outbox entry, got %d", got)
	}

	processed, err := s.DispatchDue(context.Background(), now, 10)
	if err == nil {
		t.Fatal("expected first dispatch to fail")
	}
	if processed != 0 {
		t.Fatalf("expected 0 processed on failure, got %d", processed)
	}
	entries := s.Entries()
	if entries[0].Attempts != 1 {
		t.Fatalf("expected attempts=1, got %d", entries[0].Attempts)
	}
	if entries[0].NextAttemptAt == nil {
		t.Fatal("expected next attempt to be scheduled")
	}

	processed, err = s.DispatchDue(context.Background(), entries[0].NextAttemptAt.Add(2*time.Second), 10)
	if err != nil {
		t.Fatalf("expected retry dispatch success, got err=%v", err)
	}
	if processed != 1 {
		t.Fatalf("expected 1 processed after retry, got %d", processed)
	}
	entries = s.Entries()
	if entries[0].Status != StatusDelivered {
		t.Fatalf("expected delivered status, got %s", entries[0].Status)
	}
	if entries[0].SentAt == nil {
		t.Fatal("expected sent_at to be set")
	}
}
