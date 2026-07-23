// Package account models the Account aggregate: the single ledger the risk
// manager and dashboard read from.
package account

import (
	"time"

	"trading-core/internal/domain/operation"
	"trading-core/internal/domain/shared"
)

// Account is the trading account ledger (paper, testnet or real).
type Account struct {
	ID               shared.AccountID
	Balance          shared.Money
	Equity           shared.Money
	AvailableBalance shared.Money
	PeakEquity       shared.Money
	DailyPnL         shared.Money
	WeeklyPnL        shared.Money
	CurrentDrawdown  shared.Drawdown
	OperationalMode  operation.Mode
	Version          int
	UpdatedAt        time.Time
}

// NewPaperAccount creates the initial PAPER account seeded with the given
// starting balance.
func NewPaperAccount(startingBalance shared.Money, now time.Time) (*Account, error) {
	if startingBalance.IsNegative() || startingBalance.IsZero() {
		return nil, shared.NewValidationError("starting_balance", "must be greater than zero")
	}
	return &Account{
		ID:               shared.NewAccountID(),
		Balance:          startingBalance,
		Equity:           startingBalance,
		AvailableBalance: startingBalance,
		PeakEquity:       startingBalance,
		DailyPnL:         shared.ZeroMoney(),
		WeeklyPnL:        shared.ZeroMoney(),
		CurrentDrawdown:  shared.MustNewDrawdown(shared.ZeroMoney().Decimal()),
		OperationalMode:  operation.ModePaper,
		Version:          1,
		UpdatedAt:        now,
	}, nil
}

// ApplyRealizedPnL books realized P&L into balance, equity, daily and
// weekly running totals, then recalculates drawdown from peak equity.
func (a *Account) ApplyRealizedPnL(delta shared.Money, now time.Time) {
	a.Balance = a.Balance.Add(delta)
	a.Equity = a.Equity.Add(delta)
	a.AvailableBalance = a.AvailableBalance.Add(delta)
	a.DailyPnL = a.DailyPnL.Add(delta)
	a.WeeklyPnL = a.WeeklyPnL.Add(delta)
	if a.Equity.GreaterThan(a.PeakEquity) {
		a.PeakEquity = a.Equity
	}
	a.CurrentDrawdown = shared.DrawdownFromPeak(a.PeakEquity, a.Equity)
	a.Version++
	a.UpdatedAt = now
}

// ReserveMargin moves funds from available balance into margin used by an
// open position, without affecting equity/balance.
func (a *Account) ReserveMargin(amount shared.Money, now time.Time) error {
	if amount.GreaterThan(a.AvailableBalance) {
		return shared.NewValidationError("amount", "insufficient available balance")
	}
	a.AvailableBalance = a.AvailableBalance.Sub(amount)
	a.Version++
	a.UpdatedAt = now
	return nil
}

// ReleaseMargin returns previously reserved margin back to available
// balance (e.g. when a position closes).
func (a *Account) ReleaseMargin(amount shared.Money, now time.Time) {
	a.AvailableBalance = a.AvailableBalance.Add(amount)
	a.Version++
	a.UpdatedAt = now
}

// MarkToMarket updates unrealized equity based on the current mark-to-market
// value of all open positions (balance + sum of unrealized P&L).
func (a *Account) MarkToMarket(unrealizedPnL shared.Money, now time.Time) {
	a.Equity = a.Balance.Add(unrealizedPnL)
	if a.Equity.GreaterThan(a.PeakEquity) {
		a.PeakEquity = a.Equity
	}
	a.CurrentDrawdown = shared.DrawdownFromPeak(a.PeakEquity, a.Equity)
	a.Version++
	a.UpdatedAt = now
}

// ResetDailyPnL zeroes the daily running total (called by a daily scheduler).
func (a *Account) ResetDailyPnL(now time.Time) {
	a.DailyPnL = shared.ZeroMoney()
	a.Version++
	a.UpdatedAt = now
}

// ResetWeeklyPnL zeroes the weekly running total (called by a weekly scheduler).
func (a *Account) ResetWeeklyPnL(now time.Time) {
	a.WeeklyPnL = shared.ZeroMoney()
	a.Version++
	a.UpdatedAt = now
}

// SetOperationalMode updates the cached operational mode on the account
// after a successful transition on the operation.State aggregate.
func (a *Account) SetOperationalMode(mode operation.Mode, now time.Time) {
	a.OperationalMode = mode
	a.Version++
	a.UpdatedAt = now
}
