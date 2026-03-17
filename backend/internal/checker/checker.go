package checker

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type Alternative struct {
	ASIN         string  `json:"asin"`
	Title        string  `json:"title"`
	URL          string  `json:"url"`
	ImageURL     string  `json:"image_url,omitempty"`
	PriceUSD     float64 `json:"price_usd,omitempty"`
	FreeShipping bool    `json:"free_shipping"`
}

// ShippingOption represents a country where product can be shipped with free shipping
type ShippingOption struct {
	Country       string  `json:"country"`
	CountryName   string  `json:"country_name"`
	AmazonDomain  string  `json:"amazon_domain"`
	Currency      string  `json:"currency"`
	Price         float64 `json:"price,omitempty"`
	FreeShipping  bool    `json:"free_shipping"`
}

type Result struct {
	URL                string         `json:"url"`
	Country            string         `json:"country"`
	CheckedAt          time.Time      `json:"checked_at"`
	PriceUSD           float64        `json:"price_usd,omitempty"`
	FreeShipping       bool           `json:"free_shipping"`
	FreeShippingCountry bool          `json:"free_shipping_country"`
	Signal             string         `json:"signal"`
	Method             string         `json:"method"`
	Title              string         `json:"title,omitempty"`
	ImageURL           string         `json:"image_url,omitempty"`
	Alternatives       []Alternative  `json:"alternatives,omitempty"`
	ShippingOptions    []ShippingOption `json:"shipping_options,omitempty"` // Available countries with free shipping
}

type Service struct {
	Client      *http.Client
	FetchMethod string // "auto" or "http"
	AltCache    interface{} // *altcache.Cache, nil = no caching (avoids import cycle)
}

var (
	reBuyNewText          = regexp.MustCompile(`(?is)buy\s*new[^$]{0,40}\$\s*([0-9]{1,5}(?:,[0-9]{3})*(?:\.[0-9]{2})?)`)
	reOneTimePurchaseText = regexp.MustCompile(`(?is)one[- ]time\s*purchase[^$]{0,40}\$\s*([0-9]{1,5}(?:,[0-9]{3})*(?:\.[0-9]{2})?)`)
	rePriceToPayText      = regexp.MustCompile(`(?is)price\s*to\s*pay[^$]{0,40}\$\s*([0-9]{1,5}(?:,[0-9]{3})*(?:\.[0-9]{2})?)`)
	rePriceToPayJSON      = regexp.MustCompile(`(?is)"pricetopay"\s*:\s*\{[^\}]*"priceamount"\s*:\s*([0-9]+(?:\.[0-9]{1,2})?)`)
	reOurPriceJSON        = regexp.MustCompile(`(?is)"ourprice"\s*:\s*\{[^\}]*"amount"\s*:\s*([0-9]+(?:\.[0-9]{1,2})?)`)
	reDealPriceJSON       = regexp.MustCompile(`(?is)"dealprice"\s*:\s*\{[^\}]*"amount"\s*:\s*([0-9]+(?:\.[0-9]{1,2})?)`)
	reAOffscreen          = regexp.MustCompile(`(?is)a-offscreen">\s*\$\s*([0-9]{1,5}(?:,[0-9]{3})*(?:\.[0-9]{2})?)\s*<`)
	reAnyUSD              = regexp.MustCompile(`\$\s*([0-9]{1,5}(?:,[0-9]{3})*(?:\.[0-9]{2})?)`)
)

// New creates a Service with an HTTP client configured with a 20-second timeout.
func New() *Service {
	return &Service{Client: &http.Client{Timeout: 20 * time.Second}}
}

