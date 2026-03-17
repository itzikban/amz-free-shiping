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

	"github.com/PuerkitoBio/goquery"
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
func (s *Service) decodoScrapeAlternatives(ctx context.Context, searchQuery string) ([]Alternative, error) {
	token := os.Getenv("DECODO_BASIC_AUTH")
	if token == "" {
		return nil, fmt.Errorf("missing DECODO_BASIC_AUTH")
	}

	// Build search URL for Amazon
	searchURL := fmt.Sprintf("https://www.amazon.com/s?k=%s", strings.ReplaceAll(searchQuery, " ", "+"))
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

	// Parse HTML with goquery
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		log.Printf("[DEBUG] Failed to parse HTML: %v", err)
		return alts
	}

	// Track seen ASINs and good titles per ASIN
	seenASINs := make(map[string]int)
	asinTitles := make(map[string]string)

	// First, find all product divs with data-asin attribute
	doc.Find("[data-asin]").Each(func(i int, s *goquery.Selection) {
		if len(alts) >= 5 {
			return
		}

		asin, exists := s.Attr("data-asin")
		if !exists || asin == "" || len(asin) != 10 {
			return
		}

		// Skip empty data-asin from placeholder elements
		if asin == "" {
			return
		}

		// Extract title - look for h2 or span elements with product name
		title := ""

		// Try h2 a (Amazon's typical product title link)
		if titleText := strings.TrimSpace(s.Find("h2 a").First().Text()); len(titleText) > 20 && len(titleText) < 300 {
			title = titleText
		}

		// Try h2 span if h2 a didn't work
		if title == "" {
			s.Find("h2 span").First().Each(func(_ int, titleSel *goquery.Selection) {
				text := strings.TrimSpace(titleSel.Text())
				if len(text) > 20 && len(text) < 300 && !strings.Contains(text, "$") && !strings.Contains(text, "out of") {
					title = text
				}
			})
		}

		// If still no title, use generic fallback
		if title == "" {
			title = fmt.Sprintf("Product %s", asin[:8])
		}

		// Skip this product if we couldn't extract a good title (generic only)
		if strings.HasPrefix(title, "Product ") && len(title) < 20 {
			return
		}

		// If we've already seen a good title for this ASIN, skip this variant
		if existingTitle, exists := asinTitles[asin]; exists {
			// Only include if this title is better (longer/more detailed)
			if len(existingTitle) >= len(title) {
				return
			}
		}

		// Track this ASIN
		asinTitles[asin] = title
		seenASINs[asin]++
		if seenASINs[asin] > 1 {
			return // Skip variants after the first good one
		}

		// Extract price
		price := 0.0
		s.Find(".a-price-whole, .a-price-decimal").First().Each(func(_ int, priceSel *goquery.Selection) {
			priceText := strings.TrimSpace(priceSel.Text())
			priceText = strings.TrimPrefix(priceText, "$")
			priceText = strings.ReplaceAll(priceText, ",", "")
			if p, err := strconv.ParseFloat(priceText, 64); err == nil && p > 0 && p < 10000 {
				price = p
			}
		})

		// If no price found in dedicated selector, search regex
		if price == 0 {
			pricePattern := regexp.MustCompile(`\$\s*(\d+(?:,\d{3})*(?:\.\d{2})?)`)
			priceMatches := pricePattern.FindStringSubmatch(s.Text())
			if len(priceMatches) > 1 {
				priceStr := strings.ReplaceAll(priceMatches[1], ",", "")
				if p, err := strconv.ParseFloat(priceStr, 64); err == nil && p > 0 && p < 10000 {
					price = p
				}
			}
		}

		// Extract image URL
		imageURL := ""
		s.Find("img").Each(func(_ int, imgSel *goquery.Selection) {
			if imageURL == "" {
				src, _ := imgSel.Attr("src")
				if strings.Contains(src, "amazon") || strings.Contains(src, "ssl-images") {
					imageURL = src
				}
			}
		})

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
	})

	log.Printf("[DEBUG] extractAlternativesFromHTML: found %d valid alternatives using goquery", len(alts))
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
