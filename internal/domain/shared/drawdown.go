package shared

import "github.com/shopspring/decimal"

// Drawdown is a non-negative fraction [0, 1] representing the magnitude of
// decline from a peak equity value.
type Drawdown struct {
	value decimal.Decimal
}

func NewDrawdown(value decimal.Decimal) (Drawdown, error) {
	if value.IsNegative() || value.GreaterThan(decimal.NewFromInt(1)) {
		return Drawdown{}, NewValidationError("drawdown", "must be between 0 and 1")
	}
	return Drawdown{value: value}, nil
}

func MustNewDrawdown(value decimal.Decimal) Drawdown {
	d, err := NewDrawdown(value)
	if err != nil {
		panic(err)
	}
	return d
}

// DrawdownFromPeak computes a Drawdown value from a peak and a current
// equity amount. If current >= peak the drawdown is zero.
func DrawdownFromPeak(peak, current Money) Drawdown {
	if peak.IsZero() || current.GreaterThanOrEqual(peak) {
		return Drawdown{value: decimal.Zero}
	}
	loss := peak.Sub(current)
	ratio := loss.Decimal().Div(peak.Decimal())
	if ratio.GreaterThan(decimal.NewFromInt(1)) {
		ratio = decimal.NewFromInt(1)
	}
	return Drawdown{value: ratio}
}

func (d Drawdown) Decimal() decimal.Decimal { return d.value }

func (d Drawdown) GreaterThanOrEqual(other Drawdown) bool {
	return d.value.GreaterThanOrEqual(other.value)
}

func (d Drawdown) String() string { return d.value.Mul(decimal.NewFromInt(100)).StringFixed(2) + "%" }

func (d Drawdown) MarshalJSON() ([]byte, error) { return d.value.MarshalJSON() }

func (d *Drawdown) UnmarshalJSON(data []byte) error { return d.value.UnmarshalJSON(data) }
