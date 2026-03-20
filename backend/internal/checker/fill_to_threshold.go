package checker

import (
	"sort"
)

// FillSuggestion describes a single add-on recommendation for crossing a shipping threshold.
type FillSuggestion struct {
	ASIN             string  `json:"asin"`
	Title            string  `json:"title"`
	PriceUSD         float64 `json:"price_usd"`
	ImageURL         string  `json:"image_url,omitempty"`
	URL              string  `json:"url"`
	FreeShippingHint bool    `json:"free_shipping_hint"`
	Score            float64 `json:"score"`
}

// FillCombo describes a small multi-item recommendation (2-3 items).
type FillCombo struct {
	Items []FillSuggestion `json:"items"`
	Total float64          `json:"total"`
	Score float64          `json:"score"`
}

// FillToThresholdResponse is the payload returned by /api/v1/fill-to-threshold.
type FillToThresholdResponse struct {
	CurrentPrice  float64          `json:"current_price"`
	MissingAmount float64          `json:"missing_amount"`
	Suggestions   []FillSuggestion `json:"suggestions"`
	Combos        []FillCombo      `json:"combos"`
}

// BuildFillToThreshold calculates ranked single-item and combo suggestions.
func BuildFillToThreshold(currentPrice, threshold float64, alternatives []Alternative) FillToThresholdResponse {
	missing := threshold - currentPrice
	if missing < 0 {
		missing = 0
	}

	// Already at or above threshold — no boosters needed.
	if missing == 0 {
		return FillToThresholdResponse{
			CurrentPrice:  currentPrice,
			MissingAmount: 0,
			Suggestions:   []FillSuggestion{},
			Combos:        []FillCombo{},
		}
	}

	suggestions := make([]FillSuggestion, 0, len(alternatives))
	for _, alt := range alternatives {
		// Allow items up to 3× the threshold — items above the threshold are
		// "buy this instead and qualify for free shipping" suggestions, not add-ons.
		// Items massively above threshold (3×) are unlikely to be useful.
		if alt.PriceUSD <= 0 || alt.PriceUSD > threshold*3 {
			continue
		}
		suggestions = append(suggestions, FillSuggestion{
			ASIN:             alt.ASIN,
			Title:            alt.Title,
			PriceUSD:         alt.PriceUSD,
			ImageURL:         alt.ImageURL,
			URL:              alt.URL,
			FreeShippingHint: alt.FreeShipping,
			Score:            scoreCandidate(alt.PriceUSD, missing, alt.FreeShipping),
		})
	}

	sort.SliceStable(suggestions, func(i, j int) bool {
		if suggestions[i].Score == suggestions[j].Score {
			iGap := abs(suggestions[i].PriceUSD - missing)
			jGap := abs(suggestions[j].PriceUSD - missing)
			if iGap == jGap {
				return suggestions[i].PriceUSD < suggestions[j].PriceUSD
			}
			return iGap < jGap
		}
		return suggestions[i].Score < suggestions[j].Score
	})

	combos := buildCombos(suggestions, missing)

	return FillToThresholdResponse{
		CurrentPrice:  currentPrice,
		MissingAmount: missing,
		Suggestions:   suggestions,
		Combos:        combos,
	}
}

func buildCombos(suggestions []FillSuggestion, missing float64) []FillCombo {
	combos := make([]FillCombo, 0)
	maxN := len(suggestions)
	if maxN > 6 {
		maxN = 6
	}

	for i := 0; i < maxN; i++ {
		// Include 1-item combos so the API can return combo bundles of size 1..3.
		if c := scoreCombo([]FillSuggestion{suggestions[i]}, missing); c.Total >= missing {
			combos = append(combos, c)
		}

		for j := i + 1; j < maxN; j++ {
			items := []FillSuggestion{suggestions[i], suggestions[j]}
			if c := scoreCombo(items, missing); c.Total >= missing {
				combos = append(combos, c)
			}
			for k := j + 1; k < maxN; k++ {
				items3 := []FillSuggestion{suggestions[i], suggestions[j], suggestions[k]}
				if c := scoreCombo(items3, missing); c.Total >= missing {
					combos = append(combos, c)
				}
			}
		}
	}

	sort.SliceStable(combos, func(i, j int) bool {
		if combos[i].Score == combos[j].Score {
			iGap := abs(combos[i].Total - missing)
			jGap := abs(combos[j].Total - missing)
			if iGap == jGap {
				return combos[i].Total < combos[j].Total
			}
			return iGap < jGap
		}
		return combos[i].Score < combos[j].Score
	})
	if len(combos) > 5 {
		return combos[:5]
	}
	return combos
}

func scoreCombo(items []FillSuggestion, missing float64) FillCombo {
	total := 0.0
	freeCount := 0
	for _, it := range items {
		total += it.PriceUSD
		if it.FreeShippingHint {
			freeCount++
		}
	}
	distance := abs(total - missing)
	overSpend := max(0, total-missing)
	shippingBonus := float64(freeCount) * 0.5
	score := distance + (overSpend * 1.5) - shippingBonus
	return FillCombo{Items: items, Total: total, Score: score}
}

func scoreCandidate(price, missing float64, freeShipping bool) float64 {
	distance := abs(price - missing)
	overSpend := max(0, price-missing)
	shippingBonus := 0.0
	if freeShipping {
		shippingBonus = 1.5
	}
	return distance + (overSpend * 2) - shippingBonus
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