// enrichWithAlternatives fetches PA-API alternatives when product doesn't have free shipping
func (s *Service) enrichWithAlternatives(ctx context.Context, res Result) (Result, error) {
	log.Printf("[DEBUG] enrichWithAlternatives called: free_shipping_country=%v, title_len=%d", res.FreeShippingCountry, len(res.Title))

	// Only fetch alternatives if no free shipping for country and we have a product title
	if res.FreeShippingCountry || res.Title == "" {
		log.Printf("[DEBUG] skipping alternatives: free_shipping_country=%v or title empty", res.FreeShippingCountry)
		return res, nil
	}

	// Try to extract ASIN from the product URL for caching
	// Use inline extraction to avoid import (cache is optional)
	asin := extractASINFromURL(res.URL)
	log.Printf("[DEBUG] extracted ASIN: %s", asin)

	// Check cache first (if available)
	if s.AltCache != nil && asin != "" {
		// Use reflection to call cache methods without importing altcache
		cache := s.AltCache
		// Get method: func(ctx, asin) ([]Alternative, bool)
		if getter, ok := cache.(interface {
			Get(context.Context, string) ([]Alternative, bool)
		}); ok {
			if alts, ok := getter.Get(ctx, asin); ok {
				res.Alternatives = alts
				return res, nil
			}
		}
	}

	// Cache miss or no cache: call scraper, PA-API, or decodo based on config
	altFetchMethod := os.Getenv("ALT_FETCH_METHOD")
	if altFetchMethod == "" {
		altFetchMethod = "scraper" // Default to real scraper
	}

	var alts []Alternative
	var err error

	switch altFetchMethod {
	case "scraper":
		log.Printf("[DEBUG] Using Amazon scraper to fetch alternatives for: %s", res.Title[:min(50, len(res.Title))])
		alts, err = s.scrapeAmazonAlternatives(ctx, res.Title)
		if err != nil {
			log.Printf("[DEBUG] Scraper: search failed: %v", err)
		} else {
			log.Printf("[DEBUG] Scraper: found %d alternatives", len(alts))
		}
	case "decodo":
		log.Printf("[DEBUG] Using Decodo scraper to fetch alternatives for: %s", res.Title[:min(50, len(res.Title))])
		alts, err = s.decodoScrapeAlternatives(ctx, res.Title)
		if err != nil {
			log.Printf("[DEBUG] Decodo: search failed: %v", err)
		} else {
			log.Printf("[DEBUG] Decodo: found %d alternatives", len(alts))
		}
	case "demo":
		log.Printf("[DEBUG] Using demo mode for alternatives: %s", res.Title[:min(50, len(res.Title))])
		alts = generateDemoAlternatives(res.Title, asin)
		log.Printf("[DEBUG] Demo mode: generated %d alternatives", len(alts))
	case "paapi":
		// Use PA-API
		paa := NewPAAClient()
		log.Printf("[DEBUG] PA-API: searching for alternatives: asin=%s, title=%s", asin, res.Title[:min(50, len(res.Title))])
		alts, err = paa.SearchItems(ctx, res.Title, 3)
		if err != nil {
			log.Printf("[DEBUG] PA-API: search failed: %v", err)
		} else {
			log.Printf("[DEBUG] PA-API: found %d alternatives", len(alts))
		}
	default:
		// Unknown method, try scraper as fallback
		log.Printf("[DEBUG] Unknown ALT_FETCH_METHOD=%s, falling back to scraper", altFetchMethod)
		alts, err = s.scrapeAmazonAlternatives(ctx, res.Title)
		if err != nil {
			log.Printf("[DEBUG] Scraper fallback: search failed: %v", err)
		}
	}

	if err == nil && len(alts) > 0 {
		res.Alternatives = alts
		// Write to cache (if available)
		if s.AltCache != nil && asin != "" {
			if putter, ok := s.AltCache.(interface {
				Put(context.Context, string, string, []Alternative)
			}); ok {
				putter.Put(ctx, asin, res.Title, alts)
			}
		}
	}
	// Silently ignore PA-API/scraper errors — it's a bonus feature
	return res, nil
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// extractASINFromURL extracts ASIN from various Amazon URL formats
func extractASINFromURL(rawURL string) string {
	u := strings.ToUpper(strings.TrimSpace(rawURL))

	// Check if it's already a bare ASIN
	if len(u) == 10 && isValidASIN(u) {
		return u
	}

	// Try common Amazon URL patterns
	for _, pattern := range []string{"/DP/", "/GP/PRODUCT/", "ASIN="} {
		if idx := strings.Index(u, pattern); idx >= 0 {
			start := idx + len(pattern)
			if start+10 <= len(u) {
				candidate := u[start : start+10]
				if isValidASIN(candidate) {
					return candidate
				}
			}
		}
	}

	return ""
}

// isValidASIN checks if a string is a valid 10-character ASIN
func isValidASIN(s string) bool {
	if len(s) != 10 {
		return false
	}
	for _, r := range s {
		if (r < 'A' || r > 'Z') && (r < '0' || r > '9') {
			return false
		}
	}
	return true
}

// scrapeAmazonAlternatives scrapes real Amazon search results for alternatives
func (s *Service) scrapeAmazonAlternatives(ctx context.Context, searchQuery string) ([]Alternative, error) {
	// Build Amazon search URL - use simplified query to get better results
	keywords := strings.Fields(searchQuery)
	if len(keywords) > 3 {
		keywords = keywords[:3] // Use only first 3 words to avoid blocking
	}
	simpleQuery := strings.Join(keywords, " ")
	searchURL := fmt.Sprintf("https://www.amazon.com/s?k=%s&ref=nb_sb_noss", url.QueryEscape(simpleQuery))
	log.Printf("[DEBUG] Scraper: searching for '%s'", simpleQuery)

	// Create request with browser-like headers
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Connection", "keep-alive")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("amazon status: %d", resp.StatusCode)
	}

	// Parse HTML with goquery
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("parse html: %w", err)
	}

	var alts []Alternative
	seenASINs := make(map[string]bool)

	// Try multiple selectors for product listings (Amazon changes HTML structure frequently)
	selectors := []string{
		"[data-component-type='s-search-result']",
		"[data-asin]",
		"div[data-asin]",
		".s-result-item",
	}

	for _, selector := range selectors {
		doc.Find(selector).Each(func(i int, container *goquery.Selection) {
			if len(alts) >= 5 {
				return
			}

			// Try to get ASIN from data-asin attribute or URL
			asin, ok := container.Attr("data-asin")
			if !ok || asin == "" {
				// Try to extract from URL
				if href, ok := container.Find("a[href*='/dp/']").Attr("href"); ok {
					parts := strings.Split(href, "/dp/")
					if len(parts) > 1 {
						asin = strings.Split(parts[1], "/")[0]
						asin = strings.Split(asin, "?")[0] // Remove query params
					}
				}
			}

			if asin == "" || len(asin) < 10 {
				return
			}
			if seenASINs[asin] {
				return
			}
			seenASINs[asin] = true

			// Get title - try multiple selectors
			title := ""
			titleSelectors := []string{
				"h2 a span",
				"h2 span",
				"a.a-link-normal span",
				".a-text-normal",
			}
			for _, ts := range titleSelectors {
				title = strings.TrimSpace(container.Find(ts).First().Text())
				if title != "" {
					break
				}
			}

			if title == "" {
				return
			}

			// Get price
			price := 0.0
			priceSelectors := []string{
				".a-price-whole",
				".a-price .a-offscreen",
				"span.a-price-whole",
			}
			for _, ps := range priceSelectors {
				priceText := strings.TrimSpace(container.Find(ps).First().Text())
				priceText = strings.TrimPrefix(priceText, "$")
				priceText = strings.ReplaceAll(priceText, ",", "")
				if p, err := strconv.ParseFloat(priceText, 64); err == nil && p > 0 {
					price = p
					break
				}
			}

			// Get image URL
			imageURL := ""
			img := container.Find("img").First()
			if src, ok := img.Attr("src"); ok && src != "" {
				imageURL = src
			} else if srcSet, ok := img.Attr("srcset"); ok && srcSet != "" {
				// Extract first URL from srcset
				parts := strings.Split(srcSet, ",")
				if len(parts) > 0 {
					imageURL = strings.Fields(parts[0])[0]
				}
			}

			// Check for free shipping indicators
			containerHTML, _ := goquery.OuterHtml(container)
			freeShipping := strings.Contains(strings.ToLower(containerHTML), "free shipping") ||
				strings.Contains(strings.ToLower(containerHTML), "free delivery")

			// Only include products with valid data
			if price > 0 && price < 10000 && title != "" {
				alt := Alternative{
					ASIN:         asin,
					Title:        title,
					URL:          fmt.Sprintf("https://amazon.com/dp/%s", asin),
					ImageURL:     imageURL,
					PriceUSD:     price,
					FreeShipping: freeShipping,
				}
				alts = append(alts, alt)
				log.Printf("[DEBUG] Scraper: found %s - %.0s... ($%.2f)", asin, title, price)
			}
		})

		if len(alts) > 0 {
			break // Stop trying selectors if we found results
		}
	}

	log.Printf("[DEBUG] Scraper: extracted %d alternatives from %s", len(alts), searchURL)
	return alts, nil
}

