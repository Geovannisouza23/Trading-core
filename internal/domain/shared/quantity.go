package shared

import "github.com/shopspring/decimal"

// Quantity is a non-negative amount of a tradable asset.
type Quantity struct {
	value decimal.Decimal
}

// ZeroQuantity represents a flat (no exposure) quantity.
func ZeroQuantity() Quantity {
	return Quantity{value: decimal.Zero}
}

// NewQuantity validates and builds a Quantity. Negative quantities are never
// valid; direction is expressed through Side, not sign.
func NewQuantity(value decimal.Decimal) (Quantity, error) {
	if value.IsNegative() {
		return Quantity{}, NewValidationError("quantity", "must not be negative")
	}
	return Quantity{value: value}, nil
}

func MustNewQuantity(value decimal.Decimal) Quantity {
	q, err := NewQuantity(value)
	if err != nil {
		panic(err)
	}
	return q
}

func (q Quantity) Decimal() decimal.Decimal { return q.value }

func (q Quantity) IsZero() bool { return q.value.IsZero() }

func (q Quantity) Add(other Quantity) Quantity {
	return Quantity{value: q.value.Add(other.value)}
}

func (q Quantity) Sub(other Quantity) (Quantity, error) {
	result := q.value.Sub(other.value)
	if result.IsNegative() {
		return Quantity{}, NewValidationError("quantity", "subtraction would result in negative quantity")
	}
	return Quantity{value: result}, nil
}

func (q Quantity) GreaterThan(other Quantity) bool { return q.value.GreaterThan(other.value) }

func (q Quantity) GreaterThanOrEqual(other Quantity) bool {
	return q.value.GreaterThanOrEqual(other.value)
}

func (q Quantity) LessThan(other Quantity) bool { return q.value.LessThan(other.value) }

func (q Quantity) Equal(other Quantity) bool { return q.value.Equal(other.value) }

func (q Quantity) String() string { return q.value.StringFixed(8) }

func (q Quantity) MarshalJSON() ([]byte, error) { return q.value.MarshalJSON() }

func (q *Quantity) UnmarshalJSON(data []byte) error { return q.value.UnmarshalJSON(data) }
