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
		if _, err := s.outbox.DispatchDue(ctx, time.Now().UTC(), 25); err != nil {
			return item, err
		}
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

func (s *Service) addTrackedItemFromResult(req AddTrackedItemReq, res checker.Result) (TrackedItem, Alert, bool) {
	now := time.Now().UTC()
	asin := normalizeASIN(req.URL)
	canonicalURL := canonicalProductURL(req.URL, asin)
	canonicalKey := canonicalProductKey(canonicalURL, asin)
	scopeKey := dedupScopeKey(s.user.ID, canonicalKey, req.Country, req.ZIP)

	s.mu.Lock()
	defer s.mu.Unlock()

	product, ok := s.productsByKey[canonicalKey]
	if !ok {
		s.seq++
		product = Product{ID: makeID("prod", s.seq), ASIN: asin, CanonicalURL: canonicalURL, CanonicalKey: canonicalKey, FirstSeenAt: now}
	}
	product.LastObservedAt = now
	s.productsByKey[canonicalKey] = product

	dedup := false
	var item TrackedItem
	if idx, exists := s.itemByScope[scopeKey]; exists && idx >= 0 && idx < len(s.items) {
		dedup = true
		item = s.items[idx]
		item.LastCheckedAt = res.CheckedAt
		item.LastPriceUSD = res.PriceUSD
		item.FreeShipping = res.FreeShipping
		item.FreeShippingStrict = res.FreeShippingCountry
		item.Signal = res.Signal
		item.Method = res.Method
		item.URL = req.URL
		item.ASIN = asin
		item.CanonicalURL = canonicalURL
		item.ProductID = product.ID
		s.items[idx] = item
	} else {
		s.seq++
		item = TrackedItem{
			ID:                 makeID("item", s.seq),
			UserID:             s.user.ID,
			ProductID:          product.ID,
			ASIN:               asin,
			CanonicalURL:       canonicalURL,
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
		s.rebuildItemIndex()
	}

	alert := s.appendAlert(item, dedup, now)
	queueNotification := s.prefs.InAppEnabled && s.prefs.OnItemAdded
	if queueNotification {
		key := notify.BuildIdempotencyKey(alert.ID, "in_app", s.user.ID)
		s.outbox.Enqueue(alert.ID, "in_app", s.user.ID, key, now)
	}
	return item, alert, queueNotification
}

func (s *Service) appendAlert(item TrackedItem, dedup bool, now time.Time) Alert {
	s.seq++
	a := Alert{ID: makeID("alert", s.seq), UserID: s.user.ID, CreatedAt: now}
	if dedup {
		a.Message = "♻️ Tracked item already exists (canonical dedup), refreshed latest check"
	} else if item.FreeShippingStrict {
		a.Message = "✅ Tracked item added: free shipping for destination"
	} else {
		a.Message = "ℹ️ Tracked item added: not free shipping for destination"
	}
	s.alerts = append([]Alert{a}, s.alerts...)
	if len(s.alerts) > 100 {
		s.alerts = s.alerts[:100]
	}
	return a
}

func (s *Service) rebuildItemIndex() {
	s.itemByScope = map[string]int{}
	for i := range s.items {
		s.itemByScope[dedupScopeKey(s.items[i].UserID, canonicalProductKey(s.items[i].CanonicalURL, s.items[i].ASIN), s.items[i].Country, s.items[i].ZIP)] = i
	}
}

func (s *Service) RetryFailedNotifications(ctx context.Context, limit int) (int, error) {
	return s.outbox.DispatchDue(ctx, time.Now().UTC(), limit)
}

func (s *Service) UserCounts() admin.UserStats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return admin.UserStats{TrackedItems: len(s.items), Alerts: len(s.alerts)}
}

func normalizeASIN(in string) string {
	u := strings.ToUpper(strings.TrimSpace(in))
	if len(u) == 10 && strings.HasPrefix(u, "B0") {
		return u
	}
	for _, m := range []string{"/DP/", "/GP/PRODUCT/", "ASIN="} {
		if idx := strings.Index(u, m); idx >= 0 {
			start := idx + len(m)
			if start+10 <= len(u) {
				candidate := u[start : start+10]
				if strings.HasPrefix(candidate, "B0") {
					return candidate
				}
			}
		}
	}
	if parsed, err := url.Parse(in); err == nil {
		q := strings.ToUpper(parsed.Query().Get("asin"))
		if len(q) == 10 && strings.HasPrefix(q, "B0") {
			return q
		}
	}
	return ""
}

func canonicalProductURL(rawURL, asin string) string {
	if asin != "" {
		return "https://www.amazon.com/dp/" + asin
	}
	if u, err := url.Parse(rawURL); err == nil {
		u.RawQuery = ""
		u.Fragment = ""
		return strings.TrimRight(u.String(), "/")
	}
	return strings.TrimRight(rawURL, "/")
}

func canonicalProductKey(canonicalURL, asin string) string {
	if asin != "" {
		return "asin:" + asin
	}
	sum := sha1.Sum([]byte(strings.ToLower(canonicalURL)))
	return "url:" + hex.EncodeToString(sum[:])
}

func dedupScopeKey(userID, canonicalKey, country, zip string) string {
	return strings.ToLower(strings.TrimSpace(userID)) + "|" + strings.ToUpper(strings.TrimSpace(country)) + "|" + strings.TrimSpace(zip) + "|" + canonicalKey
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
