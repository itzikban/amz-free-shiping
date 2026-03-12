package notify

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"sync"
	"time"
)

const (
	StatusPending   = "pending"
	StatusDelivered = "delivered"
	StatusFailed    = "failed"
)

type Entry struct {
	AlertID        string     `json:"alert_id"`
	Channel        string     `json:"channel"`
	Address        string     `json:"address"`
	IdempotencyKey string     `json:"idempotency_key"`
	Status         string     `json:"status"`
	CreatedAt      time.Time  `json:"created_at"`
	SentAt         *time.Time `json:"sent_at,omitempty"`
}

type Sender interface {
	Send(ctx context.Context, e Entry) error
}

type inMemorySender struct{}

func NewInMemorySender() Sender { return inMemorySender{} }

func (inMemorySender) Send(_ context.Context, _ Entry) error { return nil }

type Service struct {
	mu      sync.Mutex
	sender  Sender
	entries []Entry
}

func New(sender Sender) *Service {
	if sender == nil {
		sender = NewInMemorySender()
	}
	return &Service{sender: sender, entries: make([]Entry, 0)}
}

func (s *Service) Enqueue(alertID, channel, address, idempotencyKey string, now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, e := range s.entries {
		if e.IdempotencyKey == idempotencyKey {
			return
		}
	}
	s.entries = append(s.entries, Entry{
		AlertID:        alertID,
		Channel:        channel,
		Address:        address,
		IdempotencyKey: idempotencyKey,
		Status:         StatusPending,
		CreatedAt:      now,
	})
}

func (s *Service) DispatchDue(ctx context.Context, now time.Time, limit int) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if limit <= 0 {
		limit = len(s.entries)
	}
	processed := 0
	var errs []error
	for i := range s.entries {
		if processed >= limit {
			break
		}
		if s.entries[i].Status != StatusPending {
			continue
		}
		if err := s.sender.Send(ctx, s.entries[i]); err != nil {
			s.entries[i].Status = StatusFailed
			errs = append(errs, err)
			continue
		}
		t := now.UTC()
		s.entries[i].Status = StatusDelivered
		s.entries[i].SentAt = &t
		processed++
	}
	return processed, errors.Join(errs...)
}

func (s *Service) Entries() []Entry {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Entry, len(s.entries))
	copy(out, s.entries)
	return out
}

func BuildIdempotencyKey(alertID, channel, address string) string {
	raw := alertID + "|" + channel + "|" + address
	sum := sha1.Sum([]byte(raw))
	return hex.EncodeToString(sum[:])
}
