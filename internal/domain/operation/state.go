package operation

import (
	"time"

	"trading-core/internal/domain/shared"
)

// State is the auditable, single-row aggregate tracking the system's
// current operational mode plus who/why/when it last changed.
type State struct {
	CurrentMode Mode
	ChangedBy   string
	Origin      string
	Reason      string
	ChangedAt   time.Time
	Version     int
}

// NewState builds the initial operational state. The system always starts
// in PAPER mode.
func NewState(now time.Time) *State {
	return &State{
		CurrentMode: ModePaper,
		ChangedBy:   "system",
		Origin:      "bootstrap",
		Reason:      "initial startup",
		ChangedAt:   now,
		Version:     1,
	}
}

// RehydrateState reconstructs a State from persisted values without
// re-validating history (used by repositories).
func RehydrateState(mode Mode, changedBy, origin, reason string, changedAt time.Time, version int) *State {
	return &State{
		CurrentMode: mode,
		ChangedBy:   changedBy,
		Origin:      origin,
		Reason:      reason,
		ChangedAt:   changedAt,
		Version:     version,
	}
}

// TransitionTo attempts to move the system into a new operational mode.
// `confirmation` is only inspected when target is ModeReal; it must be
// supplied by the application layer from validated, persisted configuration.
func (s *State) TransitionTo(target Mode, actor, origin, reason string, now time.Time, confirmation RealModeConfirmation) error {
	if !s.CurrentMode.CanTransition(target) {
		return shared.NewConflictError("operational_mode", string(s.CurrentMode), string(target))
	}
	if target == ModeReal {
		if err := confirmation.Validate(); err != nil {
			return err
		}
	}
	if actor == "" {
		return shared.NewValidationError("actor", "must not be empty")
	}
	if reason == "" {
		return shared.NewValidationError("reason", "must not be empty")
	}
	s.CurrentMode = target
	s.ChangedBy = actor
	s.Origin = origin
	s.Reason = reason
	s.ChangedAt = now
	s.Version++
	return nil
}
