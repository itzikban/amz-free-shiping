package checker

import (
	"strings"
	"testing"
)

// TestBuildFillToThreshold_SuggestionsHaveNonEmptyURLs verifies that when
// alternatives carry valid ASINs the resulting suggestions all expose a
// non-empty URL field.
func TestBuildFillToThreshold_SuggestionsHaveNonEmptyURLs(t *testing.T) {
	alts := []Alternative{
		{ASIN: "B08N5LNQCX", Title: "Widget A", PriceUSD: 8.99, URL: "https://www.amazon.com/dp/B08N5LNQCX", FreeShipping: false},
		{ASIN: "B07XJ8C8F5", Title: "Widget B", PriceUSD: 12.49, URL: "https://www.amazon.com/dp/B07XJ8C8F5", FreeShipping: true},
		{ASIN: "B09G9FPHY6", Title: "Widget C", PriceUSD: 6.00, URL: "https://www.amazon.com/dp/B09G9FPHY6", FreeShipping: false},
	}

	resp := BuildFillToThreshold(40.0, 50.0, alts)

	if len(resp.Suggestions) == 0 {
		t.Fatal("expected suggestions, got none")
	}

	for i, s := range resp.Suggestions {
		if s.URL == "" {
			t.Errorf("suggestion[%d] (ASIN=%s) has empty URL", i, s.ASIN)
		}
	}
}

// TestBuildFillToThreshold_SuggestionURLsUseAmazonDPFormat verifies that all
// suggestion URLs follow the https://www.amazon.com/dp/<ASIN> pattern when
// the alternatives are built with that format.
func TestBuildFillToThreshold_SuggestionURLsUseAmazonDPFormat(t *testing.T) {
	alts := []Alternative{
		{ASIN: "B08N5LNQCX", Title: "Item X", PriceUSD: 10.00, URL: "https://www.amazon.com/dp/B08N5LNQCX", FreeShipping: false},
		{ASIN: "B07XJ8C8F5", Title: "Item Y", PriceUSD: 7.50, URL: "https://www.amazon.com/dp/B07XJ8C8F5", FreeShipping: true},
	}

	resp := BuildFillToThreshold(40.0, 50.0, alts)

	const dpPrefix = "https://www.amazon.com/dp/"

	if len(resp.Suggestions) == 0 {
		t.Fatal("expected suggestions, got none")
	}

	for i, s := range resp.Suggestions {
		if !strings.HasPrefix(s.URL, dpPrefix) {
			t.Errorf("suggestion[%d] URL %q does not start with %q", i, s.URL, dpPrefix)
		}
		// The ASIN should appear somewhere in the URL path.
		if s.ASIN != "" && !strings.Contains(s.URL, s.ASIN) {
			t.Errorf("suggestion[%d] URL %q does not contain ASIN %q", i, s.URL, s.ASIN)
		}
	}
}

// TestBuildFillToThreshold_NoSuggestionsWhenAltsEmpty verifies the function
// handles an empty alternatives slice gracefully.
func TestBuildFillToThreshold_NoSuggestionsWhenAltsEmpty(t *testing.T) {
	resp := BuildFillToThreshold(30.0, 50.0, nil)

	if resp.CurrentPrice != 30.0 {
		t.Errorf("expected CurrentPrice=30.0, got %.2f", resp.CurrentPrice)
	}
	if resp.MissingAmount != 20.0 {
		t.Errorf("expected MissingAmount=20.0, got %.2f", resp.MissingAmount)
	}
	if len(resp.Suggestions) != 0 {
		t.Errorf("expected no suggestions for empty alts, got %d", len(resp.Suggestions))
	}
	if len(resp.Combos) != 0 {
		t.Errorf("expected no combos for empty alts, got %d", len(resp.Combos))
	}
}

// TestBuildFillToThreshold_AlternativeWithEmptyURLProducesEmptySuggestionURL
// documents the current behaviour: if an Alternative.URL is empty the
// resulting FillSuggestion.URL is also empty. Callers should ensure
// alternatives carry populated URLs.
func TestBuildFillToThreshold_AlternativeWithEmptyURLProducesEmptySuggestionURL(t *testing.T) {
	alts := []Alternative{
		{ASIN: "B08N5LNQCX", Title: "Item with URL", PriceUSD: 9.00, URL: "https://www.amazon.com/dp/B08N5LNQCX", FreeShipping: false},
		{ASIN: "B07INVALID0", Title: "Item missing URL", PriceUSD: 5.00, URL: "", FreeShipping: false},
	}

	resp := BuildFillToThreshold(40.0, 50.0, alts)

	urlPresent := 0
	urlMissing := 0
	for _, s := range resp.Suggestions {
		if s.URL != "" {
			urlPresent++
		} else {
			urlMissing++
		}
	}

	// The item with a populated URL must be present.
	if urlPresent == 0 {
		t.Error("expected at least one suggestion with a non-empty URL")
	}
	// Document that missing URLs pass through unchanged.
	if urlMissing == 0 {
		t.Error("expected at least one suggestion with an empty URL (alt had no URL)")
	}
}
