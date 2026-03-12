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
)

type User struct{ ID, Name string }

type TrackedItem struct {
	ID, UserID, ProductID, ASIN, CanonicalURL, URL, Country, ZIP string
	CreatedAt, LastCheckedAt                                     time.Time
	LastPriceUSD                                                 float64
	FreeShipping, FreeShippingStrict                             bool
	Signal, Method                                               string
}

type Product struct {
	ID, ASIN, CanonicalURL, CanonicalKey string
	FirstSeenAt, LastObservedAt          time.Time
}

type Alert struct {
	ID, UserID, Message, Type string
	CreatedAt                 time.Time
}

type Notification struct {
	ID, UserID, Title, Message string
	Read                       bool
	CreatedAt                  time.Time
	ReadAt                     *time.Time
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
}

func New(c *checker.Service) *Service {
	return &Service{checker: c, user: User{"test-user", "test-user"}, prefs: NotificationPreferences{true, true}, productsByKey: map[string]Product{}, itemByScope: map[string]int{}}
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

func (s *Service) AddTrackedItem(ctx context.Context, req AddTrackedItemReq) (TrackedItem, error) {
	res, err := s.checker.CheckURL(ctx, req.URL, req.Country, req.ZIP)
	if err != nil {
		return TrackedItem{}, err
	}
	return s.addTrackedItemFromResult(req, res), nil
}

func (s *Service) addTrackedItemFromResult(req AddTrackedItemReq, res checker.Result) TrackedItem {
	now := time.Now().UTC()
	asin := normalizeASIN(req.URL)
	canonicalURL := canonicalProductURL(req.URL, asin)
	canonicalKey := canonicalProductKey(canonicalURL, asin)
	scopeKey := dedupScopeKey(s.user.ID, canonicalKey, req.Country, req.ZIP)

	s.mu.Lock()
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
		item.LastCheckedAt, item.LastPriceUSD, item.FreeShipping, item.FreeShippingStrict, item.Signal, item.Method = res.CheckedAt, res.PriceUSD, res.FreeShipping, res.FreeShippingCountry, res.Signal, res.Method
		item.URL, item.ASIN, item.CanonicalURL, item.ProductID = req.URL, asin, canonicalURL, product.ID
		s.items[idx] = item
	} else {
		s.seq++
		item = TrackedItem{ID: makeID("item", s.seq), UserID: s.user.ID, ProductID: product.ID, ASIN: asin, CanonicalURL: canonicalURL, URL: req.URL, Country: req.Country, ZIP: req.ZIP, CreatedAt: now, LastCheckedAt: res.CheckedAt, LastPriceUSD: res.PriceUSD, FreeShipping: res.FreeShipping, FreeShippingStrict: res.FreeShippingCountry, Signal: res.Signal, Method: res.Method}
		s.items = append([]TrackedItem{item}, s.items...)
		if len(s.items) > 100 {
			s.items = s.items[:100]
		}
		s.rebuildItemIndex()
	}
	alert := s.appendAlert(item, dedup, now)
	if s.prefs.InAppEnabled && s.prefs.OnItemAdded {
		s.seq++
		n := Notification{ID: makeID("notif", s.seq), UserID: s.user.ID, Title: "Tracking updated", Message: alert.Message, CreatedAt: now}
		s.notifications = append([]Notification{n}, s.notifications...)
		if len(s.notifications) > 200 {
			s.notifications = s.notifications[:200]
		}
	}
	s.mu.Unlock()
	return item
}

func (s *Service) appendAlert(item TrackedItem, dedup bool, now time.Time) Alert {
	s.seq++
	a := Alert{ID: makeID("alert", s.seq), UserID: s.user.ID, CreatedAt: now}
	if dedup {
		a.Message = "♻️ Tracked item already exists (canonical dedup), refreshed latest check"
		a.Type = "other"
	} else if item.FreeShippingStrict {
		a.Message = "✅ Tracked item added: free shipping for destination"
		a.Type = "free_shipping"
	} else {
		a.Message = "ℹ️ Tracked item added: not free shipping for destination"
		a.Type = "other"
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
	return 0, nil
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
				c := u[start : start+10]
				if strings.HasPrefix(c, "B0") {
					return c
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
	b := [20]byte{}
	p := len(b)
	for i > 0 {
		p--
		b[p] = byte('0' + i%10)
		i /= 10
	}
	return string(b[p:])
}
