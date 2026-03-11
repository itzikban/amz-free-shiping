package admin

import "testing"

type mon struct{}
func (m mon) MonitorCounts() (int, int, int, int) { return 4, 1, 3, 8 }

type usr struct{}
func (u usr) UserCounts() (int, int) { return 11, 5 }

func TestSnapshot(t *testing.T) {
	svc := &Service{Monitors: mon{}, Users: usr{}}
	s := svc.Snapshot()
	if s.MonitorsTotal != 4 || s.MonitorsRunning != 1 || s.Notifications != 8 {
		t.Fatalf("bad monitor aggregation: %+v", s)
	}
	if s.UserTrackedItems != 11 || s.UserAlerts != 5 {
		t.Fatalf("bad user aggregation: %+v", s)
	}
}
