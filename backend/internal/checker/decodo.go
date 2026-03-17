package checker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type decodoReq struct {
	Target string `json:"target"`
	Query  string `json:"query"`
}

type decodoResp struct {
	Results []struct {
		Content    string `json:"content"`
		StatusCode int    `json:"status_code"`
		Query      string `json:"query"`
	} `json:"results"`
}

func (s *Service) decodoAnalyze(ctx context.Context, url, country, zip string) (Result, error) {
	token := os.Getenv("DECODO_BASIC_AUTH")
	if token == "" {
		return Result{}, fmt.Errorf("missing DECODO_BASIC_AUTH")
	}

	payload, _ := json.Marshal(decodoReq{Target: "amazon_product", Query: extractASINorURL(url)})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://scraper-api.decodo.com/v2/scrape", bytes.NewReader(payload))
	if err != nil {
		return Result{}, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Basic "+token)

	cli := &http.Client{Timeout: 35 * time.Second}
	resp, err := cli.Do(req)
	if err != nil {
		return Result{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return Result{}, fmt.Errorf("decodo status: %d", resp.StatusCode)
	}

	var out decodoResp
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return Result{}, err
	}
	if len(out.Results) == 0 {
		return Result{}, fmt.Errorf("decodo empty results")
	}

	html := out.Results[0].Content
	res := AnalyzeHTML(url, country, html)
	res.Method = "decodo"

	low := strings.ToLower(html)
	if strings.ToUpper(country) == "US" && zip != "" {
		if strings.Contains(low, "free delivery") || strings.Contains(low, "free shipping") {
			res.FreeShippingCountry = true
			res.FreeShipping = true
			res.Signal = "us_zip_free_shipping_detected"
		}
	}
	if !res.FreeShippingCountry {
		res.FreeShipping = false
	}
	return res, nil
}

// decodoScrapeAlternatives uses Decodo to scrape Amazon search results for alternatives
func (s *Service) decodoScrapeAlternatives(ctx context.Context, searchQuery string, country string) ([]Alternative, error) {
	token := os.Getenv("DECODO_BASIC_AUTH")
	if token == "" {
		return nil, fmt.Errorf("missing DECODO_BASIC_AUTH")
	}

	// Map countries to appropriate Amazon domains
	// Only use domains that Decodo has been tested to work reliably
	amazonDomain := "com" // default
	switch country {
	case "NL":
		amazonDomain = "nl"
	case "DE":
		amazonDomain = "de"
	case "IL", "MT", "CY":
		amazonDomain = "com" // Use US domain for non-EU countries
	}

	// Build search URL for appropriate Amazon domain
	searchURL := fmt.Sprintf("https://www.amazon.%s/s?k=%s", amazonDomain, strings.ReplaceAll(searchQuery, " ", "+"))
	payload, _ := json.Marshal(decodoReq{Target: "amazon_search", Query: searchURL})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://scraper-api.decodo.com/v2/scrape", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Basic "+token)

	cli := &http.Client{Timeout: 45 * time.Second}
	resp, err := cli.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("decodo status: %d", resp.StatusCode)
	}

	var out decodoResp
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if len(out.Results) == 0 {
		return nil, fmt.Errorf("decodo empty results")
	}

	// Parse HTML to extract product cards with free shipping
	html := out.Results[0].Content
	alternatives := extractAlternativesFromHTML(html)
	return alternatives, nil
}

