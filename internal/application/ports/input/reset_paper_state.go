package input

import "context"

// ResetPaperStateUseCase powers POST /v1/paper/reset. It only ever succeeds
// in PAPER mode.
type ResetPaperStateUseCase interface {
	Execute(ctx context.Context) error
}
