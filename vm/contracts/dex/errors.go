package dex

import (
	"errors"
)

var (
	InvalidOrderIdErr       = errors.New("invalid order id")
	InvalidOrderTypeErr     = errors.New("invalid order type")
	InvalidOrderPriceErr    = errors.New("invalid order price format")
	InvalidOrderQuantityErr = errors.New("invalid order quantity")
	OrderAmountTooSmallErr  = NewDexError("order amount too small", OrderAmountTooSmallFail)

	TradeMarketExistsError    = errors.New("trade market already exists")
	TradeMarketNotExistsError = NewDexError("trade market not exists", TradeMarketNotExistsFail)
	TradeMarketInvalidQuoteTokenError    = errors.New("invalid quote token")
	TradeMarketInvalidTokenPairError    = errors.New("invalid token pair")

	GetOrderByIdFailedErr = errors.New("failed get order by orderId")
	CancelOrderOwnerInvalidErr = errors.New("order to cancel not own to initiator")
	CancelOrderInvalidStatusErr = errors.New("order status is invalid to cancel")
)

func NewDexError(str string, code int) *dexError {
	return &dexError{str, code}
}

type dexError struct {
	str string
	code int
}

func (e *dexError) Error() string {
	return e.str
}

func (e *dexError) Code() int {
	return e.code
}