// Package usecase implements every input port with real orchestration logic
// against the output ports. This is where domain, ports and transactions
// meet; it never imports a concrete infrastructure package.
package usecase

import "errors"

var (
	ErrNotFound                = errors.New("resource not found")
	ErrStaleMarketData         = errors.New("market data is older than the maximum allowed age")
	ErrSignalNotAllowed        = errors.New("risk decision did not allow this signal")
	ErrOperationalModeBlocked  = errors.New("current operational mode does not allow this action")
	ErrDuplicateExecution      = errors.New("this signal/decision is already being executed or was already executed")
	ErrBrokerOrderStateUnknown = errors.New("broker order state could not be determined")
)
