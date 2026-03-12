package userpanel

import (
	"context"
	"testing"

	"free-ship-checker-go/internal/checker"
)

func TestNotificationsReadAndPreferences(t *testing.T) {
	svc := New(checker.New())

	_, err := svc.AddTrackedItem(context.Background(), AddTrackedItemReq{
		URL:     "https://www.amazon.com/dp/B0DHCZBKW7",
		Country: "US",
		ZIP:     "10013",
	})
	if err != nil {
		t.Fatalf("AddTrackedItem failed: %v", err)
	}

	if _, err := svc.RetryFailedNotifications(context.Background(), 100); err != nil {
		t.Fatalf("RetryFailedNotifications failed: %v", err)
	}

	notifs := svc.ListNotifications(false, 10)
	if len(notifs) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(notifs))
	}
	if notifs[0].Read {
		t.Fatal("expected unread notification")
	}

	if ok := svc.MarkNotificationRead(notifs[0].ID); !ok {
		t.Fatal("expected mark read to succeed")
	}
	unread := svc.ListNotifications(true, 10)
	if len(unread) != 0 {
		t.Fatalf("expected no unread notifications, got %d", len(unread))
	}

	svc.UpdateNotificationPreferences(NotificationPreferences{InAppEnabled: false, OnItemAdded: true})
	_, err = svc.AddTrackedItem(context.Background(), AddTrackedItemReq{
		URL:     "https://www.amazon.com/dp/B0DHCZBKW7",
		Country: "US",
		ZIP:     "10013",
	})
	if err != nil {
		t.Fatalf("second AddTrackedItem failed: %v", err)
	}

	all := svc.ListNotifications(false, 10)
	if len(all) != 1 {
		t.Fatalf("expected notifications to stay at 1 when disabled, got %d", len(all))
	}
}
