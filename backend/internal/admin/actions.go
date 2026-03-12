package admin

type ActionResult struct {
	OK      bool   `json:"ok"`
	Action  string `json:"action"`
	Message string `json:"message"`
}

func ReplayFailedJobs() ActionResult {
	// TODO: Integrate with worker dispatch once the jobs replay pipeline is implemented.
	return ActionResult{OK: false, Action: "replay_failed_jobs", Message: "Not implemented: worker integration required"}
}
