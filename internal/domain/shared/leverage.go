package shared

import "github.com/shopspring/decimal"

// Leverage is a multiplier applied to notional exposure. 1 means no
// leverage; the domain never allows leverage below 1.
type Leverage struct {
	value decimal.Decimal
}

func NewLeverage(value decimal.Decimal) (Leverage, error) {
	if value.LessThan(decimal.NewFromInt(1)) {
		return Leverage{}, NewValidationError("leverage", "must be greater than or equal to 1")
	}
	return Leverage{value: value}, nil
}

func MustNewLeverage(value decimal.Decimal) Leverage {
	l, err := NewLeverage(value)
	if err != nil {
		panic(err)
	}
	return l
}

func (l Leverage) Decimal() decimal.Decimal { return l.value }

func (l Leverage) GreaterThan(other Leverage) bool { return l.value.GreaterThan(other.value) }

func (l Leverage) LessThanOrEqual(other Leverage) bool { return l.value.LessThanOrEqual(other.value) }

func (l Leverage) String() string { return l.value.StringFixed(2) + "x" }

func (l Leverage) MarshalJSON() ([]byte, error) { return l.value.MarshalJSON() }

func (l *Leverage) UnmarshalJSON(data []byte) error { return l.value.UnmarshalJSON(data) }
