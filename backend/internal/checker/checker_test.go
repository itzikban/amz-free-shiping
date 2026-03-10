package checker

import (
	"os"
	"testing"
)

func TestAnalyzeHTML_FreeShippingIsrael(t *testing.T) {
	html, err := os.ReadFile("../../testdata/free_israel.html")
	if err != nil {
		t.Fatal(err)
	}
	res := AnalyzeHTML("https://example.com/p1", "IL", string(html))
	if !res.FreeShippingCountry {
		t.Fatalf("expected FreeShippingCountry=true, got false, signal=%s", res.Signal)
	}
}

func TestAnalyzeHTML_USFreeShipping(t *testing.T) {
	html, err := os.ReadFile("../../testdata/free_us.html")
	if err != nil {
		t.Fatal(err)
	}
	res := AnalyzeHTML("https://example.com/p-us", "US", string(html))
	if !res.FreeShippingCountry {
		t.Fatalf("expected US FreeShippingCountry=true, got false, signal=%s", res.Signal)
	}
}

func TestAnalyzeHTML_NotEligible(t *testing.T) {
	html, err := os.ReadFile("../../testdata/not_free.html")
	if err != nil {
		t.Fatal(err)
	}
	res := AnalyzeHTML("https://example.com/p2", "IL", string(html))
	if res.FreeShipping {
		t.Fatalf("expected FreeShipping=false, got true, signal=%s", res.Signal)
	}
}

func TestExtractUSDPrice_PrefersBuyPriceOverPerUnit(t *testing.T) {
	html := `<div>($10.00 / count)</div><div>Buy new: $39.99</div>`
	p := extractUSDPrice(html)
	if p != 39.99 {
		t.Fatalf("expected 39.99, got %.2f", p)
	}
}

func TestExtractUSDPrice_OneTimePurchase(t *testing.T) {
	html := `<div>One-time purchase: $16.19</div><div>Save $2.00</div>`
	p := extractUSDPrice(html)
	if p != 16.19 {
		t.Fatalf("expected 16.19, got %.2f", p)
	}
}

func TestExtractUSDPrice_PrefersProductPriceOverShippingThreshold(t *testing.T) {
	html := `<div>One-time purchase: $5.99</div><div>FREE delivery Monday on orders shipped by Amazon over $35</div>`
	p := extractUSDPrice(html)
	if p != 5.99 {
		t.Fatalf("expected 5.99, got %.2f", p)
	}
}
