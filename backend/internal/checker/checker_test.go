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
