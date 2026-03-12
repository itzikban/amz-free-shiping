package userpanel

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"net/url"
	"strings"
	"sync"
	"time"

	"free-ship-checker-go/internal/admin"
	"free-ship-checker-go/internal/checker"
	"free-ship-checker-go/internal/notify"
)

type User struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type TrackedItem struct {
	ID                 string    `json:"id"`
	UserID             string    `json:"user_id"`
	ProductID          string    `json:"product_id"`
	ASIN               string    `json:"asin,omitempty"`
	CanonicalURL       string    `json:"canonical_url"`
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

type Product struct {
	ID             string    `json:"id"`
	ASIN           string    `json:"asin,omitempty"`
	CanonicalURL   string    `json:"canonical_url"`
	CanonicalKey   string    `json:"canonical_key"`
	FirstSeenAt    time.Time `json:"first_seen_at"`
	LastObservedAt time.Time `json:"last_observed_at"`
}

type Alert struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
}

type Notification struct {
	ID        string     `json:"id"`
	UserID    string     `json:"user_id"`
	Title     string     `json:"title"`
	Message   string     `json:"message"`
	Read      bool       `json:"read"`
	CreatedAt time.Time  `json:"created_at"`
	ReadAt    *time.Time `json:"read_at,omitempty"`
}

type NotificationPreferences struct {
	InAppEnabled bool `json:"in_app_enabled"`
	OnItemAdded  bool `json:"on_item_added"`
}

type AddTrackedItemReq struct {
	URL     string `json:"url"`
	Country string `json:"country"`
	ZIP     string `json:"zip"`
}

type Service struct {
	checker       *checker.Service
	mu            sync.RWMutex
	user          User
	items         []TrackedItem
	alerts        []Alert
	notifications []Notification
	prefs         NotificationPreferences
	seq           int
	productsByKey map[string]Product
	itemByScope   map[string]int
	outbox        *notify.Service
}

func New(c *checker.Service) *Service {
	return &Service{
		checker:       c,
		user:          User{ID: "test-user", Name: "test-user"},
		items:         []TrackedItem{},
		alerts:        []Alert{},
		notifications: []Notification{},
		prefs:         NotificationPreferences{InAppEnabled: true, OnItemAdded: true},
		productsByKey: map[string]Product{},
		itemByScope:   map[string]int{},
		outbox:        notify.New(notify.NewInMemorySender()),
	}
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

func (s *Service) ListNotifications(unreadOnly bool, limit int) []Notification {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if limit <= 0 {
		limit = 50
	}
	out := make([]Notification, 0, min(len(s.notifications), limit))
	for _, n := range s.notifications {
		if unreadOnly && n.Read {
			continue
		}
		out = append(out, n)
		if len(out) >= limit {
			break
		}
	}
	return out
}

func (s *Service) NotificationPreferences() NotificationPreferences {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.prefs
}

func (s *Service) UpdateNotificationPreferences(p NotificationPreferences) NotificationPreferences {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.prefs = p
	return s.prefs
}

func (s *Service) MarkNotificationRead(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	for i := range s.notifications {
		if s.notifications[i].ID == id {
			if !s.notifications[i].Read {
				s.notifications[i].Read = true
				s.notifications[i].ReadAt = &now
			}
			return true
		}
	}
	return false
}

func (s *Service) MarkAllNotificationsRead() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	count := 0
	for i := range s.notifications {
		if s.notifications[i].Read {
			continue
		}
		s.notifications[i].Read = true
		s.notifications[i].ReadAt = &now
		count++
	}
	return count
}

func (s *Service) AddTrackedItem(ctx context.Context, req AddTrackedItemReq) (TrackedItem, error) {
	res, err := s.checker.CheckURL(ctx, req.URL, req.Country, req.ZIP)
	if err != nil {
		return TrackedItem{}, err
	}
	item, alert, queueNotification := s.addTrackedItemFromResult(req, res)
	if queueNotification {
		_, _ = s.outbox.DispatchDue(ctx, time.Now().UTC(), 25)
		for _, entry := range s.outbox.Entries() {
			if entry.AlertID != alert.ID || entry.Status != notify.StatusDelivered {
				continue
			}
			s.mu.Lock()
			s.seq++
			n := Notification{ID: makeID("notif", s.seq), UserID: s.user.ID, Title: "Tracking updated", Message: alert.Message, Read: false, CreatedAt: time.Now().UTC()}
			s.notifications = append([]Notification{n}, s.notifications...)
			if len(s.notifications) > 200 {
				s.notifications = s.notifications[:200]
			}
			s.mu.Unlock()
			break
		}
	}
	return item, nil
}

func (s *Service) RetryFailedNotifications(ctx context.Context, limit int) (int, error) {
	return s.outbox.DispatchDue(ctx, time.Now().UTC(), limit)
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