// generateDemoAlternatives creates demo alternatives for testing without external APIs
func generateDemoAlternatives(productTitle string, productASIN string) []Alternative {
	demoProducts := []struct {
		asin     string
		title    string
		price    float64
		imageURL string
	}{
		{
			"B0B8QXKQHV",
			"Full Spectrum LED Plant Grow Light with Timer",
			74.99,
			"https://picsum.photos/300/300?random=1",
		},
		{
			"B09YT3V8NZ",
			"Professional Horticultural Grow Lamp 2000W",
			129.99,
			"https://picsum.photos/300/300?random=2",
		},
		{
			"B0BJ4QXLKM",
			"Compact Desk Plant Light with Auto Timer",
			45.99,
			"https://picsum.photos/300/300?random=3",
		},
		{
			"B0CQ8RLTXX",
			"Adjustable LED Grow Light Bar 150W",
			59.99,
			"https://picsum.photos/300/300?random=4",
		},
		{
			"B0CX5LQQ2K",
			"Advanced Multi-Spectrum Grow Light Panel",
			89.99,
			"https://picsum.photos/300/300?random=5",
		},
	}

	var alts []Alternative
	for _, p := range demoProducts {
		alts = append(alts, Alternative{
			ASIN:         p.asin,
			Title:        p.title,
			URL:          fmt.Sprintf("https://amazon.com/dp/%s", p.asin),
			ImageURL:     p.imageURL,
			PriceUSD:     p.price,
			FreeShipping: true,
		})
	}
	return alts
}

