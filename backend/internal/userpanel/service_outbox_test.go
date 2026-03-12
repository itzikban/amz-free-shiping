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

	req := AddTrackedItemReq{URL: "https://www.amazon.com/dp/B0DHCZBKW7", Country: "US", ZIP: "10013"}
	notFree := checker.Result{CheckedAt: time.Now().UTC(), FreeShipping: false, FreeShippingCountry: false, Signal: "not_free", Method: "mock"}
	free := checker.Result{CheckedAt: time.Now().UTC().Add(time.Minute), FreeShipping: true, FreeShippingCountry: true, Signal: "free", Method: "mock"}

	svc.addTrackedItemFromResult(context.Background(), req, notFree)
	svc.addTrackedItemFromResult(context.Background(), req, free)
	if _, err := svc.RetryFailedNotifications(context.Background(), 100); err != nil {
		t.Fatalf("RetryFailedNotifications returned err: %v", err)
	}

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

func TestRememberNotificationIndex_Bounded(t *testing.T) {
	svc := New(nil)

	for i := 0; i < maxNotificationIndexEntries+10; i++ {
		svc.rememberNotificationIndex("alert-" + fmtInt(i))
	}

	if got := len(svc.notificationOrder); got != maxNotificationIndexEntries {
		t.Fatalf("expected notificationOrder to be capped at %d, got %d", maxNotificationIndexEntries, got)
	}
	if got := len(svc.notificationIndex); got != maxNotificationIndexEntries {
		t.Fatalf("expected notificationIndex to be capped at %d, got %d", maxNotificationIndexEntries, got)
	}
	if _, exists := svc.notificationIndex["alert-0"]; exists {
		t.Fatalf("expected oldest index entry to be evicted")
	}
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

	processed, err = svc.RetryFailedNotifications(context.Background(), 100)
	if err != nil {
		t.Fatalf("RetryFailedNotifications (second call) returned err: %v", err)
	}
	if processed != 0 {
		t.Fatalf("expected 0 processed entries on second retry, got %d", processed)
	}
	if got := len(svc.ListNotifications(false, 10)); got != 1 {
		t.Fatalf("expected retry idempotency to keep 1 synced in-app notification, got %d", got)
	}
}
