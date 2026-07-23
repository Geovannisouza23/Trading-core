package binance

import (
	"context"
	"errors"
	"net/url"
	"strconv"
	"sync"

	"github.com/shopspring/decimal"

	"trading-core/internal/application/ports/output"
	"trading-core/internal/domain/shared"
)

// Broker implements output.Broker against Binance Futures (USDⓈ-M). The
// same type backs both the testnet and real adapters; they differ only in
// which base URL and safety gates the caller configures.
//
// Known limitation: output.Broker.GetOrder and FindOrderByClientOrderID
// take only an order id / client order id (per spec section 13.1), but
// Binance's order-status endpoints require the symbol too. This adapter
// keeps an in-memory id->symbol correlation cache populated by PlaceOrder/
// PlaceStop; a lookup for an order this process never placed (e.g. right
// after a restart) will fail until ReconcileBrokerState has a chance to
// rebuild it from GetOpenOrders/GetPositions.
type Broker struct {
	client *Client

	mu             sync.Mutex
	symbolByClient map[string]string
	symbolByBroker map[string]string
}

func NewBroker(client *Client) *Broker {
	return &Broker{
		client:         client,
		symbolByClient: make(map[string]string),
		symbolByBroker: make(map[string]string),
	}
}

var _ output.Broker = (*Broker)(nil)

func (b *Broker) rememberSymbol(clientOrderID, brokerOrderID, symbol string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if clientOrderID != "" {
		b.symbolByClient[clientOrderID] = symbol
	}
	if brokerOrderID != "" {
		b.symbolByBroker[brokerOrderID] = symbol
	}
}

func (b *Broker) GetAccount(ctx context.Context) (output.BrokerAccount, error) {
	var resp accountResponse
	if err := b.client.do(ctx, "GET", "/fapi/v2/account", url.Values{}, &resp); err != nil {
		return output.BrokerAccount{}, err
	}
	balance, err := decimal.NewFromString(resp.TotalWalletBalance)
	if err != nil {
		return output.BrokerAccount{}, err
	}
	equity, err := decimal.NewFromString(resp.TotalMarginBalance)
	if err != nil {
		return output.BrokerAccount{}, err
	}
	available, err := decimal.NewFromString(resp.AvailableBalance)
	if err != nil {
		return output.BrokerAccount{}, err
	}
	return output.BrokerAccount{
		Balance:          shared.NewMoney(balance),
		Equity:           shared.NewMoney(equity),
		AvailableBalance: shared.NewMoney(available),
	}, nil
}

func (b *Broker) GetPositions(ctx context.Context) ([]output.BrokerPosition, error) {
	var resp []positionRisk
	if err := b.client.do(ctx, "GET", "/fapi/v2/positionRisk", url.Values{}, &resp); err != nil {
		return nil, err
	}

	var positions []output.BrokerPosition
	for _, p := range resp {
		amount, err := decimal.NewFromString(p.PositionAmt)
		if err != nil || amount.IsZero() {
			continue
		}
		symbol, err := shared.NewSymbol(p.Symbol)
		if err != nil {
			continue
		}
		side := shared.SideBuy
		if amount.IsNegative() {
			side = shared.SideSell
			amount = amount.Neg()
		}
		quantity, err := shared.NewQuantity(amount)
		if err != nil {
			continue
		}
		entry, err := decimal.NewFromString(p.EntryPrice)
		if err != nil {
			continue
		}
		mark, err := decimal.NewFromString(p.MarkPrice)
		if err != nil {
			continue
		}
		entryPrice, err := shared.NewPrice(entry)
		if err != nil {
			continue
		}
		markPrice, err := shared.NewPrice(mark)
		if err != nil {
			continue
		}
		positions = append(positions, output.BrokerPosition{
			Symbol: symbol, Side: side, Quantity: quantity, EntryPrice: entryPrice, MarkPrice: markPrice,
		})
	}
	return positions, nil
}

func (b *Broker) GetOpenOrders(ctx context.Context) ([]output.BrokerOrder, error) {
	var resp []orderResponse
	if err := b.client.do(ctx, "GET", "/fapi/v1/openOrders", url.Values{}, &resp); err != nil {
		return nil, err
	}
	orders := make([]output.BrokerOrder, 0, len(resp))
	for _, o := range resp {
		bo, err := toBrokerOrder(o)
		if err != nil {
			continue
		}
		b.rememberSymbol(bo.ClientOrderID, bo.BrokerOrderID, o.Symbol)
		orders = append(orders, bo)
	}
	return orders, nil
}

func (b *Broker) PlaceOrder(ctx context.Context, req output.PlaceOrderRequest) (output.BrokerOrder, error) {
	params := url.Values{}
	params.Set("symbol", req.Symbol.String())
	params.Set("side", string(req.Side))
	params.Set("type", req.Type)
	params.Set("quantity", req.Quantity.Decimal().String())
	params.Set("newClientOrderId", req.ClientOrderID.String())
	if req.Type == "LIMIT" && req.Price != nil {
		params.Set("price", req.Price.Decimal().String())
		params.Set("timeInForce", "GTC")
	}

	var resp orderResponse
	if err := b.client.do(ctx, "POST", "/fapi/v1/order", params, &resp); err != nil {
		return output.BrokerOrder{}, err
	}
	bo, err := toBrokerOrder(resp)
	if err != nil {
		return output.BrokerOrder{}, err
	}
	b.rememberSymbol(bo.ClientOrderID, bo.BrokerOrderID, req.Symbol.String())
	return bo, nil
}