// europeanAmazonCountries maps European countries that have Amazon stores and ship to Israel
var europeanAmazonCountries = []struct {
	code     string
	domain   string
	currency string
}{
	{"DE", "amazon.de", "EUR"},   // Germany - best for Israel shipping
	{"UK", "amazon.co.uk", "GBP"}, // UK - good coverage
	{"NL", "amazon.nl", "EUR"},   // Netherlands
	{"FR", "amazon.fr", "EUR"},   // France
}

// checkEuropeanAlternative checks if product has free shipping in a European country
// when it doesn't have it in Israel
func (s *Service) checkEuropeanAlternative(ctx context.Context, url, ilZip string) (Result, error) {
	// Try to get ASIN from URL
	asin := extractASINFromURL(url)
	if asin == "" {
		return Result{}, fmt.Errorf("could not extract ASIN")
	}

	log.Printf("[DEBUG] Checking European alternatives for ASIN %s", asin)

	// Try each European country
	for _, country := range europeanAmazonCountries {
		// Build Amazon Europe URL
		euroURL := fmt.Sprintf("https://%s/dp/%s", country.domain, asin)

		// Use Decodo to check
		if res, err := s.decodoAnalyze(ctx, euroURL, country.code, ""); err == nil && res.FreeShippingCountry {
			log.Printf("[DEBUG] Found free shipping in %s", country.code)
			res.Signal = fmt.Sprintf("eu_alternative|%s_free_shipping", country.code)
			return res, nil
		}

		// Fall back to HTTP
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, euroURL, nil)
		req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36")

		resp, err := s.Client.Do(req)
		if err != nil {
			continue
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(io.LimitReader(resp.Body, 3<<20))
		res := AnalyzeHTML(euroURL, country.code, string(body))

		if res.FreeShippingCountry {
			log.Printf("[DEBUG] Found free shipping in %s via HTTP", country.code)
			res.Signal = fmt.Sprintf("eu_alternative|%s_free_shipping", country.code)
			return res, nil
		}
	}

	return Result{}, fmt.Errorf("no EU countries with free shipping found")
}

func (s *Service) CheckURL(ctx context.Context, url, country, zip string) (Result, error) {
	return s.CheckURLWithMethod(ctx, url, country, zip, s.FetchMethod)
}

