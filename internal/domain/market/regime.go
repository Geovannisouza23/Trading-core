package market

// Regime is a coarse classification of current market conditions, produced
// by the (external) Quant Engine and consumed by strategies and risk
// sizing.
type Regime string

const (
	RegimeTrending   Regime = "TRENDING"
	RegimeRangeBound Regime = "RANGE_BOUND"
	RegimeVolatile   Regime = "VOLATILE"
	RegimeUnknown    Regime = "UNKNOWN"
)

func (r Regime) Valid() bool {
	switch r {
	case RegimeTrending, RegimeRangeBound, RegimeVolatile, RegimeUnknown:
		return true
	default:
		return false
	}
}
