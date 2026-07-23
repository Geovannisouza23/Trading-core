package shared

import "fmt"

// ValidationError represents an invariant violation raised while constructing
// or mutating a domain object. It never crosses the domain boundary wrapped
// by infrastructure-specific error types.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// NewValidationError builds a ValidationError for the given field.
func NewValidationError(field, message string) *ValidationError {
	return &ValidationError{Field: field, Message: message}
}

// ConflictError represents an invalid state transition (e.g. an order moving
// from a terminal state to a non-terminal one).
type ConflictError struct {
	Entity string
	From   string
	To     string
}

func (e *ConflictError) Error() string {
	return fmt.Sprintf("%s: invalid transition from %s to %s", e.Entity, e.From, e.To)
}

// NewConflictError builds a ConflictError describing an illegal transition.
func NewConflictError(entity, from, to string) *ConflictError {
	return &ConflictError{Entity: entity, From: from, To: to}
}
