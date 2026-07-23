package unit_test

import (
	"testing"
	"time"

	"trading-core/internal/domain/operation"
)

func TestOperationalModeTransitions(t *testing.T) {
	cases := []struct {
		from, to operation.Mode
		allowed  bool
	}{
		{operation.ModePaper, operation.ModeTestnet, true},
		{operation.ModePaper, operation.ModeReal, true},
		{operation.ModeReal, operation.ModePaper, false}, // must pause first
		{operation.ModeReal, operation.ModePaused, true},
		{operation.ModeCloseOnly, operation.ModeReal, false},
		{operation.ModeKillSwitch, operation.ModePaused, true},
		{operation.ModeKillSwitch, operation.ModeReal, false},
		{operation.ModePaused, operation.ModeReal, true},
	}
	for _, tc := range cases {
		if got := tc.from.CanTransition(tc.to); got != tc.allowed {
			t.Errorf("%s -> %s: got allowed=%v, want %v", tc.from, tc.to, got, tc.allowed)
		}
	}
}

func TestStateTransitionToRealRequiresConfirmation(t *testing.T) {
	state := operation.NewState(time.Now())

	err := state.TransitionTo(operation.ModeReal, "operator", "test", "go live", time.Now(), operation.RealModeConfirmation{})
	if err == nil {
		t.Fatal("expected error transitioning to REAL without confirmation, got nil")
	}

	err = state.TransitionTo(operation.ModeReal, "operator", "test", "go live", time.Now(), operation.RealModeConfirmation{
		Enabled:           true,
		ConfirmationToken: "wrong-token",
	})
	if err == nil {
		t.Fatal("expected error transitioning to REAL with wrong token, got nil")
	}

	err = state.TransitionTo(operation.ModeReal, "operator", "test", "go live", time.Now(), operation.RealModeConfirmation{
		Enabled:           true,
		ConfirmationToken: "I_UNDERSTAND_THE_RISK",
	})
	if err != nil {
		t.Fatalf("expected REAL transition with correct token to succeed, got: %v", err)
	}
	if state.CurrentMode != operation.ModeReal {
		t.Fatalf("got mode %s, want REAL", state.CurrentMode)
	}
}

func TestStateTransitionRequiresActorAndReason(t *testing.T) {
	state := operation.NewState(time.Now())
	if err := state.TransitionTo(operation.ModePaused, "", "test", "reason", time.Now(), operation.RealModeConfirmation{}); err == nil {
		t.Fatal("expected error with empty actor, got nil")
	}
	if err := state.TransitionTo(operation.ModePaused, "operator", "test", "", time.Now(), operation.RealModeConfirmation{}); err == nil {
		t.Fatal("expected error with empty reason, got nil")
	}
}
