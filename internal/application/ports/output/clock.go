package output

import "time"

// Clock abstracts wall-clock time so use cases stay deterministic under
// test.
type Clock interface {
	Now() time.Time
}
