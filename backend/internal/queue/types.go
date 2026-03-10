package queue

import "time"

const (
	JobTypeCheckItem    = "check_item"
	JobTypeNotifyAlert  = "notify_alert"
	QueueMain           = "jobs:main"
	QueueDeadLetter     = "jobs:dlq"
	DefaultMaxRetries   = 5
)

type Job struct {
	ID         string          `json:"id"`
	Type       string          `json:"type"`
	Payload    []byte          `json:"payload"`
	Attempts   int             `json:"attempts"`
	MaxRetries int             `json:"max_retries"`
	CreatedAt  time.Time       `json:"created_at"`
}

type CheckItemPayload struct {
	TrackedItemID int64  `json:"tracked_item_id"`
	Country       string `json:"country"`
	ZIP           string `json:"zip,omitempty"`
	URL           string `json:"url"`
}

type NotifyAlertPayload struct {
	AlertID  int64  `json:"alert_id"`
	Channel  string `json:"channel"`
	Address  string `json:"address"`
}
