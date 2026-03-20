package checker

import "testing"

func TestBuildFillToThreshold_SortsSuggestionsByScoreAndCloseness(t *testing.T) {
	alts := []Alternative{
		{ASIN: "A1", Title: "A1", PriceUSD: 7.00, URL: "https://amazon.com/dp/A1", FreeShipping: false},
		{ASIN: "A2", Title: "A2", PriceUSD: 10.00, URL: "https://amazon.com/dp/A2", FreeShipping: false},
		{ASIN: "A3", Title: "A3", PriceUSD: 12.00, URL: "https://amazon.com/dp/A3", FreeShipping: true},
	}

	resp := BuildFillToThreshold(40.0, 50.0, alts)
	if resp.MissingAmount != 10.0 {
		t.Fatalf("expected missing amount 10.0, got %.2f", resp.MissingAmount)
	}
	if len(resp.Suggestions) != 3 {
		t.Fatalf("expected 3 suggestions, got %d", len(resp.Suggestions))
	}
	if got := resp.Suggestions[0].ASIN; got != "A2" {
		t.Fatalf("expected best suggestion A2 (exact match), got %s", got)
	}
}

func TestBuildFillToThreshold_ExcludesSinglesAboveThreshold(t *testing.T) {
	// Items up to 3× the threshold are included as "buy instead and qualify for free shipping" suggestions.
	// Items above 3× are excluded as unreasonably expensive.
	alts := []Alternative{
		{ASIN: "A1", Title: "A1", PriceUSD: 55.00, URL: "https://amazon.com/dp/A1", FreeShipping: true},   // included: 55 < 50*3=150
		{ASIN: "A2", Title: "A2", PriceUSD: 9.99, URL: "https://amazon.com/dp/A2", FreeShipping: true},    // included
		{ASIN: "A3", Title: "A3", PriceUSD: 200.00, URL: "https://amazon.com/dp/A3", FreeShipping: true},  // excluded: 200 > 50*3=150
	}

	resp := BuildFillToThreshold(40.0, 50.0, alts)
	if len(resp.Suggestions) != 2 {
		t.Fatalf("expected 2 suggestions (A1 and A2), got %d", len(resp.Suggestions))
	}
	// A3 at $200 should be excluded
	for _, s := range resp.Suggestions {
		if s.ASIN == "A3" {
			t.Fatalf("expected A3 ($200, >3× threshold) to be excluded")
		}
	}
}

func TestBuildFillToThreshold_CombosIncludeSingleItemBundles(t *testing.T) {
	alts := []Alternative{
		{ASIN: "A1", Title: "A1", PriceUSD: 10.00, URL: "https://amazon.com/dp/A1", FreeShipping: false},
		{ASIN: "A2", Title: "A2", PriceUSD: 9.50, URL: "https://amazon.com/dp/A2", FreeShipping: true},
	}

	resp := BuildFillToThreshold(40.0, 50.0, alts)
	if len(resp.Combos) == 0 {
		t.Fatalf("expected combos, got none")
	}

	foundSingle := false
	for _, combo := range resp.Combos {
		if len(combo.Items) == 1 {
			foundSingle = true
			break
		}
	}
	if !foundSingle {
		t.Fatalf("expected at least one single-item combo in response")
	}
}
