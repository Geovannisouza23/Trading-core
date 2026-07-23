package output

import "context"

type NotificationSeverity string

const (
	NotificationInfo     NotificationSeverity = "INFO"
	NotificationWarning  NotificationSeverity = "WARNING"
	NotificationCritical NotificationSeverity = "CRITICAL"
)

// Notification is a human-facing alert (e.g. sent to Telegram).
type Notification struct {
	Title    string
	Message  string
	Severity NotificationSeverity
	Metadata map[string]string
}

// Notifier is the port to outbound human alerting channels.
type Notifier interface {
	Notify(ctx context.Context, notification Notification) error
}
