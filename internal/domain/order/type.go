package order

// Type is the broker order type.
type Type string

const (
	TypeMarket     Type = "MARKET"
	TypeLimit      Type = "LIMIT"
	TypeStopMarket Type = "STOP_MARKET"
	TypeStopLimit  Type = "STOP_LIMIT"
)

func (t Type) Valid() bool {
	switch t {
	case TypeMarket, TypeLimit, TypeStopMarket, TypeStopLimit:
		return true
	default:
		return false
	}
}
