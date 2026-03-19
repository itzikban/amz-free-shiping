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
	alts := []Alternative{
		{ASIN: "A1", Title: "A1", PriceUSD: 55.00, URL: "https://amazon.com/dp/A1", FreeShipping: true},
		{ASIN: "A2", Title: "A2", PriceUSD: 9.99, URL: "https://amazon.com/dp/A2", FreeShipping: true},
	}

	resp := BuildFillToThreshold(40.0, 50.0, alts)
	if len(resp.Suggestions) != 1 {
		t.Fatalf("expected 1 suggestion under threshold, got %d", len(resp.Suggestions))
	}
	if resp.Suggestions[0].ASIN != "A2" {
		t.Fatalf("expected remaining suggestion A2, got %s", resp.Suggestions[0].ASIN)
	}
}
