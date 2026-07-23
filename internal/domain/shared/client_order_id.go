package shared

import (
	"encoding/json"
	"regexp"
)

var clientOrderIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{8,36}$`)

// ClientOrderID is the caller-generated idempotency key for an order. It is
// the single source of truth used to detect duplicate submissions before an
// order ever reaches a broker.
type ClientOrderID struct {
	value string
}

// NewClientOrderID validates and builds a ClientOrderID. The length bound
// (36) matches the strictest broker constraint we integrate with (Binance).
func NewClientOrderID(value string) (ClientOrderID, error) {
	if !clientOrderIDPattern.MatchString(value) {
		return ClientOrderID{}, NewValidationError("client_order_id", "must be 8-36 characters of [A-Za-z0-9_-]")
	}
	return ClientOrderID{value: value}, nil
}

func MustNewClientOrderID(value string) ClientOrderID {
	id, err := NewClientOrderID(value)
	if err != nil {
		panic(err)
	}
	return id
}

func (c ClientOrderID) String() string { return c.value }

func (c ClientOrderID) Equal(other ClientOrderID) bool { return c.value == other.value }

func (c ClientOrderID) IsEmpty() bool { return c.value == "" }

func (c ClientOrderID) MarshalJSON() ([]byte, error) { return json.Marshal(c.value) }

func (c *ClientOrderID) UnmarshalJSON(data []byte) error {
	var raw string
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	parsed, err := NewClientOrderID(raw)
	if err != nil {
		return err
	}
	*c = parsed
	return nil
}
