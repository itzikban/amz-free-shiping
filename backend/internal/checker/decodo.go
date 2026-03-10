package checker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
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
