package shared

import "github.com/shopspring/decimal"

// Price is a strictly positive monetary quotation for one unit of an asset.
type Price struct {
	value decimal.Decimal
}

// NewPrice validates and builds a Price. A price must always be greater than
// zero: zero or negative prices are not representable market quotations.
func NewPrice(value decimal.Decimal) (Price, error) {
	if value.LessThanOrEqual(decimal.Zero) {
		return Price{}, NewValidationError("price", "must be greater than zero")
	}
	return Price{value: value}, nil
}

// MustNewPrice panics on invalid input; reserved for compile-time-safe
// literals (tests, seeds) where the value is known to be valid.
func MustNewPrice(value decimal.Decimal) Price {
	p, err := NewPrice(value)
	if err != nil {
		panic(err)
	}
	return p
}

func (p Price) Decimal() decimal.Decimal { return p.value }

func (p Price) GreaterThan(other Price) bool { return p.value.GreaterThan(other.value) }

func (p Price) GreaterThanOrEqual(other Price) bool { return p.value.GreaterThanOrEqual(other.value) }

func (p Price) LessThan(other Price) bool { return p.value.LessThan(other.value) }

func (p Price) LessThanOrEqual(other Price) bool { return p.value.LessThanOrEqual(other.value) }

func (p Price) Equal(other Price) bool { return p.value.Equal(other.value) }

// DistanceTo returns the absolute distance between two prices.
func (p Price) DistanceTo(other Price) decimal.Decimal {
	return p.value.Sub(other.value).Abs()
}

// PercentageDistanceTo returns the absolute distance to another price
// expressed as a fraction of this price (e.g. 0.02 == 2%).
func (p Price) PercentageDistanceTo(other Price) decimal.Decimal {
	if p.value.IsZero() {
		return decimal.Zero
	}
	return p.DistanceTo(other).Div(p.value)
}

func (p Price) String() string { return p.value.StringFixed(8) }

func (p Price) MarshalJSON() ([]byte, error) { return p.value.MarshalJSON() }

func (p *Price) UnmarshalJSON(data []byte) error { return p.value.UnmarshalJSON(data) }
