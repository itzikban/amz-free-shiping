package checker

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type Result struct {
	URL                 string    `json:"url"`
	Country             string    `json:"country"`
	CheckedAt           time.Time `json:"checked_at"`
	FreeShipping        bool      `json:"free_shipping"`
	FreeShippingCountry bool      `json:"free_shipping_country"`
	Signal              string    `json:"signal"`
	Method              string    `json:"method"`
}

type Service struct {
	Client *http.Client
}

func New() *Service {
	return &Service{Client: &http.Client{Timeout: 20 * time.Second}}
}

func (s *Service) CheckURL(ctx context.Context, url, country, zip string) (Result, error) {
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
	if dres, derr := s.decodoAnalyze(ctx, url, country, zip); derr == nil {
		return dres, nil
	}
	if res.FreeShippingCountry {
		return res, nil
	}

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
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err == nil {
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
