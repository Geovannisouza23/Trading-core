package shared

import (
	"encoding/json"

	"github.com/google/uuid"
)

// idValue is the common representation backing every strongly-typed
// identifier in the domain. Using distinct Go types (instead of raw
// uuid.UUID or string) prevents accidentally passing an OrderID where a
// SignalID is expected.
type idValue struct {
	value uuid.UUID
}

func newID() idValue { return idValue{value: uuid.New()} }

func parseID(value string, field string) (idValue, error) {
	parsed, err := uuid.Parse(value)
	if err != nil {
		return idValue{}, NewValidationError(field, "must be a valid UUID")
	}
	return idValue{value: parsed}, nil
}

func (i idValue) String() string       { return i.value.String() }
func (i idValue) IsZero() bool         { return i.value == uuid.Nil }
func (i idValue) equal(o idValue) bool { return i.value == o.value }

func (i idValue) MarshalJSON() ([]byte, error) { return json.Marshal(i.value.String()) }

func (i *idValue) UnmarshalJSON(data []byte) error {
	var raw string
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	parsed, err := uuid.Parse(raw)
	if err != nil {
		return NewValidationError("id", "must be a valid UUID")
	}
	i.value = parsed
	return nil
}

// OrderID uniquely identifies an Order aggregate.
type OrderID struct{ idValue }

func NewOrderID() OrderID { return OrderID{newID()} }
func ParseOrderID(value string) (OrderID, error) {
	id, err := parseID(value, "order_id")
	return OrderID{id}, err
}
func (a OrderID) Equal(b OrderID) bool { return a.equal(b.idValue) }

// SignalID uniquely identifies a TradeSignal aggregate.
type SignalID struct{ idValue }

func NewSignalID() SignalID { return SignalID{newID()} }
func ParseSignalID(value string) (SignalID, error) {
	id, err := parseID(value, "signal_id")
	return SignalID{id}, err
}
func (a SignalID) Equal(b SignalID) bool { return a.equal(b.idValue) }

// RiskDecisionID uniquely identifies a RiskDecision aggregate.
type RiskDecisionID struct{ idValue }

func NewRiskDecisionID() RiskDecisionID { return RiskDecisionID{newID()} }
func ParseRiskDecisionID(value string) (RiskDecisionID, error) {
	id, err := parseID(value, "risk_decision_id")
	return RiskDecisionID{id}, err
}
func (a RiskDecisionID) Equal(b RiskDecisionID) bool { return a.equal(b.idValue) }

// AccountID uniquely identifies an Account aggregate.
type AccountID struct{ idValue }

func NewAccountID() AccountID { return AccountID{newID()} }
func ParseAccountID(value string) (AccountID, error) {
	id, err := parseID(value, "account_id")
	return AccountID{id}, err
}
func (a AccountID) Equal(b AccountID) bool { return a.equal(b.idValue) }

// PositionID uniquely identifies a Position aggregate.
type PositionID struct{ idValue }

func NewPositionID() PositionID { return PositionID{newID()} }
func ParsePositionID(value string) (PositionID, error) {
	id, err := parseID(value, "position_id")
	return PositionID{id}, err
}
func (a PositionID) Equal(b PositionID) bool { return a.equal(b.idValue) }

// MarketEventID uniquely identifies a MarketEvent aggregate.
type MarketEventID struct{ idValue }

func NewMarketEventID() MarketEventID { return MarketEventID{newID()} }
func ParseMarketEventID(value string) (MarketEventID, error) {
	id, err := parseID(value, "market_event_id")
	return MarketEventID{id}, err
}
func (a MarketEventID) Equal(b MarketEventID) bool { return a.equal(b.idValue) }

// IncidentID uniquely identifies a SystemIncident aggregate.
type IncidentID struct{ idValue }

func NewIncidentID() IncidentID { return IncidentID{newID()} }
func ParseIncidentID(value string) (IncidentID, error) {
	id, err := parseID(value, "incident_id")
	return IncidentID{id}, err
}
func (a IncidentID) Equal(b IncidentID) bool { return a.equal(b.idValue) }

// SnapshotID uniquely identifies an AccountSnapshot aggregate.
type SnapshotID struct{ idValue }

func NewSnapshotID() SnapshotID { return SnapshotID{newID()} }
func ParseSnapshotID(value string) (SnapshotID, error) {
	id, err := parseID(value, "snapshot_id")
	return SnapshotID{id}, err
}
func (a SnapshotID) Equal(b SnapshotID) bool { return a.equal(b.idValue) }
