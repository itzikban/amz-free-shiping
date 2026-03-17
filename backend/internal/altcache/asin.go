package altcache

import (
	"net/url"
	"strings"
)

// ExtractASIN extracts a 10-character Amazon ASIN from various URL formats.
// Covers: /dp/XXXXXXXXXX, /gp/product/XXXXXXXXXX, ?ASIN=XXXXXXXXXX, and bare ASINs.
// Returns empty string if no valid ASIN found.
func ExtractASIN(rawURL string) string {
	u := strings.ToUpper(strings.TrimSpace(rawURL))

	// Check if it's already a bare ASIN
	if isASIN(u) {
		return u
	}

	// Try common Amazon URL patterns
	for _, pattern := range []string{"/DP/", "/GP/PRODUCT/", "ASIN="} {
		if idx := strings.Index(u, pattern); idx >= 0 {
			start := idx + len(pattern)
			if start+10 <= len(u) {
				candidate := u[start : start+10]
				if isASIN(candidate) {
					return candidate
				}
			}
		}
	}

	// Try parsing as URL and check query parameters
	if parsed, err := url.Parse(rawURL); err == nil {
		q := strings.ToUpper(parsed.Query().Get("asin"))
		if isASIN(q) {
			return q
		}
	}

	return ""
}

// isASIN checks if a string is a valid 10-character ASIN (uppercase alphanumeric).
func isASIN(s string) bool {
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
