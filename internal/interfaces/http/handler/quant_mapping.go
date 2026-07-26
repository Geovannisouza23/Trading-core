package handler

import (
	"fmt"
	"time"

	"github.com/shopspring/decimal"

	"trading-core/internal/application/ports/output"
	"trading-core/internal/domain/market"
	"trading-core/internal/domain/shared"
	"trading-core/internal/interfaces/http/request"
)

// candlesFromRequest converts the caller-supplied candle window into
// domain candles. Trading-core does not fetch historical data on the
// caller's behalf (see request.CandleRequest) — every field here must
// round-trip through the same domain constructors used everywhere else,
// so a malformed candle is rejected the same way it would be anywhere
// else in the system, not silently accepted. Every error is a
// *shared.ValidationError (not a bare fmt.Errorf) so presenter.classify
// maps it to 400, not the 500 a generic error falls back to.
func candlesFromRequest(symbolRaw, timeframeRaw string, candles []request.CandleRequest) ([]market.Candle, error) {
	symbol, err := shared.NewSymbol(symbolRaw)
	if err != nil {
		return nil, shared.NewValidationError("symbol", err.Error())
	}
	timeframe, err := shared.NewTimeframe(timeframeRaw)
	if err != nil {
		return nil, shared.NewValidationError("timeframe", err.Error())
	}

	out := make([]market.Candle, 0, len(candles))
	for i, c := range candles {
		open, err := priceFromString(c.Open)
		if err != nil {
			return nil, shared.NewValidationError(fmt.Sprintf("candles[%d].open", i), err.Error())
		}
		high, err := priceFromString(c.High)
		if err != nil {
			return nil, shared.NewValidationError(fmt.Sprintf("candles[%d].high", i), err.Error())
		}
		low, err := priceFromString(c.Low)
		if err != nil {
			return nil, shared.NewValidationError(fmt.Sprintf("candles[%d].low", i), err.Error())
		}
		close, err := priceFromString(c.Close)
		if err != nil {
			return nil, shared.NewValidationError(fmt.Sprintf("candles[%d].close", i), err.Error())
		}
		volumeValue, err := decimal.NewFromString(c.Volume)
		if err != nil {
			return nil, shared.NewValidationError(fmt.Sprintf("candles[%d].volume", i), err.Error())
		}
		volume, err := shared.NewQuantity(volumeValue)
		if err != nil {
			return nil, shared.NewValidationError(fmt.Sprintf("candles[%d].volume", i), err.Error())
		}
		openTime, err := time.Parse(time.RFC3339, c.OpenTime)
		if err != nil {
			return nil, shared.NewValidationError(fmt.Sprintf("candles[%d].open_time", i), err.Error())
		}
		closeTime, err := time.Parse(time.RFC3339, c.CloseTime)
		if err != nil {
			return nil, shared.NewValidationError(fmt.Sprintf("candles[%d].close_time", i), err.Error())
		}

		candle, err := market.NewClosedCandle(symbol, timeframe, open, high, low, close, volume, openTime, closeTime)
		if err != nil {
			return nil, shared.NewValidationError(fmt.Sprintf("candles[%d]", i), err.Error())
		}
		out = append(out, candle)
	}
	return out, nil
}

func priceFromString(raw string) (shared.Price, error) {
	value, err := decimal.NewFromString(raw)
	if err != nil {
		return shared.Price{}, err
	}
	return shared.NewPrice(value)
}

func backtestConfigFromRequest(req request.BacktestConfigRequest) (output.BacktestConfig, error) {
	startAt, err := time.Parse(time.RFC3339, req.From)
	if err != nil {
		return output.BacktestConfig{}, shared.NewValidationError("from", err.Error())
	}
	endAt, err := time.Parse(time.RFC3339, req.To)
	if err != nil {
		return output.BacktestConfig{}, shared.NewValidationError("to", err.Error())
	}
	return output.BacktestConfig{
		Symbol:                  req.Symbol,
		Timeframe:               req.Timeframe,
		StrategyName:            req.StrategyName,
		StrategyVersion:         req.StrategyVersion,
		StrategyParams:          req.StrategyParams,
		InitialCapital:          req.InitialCapital,
		FeeRate:                 req.FeeRate,
		Slippage:                req.Slippage,
		ExecutionModel:          req.ExecutionModel,
		StartAt:                 startAt,
		EndAt:                   endAt,
		AllowShort:              req.AllowShort,
		ModelName:               req.ModelName,
		ModelVersion:            req.ModelVersion,
		GenerateTrainingRecords: req.GenerateTrainingRecords,
	}, nil
}

func parameterRangesFromRequest(ranges []request.ParameterRangeRequest) []output.ParameterRange {
	out := make([]output.ParameterRange, len(ranges))
	for i, r := range ranges {
		out[i] = output.ParameterRange{Name: r.Name, Min: r.Min, Max: r.Max, Step: r.Step, Choices: r.Choices}
	}
	return out
}

func featureValuesFromRequest(values []request.FeatureValueRequest) []output.FeatureValue {
	out := make([]output.FeatureValue, len(values))
	for i, v := range values {
		out[i] = output.FeatureValue{
			Name:             v.Name,
			Kind:             v.Kind,
			NumericValue:     v.NumericValue,
			CategoricalValue: v.CategoricalValue,
			BoolValue:        v.BoolValue,
		}
	}
	return out
}