func (s *Service) CheckURLWithMethod(ctx context.Context, url, country, zip, method string) (Result, error) {
	// 1. Try Decodo API first (preferred method) — trust its result when it succeeds
	if method != "http" {
		if dres, derr := s.decodoAnalyze(ctx, url, country, zip); derr == nil {
			return s.enrichWithAlternatives(ctx, dres)
		}
		// Decodo failed (API error, no auth, etc.) — fall through to HTTP/browser
	}

	// 2. Fallback: plain HTTP fetch
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Result{}, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := s.Client.Do(req)
	if err != nil {
		return Result{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return Result{}, fmt.Errorf("upstream status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 3<<20))
	if err != nil {
		return Result{}, err
	}
	res := AnalyzeHTML(url, country, string(body))
	res.Method = "http"
	if res.FreeShippingCountry {
		return res, nil
	}

	// 3. Fallback: headless browser
	browserRes, berr := s.browserAnalyze(ctx, url, country, zip)
	if berr == nil {
		return s.enrichWithAlternatives(ctx, browserRes)
	}
	errMsg := strings.ReplaceAll(berr.Error(), " ", "_")
	if len(errMsg) > 80 {
		errMsg = errMsg[:80]
	}
	res.Signal = res.Signal + "|browser_unavailable:" + errMsg

	// If Israel has no free shipping, try European alternatives
	if strings.ToUpper(country) == "IL" && !res.FreeShippingCountry {
		if euRes, err := s.checkEuropeanAlternative(ctx, url, zip); err == nil {
			log.Printf("[DEBUG] Using European alternative instead of IL")
			return s.enrichWithAlternatives(ctx, euRes)
		}
	}

	return s.enrichWithAlternatives(ctx, res)
}

// AnalyzeHTML analyzes the provided HTML for pricing, shipping signals and basic product metadata.
// 
// AnalyzeHTML normalizes the country (defaults to "IL"), extracts a USD price, and fills a Result
// with URL, Country, CheckedAt, PriceUSD, FreeShipping, FreeShippingCountry, Signal, Title and ImageURL.
// It detects CAPTCHA indicators and returns early when found, checks country-specific and general
// free-shipping phrases, applies a small DOM fallback for common delivery selectors, and recognizes
// explicit negative markers (e.g., "not eligible for free shipping"). If no signal is detected,
// Signal is set to "no_free_shipping_signal".
func AnalyzeHTML(url, country, html string) Result {
	country = strings.ToUpper(strings.TrimSpace(country))
	if country == "" {
		country = "IL"
	}
	lower := strings.ToLower(html)

	countryPatterns := map[string][]string{
		"IL": {"free shipping to israel", "eligible for free shipping to israel", "free delivery to israel"},
		"US": {"free shipping to united states", "free shipping to the united states", "free delivery to united states"},
	}
	freeGeneralPatterns := []string{
		"free shipping",
		"free delivery",
	}

	res := Result{URL: url, Country: country, CheckedAt: time.Now().UTC()}
	res.PriceUSD = extractUSDPrice(html)

	// Extract product metadata from HTML.
	doc, docErr := goquery.NewDocumentFromReader(strings.NewReader(html))
	if docErr == nil {
		if title := strings.TrimSpace(doc.Find("#productTitle").Text()); title != "" {
			res.Title = title
		} else {
			res.Title = strings.TrimSpace(doc.Find("title").First().Text())
		}
		doc.Find("#landingImage, #imgTagWrapperId img, #main-image-container img").EachWithBreak(func(_ int, s *goquery.Selection) bool {
			for _, attr := range []string{"data-old-hires", "src"} {
				if v, ok := s.Attr(attr); ok && strings.HasPrefix(v, "http") {
					res.ImageURL = v
					return false
				}
			}
			return true
		})
	}

	captchaSignals := []string{"opfcaptcha", "validatecaptcha", "automated access to amazon data", "enter the characters you see below"}
	for _, c := range captchaSignals {
		if strings.Contains(lower, c) {
			res.Signal = "captcha_detected"
			return res
		}
	}

	for _, p := range countryPatterns[country] {
		if strings.Contains(lower, p) {
			res.FreeShippingCountry = true
			res.FreeShipping = true
			res.Signal = p
			return res
		}
	}

	for _, p := range freeGeneralPatterns {
		if strings.Contains(lower, p) {
			res.FreeShipping = true
			res.Signal = p
			break
		}
	}

	// Small DOM fallback for common selectors.
	if docErr == nil {
		txt := strings.ToLower(strings.TrimSpace(doc.Find("#mir-layout-DELIVERY_BLOCK-slot-PRIMARY_DELIVERY_MESSAGE_LARGE").Text()))
		if strings.Contains(txt, "free") {
			if country == "IL" && strings.Contains(txt, "israel") {
				res.FreeShippingCountry = true
				res.FreeShipping = true
				res.Signal = "delivery_block_primary"
				return res
			}
			if country == "US" && strings.Contains(txt, "united states") {
				res.FreeShippingCountry = true
				res.FreeShipping = true
				res.Signal = "delivery_block_primary"
				return res
			}
		}
	}

	// Guard against false positives like "not eligible for free shipping".
	notEligible := regexp.MustCompile(`(?i)not\s+eligible\s+for\s+free\s+shipping`)
	if notEligible.MatchString(html) {
		res.FreeShipping = false
		res.FreeShippingCountry = false
		res.Signal = "not_eligible_marker"
	}

	if !res.FreeShippingCountry {
		res.FreeShipping = false
	}
	if res.Signal == "" {
		res.Signal = "no_free_shipping_signal"
	}
	return res
}

func extractUSDPrice(html string) float64 {
	lower := strings.ToLower(html)

	// 0) Explicit textual anchors first
	for _, re := range []*regexp.Regexp{reBuyNewText, reOneTimePurchaseText, rePriceToPayText} {
		m := re.FindStringSubmatch(lower)
		if len(m) >= 2 {
			n := strings.ReplaceAll(m[1], ",", "")
			if v, err := strconv.ParseFloat(n, 64); err == nil && v > 0 {
				return v
			}
		}
	}

	// 1) Strong structured signals first (Amazon JSON blobs)
	for _, re := range []*regexp.Regexp{rePriceToPayJSON, reOurPriceJSON, reDealPriceJSON} {
		m := re.FindStringSubmatch(lower)
		if len(m) >= 2 {
			if v, err := strconv.ParseFloat(m[1], 64); err == nil && v > 0 {
				return v
			}
		}
	}

	// 2) Common buy-box markup (<span class="a-offscreen">$xx.xx</span>) near buy new/price
	offs := reAOffscreen.FindAllStringSubmatchIndex(lower, -1)
	bestOff := 0.0
	bestScore := -999
	for _, m := range offs {
		start, end := m[0], m[1]
		g1s, g1e := m[2], m[3]
		n := strings.ReplaceAll(lower[g1s:g1e], ",", "")
		v, err := strconv.ParseFloat(n, 64)
		if err != nil {
			continue
		}
		left := start - 180
		if left < 0 {
			left = 0
		}
		right := end + 180
		if right > len(lower) {
			right = len(lower)
		}
		ctx := lower[left:right]
		s := 0
		if strings.Contains(ctx, "buy new") || strings.Contains(ctx, "price to pay") || strings.Contains(ctx, "priceblock") {
			s += 8
		}
		if strings.Contains(ctx, "a-price") {
			s += 3
		}
		if strings.Contains(ctx, "/count") || strings.Contains(ctx, "per ") || strings.Contains(ctx, "shipping") || strings.Contains(ctx, "over $") {
			s -= 6
		}
		if s > bestScore || (s == bestScore && (bestOff == 0 || v < bestOff)) {
			bestScore, bestOff = s, v
		}
	}
	if bestOff > 0 && bestScore >= 2 {
		return bestOff
	}

	// 3) Fallback heuristic scan
	idxs := reAnyUSD.FindAllStringSubmatchIndex(lower, -1)
	if len(idxs) == 0 {
		return 0
	}
	type cand struct{ value float64; score int }
	cands := make([]cand, 0, len(idxs))
	for _, m := range idxs {
		start, end := m[0], m[1]
		g1s, g1e := m[2], m[3]
		n := strings.ReplaceAll(lower[g1s:g1e], ",", "")
		v, err := strconv.ParseFloat(n, 64)
		if err != nil {
			continue
		}
		left := start - 120
		if left < 0 { left = 0 }
		right := end + 120
		if right > len(lower) { right = len(lower) }
		ctx := lower[left:right]
		score := 0
		if strings.Contains(ctx, "buy new") || strings.Contains(ctx, "price to pay") || strings.Contains(ctx, "ourprice") || strings.Contains(ctx, "dealprice") { score += 6 }
		if strings.Contains(ctx, "one-time purchase") { score += 3 }
		if strings.Contains(ctx, "/count") || strings.Contains(ctx, "per ounce") || strings.Contains(ctx, "shipping") || strings.Contains(ctx, "coupon") || strings.Contains(ctx, "over $") { score -= 6 }
		cands = append(cands, cand{value:v, score:score})
	}
	if len(cands) == 0 { return 0 }
	best := cands[0]
	for _, c := range cands[1:] {
		if c.score > best.score || (c.score == best.score && c.value < best.value) { best = c }
	}
	return best.value
}
