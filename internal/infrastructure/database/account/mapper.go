package account

import (
	domainaccount "trading-core/internal/domain/account"
	"trading-core/internal/domain/operation"
	"trading-core/internal/domain/shared"
)

func toDomain(m model) (*domainaccount.Account, error) {
	id, err := shared.ParseAccountID(m.ID)
	if err != nil {
		return nil, err
	}
	drawdown, err := shared.NewDrawdown(m.CurrentDrawdown)
	if err != nil {
		return nil, err
	}
	return &domainaccount.Account{
		ID:               id,
		Balance:          shared.NewMoney(m.Balance),
		Equity:           shared.NewMoney(m.Equity),
		AvailableBalance: shared.NewMoney(m.AvailableBalance),
		PeakEquity:       shared.NewMoney(m.PeakEquity),
		DailyPnL:         shared.NewMoney(m.DailyPnL),
		WeeklyPnL:        shared.NewMoney(m.WeeklyPnL),
		CurrentDrawdown:  drawdown,
		OperationalMode:  operation.Mode(m.OperationalMode),
		Version:          m.Version,
		UpdatedAt:        m.UpdatedAt,
	}, nil
}

func toModel(a *domainaccount.Account) model {
	return model{
		ID:               a.ID.String(),
		Balance:          a.Balance.Decimal(),
		Equity:           a.Equity.Decimal(),
		AvailableBalance: a.AvailableBalance.Decimal(),
		PeakEquity:       a.PeakEquity.Decimal(),
		DailyPnL:         a.DailyPnL.Decimal(),
		WeeklyPnL:        a.WeeklyPnL.Decimal(),
		CurrentDrawdown:  a.CurrentDrawdown.Decimal(),
		OperationalMode:  string(a.OperationalMode),
		Version:          a.Version,
		UpdatedAt:        a.UpdatedAt,
	}
}
