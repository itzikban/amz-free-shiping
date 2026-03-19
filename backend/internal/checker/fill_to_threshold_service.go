package checker

import (
	"context"
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

	alts := res.Alternatives

	// Prefer Decodo markdown alternatives for this feature when they are available.
	asin := extractASINorURL(productURL)
	if len(asin) == 10 {
		domain := amazonDomainOrDefault(productURL)
		if dAlts, derr := s.decodoFetchAlternatives(ctx, asin, domain); derr == nil && len(dAlts) > 0 {
			alts = dAlts
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
