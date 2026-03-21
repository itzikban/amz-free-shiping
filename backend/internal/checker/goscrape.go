package checker

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
)

var reASINFromHref = regexp.MustCompile(`/dp/([A-Z0-9]{10})`)

// goScrapeHTTPClient returns an http.Client whose outbound connections are
// bound to GOSCRAPE_BIND_IP (the WireGuard interface IP), so only goscrape
// traffic exits through the VPN tunnel. Falls back to direct if unset.
func goScrapeHTTPClient() *http.Client {
	bindIP := os.Getenv("GOSCRAPE_BIND_IP")
	if bindIP == "" {
		return &http.Client{Timeout: 15 * time.Second}
	}
	localAddr := &net.TCPAddr{IP: net.ParseIP(bindIP)}
	dialer := &net.Dialer{LocalAddr: localAddr, Timeout: 10 * time.Second}
	transport := &http.Transport{
		DialContext: dialer.DialContext,
	}
	log.Printf("[GOSCRAPE] binding to WireGuard IP %s", bindIP)
	return &http.Client{Transport: transport, Timeout: 15 * time.Second}
}

// GoScrapeAlternatives is the exported wrapper around goScrapeAlternatives.
func (s *Service) GoScrapeAlternatives(ctx context.Context, query string) ([]Alternative, error) {
	return s.goScrapeAlternatives(ctx, query)
}

func (s *Service) goScrapeAlternatives(ctx context.Context, searchQuery string) ([]Alternative, error) {
	// Trim query to first 5 words — long product titles hurt Amazon search relevance.
	searchQuery = trimToWords(searchQuery, 5)
	baseURL := "https://www.amazon.com/s?k=" + url.QueryEscape(searchQuery) + "&ref=nb_sb_noss"

	type pageResult struct {
		alts []Alternative
		err  error
	}

	pages := []int{1, 2}
	results := make(chan pageResult, len(pages))
	sem := make(chan struct{}, 3)
	var wg sync.WaitGroup

	for _, page := range pages {
		wg.Add(1)
		go func(p int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			pageURL := baseURL
			if p > 1 {
				pageURL = fmt.Sprintf("%s&page=%d", baseURL, p)
			}

			// Use a fresh context so a near-expired parent (after Decodo+scraper retries)
			// doesn't immediately cancel our request.
			reqCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()

			req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, pageURL, nil)
			if err != nil {
				results <- pageResult{err: err}
				return
			}
			req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
			req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
			req.Header.Set("Accept-Language", "en-US,en;q=0.5")
			req.Header.Set("Accept-Encoding", "gzip, deflate, br")

			resp, err := goScrapeHTTPClient().Do(req)
			if err != nil {
				results <- pageResult{err: err}
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				log.Printf("[GOSCRAPE] page %d: HTTP %d (Amazon may be blocking this IP/environment)", p, resp.StatusCode)
				results <- pageResult{err: fmt.Errorf("HTTP %d for page %d", resp.StatusCode, p)}
				return
			}

			// Handle gzip-encoded response bodies
			var bodyReader io.Reader = resp.Body
			if resp.Header.Get("Content-Encoding") == "gzip" {
				gz, gzErr := gzip.NewReader(resp.Body)
				if gzErr != nil {
					results <- pageResult{err: fmt.Errorf("gzip decode error: %w", gzErr)}
					return
				}
				defer gz.Close()
				bodyReader = gz
			}

			// Read full body for debug logging and parsing
			bodyBytes, err := io.ReadAll(bodyReader)
			if err != nil {
				results <- pageResult{err: fmt.Errorf("read body error: %w", err)}
				return
			}

			// Debug: log first 2000 chars of the response body
			preview := string(bodyBytes)
			if len(preview) > 2000 {
				preview = preview[:2000]
			}
			log.Printf("[GOSCRAPE] page %d response preview (%d bytes total):\n%s", p, len(bodyBytes), preview)

			doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(bodyBytes)))
			if err != nil {
				results <- pageResult{err: err}
				return
			}

			var alts []Alternative
			doc.Find("div[data-component-type='s-search-result']").Each(func(_ int, sel *goquery.Selection) {
				asin, exists := sel.Attr("data-asin")
				if !exists || len(asin) != 10 {
					return
				}
				alt := parseSearchResult(sel)
				if alt.ASIN == "" {
					alt.ASIN = asin
				}
				if len(alt.ASIN) == 10 && alt.PriceUSD > 0 {
					alts = append(alts, alt)
				}
			})

			results <- pageResult{alts: alts}
		}(page)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	seen := make(map[string]bool)
	var allAlts []Alternative
	totalSeen := 0

	for pr := range results {
		if pr.err != nil {
			continue
		}
		for _, a := range pr.alts {
			totalSeen++
			if !seen[a.ASIN] {
				seen[a.ASIN] = true
				allAlts = append(allAlts, a)
			}
		}
	}

	// Build markdown table for logging
	mdTable := "| # | ASIN       | Title                        | Price  | Free |\n|---|------------|------------------------------|--------|------|\n"
	for i, a := range allAlts {
		title := a.Title
		if len(title) > 28 {
			title = title[:28]
		}
		free := "N"
		if a.FreeShipping {
			free = "Y"
		}
		mdTable += fmt.Sprintf("| %d | %s | %-28s | $%.2f | %s |\n", i+1, a.ASIN, title, a.PriceUSD, free)
	}

	log.Printf("[GOSCRAPE] === Search Results: %q ===\n%s\n[GOSCRAPE] Found %d/%d valid alternatives",
		searchQuery, mdTable, len(allAlts), totalSeen)

	if len(allAlts) == 0 {
		return nil, fmt.Errorf("goscrape: no results for %q", searchQuery)
	}

	if len(allAlts) > 5 {
		allAlts = allAlts[:5]
	}

	return allAlts, nil
}

