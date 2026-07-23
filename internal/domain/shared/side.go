package shared

// Side represents the market direction of a signal, order or position.
type Side string

const (
	SideBuy  Side = "BUY"
	SideSell Side = "SELL"
)

// Valid reports whether the side is one of the known enumeration values.
func (s Side) Valid() bool {
	switch s {
	case SideBuy, SideSell:
		return true
	default:
		return false
	}
}

// Opposite returns the inverse side, useful for computing closing orders.
func (s Side) Opposite() Side {
	if s == SideBuy {
		return SideSell
	}
	return SideBuy
}

func (s Side) String() string { return string(s) }
