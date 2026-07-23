package shared

import "encoding/json"

// Timeframe is a bounded enumeration of candle intervals the system
// understands.
type Timeframe struct {
	value string
}

var validTimeframes = map[string]struct{}{
	"1m": {}, "3m": {}, "5m": {}, "15m": {}, "30m": {},
	"1h": {}, "2h": {}, "4h": {}, "6h": {}, "8h": {}, "12h": {},
	"1d": {}, "3d": {}, "1w": {},
}

// NewTimeframe validates and builds a Timeframe.
func NewTimeframe(value string) (Timeframe, error) {
	if _, ok := validTimeframes[value]; !ok {
		return Timeframe{}, NewValidationError("timeframe", "unsupported timeframe value")
	}
	return Timeframe{value: value}, nil
}

func MustNewTimeframe(value string) Timeframe {
	t, err := NewTimeframe(value)
	if err != nil {
		panic(err)
	}
	return t
}

func (t Timeframe) String() string { return t.value }

func (t Timeframe) Equal(other Timeframe) bool { return t.value == other.value }

func (t Timeframe) MarshalJSON() ([]byte, error) { return json.Marshal(t.value) }

func (t *Timeframe) UnmarshalJSON(data []byte) error {
	var raw string
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	parsed, err := NewTimeframe(raw)
	if err != nil {
		return err
	}
	*t = parsed
	return nil
}
