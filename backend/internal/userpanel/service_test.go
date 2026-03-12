package userpanel

import (
	"context"
	"testing"
	"time"

	"free-ship-checker-go/internal/checker"
)

func TestNormalizeASIN(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"b0dhczbkw7", "B0DHCZBKW7"},
		{"https://www.amazon.com/Anything/dp/b0dhczbkw7/ref=abc", "B0DHCZBKW7"},
		{"https://www.amazon.com/gp/product/B0DHCZBKW7?th=1", "B0DHCZBKW7"},
		{"https://example.com/path?asin=b0dhczbkw7", "B0DHCZBKW7"},
		{"https://example.com/no-asin", ""},
	}
	for _, tc := range cases {
		if got := normalizeASIN(tc.in); got != tc.want {
			t.Fatalf("normalizeASIN(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
}

func TestCanonicalDedupByASINAndScope(t *testing.T) {
	svc := New(nil)
	res1 := checker.Result{CheckedAt: time.Now().UTC(), FreeShippingCountry: true, FreeShipping: true, Signal: "signal-1", Method: "mock"}
	res2 := checker.Result{CheckedAt: time.Now().UTC().Add(1 * time.Minute), FreeShippingCountry: false, FreeShipping: false, Signal: "signal-2", Method: "mock"}

	item1 := svc.addTrackedItemFromResult(context.Background(), AddTrackedItemReq{URL: "https://www.amazon.com/dp/B0DHCZBKW7?ref=123", Country: "US", ZIP: "10013"}, res1)
	item2 := svc.addTrackedItemFromResult(context.Background(), AddTrackedItemReq{URL: "https://www.amazon.com/gp/product/b0dhczbkw7", Country: "US", ZIP: "10013"}, res2)

	if item1.ID != item2.ID {
		t.Fatalf("expected deduped tracked item, got different IDs: %s vs %s", item1.ID, item2.ID)
	}
	if got := len(svc.ListItems()); got != 1 {
		t.Fatalf("expected 1 tracked item after dedup, got %d", got)
	}
	if item2.Signal != "signal-2" {
		t.Fatalf("expected latest check values to refresh existing item")
	}
	if item2.ProductID == "" {
		t.Fatalf("expected product ID to be assigned")
	}
	if item2.CanonicalURL != "https://www.amazon.com/dp/B0DHCZBKW7" {
		t.Fatalf("unexpected canonical URL: %s", item2.CanonicalURL)
	}
}

func TestCanonicalDedupRespectsCountryZipScope(t *testing.T) {
	svc := New(nil)
	res := checker.Result{CheckedAt: time.Now().UTC(), FreeShippingCountry: true, FreeShipping: true, Signal: "signal", Method: "mock"}

	_ = svc.addTrackedItemFromResult(context.Background(), AddTrackedItemReq{URL: "https://www.amazon.com/dp/B0DHCZBKW7", Country: "US", ZIP: "10013"}, res)
	_ = svc.addTrackedItemFromResult(context.Background(), AddTrackedItemReq{URL: "https://www.amazon.com/dp/B0DHCZBKW7", Country: "US", ZIP: "90210"}, res)
	_ = svc.addTrackedItemFromResult(context.Background(), AddTrackedItemReq{URL: "https://www.amazon.com/dp/B0DHCZBKW7", Country: "IL"}, res)

	if got := len(svc.ListItems()); got != 3 {
		t.Fatalf("expected 3 scoped tracked items, got %d", got)
	}
}
