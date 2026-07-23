package output

import "errors"

// ErrNotFound is the sentinel every repository implementation must return
// (via errors.Is-compatible wrapping) when a lookup finds nothing. Living in
// this package lets both application/usecase and infrastructure/database
// depend on it without either depending on the other.
var ErrNotFound = errors.New("not found")

// ErrOptimisticLock is the sentinel repositories return when a Save call's
// expected version no longer matches the persisted row (someone else wrote
// first).
var ErrOptimisticLock = errors.New("optimistic lock conflict")
