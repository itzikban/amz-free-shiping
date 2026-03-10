package monitor

import (
	"context"
	"fmt"
	"sync"
	"time"

	"free-ship-checker-go/internal/checker"
	"github.com/google/uuid"
)

type Notification struct {
	MonitorID string    `json:"monitor_id"`
	At        time.Time `json:"at"`
	Message   string    `json:"message"`
}

type HistoryItem struct {
	At                  time.Time `json:"at"`
	FreeShipping        bool      `json:"free_shipping"`
	FreeShippingCountry bool      `json:"free_shipping_country"`
	Signal              string    `json:"signal"`
	Method              string    `json:"method"`
}

type Monitor struct {
	ID              string        `json:"id"`
	URL             string        `json:"url"`
	Country         string        `json:"country"`
	ZIP             string        `json:"zip,omitempty"`
	IntervalSeconds int           `json:"interval_seconds"`
	Running         bool          `json:"running"`
	LastCheckedAt   *time.Time    `json:"last_checked_at,omitempty"`
	LastStatus      *bool         `json:"last_status,omitempty"`
	LastSignal      string        `json:"last_signal,omitempty"`
	LastMethod      string        `json:"last_method,omitempty"`
	History         []HistoryItem `json:"history"`
}

type StartReq struct {
	URL             string `json:"url"`
	Country         string `json:"country"`
	ZIP             string `json:"zip"`
	IntervalSeconds int    `json:"interval_seconds"`
}

type Service struct {
	checker       *checker.Service
	mu            sync.RWMutex
	monitors      map[string]*Monitor
	cancels       map[string]context.CancelFunc
	notifications []Notification
}

func New(svc *checker.Service) *Service {
	return &Service{checker: svc, monitors: map[string]*Monitor{}, cancels: map[string]context.CancelFunc{}, notifications: []Notification{}}
}

func (s *Service) Start(ctx context.Context, r StartReq) (*Monitor, error) {
	if r.URL == "" {
		return nil, fmt.Errorf("missing url")
	}
	if r.Country == "" {
		r.Country = "US"
	}
	if r.IntervalSeconds <= 0 {
		r.IntervalSeconds = 30
	}
	id := uuid.NewString()
	m := &Monitor{ID: id, URL: r.URL, Country: r.Country, ZIP: r.ZIP, IntervalSeconds: r.IntervalSeconds, Running: true, History: []HistoryItem{}}

	mctx, cancel := context.WithCancel(context.Background())
	s.mu.Lock()
	s.monitors[id] = m
	s.cancels[id] = cancel
	s.mu.Unlock()

	go s.loop(mctx, id)
	return m, nil
}

func (s *Service) loop(ctx context.Context, id string) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	nextRun := time.Now()
	for {
		select {
		case <-ctx.Done():
			s.mu.Lock()
			if m, ok := s.monitors[id]; ok {
				m.Running = false
			}
			s.mu.Unlock()
			return
		case now := <-ticker.C:
			s.mu.RLock()
			m, ok := s.monitors[id]
			s.mu.RUnlock()
			if !ok {
				return
			}
			if now.Before(nextRun) {
				continue
			}
			nextRun = now.Add(time.Duration(m.IntervalSeconds) * time.Second)

			res, err := s.checker.CheckURL(ctx, m.URL, m.Country, m.ZIP)
			if err != nil {
				continue
			}
			status := res.FreeShippingCountry
			at := time.Now().UTC()

			s.mu.Lock()
			prev := m.LastStatus
			m.LastCheckedAt = &at
			m.LastStatus = &status
			m.LastSignal = res.Signal
			m.LastMethod = res.Method
			m.History = append([]HistoryItem{{At: at, FreeShipping: res.FreeShipping, FreeShippingCountry: res.FreeShippingCountry, Signal: res.Signal, Method: res.Method}}, m.History...)
			if len(m.History) > 20 {
				m.History = m.History[:20]
			}
			if prev != nil && *prev != status {
				s.notifications = append([]Notification{{MonitorID: id, At: at, Message: fmt.Sprintf("Status changed: %v -> %v", *prev, status)}}, s.notifications...)
				if len(s.notifications) > 100 {
					s.notifications = s.notifications[:100]
				}
			}
			s.mu.Unlock()
		}
	}
}

func (s *Service) Stop(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if c, ok := s.cancels[id]; ok {
		c()
		delete(s.cancels, id)
	}
	if m, ok := s.monitors[id]; ok {
		m.Running = false
	}
}

func (s *Service) List() []*Monitor {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Monitor, 0, len(s.monitors))
	for _, m := range s.monitors {
		cp := *m
		out = append(out, &cp)
	}
	return out
}

func (s *Service) Notifications() []Notification {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cp := make([]Notification, len(s.notifications))
	copy(cp, s.notifications)
	return cp
}
