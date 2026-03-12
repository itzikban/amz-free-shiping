package userpanel

import (
	"context"
	"testing"
	"time"

	"free-ship-checker-go/internal/checker"
	"free-ship-checker-go/internal/notify"
)

func TestTransitionNotFreeToFree_CreatesTransitionAlertAndNotification(t *testing.T) {
	svc := New(nil)
	svc.outbox = notify.New(notify.NewInMemorySender())

	req := AddTrackedItemReq{URL: "https://www.amazon.com/dp/B0DHCZBKW7", Country: "US", ZIP: "10013"}
	notFree := checker.Result{CheckedAt: time.Now().UTC(), FreeShipping: false, FreeShippingCountry: false, Signal: "not_free", Method: "mock"}
	free := checker.Result{CheckedAt: time.Now().UTC().Add(time.Minute), FreeShipping: true, FreeShippingCountry: true, Signal: "free", Method: "mock"}

	svc.addTrackedItemFromResult(req, notFree)
	svc.addTrackedItemFromResult(req, free)

	alerts := svc.ListAlerts()
	if len(alerts) < 2 {
		t.Fatalf("expected at least 2 alerts, got %d", len(alerts))
	}
	found := false
	transitionAlertID := ""
	for _, a := range alerts {
		if a.Type == "free_shipping_available" {
			found = true
			transitionAlertID = a.ID
			break
		}
	}
	if !found {
		t.Fatalf("expected transition alert type free_shipping_available, got %+v", alerts)
	}

	notifs := svc.ListNotifications(false, 20)
	for _, n := range notifs {
		if n.AlertID == transitionAlertID {
			return
		}
	}
	t.Fatalf("expected notification for transition alert %s, got %+v", transitionAlertID, notifs)
}

func TestRetryFailedNotifications_ProcessesDueEntries(t *testing.T) {
	svc := New(nil)
	now := time.Now().UTC()
	svc.outbox.Enqueue("alert-1", "in_app", svc.user.ID, "hello", notify.BuildIdempotencyKey("alert-1", "in_app", svc.user.ID), now)

	processed, err := svc.RetryFailedNotifications(context.Background(), 100)
	if err != nil {
		t.Fatalf("RetryFailedNotifications returned err: %v", err)
	}
	if processed != 1 {
		t.Fatalf("expected 1 processed entry, got %d", processed)
	}
	if got := len(svc.ListNotifications(false, 10)); got != 1 {
		t.Fatalf("expected 1 synced in-app notification, got %d", got)
	}
}
