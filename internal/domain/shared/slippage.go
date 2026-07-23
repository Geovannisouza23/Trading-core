package shared

import "github.com/shopspring/decimal"

// Slippage is a non-negative fraction representing the deviation between an
// expected and an executed price.
type Slippage struct {
	value decimal.Decimal
}

func NewSlippage(value decimal.Decimal) (Slippage, error) {
	if value.IsNegative() {
		return Slippage{}, NewValidationError("slippage", "must not be negative")
	}
	return Slippage{value: value}, nil
}

func MustNewSlippage(value decimal.Decimal) Slippage {
	s, err := NewSlippage(value)
	if err != nil {
		panic(err)
	}
	return s
}

// SlippageBetween computes the slippage fraction between an expected and an
// executed price, relative to the expected price.
func SlippageBetween(expected, executed Price) Slippage {
	if expected.Decimal().IsZero() {
		return Slippage{value: decimal.Zero}
	}
	return Slippage{value: expected.DistanceTo(executed).Div(expected.Decimal())}
}

func (s Slippage) Decimal() decimal.Decimal { return s.value }

func (s Slippage) GreaterThan(other Slippage) bool { return s.value.GreaterThan(other.value) }

func (s Slippage) String() string { return s.value.Mul(decimal.NewFromInt(100)).StringFixed(4) + "%" }

func (s Slippage) MarshalJSON() ([]byte, error) { return s.value.MarshalJSON() }

func (s *Slippage) UnmarshalJSON(data []byte) error { return s.value.UnmarshalJSON(data) }
