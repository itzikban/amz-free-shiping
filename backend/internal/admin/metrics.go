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

type MonitorStats struct {
	Total         int
	Running       int
	Stopped       int
	Notifications int
}

type MonitorStatsProvider interface {
	MonitorCounts() MonitorStats
}

type UserStats struct {
	TrackedItems int
	Alerts       int
}

type UserStatsProvider interface {
	UserCounts() UserStats
}

type Service struct {
	monitors MonitorStatsProvider
	users    UserStatsProvider
}

func NewService(monitors MonitorStatsProvider, users UserStatsProvider) *Service {
	return &Service{monitors: monitors, users: users}
}

func (s *Service) Snapshot() Metrics {
	if s == nil {
		return Metrics{}
	}

	monitorStats := MonitorStats{}
	userStats := UserStats{}
	if s.monitors != nil {
		monitorStats = s.monitors.MonitorCounts()
	}
	if s.users != nil {
		userStats = s.users.UserCounts()
	}
	return Metrics{
		GeneratedAt:      time.Now().UTC(),
		MonitorsTotal:    monitorStats.Total,
		MonitorsRunning:  monitorStats.Running,
		MonitorsStopped:  monitorStats.Stopped,
		Notifications:    monitorStats.Notifications,
		UserTrackedItems: userStats.TrackedItems,
		UserAlerts:       userStats.Alerts,
	}
}
