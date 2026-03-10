package monitor

import (
	"context"
	"testing"

	"free-ship-checker-go/internal/checker"
)

func TestStart_ValidatesURLAndHost(t *testing.T) {
	s := New(checker.New())

	if _, err := s.Start(context.Background(), StartReq{URL: "", Country: "US", IntervalSeconds: 10}); err == nil {
		t.Fatal("expected error for empty url")
	}
	if _, err := s.Start(context.Background(), StartReq{URL: "http://example.com/x", Country: "US", IntervalSeconds: 10}); err == nil {
		t.Fatal("expected error for non-https/unsupported host")
	}
	if _, err := s.Start(context.Background(), StartReq{URL: "https://google.com", Country: "US", IntervalSeconds: 10}); err == nil {
		t.Fatal("expected error for unsupported host")
	}
}

func TestStart_ValidatesIntervalRange(t *testing.T) {
	s := New(checker.New())
	if _, err := s.Start(context.Background(), StartReq{URL: "https://www.amazon.com/dp/B0DHCZBKW7", Country: "US", IntervalSeconds: 1}); err == nil {
		t.Fatal("expected range error for too small interval")
	}
	if _, err := s.Start(context.Background(), StartReq{URL: "https://www.amazon.com/dp/B0DHCZBKW7", Country: "US", IntervalSeconds: 999999}); err == nil {
		t.Fatal("expected range error for too large interval")
	}
}