func parseSearchResult(sel *goquery.Selection) Alternative {
	var alt Alternative

	// ASIN
	alt.ASIN, _ = sel.Attr("data-asin")
	if len(alt.ASIN) != 10 {
		// Fallback: extract from href
		href, exists := sel.Find("a").Attr("href")
		if exists {
			if m := reASINFromHref.FindStringSubmatch(href); len(m) == 2 {
				alt.ASIN = m[1]
			}
		}
	}

	// Title: try multiple selectors in order of specificity
	titleSel := sel.Find("h2 a span")
	if titleSel.Length() == 0 {
		titleSel = sel.Find("h2 span.a-text-normal")
	}
	if titleSel.Length() == 0 {
		titleSel = sel.Find("span.a-size-medium.a-color-base.a-text-normal")
	}
	if titleSel.Length() == 0 {
		titleSel = sel.Find("a.a-link-normal h2 span")
	}
	if titleSel.Length() == 0 {
		titleSel = sel.Find("h2 span")
	}
	alt.Title = strings.TrimSpace(titleSel.First().Text())

	// Price: try span.a-price span.a-offscreen first
	priceText := strings.TrimSpace(sel.Find("span.a-price span.a-offscreen").First().Text())
	if priceText != "" {
		alt.PriceUSD = parsePrice(priceText)
	}
	if alt.PriceUSD == 0 {
		whole := strings.TrimSpace(sel.Find(".a-price-whole").First().Text())
		fraction := strings.TrimSpace(sel.Find(".a-price-fraction").First().Text())
		if whole != "" {
			whole = strings.ReplaceAll(whole, ",", "")
			whole = strings.TrimRight(whole, ".")
			if fraction == "" {
				fraction = "00"
			}
			alt.PriceUSD = parsePrice(whole + "." + fraction)
		}
	}

	// Image
	alt.ImageURL, _ = sel.Find("img.s-image").Attr("src")

	// Free shipping
	containerHTML, _ := sel.Html()
	alt.FreeShipping = strings.Contains(strings.ToLower(containerHTML), "free shipping")

	// Product URL
	if len(alt.ASIN) == 10 {
		alt.URL = "https://www.amazon.com/dp/" + alt.ASIN
	}

	return alt
}

// trimToWords returns the first n whitespace-separated words of s.
func trimToWords(s string, n int) string {
	words := strings.Fields(s)
	if len(words) <= n {
		return s
	}
	return strings.Join(words[:n], " ")
}

func parsePrice(s string) float64 {
	s = strings.ReplaceAll(s, "$", "")
	s = strings.ReplaceAll(s, ",", "")
	s = strings.TrimSpace(s)
	v, _ := strconv.ParseFloat(s, 64)
	return v
}
