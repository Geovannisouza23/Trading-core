package shared

import "github.com/shopspring/decimal"

// Money represents a fixed-point monetary amount. It never uses float64 to
// avoid rounding drift on financial calculations.
type Money struct {
	amount decimal.Decimal
}

// ZeroMoney returns the additive identity.
func ZeroMoney() Money {
	return Money{amount: decimal.Zero}
}

// NewMoney builds a Money value from a decimal amount.
func NewMoney(amount decimal.Decimal) Money {
	return Money{amount: amount}
}

// NewMoneyFromString parses a decimal string into Money.
func NewMoneyFromString(value string) (Money, error) {
	d, err := decimal.NewFromString(value)
	if err != nil {
		return Money{}, NewValidationError("money", "must be a valid decimal string")
	}
	return Money{amount: d}, nil
}

func (m Money) Decimal() decimal.Decimal { return m.amount }

func (m Money) Add(other Money) Money {
	return Money{amount: m.amount.Add(other.amount)}
}

func (m Money) Sub(other Money) Money {
	return Money{amount: m.amount.Sub(other.amount)}
}

func (m Money) Mul(factor decimal.Decimal) Money {
	return Money{amount: m.amount.Mul(factor)}
}

func (m Money) Neg() Money {
	return Money{amount: m.amount.Neg()}
}

func (m Money) IsZero() bool { return m.amount.IsZero() }

func (m Money) IsNegative() bool { return m.amount.IsNegative() }

func (m Money) IsPositive() bool { return m.amount.IsPositive() }

func (m Money) GreaterThan(other Money) bool { return m.amount.GreaterThan(other.amount) }

func (m Money) GreaterThanOrEqual(other Money) bool {
	return m.amount.GreaterThanOrEqual(other.amount)
}

func (m Money) LessThan(other Money) bool { return m.amount.LessThan(other.amount) }

func (m Money) LessThanOrEqual(other Money) bool { return m.amount.LessThanOrEqual(other.amount) }

func (m Money) Equal(other Money) bool { return m.amount.Equal(other.amount) }

func (m Money) String() string { return m.amount.StringFixed(8) }

func (m Money) MarshalJSON() ([]byte, error) { return m.amount.MarshalJSON() }

func (m *Money) UnmarshalJSON(data []byte) error { return m.amount.UnmarshalJSON(data) }
