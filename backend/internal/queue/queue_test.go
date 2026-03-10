package queue

import (
	"context"
	"encoding/json"
	"testing"
)

func TestJobPayloadRoundTrip(t *testing.T) {
	p := CheckItemPayload{TrackedItemID: 1, Country: "US", ZIP: "10013", URL: "https://www.amazon.com/dp/B0DHCZBKW7"}
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	var out CheckItemPayload
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if out.Country != "US" || out.ZIP != "10013" {
		t.Fatalf("unexpected payload: %+v", out)
	}
}

func TestWorkerRequiresQueue(t *testing.T) {
	w := &Worker{}
	if err := w.Run(context.Background()); err == nil {
		t.Fatal("expected error for nil queue")
	}
}
