// Package system implements output.Clock with the real wall clock.
package system

import (
	"time"

	"trading-core/internal/application/ports/output"
)

type Clock struct{}

func NewClock() *Clock { return &Clock{} }

var _ output.Clock = (*Clock)(nil)

func (Clock) Now() time.Time { return time.Now().UTC() }
