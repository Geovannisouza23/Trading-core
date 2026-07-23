package shared

import "github.com/shopspring/decimal"

var (
	percentageLowerBound = decimal.NewFromInt(-10)
	percentageUpperBound = decimal.NewFromInt(10)
)

// Percentage represents a fraction (0.03 == 3%) used across risk and P&L
// calculations. Bounded to [-1000%, 1000%] to catch unit mistakes (e.g.
// passing 30 instead of 0.30) at construction time.
type Percentage struct {
	value decimal.Decimal
}

// NewPercentage validates and builds a Percentage from a fraction.
func NewPercentage(value decimal.Decimal) (Percentage, error) {
	if value.LessThan(percentageLowerBound) || value.GreaterThan(percentageUpperBound) {
		return Percentage{}, NewValidationError("percentage", "out of representable range")
	}
	return Percentage{value: value}, nil
}

func MustNewPercentage(value decimal.Decimal) Percentage {
	p, err := NewPercentage(value)
	if err != nil {
		panic(err)
	}
	return p
}

func (p Percentage) Decimal() decimal.Decimal { return p.value }

// Of applies the percentage to a Money amount.
func (p Percentage) Of(m Money) Money {
	return Money{amount: m.Decimal().Mul(p.value)}
}

func (p Percentage) GreaterThan(other Percentage) bool { return p.value.GreaterThan(other.value) }

func (p Percentage) GreaterThanOrEqual(other Percentage) bool {
	return p.value.GreaterThanOrEqual(other.value)
}

func (p Percentage) LessThan(other Percentage) bool { return p.value.LessThan(other.value) }

func (p Percentage) Abs() Percentage { return Percentage{value: p.value.Abs()} }

func (p Percentage) IsNegative() bool { return p.value.IsNegative() }

func (p Percentage) String() string { return p.value.Mul(decimal.NewFromInt(100)).StringFixed(2) + "%" }

func (p Percentage) MarshalJSON() ([]byte, error) { return p.value.MarshalJSON() }

func (p *Percentage) UnmarshalJSON(data []byte) error { return p.value.UnmarshalJSON(data) }
