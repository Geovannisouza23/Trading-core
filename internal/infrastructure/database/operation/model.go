// Package operation implements output.OperationalModeRepository against the
// singleton operational_modes table.
package operation

import "time"

type model struct {
	CurrentMode string
	ChangedBy   string
	Origin      string
	Reason      string
	ChangedAt   time.Time
	Version     int
}
