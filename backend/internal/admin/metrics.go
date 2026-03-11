package admin

import "time"

type Metrics struct {
	GeneratedAt      time.Time `json:"generated_at"`
	MonitorsTotal    int       `json:"monitors_total"`
	MonitorsRunning  int       `json:"monitors_running"`
	MonitorsStopped  int       `json:"monitors_stopped"`
	Notifications    int       `json:"notifications"`
	UserTrackedItems int       `json:"user_tracked_items"`
	UserAlerts       int       `json:"user_alerts"`
}

type MonitorStatsProvider interface {
	MonitorCounts() (total, running, stopped, notifications int)
}

type UserStatsProvider interface {
	UserCounts() (trackedItems, alerts int)
}

type Service struct {
	Monitors MonitorStatsProvider
	Users    UserStatsProvider
}

func (s *Service) Snapshot() Metrics {
	total, running, stopped, notifications := 0, 0, 0, 0
	tracked, alerts := 0, 0
	if s.Monitors != nil {
		total, running, stopped, notifications = s.Monitors.MonitorCounts()
	}
	if s.Users != nil {
		tracked, alerts = s.Users.UserCounts()
	}
	return Metrics{
		GeneratedAt:      time.Now().UTC(),
		MonitorsTotal:    total,
		MonitorsRunning:  running,
		MonitorsStopped:  stopped,
		Notifications:    notifications,
		UserTrackedItems: tracked,
		UserAlerts:       alerts,
	}
}
