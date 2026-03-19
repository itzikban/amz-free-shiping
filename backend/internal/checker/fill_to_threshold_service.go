package checker

import (
	"context"
	"log"
	"net/url"
	"strings"
)

// BuildFillToThresholdForURL resolves a product check and then builds ranked top-up suggestions.
// It prefers Decodo markdown alternatives when available because they provide richer clickable data
// (ASIN/title/image/url/price) for the Fill-to-Threshold UX.
func (s *Service) BuildFillToThresholdForURL(ctx context.Context, productURL, country, zip string, threshold float64, method string) (FillToThresholdResponse, error) {
	res, err := s.CheckURLWithMethod(ctx, productURL, country, zip, method)
	if err != nil {
		return FillToThresholdResponse{}, err
	}

	// Start with whatever alternatives the check already returned (may be empty when
	// FreeShippingCountry=true because enrichWithAlternatives skips that case).
	alts := res.Alternatives

	// Prefer Decodo markdown alternatives for this feature when they are available.
	asin := extractASINorURL(productURL)
	if len(asin) == 10 {
		domain := amazonDomainOrDefault(productURL)
		if dAlts, derr := s.decodoFetchAlternatives(ctx, asin, domain); derr == nil && len(dAlts) > 0 {
			alts = dAlts
		}
	}

	// If still no alternatives (e.g. FreeShippingCountry=true skipped enrichWithAlternatives,
	// or Decodo returned nothing), fall back to the Amazon scraper using the product title.
	if len(alts) == 0 && res.Title != "" {
		log.Printf("[DEBUG] fill-to-threshold: no alternatives yet, falling back to scraper for: %s", res.Title)
		if scraped, serr := s.scrapeAmazonAlternatives(ctx, res.Title); serr == nil && len(scraped) > 0 {
			alts = scraped
		} else if serr != nil {
			log.Printf("[DEBUG] fill-to-threshold: scraper fallback failed: %v", serr)
		}
	}

	return BuildFillToThreshold(res.PriceUSD, threshold, alts), nil
}

func amazonDomainOrDefault(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Hostname() == "" {
		return "www.amazon.com"
	}
	h := strings.ToLower(u.Hostname())
	if strings.Contains(h, "amazon.") {
		return h
	}
	return "www.amazon.com"
}
