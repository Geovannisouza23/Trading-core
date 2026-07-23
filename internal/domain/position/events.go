package position

import "time"

type PositionOpened struct {
	Position    *Position
	OccurredAt_ time.Time
}

func (e PositionOpened) EventName() string     { return "PositionOpened" }
func (e PositionOpened) OccurredAt() time.Time { return e.OccurredAt_ }

type PositionUpdated struct {
	Position    *Position
	OccurredAt_ time.Time
}

func (e PositionUpdated) EventName() string     { return "PositionUpdated" }
func (e PositionUpdated) OccurredAt() time.Time { return e.OccurredAt_ }

type PositionClosed struct {
	Position    *Position
	OccurredAt_ time.Time
}

func (e PositionClosed) EventName() string     { return "PositionClosed" }
func (e PositionClosed) OccurredAt() time.Time { return e.OccurredAt_ }
