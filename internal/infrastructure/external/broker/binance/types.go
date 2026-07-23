package binance

// accountResponse maps the fields of GET /fapi/v2/account we need.
type accountResponse struct {
	TotalWalletBalance string `json:"totalWalletBalance"`
	TotalMarginBalance string `json:"totalMarginBalance"`
	AvailableBalance   string `json:"availableBalance"`
}

// positionRisk maps one element of GET /fapi/v2/positionRisk.
type positionRisk struct {
	Symbol      string `json:"symbol"`
	PositionAmt string `json:"positionAmt"`
	EntryPrice  string `json:"entryPrice"`
	MarkPrice   string `json:"markPrice"`
}

// orderResponse maps the common shape of order-related endpoints (place,
// get, cancel, open orders).
type orderResponse struct {
	OrderID       int64  `json:"orderId"`
	ClientOrderID string `json:"clientOrderId"`
	Symbol        string `json:"symbol"`
	Side          string `json:"side"`
	Type          string `json:"type"`
	OrigQty       string `json:"origQty"`
	ExecutedQty   string `json:"executedQty"`
	AvgPrice      string `json:"avgPrice"`
	Status        string `json:"status"`
}