// extractAlternativesFromHTML parses Amazon search results HTML to extract product info
func extractAlternativesFromHTML(html string) []Alternative {
	var alts []Alternative
	lower := strings.ToLower(html)
	log.Printf("[DEBUG] Searching in HTML of length %d bytes", len(html))

	// Extract all unique ASINs from data-asin attributes
	asinPattern := regexp.MustCompile(`data-asin="([A-Z0-9]{10})"`)
	asinMatches := asinPattern.FindAllStringSubmatch(html, 100)

	seenASINs := make(map[string]bool)

	for _, match := range asinMatches {
		if len(alts) >= 5 {
			break
		}

		if len(match) < 2 || len(match[1]) != 10 {
			continue
		}

		asin := match[1]
		if seenASINs[asin] {
			continue // Skip duplicates
		}
		seenASINs[asin] = true

		// Extract title - look for h2/span aria-label or text near this ASIN
		title := ""

		// Pattern: Look for h2 with aria-label containing product name after data-asin
		// Amazon structure: ... data-asin="ASIN"....<h2 aria-label="Product Name">
		ariaPattern := regexp.MustCompile(fmt.Sprintf(
			`data-asin="%s"[^>]*>.*?<h2[^>]*aria-label="([^"]{20,300})"`,
			regexp.QuoteMeta(asin)))
		if matches := ariaPattern.FindStringSubmatch(html); len(matches) > 1 {
			title = strings.TrimSpace(matches[1])
		}

		// Fallback: Look for h2 > span pattern
		if title == "" {
			spanPattern := regexp.MustCompile(fmt.Sprintf(
				`data-asin="%s"[^>]*>.*?<h2[^>]*>.*?<span[^>]*>([^<]{20,300})</span>`,
				regexp.QuoteMeta(asin)))
			if matches := spanPattern.FindStringSubmatch(html); len(matches) > 1 {
				title = strings.TrimSpace(matches[1])
			}
		}

		// Fallback: Look for any h2 text near the ASIN
		if title == "" {
			h2Pattern := regexp.MustCompile(fmt.Sprintf(
				`data-asin="%s"[^>]*>.*?<h2[^>]*>([^<]{20,300})</h2>`,
				regexp.QuoteMeta(asin)))
			if matches := h2Pattern.FindStringSubmatch(html); len(matches) > 1 {
				candidate := strings.TrimSpace(matches[1])
				if !strings.Contains(candidate, "<") && len(candidate) > 20 {
					title = candidate
				}
			}
		}

		// If no title found, skip this product
		if title == "" || len(title) < 20 {
			continue
		}

		// Extract price from the search results page (general, not product-specific)
		price := 0.0
		pricePattern := regexp.MustCompile(`\$\s*(\d+(?:,\d{3})*(?:\.\d{2})?)`)
		if matches := pricePattern.FindStringSubmatch(html); len(matches) > 1 {
			priceStr := strings.ReplaceAll(matches[1], ",", "")
			if p, err := strconv.ParseFloat(priceStr, 64); err == nil && p > 0 && p < 10000 {
				price = p
			}
		}

		// Extract image URL
		imageURL := ""
		imgPattern := regexp.MustCompile(`src="([^"]*(?:ssl-images-amazon|m\.media-amazon)[^"]*)"`)
		if matches := imgPattern.FindStringSubmatch(html); len(matches) > 1 {
			imageURL = matches[1]
		}

		// Check for free shipping
		freeShipping := strings.Contains(lower, "free shipping") || strings.Contains(lower, "free delivery")

		alt := Alternative{
			ASIN:         asin,
			Title:        title,
			URL:          fmt.Sprintf("https://amazon.com/dp/%s", asin),
			ImageURL:     imageURL,
			PriceUSD:     price,
			FreeShipping: freeShipping,
		}
		alts = append(alts, alt)
		log.Printf("[DEBUG] Extracted product: %s - %s (price: $%.2f)", asin, title[:min(50, len(title))], price)
	}

	log.Printf("[DEBUG] extractAlternativesFromHTML: found %d valid alternatives", len(alts))
	return alts
}

func extractASINorURL(s string) string {
	for i := 0; i+14 <= len(s); i++ {
		if s[i:i+4] == "/dp/" {
			asin := s[i+4:]
			if len(asin) >= 10 {
				return asin[:10]
			}
		}
	}
	return s
}
