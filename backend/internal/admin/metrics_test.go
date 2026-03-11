package admin

import "testing"

type mon struct{}

func (m mon) MonitorCounts() MonitorStats {
	return MonitorStats{Total: 4, Running: 1, Stopped: 3, Notifications: 8}
}

type usr struct{}

func (u usr) UserCounts() UserStats { return UserStats{TrackedItems: 11, Alerts: 5} }

func TestSnapshot(t *testing.T) {
	svc := &Service{Monitors: mon{}, Users: usr{}}
	s := svc.Snapshot()
	if s.MonitorsTotal != 4 || s.MonitorsRunning != 1 || s.MonitorsStopped != 3 || s.Notifications != 8 {
		t.Fatalf("bad monitor aggregation: %+v", s)
	}
	if s.UserTrackedItems != 11 || s.UserAlerts != 5 {
		t.Fatalf("bad user aggregation: %+v", s)
	}
}

func TestSnapshotNilProviders(t *testing.T) {
	svc := &Service{}
	s := svc.Snapshot()
	if s.MonitorsTotal != 0 || s.MonitorsRunning != 0 || s.Notifications != 0 {
		t.Fatalf("expected zero monitor counts with nil provider: %+v", s)
	}
	if s.UserTrackedItems != 0 || s.UserAlerts != 0 {
		t.Fatalf("expected zero user counts with nil provider: %+v", s)
	}
}
