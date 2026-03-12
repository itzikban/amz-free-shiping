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
	Attempts       int        `json:"attempts"`
	CreatedAt      time.Time  `json:"created_at"`
	SentAt         *time.Time `json:"sent_at,omitempty"`
	NextAttemptAt  *time.Time `json:"next_attempt_at,omitempty"`
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
	for i, e := range s.entries {
		if e.IdempotencyKey != idempotencyKey {
			continue
		}
		if e.Status != StatusDelivered {
			s.entries[i].Status = StatusPending
			s.entries[i].NextAttemptAt = nil
		}
		return
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
	if limit <= 0 {
		s.mu.Lock()
		limit = len(s.entries)
		s.mu.Unlock()
	}

	processed := 0
	var errs []error
	cursor := 0
	now = now.UTC()

	for processed < limit {
		s.mu.Lock()
		if cursor >= len(s.entries) {
			s.mu.Unlock()
			break
		}

		entry := s.entries[cursor]
		idx := cursor
		cursor++

		if entry.Status != StatusPending {
			s.mu.Unlock()
			continue
		}
		if entry.NextAttemptAt != nil && entry.NextAttemptAt.After(now) {
			s.mu.Unlock()
			continue
		}

		entry.Attempts++
		s.entries[idx].Attempts = entry.Attempts
		s.mu.Unlock()

		err := s.sender.Send(ctx, entry)

		s.mu.Lock()
		if idx >= len(s.entries) || s.entries[idx].IdempotencyKey != entry.IdempotencyKey {
			idx = -1
			for i := range s.entries {
				if s.entries[i].IdempotencyKey == entry.IdempotencyKey {
					idx = i
					break
				}
			}
		}
		if idx >= 0 {
			if err != nil {
				delay := time.Duration(min(s.entries[idx].Attempts, 5)) * time.Minute
				next := now.Add(delay)
				s.entries[idx].Status = StatusPending
				s.entries[idx].NextAttemptAt = &next
				err = errors.New("dispatch failed: " + err.Error())
				errs = append(errs, err)
			} else {
				t := now
				s.entries[idx].Status = StatusDelivered
				s.entries[idx].SentAt = &t
				s.entries[idx].NextAttemptAt = nil
				processed++
			}
		}
		s.mu.Unlock()
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
