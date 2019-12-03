package dex

import (
	"errors"
)

var (
	InvalidInputParamErr = errors.New("invalid input param")

	InvalidOrderIdErr       = errors.New("invalid order id")
	InvalidOrderHashErr     = errors.New("invalid order hash")
	InvalidOrderTypeErr     = errors.New("invalid order type")
	InvalidOrderPriceErr    = errors.New("invalid order price format")
	InvalidOrderQuantityErr = errors.New("invalid order quantity")
	OrderNotExistsErr       = errors.New("order not exists")
	OrderAmountTooSmallErr  = errors.New("order amount too small")

	DexStoppedErr                   = errors.New("dex stopped")
	TradeMarketExistsErr            = errors.New("trade market already exists")
	TradeMarketNotExistsErr         = errors.New("trade market not exists")
	TradeMarketStoppedErr           = errors.New("trade market stopped")
	TradeMarketNotGrantedErr        = errors.New("trade market not granted")
	ComposeOrderIdFailErr           = errors.New("compose order id fail")
	DeComposeOrderIdFailErr         = errors.New("decompose order id fail")
	TradeMarketInvalidQuoteTokenErr = errors.New("invalid quote token")
	TradeMarketInvalidTokenPairErr  = errors.New("invalid token pair")
	TradeMarketAllowMineErr         = errors.New("token pair already allow mine")
	TradeMarketNotAllowMineErr      = errors.New("token pair already not allow mine")

	CancelOrderOwnerInvalidErr  = errors.New("order to cancel not own to initiator")
	CancelOrderInvalidStatusErr = errors.New("order status is invalid to cancel")

	OnlyOwnerAllowErr   = errors.New("only owner allow")
	InvalidOperationErr = errors.New("invalid operation")

	ExceedFundAvailableErr         = errors.New("exceed fund available")
	ExceedFundLockedErr            = errors.New("try release locked amount exceed locked")
	InvalidStakeAmountErr          = errors.New("invalid stake amount")
	InvalidStakeActionTypeErr      = errors.New("invalid stake action type")
	ExceedStakedAvailableErr       = errors.New("exceed staked available")
	StakingAmountLeavedNotValidErr = errors.New("staking amount leaved not valid")
	VIPStakingExistsErr            = errors.New("VIP staking exists")
	VIPStakingNotExistsErr         = errors.New("VIP staking not exists")
	SuperVipStakingExistsErr       = errors.New("super VIP staking exists")
	SuperVIPStakingNotExistsErr    = errors.New("super VIP staking not exists")
	StakingInfoByIdNotExistsErr    = errors.New("staking info by id not exists")

	InvalidSourceAddressErr          = errors.New("invalid source address")
	InvalidAmountForStakeCallbackErr = errors.New("invalid amount for stake callback")
	InvalidIdForStakeCallbackErr     = errors.New("invalid id for stake callback")

	InvalidTokenErr                      = errors.New("invalid token")
	PendingNewMarketInnerConflictErr     = errors.New("pending new market inner conflict")
	GetTokenInfoCallbackInnerConflictErr = errors.New("get token info callback inner conflict")
	InvalidTimestampFromTimeOracleErr    = errors.New("invalid timestamp from time oracle")
	InvalidOperatorFeeRateErr            = errors.New("invalid operator fee rate")

	NoDexFeesFoundForValidPeriodErr = errors.New("no fee sum found for valid period")
	NotSetTimestampErr              = errors.New("not set timestamp")
	IterateVmDbFailedErr            = errors.New("iterate vm db failed")
	NotSetMaintainerErr             = errors.New("not set maintainer")
	NotSetMakerMiningAdmin          = errors.New("not set maker mining admin")

	AlreadyIsInviterErr = errors.New("already is inviter")

	InvalidInviteCodeErr     = errors.New("invalid invite code")
	NotBindInviterErr        = errors.New("not bind invite code")
	AlreadyBindInviterErr    = errors.New("already bind inviter")
	NewInviteCodeFailErr     = errors.New("new invite code fail")
	AlreadyQuoteType         = errors.New("already quote type")
	InvalidQuoteTokenTypeErr = errors.New("invalid quote token type")
	FundOwnerNotConfigErr    = errors.New("fund owner not config")

	MultiMarketsInOneActionErr = errors.New("multi markets one action")

	DexFundUserNotExists            = errors.New("fund user doesn't exist.")
	LockedVxAmountLeavedNotValidErr = errors.New("locked vx amount leaved not valid")

	InternalErr = errors.New("internal error")
)
