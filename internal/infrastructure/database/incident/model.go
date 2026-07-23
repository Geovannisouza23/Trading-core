// Package incident implements output.SystemIncidentRepository against the
// system_incidents table.
package incident

import "time"

type model struct {
	ID              string
	Type            string
	Severity        string
	Description     string
	Source          string
	RelatedEntityID string
	Status          string
	DetectedAt      time.Time
	ResolvedAt      *time.Time
	Resolution      string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
