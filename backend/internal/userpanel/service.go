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

type User struct{ ID, Name string }

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
	ID, UserID, Message, Type string
	CreatedAt                 time.Time
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

type NotificationPreferences struct{ InAppEnabled, OnItemAdded bool }

type AddTrackedItemReq struct{ URL, Country, ZIP string }

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
	out := append([]TrackedItem(nil), s.items...)
	return out
}
func (s *Service) ListAlerts() []Alert {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]Alert(nil), s.alerts...)
}
func (s *Service) GetItem(id string) (TrackedItem, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, it := range s.items {
		if it.ID == id {
			return it, true
		}
	}
	return TrackedItem{}, false
}
func (s *Service) ListNotifications(unreadOnly bool, limit int) []Notification {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if limit <= 0 {
		limit = 50
	}
	out := []Notification{}
	for _, n := range s.notifications {
		if unreadOnly && n.Read {
			continue
		}
		copied := n
		if n.ReadAt != nil {
			t := *n.ReadAt
			copied.ReadAt = &t
		}
		out = append(out, copied)
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
	return p
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
	c := 0
	for i := range s.notifications {
		if s.notifications[i].Read {
			continue
		}
		s.notifications[i].Read = true
		s.notifications[i].ReadAt = &now
		c++
	}
	return c
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
		if n.ReadAt != nil {
			readAt := *n.ReadAt
			n.ReadAt = &readAt
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
	raw := strings.TrimSpace(req.URL)
	if asin := strings.ToUpper(raw); isASIN(asin) {
		req.URL = "https://www.amazon.com/dp/" + asin
	} else {
		req.URL = raw
	}

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
	product.LastObservedAt = now
	s.productsByKey[canonicalKey] = product

	alert := s.appendAlert(item, dedup, now)
	queueNotification := !dedup && s.prefs.InAppEnabled && s.prefs.OnItemAdded
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

func isASIN(s string) bool {
	if len(s) != 10 {
		return false
	}
	for _, r := range s {
		if (r < 'A' || r > 'Z') && (r < '0' || r > '9') {
			return false
		}
	}
	return true
}

func normalizeASIN(in string) string {
	u := strings.ToUpper(strings.TrimSpace(in))
	if isASIN(u) {
		return u
	}
	for _, m := range []string{"/DP/", "/GP/PRODUCT/", "ASIN="} {
		if idx := strings.Index(u, m); idx >= 0 {
			start := idx + len(m)
			if start+10 <= len(u) {
				candidate := u[start : start+10]
				if isASIN(candidate) {
					return candidate
				}
			}
		}
	}
	if parsed, err := url.Parse(in); err == nil {
		q := strings.ToUpper(parsed.Query().Get("asin"))
		if isASIN(q) {
			return q
		}
	}
	return ""
}

func canonicalProductURL(rawURL, asin string) string {
	if asin != "" {
		if u, err := url.Parse(rawURL); err == nil && strings.TrimSpace(u.Host) != "" {
			scheme := u.Scheme
			if scheme == "" {
				scheme = "https"
			}
			return scheme + "://" + u.Host + "/dp/" + asin
		}
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
	normalizedZIP := strings.ToUpper(strings.TrimSpace(zip))
	return strings.ToLower(strings.TrimSpace(userID)) + "|" + strings.ToUpper(strings.TrimSpace(country)) + "|" + normalizedZIP + "|" + canonicalKey
}

func makeID(prefix string, n int) string {
	return prefix + "-" + time.Now().UTC().Format("150405") + "-" + fmtInt(n)
}
func fmtInt(i int) string {
	if i == 0 {
		return "0"
	}
	b := [20]byte{}
	p := len(b)
	for i > 0 {
		p--
		b[p] = byte('0' + i%10)
		i /= 10
	}
	return string(b[p:])
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
