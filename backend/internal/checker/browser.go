package checker

import (
	"context"
	"os/exec"
	"strings"
	"time"
)

func (s *Service) browserAnalyze(ctx context.Context, url, country, zip string) (Result, error) {
	_ = zip // reserved for next step: explicit location setting via browser profile/session.

	bctx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()

	cmd := exec.CommandContext(
		bctx,
		"/usr/bin/chromium-browser",
		"--headless",
		"--disable-gpu",
		"--no-sandbox",
		"--dump-dom",
		url,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return Result{}, err
	}

	text := string(out)
	res := AnalyzeHTML(url, country, text)
	res.Method = "browser"

	lower := strings.ToLower(text)
	if strings.Contains(lower, "captcha") || strings.Contains(lower, "automated access to amazon data") {
		res.Signal = "captcha_detected"
		res.FreeShipping = false
		res.FreeShippingCountry = false
		return res, nil
	}

	if strings.ToUpper(country) == "US" && res.FreeShipping && !res.FreeShippingCountry {
		res.FreeShippingCountry = true
		res.Signal = "browser_generic_free_shipping"
	}

	return res, nil
}
