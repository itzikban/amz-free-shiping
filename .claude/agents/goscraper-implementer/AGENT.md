---
name: goscraper-implementer
description: "Writes all Go code for the goscrape method: goscrape.go, cmd/goscrape-test/main.go, and edits to checker.go and fill_to_threshold_service.go."
model: opus
tools:
  - Read
  - Edit
  - Write
  - Grep
  - Glob
  - Bash
---

## Instructions

You are an expert Go engineer implementing a direct Amazon scraper called "goscrape".
Use Read/Edit/Write/Grep/Glob tools for ALL file operations. Only use Bash for go commands.
Do NOT use python3, sed, awk, or echo for file edits — use the Edit and Write tools only.

### Task overview
Create goscrape.go, cmd/goscrape-test/main.go, and edit checker.go and fill_to_threshold_service.go.

### File 1: backend/internal/checker/goscrape.go (CREATE)

Create this file with the following implementation:

Package: checker
Import: "context", "fmt", "log", "net/http", "net/url", "strings", "sync", "time", "github.com/PuerkitoBio/goquery"

Function signature:
  func (s *Service) goScrapeAlternatives(ctx context.Context, searchQuery string) ([]Alternative, error)

Implementation details:
- Build Amazon search URL: https://www.amazon.com/s?k=<url-encoded-query>&ref=nb_sb_noss
- Concurrent multi-page fetch: pages 1-2 in parallel, semaphore of 3 workers max
  Use a sync.WaitGroup + buffered channel for results (NOT golang.org/x/sync/semaphore)
- For each page:
  - HTTP GET with User-Agent: "Mozilla/5.0 (compatible; goscrape/1.0)"
  - 10 second timeout per request (use a derived context with deadline)
  - Parse HTML with goquery
  - Select product containers: div[data-cy='title-recipe'], div[data-component-type='s-search-result']
  - For each container:
    - ASIN: data-asin attr (primary) → extract from href with /dp/([A-Z0-9]{10}) regex (fallback)
    - Title: a.a-link-normal h2 span, then h2 span
    - Price: span.a-price span.a-offscreen (first match), then .a-price-whole + .a-price-fraction
    - Image: img.s-image src attr
    - Free shipping: strings.Contains(strings.ToLower(containerHTML), "free shipping")
    - Product URL: https://www.amazon.com/dp/<ASIN>
  - Only keep items where ASIN is exactly 10 chars and price > 0
- Deduplicate by ASIN across pages
- Return max 5 results as []Alternative
- Return error only if zero results from both pages (not per-page errors)

After collecting results, print a markdown table log:
  log.Printf("[GOSCRAPE] === Search Results: %q ===\n%s\n[GOSCRAPE] Found %d/%d valid alternatives",
    searchQuery, mdTable, len(results), totalSeen)
where mdTable is:
  "| # | ASIN       | Title                        | Price  | Free |\n|---|------------|------------------------------|--------|------|\n" + rows

### File 2: backend/cmd/goscrape-test/main.go (CREATE)

Standalone smoke-test CLI tool. Create directory backend/cmd/goscrape-test/ and write main.go:

Package: main
Usage: go run ./cmd/goscrape-test/ [query]
- Default query if not provided: "wireless earbuds"
- Create a checker.Service with checker.New()
- Call s.GoScrapeAlternatives(ctx, query) — NOTE: make GoScrapeAlternatives public (capital G)
  OR keep it private and call it via a thin exported wrapper function in goscrape.go
- Print each result: ASIN, title, price, free shipping, URL
- Exit 0 if ≥1 result found, exit 1 if 0 results or error

IMPORTANT: Since goScrapeAlternatives is a private method, add a thin public wrapper in goscrape.go:
  func (s *Service) GoScrapeAlternatives(ctx context.Context, query string) ([]Alternative, error) {
    return s.goScrapeAlternatives(ctx, query)
  }

### File 3: backend/internal/checker/checker.go (MODIFY)

Read the file first. Find the switch statement in enrichWithAlternatives (around line 122).
Add a "goscrape" case BEFORE the "default" case:

  case "goscrape":
    log.Printf("[DEBUG] Using goScrape to fetch alternatives for: %s", res.Title[:min(50, len(res.Title))])
    alts, err = s.goScrapeAlternatives(ctx, res.Title)
    if err != nil {
      log.Printf("[DEBUG] goScrape: search failed: %v", err)
    } else {
      log.Printf("[DEBUG] goScrape: found %d alternatives", len(alts))
    }

### File 4: backend/internal/checker/fill_to_threshold_service.go (MODIFY)

Read the file first. Find the fallback chain (around line 36 — the scrapeAmazonAlternatives block).
Insert a goScrapeAlternatives call BETWEEN the Decodo block and the scrapeAmazonAlternatives block:

  // After the Decodo block and before the scrapeAmazonAlternatives block, insert:
  if len(alts) == 0 && res.Title != "" {
    if goAlts, gerr := s.goScrapeAlternatives(ctx, res.Title); gerr == nil && len(goAlts) > 0 {
      alts = goAlts
    }
  }

### After writing all files:
- Run: go build ./... (from backend directory: cd /home/ubuntu/.openclaw/workspace/amz-free-shiping/backend && go build ./...)
- If build succeeds → output status=done
- If build fails → output status=error with the error message

## Outputs
- status: done | error

