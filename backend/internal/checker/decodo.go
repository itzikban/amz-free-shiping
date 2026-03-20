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
		// Amazon uses "free delivery" (EU/UK) and "free shipping" (US) — check both
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

// decodoFetchAlternatives fetches alternatives using amazon_product + markdown:true,
// then parses the "Similar items that may deliver to you quickly" section.
// This is more reliable than amazon_search because Amazon curates the similar-items
// list itself — guaranteeing relevant, in-category results.
func (s *Service) decodoFetchAlternatives(ctx context.Context, asin, domain string) ([]Alternative, error) {
	token := os.Getenv("DECODO_BASIC_AUTH")
	if token == "" {
		return nil, fmt.Errorf("missing DECODO_BASIC_AUTH")
	}
	if asin == "" {
		return nil, fmt.Errorf("no ASIN to look up alternatives for")
	}
	if domain == "" {
		domain = "www.amazon.com"
	}

	payload, _ := json.Marshal(struct {
		Target            string `json:"target"`
		Query             string `json:"query"`
		Headless          string `json:"headless"`
		AutoselectVariant bool   `json:"autoselect_variant"`
		Markdown          bool   `json:"markdown"`
	}{
		Target:            "amazon_product",
		Query:             asin,
		Headless:          "html",
		AutoselectVariant: true,
		Markdown:          true,
	})

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
		return nil, fmt.Errorf("decodo decode: %w", err)
	}
	if len(out.Results) == 0 || len(out.Results[0].Content) < 1000 {
		return nil, fmt.Errorf("decodo empty or schema-only response")
	}

	markdown := out.Results[0].Content
	log.Printf("[DEBUG] decodoFetchAlternatives: markdown len=%d", len(markdown))
	alts := extractAlternativesFromMarkdown(markdown, domain)
	log.Printf("[DEBUG] decodoFetchAlternatives: extracted %d alternatives", len(alts))
	return alts, nil
}

var (
	reMarkdownAsin     = regexp.MustCompile(`/dp/([A-Z0-9]{10})`)
	reMarkdownImage    = regexp.MustCompile(`!\[[^\]]*\]\((https?://[^\)]+)\)`)
	reMarkdownTitle    = regexp.MustCompile(`(?:^|[^!])\[([^\]]{20,})\]\((?:/[A-Za-z]|https?://www\.amazon)`)
	reMarkdownPrice    = regexp.MustCompile(`\[\$([0-9,]+\.[0-9]{2})|\$([0-9,]+\.[0-9]{2})`)
	reMarkdownAltText  = regexp.MustCompile(`!\[([^\]]{20,})\]`)
	reMarkdownListItem = regexp.MustCompile(`\n\d+\. `)
)

// extractAlternativesFromMarkdown parses the "Similar items that may deliver to you quickly"
// numbered list from an Amazon product page returned as markdown by Decodo.
func extractAlternativesFromMarkdown(markdown, domain string) []Alternative {
	// Locate the similar-items section
	sectionStart := strings.Index(markdown, "## Similar items that may deliver to you quickly")
	if sectionStart < 0 {
		sectionStart = strings.Index(markdown, "## More items to explore")
	}
	if sectionStart < 0 {
		log.Printf("[DEBUG] extractAlternativesFromMarkdown: similar-items section not found")
		return nil
	}

	// Find end of section (next major heading)
	endIdx := len(markdown)
	for _, marker := range []string{"## Product information", "## Frequently bought together", "## Product description"} {
		if pos := strings.Index(markdown[sectionStart+50:], marker); pos >= 0 {
			if abs := sectionStart + 50 + pos; abs < endIdx {
				endIdx = abs
			}
		}
	}

	section := markdown[sectionStart:endIdx]
	itemParts := reMarkdownListItem.Split(section, -1)

	var alts []Alternative
	for _, item := range itemParts[1:] { // itemParts[0] is the section header
		if len(alts) >= 3 {
			break
		}

		// ASIN — first /dp/ link in the item
		asinMatch := reMarkdownAsin.FindStringSubmatch(item)
		if len(asinMatch) < 2 {
			continue
		}
		asin := asinMatch[1]

		// Image URL
		imgURL := ""
		if m := reMarkdownImage.FindStringSubmatch(item); len(m) >= 2 {
			imgURL = m[1]
		}

		// Title — prefer the standalone text link over the image alt
		title := ""
		if m := reMarkdownTitle.FindStringSubmatch(item); len(m) >= 2 {
			title = strings.TrimSpace(m[1])
		}
		if len(title) < 15 {
			// fallback: image alt text
			if m := reMarkdownAltText.FindStringSubmatch(item); len(m) >= 2 {
				title = strings.TrimSpace(m[1])
			}
		}
		if len(title) < 15 {
			continue
		}

		// Price — markdown duplicates like [$59.49$59.49](url); first capture wins
		price := 0.0
		if m := reMarkdownPrice.FindStringSubmatch(item); len(m) >= 3 {
			priceStr := m[1]
			if priceStr == "" {
				priceStr = m[2]
			}
			if p, err := strconv.ParseFloat(strings.ReplaceAll(priceStr, ",", ""), 64); err == nil && p > 0 {
				price = p
			}
		}

		freeShip := strings.Contains(strings.ToLower(item), "free shipping")

		alts = append(alts, Alternative{
			ASIN:         asin,
			Title:        title,
			URL:          fmt.Sprintf("https://%s/dp/%s", domain, asin),
			ImageURL:     imgURL,
			PriceUSD:     price,
			FreeShipping: freeShip,
		})
		log.Printf("[DEBUG] alt: %s — %s (price=%.2f free=%v)", asin, title[:min(50, len(title))], price, freeShip)
	}

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
