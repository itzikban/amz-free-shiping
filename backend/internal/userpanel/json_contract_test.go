package userpanel

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestJSONContract_UsesSnakeCaseFields(t *testing.T) {
	now := time.Now().UTC()
	n := Notification{ID: "n1", UserID: "u1", Title: "t", Message: "m", Read: false, CreatedAt: now}
	a := Alert{ID: "a1", UserID: "u1", Message: "msg", Type: "other", CreatedAt: now}
	u := User{ID: "u1", Name: "test-user"}
	it := TrackedItem{ID: "i1", UserID: "u1", ProductID: "p1", CanonicalURL: "https://www.amazon.com/dp/B09H74FXNW", URL: "https://www.amazon.com/dp/B09H74FXNW", Country: "US", CreatedAt: now, LastCheckedAt: now, FreeShippingStrict: true}

	blob, err := json.Marshal(map[string]any{
		"user":          u,
		"alerts":        []Alert{a},
		"notifications": []Notification{n},
		"items":         []TrackedItem{it},
		"prefs":         NotificationPreferences{InAppEnabled: true, OnItemAdded: true},
	})
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	s := string(blob)
	mustContain := []string{
		`"name":"test-user"`,
		`"created_at"`,
		`"last_checked_at"`,
		`"free_shipping_country"`,
		`"in_app_enabled"`,
	}
	for _, token := range mustContain {
		if !strings.Contains(s, token) {
			t.Fatalf("expected json to contain %s; got: %s", token, s)
		}
	}

	mustNotContain := []string{`"CreatedAt"`, `"LastCheckedAt"`, `"FreeShippingStrict"`, `"Name"`}
	for _, token := range mustNotContain {
		if strings.Contains(s, token) {
			t.Fatalf("expected json to not contain %s; got: %s", token, s)
		}
	}
}
