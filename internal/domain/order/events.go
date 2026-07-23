package order

import "time"

type OrderCreated struct {
	Order       *Order
	OccurredAt_ time.Time
}

func (e OrderCreated) EventName() string     { return "OrderCreated" }
func (e OrderCreated) OccurredAt() time.Time { return e.OccurredAt_ }

type OrderSubmitted struct {
	Order       *Order
	OccurredAt_ time.Time
}

func (e OrderSubmitted) EventName() string     { return "OrderSubmitted" }
func (e OrderSubmitted) OccurredAt() time.Time { return e.OccurredAt_ }

type OrderPartiallyFilled struct {
	Order       *Order
	OccurredAt_ time.Time
}

func (e OrderPartiallyFilled) EventName() string     { return "OrderPartiallyFilled" }
func (e OrderPartiallyFilled) OccurredAt() time.Time { return e.OccurredAt_ }

type OrderFilled struct {
	Order       *Order
	OccurredAt_ time.Time
}

func (e OrderFilled) EventName() string     { return "OrderFilled" }
func (e OrderFilled) OccurredAt() time.Time { return e.OccurredAt_ }

type OrderCancelled struct {
	Order       *Order
	OccurredAt_ time.Time
}

func (e OrderCancelled) EventName() string     { return "OrderCancelled" }
func (e OrderCancelled) OccurredAt() time.Time { return e.OccurredAt_ }
