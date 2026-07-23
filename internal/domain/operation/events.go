package operation

import "time"

// KillSwitchActivated is published whenever the system enters ModeKillSwitch.
type KillSwitchActivated struct {
	ActivatedBy string
	Origin      string
	Reason      string
	AllowClose  bool
	OccurredAt_ time.Time
}

func (e KillSwitchActivated) EventName() string     { return "KillSwitchActivated" }
func (e KillSwitchActivated) OccurredAt() time.Time { return e.OccurredAt_ }

// OperationalModeChanged is published on every successful mode transition.
type OperationalModeChanged struct {
	From        Mode
	To          Mode
	ChangedBy   string
	Origin      string
	Reason      string
	OccurredAt_ time.Time
}

func (e OperationalModeChanged) EventName() string     { return "OperationalModeChanged" }
func (e OperationalModeChanged) OccurredAt() time.Time { return e.OccurredAt_ }

// ReconciliationDivergenceDetected is published whenever ReconcileBrokerState
// finds a mismatch between local and broker-reported state.
type ReconciliationDivergenceDetected struct {
	IncidentID  string
	Severity    IncidentSeverity
	Description string
	OccurredAt_ time.Time
}

func (e ReconciliationDivergenceDetected) EventName() string {
	return "ReconciliationDivergenceDetected"
}
func (e ReconciliationDivergenceDetected) OccurredAt() time.Time { return e.OccurredAt_ }
