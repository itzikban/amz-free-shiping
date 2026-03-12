package userpanel

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"log"
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
	Type      string    `json:"type"`
	CreatedAt time.Time `json:"created_at"`
}

type Notification struct {
	ID        string     `json:"id"`
	UserID    string     `json:"user_id"`
	AlertID   string     `json:"alert_id,omitempty"`
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

const (
	maxNotificationIndexEntries = 2000
	maxAlertMessageEntries      = 1000
)

type Service struct {
	checker           *checker.Service
	mu                sync.RWMutex
	user              User
	items             []TrackedItem
	alerts            []Alert
	notifications     []Notification
	prefs             NotificationPreferences
	seq               int
	productsByKey     map[string]Product
	itemByScope       map[string]int
	outbox            *notify.Service
	notificationIndex map[string]struct{}
	notificationOrder []string
	alertMessages     map[string]string
	alertMessageOrder []string
}

func New(c *checker.Service) *Service {
	return &Service{checker: c, user: User{"test-user", "test-user"}, prefs: NotificationPreferences{true, true}, productsByKey: map[string]Product{}, itemByScope: map[string]int{}, outbox: notify.New(notify.NewInMemorySender()), notificationIndex: map[string]struct{}{}, notificationOrder: []string{}, alertMessages: map[string]string{}, alertMessageOrder: []string{}}
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
	return s.addTrackedItemFromResult(ctx, req, res), nil
}

func (s *Service) addTrackedItemFromResult(ctx context.Context, req AddTrackedItemReq, res checker.Result) TrackedItem {
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
	prevStrict := false
	var item TrackedItem
	if idx, exists := s.itemByScope[scopeKey]; exists && idx >= 0 && idx < len(s.items) {
		dedup = true
		item = s.items[idx]
		prevStrict = item.FreeShippingStrict
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
	alert := s.appendAlert(item, dedup, prevStrict, now)
	enqueue := false
	if s.prefs.InAppEnabled && s.prefs.OnItemAdded && s.outbox != nil {
		key := notify.BuildIdempotencyKey(alert.ID, "in_app", s.user.ID)
		s.outbox.Enqueue(alert.ID, "in_app", s.user.ID, alert.Message, key, now)
		enqueue = true
	}
	s.mu.Unlock()

	if enqueue {
		now := time.Now().UTC()
		if _, err := s.outbox.DispatchDue(ctx, now, 20); err != nil {
			log.Printf("userpanel: dispatching due notifications failed: %v", err)
		}
		s.syncDeliveredNotifications(now)
	}
	return item
}

func (s *Service) appendAlert(item TrackedItem, dedup bool, prevStrict bool, now time.Time) Alert {
	s.seq++
	a := Alert{ID: makeID("alert", s.seq), UserID: s.user.ID, CreatedAt: now}
	if dedup {
		if !prevStrict && item.FreeShippingStrict {
			a.Message = "🎉 Free shipping became available"
			a.Type = "free_shipping_available"
		} else {
			a.Message = "♻️ Tracked item already exists (canonical dedup), refreshed latest check"
			a.Type = "other"
		}
	} else if item.FreeShippingStrict {
		a.Message = "✅ Tracked item added: free shipping for destination"
		a.Type = "free_shipping"
	} else {
		a.Message = "ℹ️ Tracked item added: not free shipping for destination"
		a.Type = "other"
	}
	s.alerts = append([]Alert{a}, s.alerts...)
	s.rememberAlertMessage(a.ID, a.Message)
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
	if s.outbox == nil {
		return 0, nil
	}
	processed, err := s.outbox.DispatchDue(ctx, time.Now().UTC(), limit)
	s.syncDeliveredNotifications(time.Now().UTC())
	return processed, err
}

func (s *Service) syncDeliveredNotifications(now time.Time) {
	if s.outbox == nil {
		return
	}
	entries := s.outbox.Entries()
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.notificationIndex == nil {
		s.notificationIndex = map[string]struct{}{}
	}
	for _, n := range s.notifications {
		s.rememberNotificationIndex(n.AlertID)
	}
	for _, e := range entries {
		if e.Status != notify.StatusDelivered || e.Channel != "in_app" {
			continue
		}
		if _, exists := s.notificationIndex[e.AlertID]; exists {
			continue
		}
		msg := strings.TrimSpace(e.Message)
		if msg == "" {
			if saved := strings.TrimSpace(s.alertMessages[e.AlertID]); saved != "" {
				msg = saved
			} else {
				msg = "Notification delivered"
			}
		}
		s.seq++
		n := Notification{ID: makeID("notif", s.seq), UserID: s.user.ID, AlertID: e.AlertID, Title: "Tracking updated", Message: msg, CreatedAt: now}
		s.notifications = append([]Notification{n}, s.notifications...)
		s.rememberNotificationIndex(e.AlertID)
		if len(s.notifications) > 200 {
			s.notifications = s.notifications[:200]
		}
	}
}

func (s *Service) rememberNotificationIndex(alertID string) {
	if alertID == "" {
		return
	}
	if s.notificationIndex == nil {
		s.notificationIndex = map[string]struct{}{}
	}
	if _, exists := s.notificationIndex[alertID]; exists {
		return
	}
	s.notificationIndex[alertID] = struct{}{}
	s.notificationOrder = append(s.notificationOrder, alertID)
	for len(s.notificationOrder) > maxNotificationIndexEntries {
		drop := s.notificationOrder[0]
		s.notificationOrder = s.notificationOrder[1:]
		delete(s.notificationIndex, drop)
	}
}

func (s *Service) rememberAlertMessage(alertID, message string) {
	if alertID == "" {
		return
	}
	if s.alertMessages == nil {
		s.alertMessages = map[string]string{}
	}
	if _, exists := s.alertMessages[alertID]; !exists {
		s.alertMessageOrder = append(s.alertMessageOrder, alertID)
	}
	s.alertMessages[alertID] = message
	for len(s.alertMessageOrder) > maxAlertMessageEntries {
		drop := s.alertMessageOrder[0]
		s.alertMessageOrder = s.alertMessageOrder[1:]
		delete(s.alertMessages, drop)
	}
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
				c := u[start : start+10]
				if isASIN(c) {
					return c
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
