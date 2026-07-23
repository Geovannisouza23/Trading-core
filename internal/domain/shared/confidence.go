package shared

import "github.com/shopspring/decimal"

// Confidence is a bounded [0, 1] score describing how strongly a signal or
// event assessment should be trusted.
type Confidence struct {
	value decimal.Decimal
}

func NewConfidence(value decimal.Decimal) (Confidence, error) {
	if value.IsNegative() || value.GreaterThan(decimal.NewFromInt(1)) {
		return Confidence{}, NewValidationError("confidence", "must be between 0 and 1")
	}
	return Confidence{value: value}, nil
}

func MustNewConfidence(value decimal.Decimal) Confidence {
	c, err := NewConfidence(value)
	if err != nil {
		panic(err)
	}
	return c
}

func (c Confidence) Decimal() decimal.Decimal { return c.value }

func (c Confidence) GreaterThanOrEqual(other Confidence) bool {
	return c.value.GreaterThanOrEqual(other.value)
}

func (c Confidence) String() string { return c.value.StringFixed(4) }

func (c Confidence) MarshalJSON() ([]byte, error) { return c.value.MarshalJSON() }

func (c *Confidence) UnmarshalJSON(data []byte) error { return c.value.UnmarshalJSON(data) }
