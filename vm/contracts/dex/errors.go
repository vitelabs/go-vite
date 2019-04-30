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
	TradeMarketAllowMineError    = errors.New("token pair already allow mine")
	TradeMarketNotAllowMineError    = errors.New("token pair already not allow mine")

	GetOrderByIdFailedErr = errors.New("failed get order by orderId")
	CancelOrderOwnerInvalidErr = errors.New("order to cancel not own to initiator")
	CancelOrderInvalidStatusErr = errors.New("order status is invalid to cancel")

	OnlyOwnerAllowErr = errors.New("only owner allow")

	ExceedFundAvailableErr = errors.New("exceed fund available")
	InvalidPledgeAmountErr = errors.New("invalid pledge amount")
	InvalidPledgeActionTypeErr = errors.New("invalid pledge action type")
	ExceedPledgeAvailableErr = errors.New("exceed pledge available")
	PledgeForVipExistsErr = errors.New("pledge for vip exists")
	PledgeForVipNotExistsErr = errors.New("pledge for vip not exists")
	PledgeForVipNotExpireErr = errors.New("pledge for vip not expire")

	InvalidSourceAddressErr           = errors.New("invalid source address")
	InvalidAmountForPledgeCallbackErr = errors.New("invalid amount for pledge callback")

	InvalidTokenErr = errors.New("invalid token")
	PendingDonateAmountSubExceedErr = errors.New("pending donate amount sub exceed")
	PendingNewMarketInnerConflictErr = errors.New("pending new market inner conflict")
	GetTokenInfoCallbackInnerConflictErr = errors.New("get token info callback inner conflict")
	InvalidStatusForPendingMarketInfoErr = errors.New("invalid status for pending market info")
	InvalidTimestampFromTimerErr = errors.New("invalid timestamp from timer")

	NotFoundValueFromDb               = errors.New("not found value from db")
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