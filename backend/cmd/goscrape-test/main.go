package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"free-ship-checker-go/internal/checker"
)

func main() {
	query := "wireless earbuds"
	if len(os.Args) > 1 {
		query = strings.Join(os.Args[1:], " ")
	}

	s := checker.New()
	ctx := context.Background()

	alts, err := s.GoScrapeAlternatives(ctx, query)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if len(alts) == 0 {
		fmt.Fprintln(os.Stderr, "No results found")
		os.Exit(1)
	}

	for i, a := range alts {
		fmt.Printf("%d. ASIN=%s  Price=$%.2f  FreeShip=%v\n   Title: %s\n   URL:   %s\n\n",
			i+1, a.ASIN, a.PriceUSD, a.FreeShipping, a.Title, a.URL)
	}
}
