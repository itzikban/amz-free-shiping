package checker

import (
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
)

func TestParseSearchResult(t *testing.T) {
	tests := []struct {
		name         string
		html         string
		wantASIN     string
		wantTitle    string
		wantPrice    float64
		wantImage    string
		wantFreeship bool
	}{
		{
			name: "happy path with all fields and free shipping",
			html: `<div data-component-type="s-search-result" data-asin="B09TEST1AB">
				<h2><a><span>Great Wireless Mouse</span></a></h2>
				<span class="a-price"><span class="a-offscreen">$29.99</span></span>
				<img class="s-image" src="https://images.example.com/mouse.jpg" />
				<span class="a-color-base a-text-bold">FREE Shipping</span>
			</div>`,
			wantASIN:     "B09TEST1AB",
			wantTitle:    "Great Wireless Mouse",
			wantPrice:    29.99,
			wantImage:    "https://images.example.com/mouse.jpg",
			wantFreeship: true,
		},
		{
			name: "missing price element returns PriceUSD 0",
			html: `<div data-component-type="s-search-result" data-asin="B09NOPRIC0">
				<h2><a><span>Budget Widget</span></a></h2>
				<img class="s-image" src="https://images.example.com/widget.jpg" />
			</div>`,
			wantASIN:     "B09NOPRIC0",
			wantTitle:    "Budget Widget",
			wantPrice:    0.0,
			wantImage:    "https://images.example.com/widget.jpg",
			wantFreeship: false,
		},
		{
			name: "ASIN from href fallback when data-asin is missing",
			html: `<div data-component-type="s-search-result">
				<a href="/dp/B0TESTASN1/ref=something">Link</a>
				<h2><a><span>Fallback ASIN Product</span></a></h2>
				<span class="a-price"><span class="a-offscreen">$15.00</span></span>
				<img class="s-image" src="https://images.example.com/fallback.jpg" />
			</div>`,
			wantASIN:     "B0TESTASN1",
			wantTitle:    "Fallback ASIN Product",
			wantPrice:    15.00,
			wantImage:    "https://images.example.com/fallback.jpg",
			wantFreeship: false,
		},
		{
			name: "non-free delivery text does not set FreeShipping",
			html: `<div data-component-type="s-search-result" data-asin="B09NOFREE0">
				<h2><a><span>Standard Delivery Item</span></a></h2>
				<span class="a-price"><span class="a-offscreen">$42.50</span></span>
				<img class="s-image" src="https://images.example.com/std.jpg" />
				<span class="a-color-base a-text-bold">Get it by Tomorrow</span>
			</div>`,
			wantASIN:     "B09NOFREE0",
			wantTitle:    "Standard Delivery Item",
			wantPrice:    42.50,
			wantImage:    "https://images.example.com/std.jpg",
			wantFreeship: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			doc, err := goquery.NewDocumentFromReader(strings.NewReader(tc.html))
			if err != nil {
				t.Fatalf("failed to parse HTML: %v", err)
			}

			sel := doc.Find("div[data-component-type='s-search-result']").First()
			alt := parseSearchResult(sel)

			if alt.ASIN != tc.wantASIN {
				t.Errorf("ASIN: got %q, want %q", alt.ASIN, tc.wantASIN)
			}
			if alt.Title != tc.wantTitle {
				t.Errorf("Title: got %q, want %q", alt.Title, tc.wantTitle)
			}
			if alt.PriceUSD != tc.wantPrice {
				t.Errorf("PriceUSD: got %v, want %v", alt.PriceUSD, tc.wantPrice)
			}
			if alt.ImageURL != tc.wantImage {
				t.Errorf("ImageURL: got %q, want %q", alt.ImageURL, tc.wantImage)
			}
			if alt.FreeShipping != tc.wantFreeship {
				t.Errorf("FreeShipping: got %v, want %v", alt.FreeShipping, tc.wantFreeship)
			}
		})
	}
}
