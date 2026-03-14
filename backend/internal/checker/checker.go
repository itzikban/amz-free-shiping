package checker

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type Result struct {
	URL                 string    `json:"url"`
	Country             string    `json:"country"`
	CheckedAt           time.Time `json:"checked_at"`
	PriceUSD            float64   `json:"price_usd,omitempty"`
	FreeShipping        bool      `json:"free_shipping"`
	FreeShippingCountry bool      `json:"free_shipping_country"`
	Signal              string    `json:"signal"`
	Method              string    `json:"method"`
	Title               string    `json:"title,omitempty"`
	ImageURL            string    `json:"image_url,omitempty"`
}

type Service struct {
	Client      *http.Client
	FetchMethod string // "auto" or "http"
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

func New() *Service {
	return &Service{Client: &http.Client{Timeout: 20 * time.Second}}
}

func (s *Service) CheckURL(ctx context.Context, url, country, zip string) (Result, error) {
	return s.CheckURLWithMethod(ctx, url, country, zip, s.FetchMethod)
}

func (s *Service) CheckURLWithMethod(ctx context.Context, url, country, zip, method string) (Result, error) {
	// 1. Try Decodo API first (preferred method) — trust its result when it succeeds
	if method != "http" {
		if dres, derr := s.decodoAnalyze(ctx, url, country, zip); derr == nil {
			return dres, nil
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
		return browserRes, nil
	}
	errMsg := strings.ReplaceAll(berr.Error(), " ", "_")
	if len(errMsg) > 80 {
		errMsg = errMsg[:80]
	}
	res.Signal = res.Signal + "|browser_unavailable:" + errMsg
	return res, nil
}

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
