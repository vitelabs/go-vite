package dex

import (
	"errors"
)

var (
	InvalidOrderIdErr       = errors.New("invalid order id")
	InvalidOrderTypeErr     = errors.New("invalid order type")
	InvalidOrderPriceErr    = errors.New("invalid order price format")
	InvalidOrderQuantityErr = errors.New("invalid order quantity")
	OrderAmountTooSmallErr  = errors.New("order amount too small")

	TradeMarketExistsError    = errors.New("trade market already exists")
	TradeMarketNotExistsError = errors.New("trade market not exists")
	TradeMarketInvalidQuoteTokenError    = errors.New("invalid quote token")
	TradeMarketInvalidTokenPairError    = errors.New("invalid token pair")

	GetOrderByIdFailedErr = errors.New("failed get order by orderId")
	CancelOrderOwnerInvalidErr = errors.New("order to cancel not own to initiator")
	CancelOrderInvalidStatusErr = errors.New("order status is invalid to cancel")
)