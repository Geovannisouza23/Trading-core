// Package query holds the read-intent inputs accepted by the query use case.
package query

type GetTradingDashboardQuery struct{}

type ListOrdersQuery struct {
	Limit int
}

type GetOrderByIDQuery struct {
	OrderID string
}

type ListSignalsQuery struct {
	Limit int
}

type ListRiskDecisionsQuery struct {
	Limit int
}

type ListEventsQuery struct {
	Limit int
}

type ListIncidentsQuery struct {
	Limit int
}

// --- Quant Engine capability queries ---------------------------------------

type GetBacktestResultQuery struct {
	BacktestID string
}

type StreamBacktestProgressQuery struct {
	BacktestID string
}

type GetOptimizationResultQuery struct {
	OptimizationID string
}

type GetFeatureSchemaQuery struct {
	SchemaVersion string // empty = current
}

type GetModelMetadataQuery struct {
	ModelName    string
	ModelVersion string // empty = currently loaded version
}
