package shared

import (
	"encoding/json"
	"regexp"
	"strings"
)

var symbolPattern = regexp.MustCompile(`^[A-Z0-9]{3,20}$`)

// Symbol identifies a tradable instrument (e.g. BTCUSDT).
type Symbol struct {
	value string
}

// NewSymbol validates and builds a Symbol. Input is normalized to upper case
// before validation.
func NewSymbol(value string) (Symbol, error) {
	normalized := strings.ToUpper(strings.TrimSpace(value))
	if !symbolPattern.MatchString(normalized) {
		return Symbol{}, NewValidationError("symbol", "must be 3-20 alphanumeric upper-case characters")
	}
	return Symbol{value: normalized}, nil
}

func MustNewSymbol(value string) Symbol {
	s, err := NewSymbol(value)
	if err != nil {
		panic(err)
	}
	return s
}

func (s Symbol) String() string { return s.value }

func (s Symbol) Equal(other Symbol) bool { return s.value == other.value }

func (s Symbol) MarshalJSON() ([]byte, error) { return json.Marshal(s.value) }

func (s *Symbol) UnmarshalJSON(data []byte) error {
	var raw string
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	parsed, err := NewSymbol(raw)
	if err != nil {
		return err
	}
	*s = parsed
	return nil
}
