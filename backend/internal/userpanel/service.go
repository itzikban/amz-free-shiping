package userpanel

import (
	"context"
	"sync"
	"time"

	"free-ship-checker-go/internal/admin"
	"free-ship-checker-go/internal/checker"
)

type User struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type TrackedItem struct {
	ID                 string    `json:"id"`
	UserID             string    `json:"user_id"`
	URL                string    `json:"url"`
	Country            string    `json:"country"`
	ZIP                string    `json:"zip,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
	LastCheckedAt      time.Time `json:"last_checked_at"`
	LastPriceUSD       float64   `json:"last_price_usd,omitempty"`
	FreeShipping       bool      `json:"free_shipping"`
	FreeShippingStrict bool      `json:"free_shipping_country"`
	Signal             string    `json:"signal"`
	Method             string    `json:"method"`
}

type Alert struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
}

type AddTrackedItemReq struct {
	URL     string `json:"url"`
	Country string `json:"country"`
	ZIP     string `json:"zip"`
}

type Service struct {
	checker *checker.Service
	mu      sync.RWMutex
	user    User
	items   []TrackedItem
	alerts  []Alert
	seq     int
}

func New(c *checker.Service) *Service {
	return &Service{checker: c, user: User{ID: "test-user", Name: "test-user"}, items: []TrackedItem{}, alerts: []Alert{}}
}

func (s *Service) Me() User { return s.user }

func (s *Service) ListItems() []TrackedItem {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]TrackedItem, len(s.items))
	copy(out, s.items)
	return out
}

func (s *Service) ListAlerts() []Alert {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Alert, len(s.alerts))
	copy(out, s.alerts)
	return out
}

func (s *Service) AddTrackedItem(ctx context.Context, req AddTrackedItemReq) (TrackedItem, error) {
	res, err := s.checker.CheckURL(ctx, req.URL, req.Country, req.ZIP)
	if err != nil {
		return TrackedItem{}, err
	}
	now := time.Now().UTC()

	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	id := makeID("item", s.seq)
	item := TrackedItem{
		ID:                 id,
		UserID:             s.user.ID,
		URL:                req.URL,
		Country:            req.Country,
		ZIP:                req.ZIP,
		CreatedAt:          now,
		LastCheckedAt:      res.CheckedAt,
		LastPriceUSD:       res.PriceUSD,
		FreeShipping:       res.FreeShipping,
		FreeShippingStrict: res.FreeShippingCountry,
		Signal:             res.Signal,
		Method:             res.Method,
	}
	s.items = append([]TrackedItem{item}, s.items...)
	if len(s.items) > 100 {
		s.items = s.items[:100]
	}

	s.seq++
	alert := Alert{ID: makeID("alert", s.seq), UserID: s.user.ID, CreatedAt: now}
	if item.FreeShippingStrict {
		alert.Message = "✅ Tracked item added: free shipping for destination"
	} else {
		alert.Message = "ℹ️ Tracked item added: not free shipping for destination"
	}
	s.alerts = append([]Alert{alert}, s.alerts...)
	if len(s.alerts) > 100 {
		s.alerts = s.alerts[:100]
	}
	return item, nil
}

func (s *Service) UserCounts() admin.UserStats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return admin.UserStats{TrackedItems: len(s.items), Alerts: len(s.alerts)}
}

func makeID(prefix string, n int) string {
	return prefix + "-" + time.Now().UTC().Format("150405") + "-" + fmtInt(n)
}
func fmtInt(i int) string {
	if i == 0 {
		return "0"
	}
	buf := [20]byte{}
	bp := len(buf)
	for i > 0 {
		bp--
		buf[bp] = byte('0' + i%10)
		i /= 10
	}
	return string(buf[bp:])
}