func (b *Broker) PlaceStop(ctx context.Context, req output.PlaceStopRequest) (output.BrokerOrder, error) {
	params := url.Values{}
	params.Set("symbol", req.Symbol.String())
	params.Set("side", string(req.Side))
	params.Set("type", "STOP_MARKET")
	params.Set("quantity", req.Quantity.Decimal().String())
	params.Set("stopPrice", req.StopPrice.Decimal().String())
	params.Set("newClientOrderId", req.ClientOrderID.String())
	params.Set("reduceOnly", "true")

	var resp orderResponse
	if err := b.client.do(ctx, "POST", "/fapi/v1/order", params, &resp); err != nil {
		return output.BrokerOrder{}, err
	}
	bo, err := toBrokerOrder(resp)
	if err != nil {
		return output.BrokerOrder{}, err
	}
	b.rememberSymbol(bo.ClientOrderID, bo.BrokerOrderID, req.Symbol.String())
	return bo, nil
}

func (b *Broker) CancelOrder(ctx context.Context, brokerOrderID string) error {
	symbol, ok := b.knownSymbolForBrokerID(brokerOrderID)
	if !ok {
		return errors.New("binance: cannot cancel order with unknown symbol (never seen in this process)")
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("orderId", brokerOrderID)
	return b.client.do(ctx, "DELETE", "/fapi/v1/order", params, nil)
}

func (b *Broker) GetOrder(ctx context.Context, brokerOrderID string) (output.BrokerOrder, error) {
	symbol, ok := b.knownSymbolForBrokerID(brokerOrderID)
	if !ok {
		return output.BrokerOrder{}, errors.New("binance: cannot look up order with unknown symbol (never seen in this process)")
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("orderId", brokerOrderID)

	var resp orderResponse
	if err := b.client.do(ctx, "GET", "/fapi/v1/order", params, &resp); err != nil {
		var apiErr *APIError
		if errors.As(err, &apiErr) && apiErr.IsNotFound() {
			return output.BrokerOrder{}, output.ErrNotFound
		}
		return output.BrokerOrder{}, err
	}
	return toBrokerOrder(resp)
}

func (b *Broker) FindOrderByClientOrderID(ctx context.Context, clientOrderID string) (output.BrokerOrder, error) {
	symbol, ok := b.knownSymbolForClientID(clientOrderID)
	if !ok {
		return output.BrokerOrder{}, errors.New("binance: cannot look up order with unknown symbol (never seen in this process)")
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("origClientOrderId", clientOrderID)

	var resp orderResponse
	if err := b.client.do(ctx, "GET", "/fapi/v1/order", params, &resp); err != nil {
		var apiErr *APIError
		if errors.As(err, &apiErr) && apiErr.IsNotFound() {
			return output.BrokerOrder{}, output.ErrNotFound
		}
		return output.BrokerOrder{}, err
	}
	return toBrokerOrder(resp)
}

func (b *Broker) knownSymbolForBrokerID(id string) (string, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	s, ok := b.symbolByBroker[id]
	return s, ok
}

func (b *Broker) knownSymbolForClientID(id string) (string, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	s, ok := b.symbolByClient[id]
	return s, ok
}

func toBrokerOrder(o orderResponse) (output.BrokerOrder, error) {
	symbol, err := shared.NewSymbol(o.Symbol)
	if err != nil {
		return output.BrokerOrder{}, err
	}
	quantity, err := decimal.NewFromString(o.OrigQty)
	if err != nil {
		return output.BrokerOrder{}, err
	}
	filled, err := decimal.NewFromString(o.ExecutedQty)
	if err != nil {
		filled = decimal.Zero
	}
	qty, err := shared.NewQuantity(quantity)
	if err != nil {
		return output.BrokerOrder{}, err
	}
	filledQty, err := shared.NewQuantity(filled)
	if err != nil {
		filledQty = shared.ZeroQuantity()
	}

	var avgPrice *shared.Price
	if o.AvgPrice != "" {
		if avg, err := decimal.NewFromString(o.AvgPrice); err == nil && !avg.IsZero() {
			if p, err := shared.NewPrice(avg); err == nil {
				avgPrice = &p
			}
		}
	}

	return output.BrokerOrder{
		BrokerOrderID:  strconv.FormatInt(o.OrderID, 10),
		ClientOrderID:  o.ClientOrderID,
		Symbol:         symbol,
		Side:           shared.Side(o.Side),
		Type:           o.Type,
		Quantity:       qty,
		FilledQuantity: filledQty,
		AveragePrice:   avgPrice,
		RawStatus:      o.Status,
	}, nil
}
