// Package order models the Order aggregate and its state machine. This is
// the only aggregate allowed to represent a broker-facing order; nothing
// outside internal/interfaces/http (via the execution use case) is allowed
// to send it to a broker.
package order

// Status is the lifecycle state of an Order.
type Status string

const (
	StatusPending         Status = "PENDING"
	StatusSubmitted       Status = "SUBMITTED"
	StatusPartiallyFilled Status = "PARTIALLY_FILLED"
	StatusFilled          Status = "FILLED"
	StatusCancelled       Status = "CANCELLED"
	StatusRejected        Status = "REJECTED"
	StatusFailed          Status = "FAILED"
	StatusUnknown         Status = "UNKNOWN"
)

func (s Status) Valid() bool {
	switch s {
	case StatusPending, StatusSubmitted, StatusPartiallyFilled, StatusFilled,
		StatusCancelled, StatusRejected, StatusFailed, StatusUnknown:
		return true
	default:
		return false
	}
}

// IsTerminal reports whether no further transition is possible.
func (s Status) IsTerminal() bool {
	switch s {
	case StatusFilled, StatusCancelled, StatusRejected, StatusFailed:
		return true
	default:
		return false
	}
}

// allowedStatusTransitions mirrors section 9.2 of the spec exactly.
var allowedStatusTransitions = map[Status]map[Status]bool{
	StatusPending: {
		StatusSubmitted: true,
		StatusFailed:    true,
	},
	StatusSubmitted: {
		StatusPartiallyFilled: true,
		StatusFilled:          true,
		StatusCancelled:       true,
		StatusRejected:        true,
		StatusUnknown:         true,
	},
	StatusPartiallyFilled: {
		StatusFilled:    true,
		StatusCancelled: true,
		StatusUnknown:   true,
	},
	StatusUnknown: {
		StatusSubmitted:       true,
		StatusPartiallyFilled: true,
		StatusFilled:          true,
		StatusCancelled:       true,
		StatusRejected:        true,
		StatusFailed:          true,
	},
}

// CanTransition reports whether moving from `s` to `target` is legal.
func (s Status) CanTransition(target Status) bool {
	if !s.Valid() || !target.Valid() {
		return false
	}
	return allowedStatusTransitions[s][target]
}
