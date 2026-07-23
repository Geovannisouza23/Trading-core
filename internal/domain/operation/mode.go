// Package operation models the system-wide operational mode state machine
// (PAPER, TESTNET, REAL, PAUSED, CLOSE_ONLY, KILL_SWITCH) and the system
// incidents that can force a mode transition.
package operation

import "trading-core/internal/domain/shared"

// Mode is the operational mode the trading engine currently runs in.
type Mode string

const (
	ModePaper      Mode = "PAPER"
	ModeTestnet    Mode = "TESTNET"
	ModeReal       Mode = "REAL"
	ModePaused     Mode = "PAUSED"
	ModeCloseOnly  Mode = "CLOSE_ONLY"
	ModeKillSwitch Mode = "KILL_SWITCH"
)

func (m Mode) Valid() bool {
	switch m {
	case ModePaper, ModeTestnet, ModeReal, ModePaused, ModeCloseOnly, ModeKillSwitch:
		return true
	default:
		return false
	}
}

// AllowsNewEntries reports whether the mode permits opening new positions.
func (m Mode) AllowsNewEntries() bool {
	return m == ModePaper || m == ModeTestnet || m == ModeReal
}

// AllowsReconciliation reports whether reconciliation must keep running.
// Reconciliation always runs regardless of mode.
func (m Mode) AllowsReconciliation() bool { return true }

// allowedTransitions encodes the safety matrix from section 14 of the spec:
// REAL can only be reached from PAPER/TESTNET/PAUSED (never directly from
// CLOSE_ONLY or KILL_SWITCH), and KILL_SWITCH can only be recovered through
// PAUSED, forcing an explicit manual review step.
var allowedTransitions = map[Mode]map[Mode]bool{
	ModePaper:      {ModeTestnet: true, ModeReal: true, ModePaused: true, ModeCloseOnly: true, ModeKillSwitch: true},
	ModeTestnet:    {ModePaper: true, ModeReal: true, ModePaused: true, ModeCloseOnly: true, ModeKillSwitch: true},
	ModeReal:       {ModePaused: true, ModeCloseOnly: true, ModeKillSwitch: true},
	ModePaused:     {ModePaper: true, ModeTestnet: true, ModeReal: true, ModeCloseOnly: true, ModeKillSwitch: true},
	ModeCloseOnly:  {ModePaused: true, ModeKillSwitch: true},
	ModeKillSwitch: {ModePaused: true},
}

// CanTransition reports whether moving from `m` to `target` is structurally
// allowed by the state machine (config-level guards for REAL are enforced by
// the caller; the domain only knows about safe state shape).
func (m Mode) CanTransition(target Mode) bool {
	if !m.Valid() || !target.Valid() {
		return false
	}
	if m == target {
		return true
	}
	return allowedTransitions[m][target]
}

// RealModeConfirmation must be presented by the caller (application layer,
// sourced from internal/config) whenever a transition into REAL is
// attempted. This keeps the "no unsafe REAL activation" invariant inside the
// domain without the domain importing configuration.
type RealModeConfirmation struct {
	Enabled           bool
	ConfirmationToken string
}

const requiredRealConfirmationToken = "I_UNDERSTAND_THE_RISK"

// Validate reports whether the confirmation satisfies the domain invariant
// required to enter REAL mode.
func (c RealModeConfirmation) Validate() error {
	if !c.Enabled || c.ConfirmationToken != requiredRealConfirmationToken {
		return shared.NewValidationError("operational_mode", "REAL mode requires explicit persisted confirmation")
	}
	return nil
}
