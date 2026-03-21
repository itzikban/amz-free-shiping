package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type ScrapedProduct struct {
	ASIN         string
	Title        string
	Price        string
	ImageURL     string
	URL          string
	Rating       string
	ReviewCnt    string
	FreeShipUS   bool
	ShippingInfo string
}

func main() {
	// Search term: the product from the user
	searchTerm := "Barrina T8 Grow Lights"
	maxResults := 3 // Fewer to avoid rate limiting

	fmt.Printf("🔍 Searching Amazon for: %s\n\n", searchTerm)

	products, err := searchAmazon(searchTerm, maxResults)
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return
	}

	fmt.Printf("✅ Found %d products\n\n", len(products))
	for i, p := range products {
		fmt.Printf("%d. %s\n", i+1, p.Title)
		fmt.Printf("   ASIN: %s\n", p.ASIN)
		fmt.Printf("   Price: %s\n", p.Price)
		fmt.Printf("   Rating: %s | Reviews: %s\n", p.Rating, p.ReviewCnt)
		fmt.Printf("   URL: %s\n", p.URL)
		if p.ImageURL != "" {
			fmt.Printf("   Image: %s\n", p.ImageURL)
		}
		fmt.Printf("   📦 Shipping (US): ")
		if p.FreeShipUS {
			fmt.Printf("✅ FREE\n")
		} else if p.ShippingInfo != "" {
			fmt.Printf("❌ %s\n", p.ShippingInfo)
		} else {
			fmt.Printf("❓ Unknown\n")
		}
		fmt.Printf("\n")
	}
}

func searchAmazon(query string, maxResults int) ([]ScrapedProduct, error) {
	// Build search URL
	searchURL := fmt.Sprintf(
		"https://www.amazon.com/s?k=%s",
		url.QueryEscape(query),
	)

	fmt.Printf("📡 Fetching search results: %s\n", searchURL)

	// Create request with browser-like headers
	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 5<<20))
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	// Parse HTML
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("parse html: %w", err)
	}

	var products []ScrapedProduct
	seen := make(map[string]bool)

	// Try multiple selectors for product containers (Amazon changes these frequently)
	selectors := []string{
		"div[data-component-type='s-search-result']",
		"div.s-result-item",
		"div[data-asin]",
	}

	for _, selector := range selectors {
		doc.Find(selector).Each(func(i int, s *goquery.Selection) {
			if len(products) >= maxResults {
				return
			}

			p := ScrapedProduct{}

			// Get ASIN - try multiple ways
			if asin, ok := s.Attr("data-asin"); ok && asin != "" {
				p.ASIN = asin
			}

			// Get title - try multiple selectors
			titleSelectors := []string{
				"h2 a span",
				"h2 span",
				"span.a-size-medium",
				"a.a-link-normal h2 span",
			}
			for _, ts := range titleSelectors {
				if title := strings.TrimSpace(s.Find(ts).First().Text()); title != "" && len(title) > 10 {
					p.Title = title
					break
				}
			}

			// Get price - try multiple selectors
			priceSelectors := []string{
				".a-price-whole",
				"span.a-price span",
				".a-price",
			}
			for _, ps := range priceSelectors {
				if price := strings.TrimSpace(s.Find(ps).First().Text()); price != "" {
					p.Price = price
					break
				}
			}

			// Get image
			imgSelectors := []string{
				"img",
				"img.s-image",
			}
			for _, is := range imgSelectors {
				if imgURL, ok := s.Find(is).First().Attr("src"); ok && imgURL != "" {
					p.ImageURL = imgURL
					break
				}
			}

			// Get rating
			ratingNode := s.Find(".a-icon-star-small span, .a-star-small span")
			if rating := strings.TrimSpace(ratingNode.First().Text()); rating != "" {
				p.Rating = rating
			}

			// Get review count
			reviewNode := s.Find(".a-size-base")
			reviewTexts := reviewNode.Map(func(_ int, sel *goquery.Selection) string {
				return strings.TrimSpace(sel.Text())
			})
			for _, rt := range reviewTexts {
				if strings.Contains(rt, "K") || strings.Contains(rt, ",") {
					p.ReviewCnt = rt
					break
				}
			}

			// Get product URL
			linkNode := s.Find("a.a-link-normal")
			if href, ok := linkNode.Attr("href"); ok && href != "" {
				if !strings.HasPrefix(href, "http") {
					p.URL = "https://www.amazon.com" + href
				} else {
					p.URL = href
				}
				// Extract ASIN from URL if not found
				if p.ASIN == "" {
					parts := strings.Split(href, "/dp/")
					if len(parts) > 1 {
						asins := strings.Split(parts[1], "/")
						if len(asins) > 0 && len(asins[0]) == 10 {
							p.ASIN = asins[0]
						}
					}
				}
			}

			// Only add if we have minimal data and not duplicate
			if p.Title != "" && len(p.Title) > 10 {
				key := p.ASIN
				if key == "" {
					key = p.URL
				}
				if key != "" && !seen[key] {
					seen[key] = true
					products = append(products, p)
				}
			}
		})

		if len(products) >= maxResults {
			break
		}
	}

	if len(products) == 0 {
		return nil, fmt.Errorf("no products found - Amazon may require JavaScript or CAPTCHA")
	}

	// Now fetch detail pages for shipping info
	fmt.Printf("\n📦 Fetching shipping info for %d products...\n\n", len(products))
	for i := range products {
		if err := fetchShippingInfo(&products[i]); err != nil {
			products[i].ShippingInfo = "shipping check failed"
			fmt.Printf("⚠️ shipping lookup failed for %s: %v\n", products[i].ASIN, err)
		}
		time.Sleep(1 * time.Second) // Be nice to Amazon's servers
	}

	return products, nil
}

func fetchShippingInfo(p *ScrapedProduct) error {
	if strings.TrimSpace(p.ASIN) == "" {
		return fmt.Errorf("missing ASIN for product")
	}

	// Use simple product URL format
	detailURL := fmt.Sprintf("https://www.amazon.com/dp/%s", p.ASIN)

	req, err := http.NewRequest("GET", detailURL, nil)
	if err != nil {
		return fmt.Errorf("create detail request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("fetch detail page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("detail status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 3<<20))
	if err != nil {
		return fmt.Errorf("read detail body: %w", err)
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("parse detail html: %w", err)
	}

	shippingSelectors := []string{
		"#delivery",
		"#shippingMessage",
		"#delivery-message",
		"#mir-layout-DELIVERY_BLOCK-slot-PRIMARY_DELIVERY_MESSAGE_LARGE",
		".shipping",
	}

	foundSignal := false
	for _, sel := range shippingSelectors {
		doc.Find(sel).EachWithBreak(func(_ int, s *goquery.Selection) bool {
			text := strings.ToLower(strings.TrimSpace(s.Text()))
			if text == "" {
				return true
			}
			if strings.Contains(text, "free shipping") || strings.Contains(text, "ships free") ||
				strings.Contains(text, "free delivery") {
				p.FreeShipUS = true
				p.ShippingInfo = "Free shipping"
				foundSignal = true
				return false
			}
			if strings.Contains(text, "shipping") && strings.Contains(text, "$") {
				p.ShippingInfo = text
				foundSignal = true
				return false
			}
			return true
		})
		if foundSignal {
			return nil
		}
	}

	return fmt.Errorf("no shipping signal found")
}
