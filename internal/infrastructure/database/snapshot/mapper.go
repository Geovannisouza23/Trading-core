package snapshot

import (
	"encoding/json"
	"fmt"

	"trading-core/internal/domain/account"
	"trading-core/internal/domain/order"
	"trading-core/internal/domain/position"
	"trading-core/internal/domain/shared"
)

func toDomain(m model) (*account.Snapshot, error) {
	id, err := shared.ParseSnapshotID(m.ID)
	if err != nil {
		return nil, err
	}
	drawdown, err := shared.NewDrawdown(m.Drawdown)
	if err != nil {
		return nil, err
	}

	var positions []position.Position
	if len(m.Positions) > 0 {
		if err := json.Unmarshal(m.Positions, &positions); err != nil {
			return nil, fmt.Errorf("decoding snapshot positions: %w", err)
		}
	}
	var openOrders []order.Order
	if len(m.OpenOrders) > 0 {
		if err := json.Unmarshal(m.OpenOrders, &openOrders); err != nil {
			return nil, fmt.Errorf("decoding snapshot open orders: %w", err)
		}
	}

	return &account.Snapshot{
		ID:         id,
		Balance:    shared.NewMoney(m.Balance),
		Equity:     shared.NewMoney(m.Equity),
		Positions:  positions,
		OpenOrders: openOrders,
		DailyPnL:   shared.NewMoney(m.DailyPnL),
		Drawdown:   drawdown,
		Timestamp:  m.Timestamp,
	}, nil
}

func toModel(s *account.Snapshot) (model, error) {
	positionsJSON, err := json.Marshal(s.Positions)
	if err != nil {
		return model{}, err
	}
	openOrdersJSON, err := json.Marshal(s.OpenOrders)
	if err != nil {
		return model{}, err
	}
	return model{
		ID:         s.ID.String(),
		Balance:    s.Balance.Decimal(),
		Equity:     s.Equity.Decimal(),
		Positions:  positionsJSON,
		OpenOrders: openOrdersJSON,
		DailyPnL:   s.DailyPnL.Decimal(),
		Drawdown:   s.Drawdown.Decimal(),
		Timestamp:  s.Timestamp,
	}, nil
}
