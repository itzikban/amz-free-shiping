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

func TestAnalyzeHTML_ExtractsProductTitle(t *testing.T) {
	html := `<html><body><span id="productTitle"> Test Product Title </span></body></html>`
	res := AnalyzeHTML("https://example.com/dp/B000TEST", "US", html)
	if res.Title != "Test Product Title" {
		t.Fatalf("expected 'Test Product Title', got %q", res.Title)
	}
}

func TestAnalyzeHTML_FallsBackToTitleTag(t *testing.T) {
	html := `<html><head><title>Amazon.com: Fallback Title</title></head><body></body></html>`
	res := AnalyzeHTML("https://example.com/dp/B000TEST", "US", html)
	if res.Title != "Amazon.com: Fallback Title" {
		t.Fatalf("expected 'Amazon.com: Fallback Title', got %q", res.Title)
	}
}

func TestAnalyzeHTML_ExtractsMainImage(t *testing.T) {
	html := `<html><body><img id="landingImage" data-old-hires="https://images-na.ssl-images-amazon.com/images/I/large.jpg" src="https://images-na.ssl-images-amazon.com/images/I/small.jpg" /></body></html>`
	res := AnalyzeHTML("https://example.com/dp/B000TEST", "US", html)
	if res.ImageURL != "https://images-na.ssl-images-amazon.com/images/I/large.jpg" {
		t.Fatalf("expected large image URL, got %q", res.ImageURL)
	}
}

func TestAnalyzeHTML_ExtractsImageFromSrc(t *testing.T) {
	html := `<html><body><div id="imgTagWrapperId"><img src="https://images-na.ssl-images-amazon.com/images/I/product.jpg" /></div></body></html>`
	res := AnalyzeHTML("https://example.com/dp/B000TEST", "US", html)
	if res.ImageURL != "https://images-na.ssl-images-amazon.com/images/I/product.jpg" {
		t.Fatalf("expected product image URL, got %q", res.ImageURL)
	}
}

func TestAnalyzeHTML_NoImageWhenMissing(t *testing.T) {
	html := `<html><body><p>no images here</p></body></html>`
	res := AnalyzeHTML("https://example.com/dp/B000TEST", "US", html)
	if res.ImageURL != "" {
		t.Fatalf("expected empty ImageURL, got %q", res.ImageURL)
	}
}
