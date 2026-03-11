package admin

type ActionResult struct {
	OK      bool   `json:"ok"`
	Action  string `json:"action"`
	Message string `json:"message"`
}

func ReplayFailedJobs() ActionResult {
	return ActionResult{OK: true, Action: "replay_failed_jobs", Message: "Replay accepted (stub for next chunk)"}
}

func RetryFailedNotifications() ActionResult {
	return ActionResult{OK: true, Action: "retry_failed_notifications", Message: "Retry accepted (stub for next chunk)"}
}
