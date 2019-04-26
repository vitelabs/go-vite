package dex

import (
	"bytes"
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/contracts/abi"
	dexproto "github.com/vitelabs/go-vite/vm/contracts/dex/proto"
	"github.com/vitelabs/go-vite/vm_context/vmctxt_interface"
	"math/big"
	"strconv"
	"strings"
)

func CheckMarketParam(marketParam *ParamDexFundNewMarket, feeTokenId types.TokenTypeId, feeAmount *big.Int) (err error) {
	if feeTokenId != ledger.ViteTokenId {
		return fmt.Errorf("token type of fee for create market not valid")
	}
	if feeAmount.Cmp(NewMarketFeeAmount) < 0 {
		return fmt.Errorf("fee for create market not enough")
	}
	if _, ok := QuoteTokenMinAmount[marketParam.QuoteToken]; !ok {
		return TradeMarketInvalidQuoteTokenError
	}
	if marketParam.TradeToken == marketParam.QuoteToken {
		return TradeMarketInvalidTokenPairError
	}
	if marketParam.QuoteToken == bitcoinToken && marketParam.TradeToken == usdtToken ||
		marketParam.QuoteToken == ethToken && (marketParam.TradeToken == usdtToken || marketParam.TradeToken == bitcoinToken) ||
		marketParam.QuoteToken == ledger.ViteTokenId && (marketParam.TradeToken == usdtToken || marketParam.TradeToken == bitcoinToken || marketParam.TradeToken == ethToken) {
		return TradeMarketInvalidTokenPairError
	}
	return nil
}

func RenderMarketInfo(db vmctxt_interface.VmDatabase, marketInfo *MarketInfo, newMarketEvent *NewMarketEvent, marketParam *ParamDexFundNewMarket, address types.Address) error {
	if marketInfo, _ := GetMarketInfo(db, marketParam.TradeToken, marketParam.QuoteToken); marketInfo != nil {
		return TradeMarketExistsError
	}
	if tradeTokenInfo, err := GetTokenInfo(db, marketParam.TradeToken); err != nil {
		return err
	} else {
		marketInfo.TradeTokenDecimals = int32(tradeTokenInfo.Decimals)
		newMarketEvent.TradeToken = marketParam.TradeToken.Bytes()
		newMarketEvent.TradeTokenDecimals = marketInfo.TradeTokenDecimals
		newMarketEvent.TradeTokenSymbol = tradeTokenInfo.Symbol
	}
	if quoteTokenInfo, err := GetTokenInfo(db, marketParam.QuoteToken); err != nil {
		return err
	} else {
		marketInfo.QuoteTokenDecimals = int32(quoteTokenInfo.Decimals)
		newMarketEvent.QuoteToken = marketParam.QuoteToken.Bytes()
		newMarketEvent.QuoteTokenDecimals = marketInfo.QuoteTokenDecimals
		newMarketEvent.QuoteTokenSymbol = quoteTokenInfo.Symbol
	}
	marketInfo.Creator = address.Bytes()
	marketInfo.Timestamp = GetTimestampInt64(db)
	newMarketEvent.Creator = address.Bytes()
	return nil
}

func PreCheckOrderParam(orderParam *ParamDexFundNewOrder) error {
	var (
		orderId OrderId
		err     error
	)
	if orderId, err = NewOrderId(orderParam.OrderId); err != nil {
		return InvalidOrderIdErr
	}
	if !orderId.IsNormal() {
		return InvalidOrderIdErr
	}
	if orderParam.Quantity.Sign() <= 0 {
		return InvalidOrderQuantityErr
	}
	// TODO add market order support
	if orderParam.OrderType != Limited {
		return InvalidOrderTypeErr
	}
	if orderParam.OrderType == Limited {
		if !ValidPrice(orderParam.Price) {
			return InvalidOrderPriceErr
		}
	}
	return nil
}

func RenderOrder(orderInfo *dexproto.OrderInfo, param *ParamDexFundNewOrder, db vmctxt_interface.VmDatabase, address types.Address) *dexError {
	order := &dexproto.Order{}
	orderInfo.Order = order
	order.Id = param.OrderId
	order.Address = address.Bytes()
	order.Side = param.Side
	order.Type = int32(param.OrderType)
	order.Price = PriceToBytes(param.Price)
	order.Quantity = param.Quantity.Bytes()
	var marketInfo *MarketInfo
	if marketInfo, _ = GetMarketInfo(db, param.TradeToken, param.QuoteToken); marketInfo == nil {
		return TradeMarketNotExistsError
	}
	orderMarketInfo := &dexproto.OrderMarketInfo{}
	orderMarketInfo.MarketId = marketInfo.MarketId
	orderMarketInfo.TradeToken = param.TradeToken.Bytes()
	orderMarketInfo.QuoteToken = param.QuoteToken.Bytes()
	orderMarketInfo.DecimalsDiff = marketInfo.TradeTokenDecimals - marketInfo.QuoteTokenDecimals
	if order.Type == Limited {
		order.Amount = CalculateRawAmount(order.Quantity, order.Price, orderMarketInfo.DecimalsDiff)
		if !order.Side { //buy
			order.LockedBuyFee = CalculateRawFee(order.Amount, MaxFeeRate())
		}
		totalAmount := AddBigInt(order.Amount, order.LockedBuyFee)
		if new(big.Int).SetBytes(totalAmount).Cmp(QuoteTokenMinAmount[param.QuoteToken]) < 0 {
			return OrderAmountTooSmallErr
		}
	}
	order.Status = Pending
	order.ExecutedQuantity = big.NewInt(0).Bytes()
	order.ExecutedAmount = big.NewInt(0).Bytes()
	order.RefundToken = []byte{}
	order.RefundQuantity = big.NewInt(0).Bytes()
	order.Timestamp = GetTimestampInt64(db)
	orderInfo.OrderMarketInfo = orderMarketInfo
	return nil
}

func CheckSettleActions(actions *dexproto.SettleActions) error {
	if actions == nil || len(actions.FundActions) == 0 && len(actions.FeeActions) == 0 {
		return fmt.Errorf("settle actions is emtpy")
	}
	for _, fund := range actions.FundActions {
		if len(fund.Address) != 20 {
			return fmt.Errorf("invalid address format for settle")
		}
		if len(fund.FundSettles) == 0 {
			return fmt.Errorf("no user funds to settle")
		}
	}

	for _, fee := range actions.FeeActions {
		if len(fee.Token) != 10 {
			return fmt.Errorf("invalid tokenId format for fee settle")
		}
		if len(fee.UserFeeSettles) == 0 {
			return fmt.Errorf("no userFees to settle")
		}
	}
	return nil
}

func DepositAccount(db vmctxt_interface.VmDatabase, address types.Address, tokenId types.TokenTypeId, amount *big.Int) (*dexproto.Account, error) {
	if dexFund, err := GetUserFundFromStorage(db, address); err != nil {
		return nil, err
	} else {
		account, exists := GetAccountByTokeIdFromFund(dexFund, tokenId)
		available := new(big.Int).SetBytes(account.Available)
		account.Available = available.Add(available, amount).Bytes()
		if !exists {
			dexFund.Accounts = append(dexFund.Accounts, account)
		}
		return account, SaveUserFundToStorage(db, address, dexFund)
	}
}

func CheckAndLockFundForNewOrder(dexFund *UserFund, orderInfo *dexproto.OrderInfo) (needUpdate bool, err error) {
	var (
		lockToken, lockAmount []byte
		lockTokenId           *types.TokenTypeId
		lockAmountToInc       *big.Int
	)
	switch orderInfo.Order.Side {
	case false: //buy
		lockToken = orderInfo.OrderMarketInfo.QuoteToken
		if orderInfo.Order.Type == Limited {
			lockAmount = AddBigInt(orderInfo.Order.Amount, orderInfo.Order.LockedBuyFee)
		}
	case true: // sell
		lockToken = orderInfo.OrderMarketInfo.TradeToken
		lockAmount = orderInfo.Order.Quantity
	}
	if tkId, err := types.BytesToTokenTypeId(lockToken); err != nil {
		return false, err
	} else {
		lockTokenId = &tkId
	}
	//var tokenName string
	//if tokenInfo := cabi.GetTokenById(db, *lockTokenId); tokenInfo != nil {
	//	tokenName = tokenInfo.TokenName
	//}
	account, exists := GetAccountByTokeIdFromFund(dexFund, *lockTokenId)
	available := new(big.Int).SetBytes(account.Available)
	if orderInfo.Order.Type != Market || orderInfo.Order.Side { // limited or sell orderInfo
		lockAmountToInc = new(big.Int).SetBytes(lockAmount)
		//fmt.Printf("token %s, available %s , lockAmountToInc %s\n", tokenName, available.String(), lockAmountToInc.String())
		if available.Cmp(lockAmountToInc) < 0 {
			return false, fmt.Errorf("orderInfo lock amount exceed fund available")
		}
	}
	if !orderInfo.Order.Side && orderInfo.Order.Type == Market { // buy or market orderInfo
		if available.Sign() <= 0 {
			return false, fmt.Errorf("no quote amount available for market sell orderInfo")
		} else {
			lockAmount = available.Bytes()
			//NOTE: use amount available for orderInfo amount to full fill
			orderInfo.Order.Amount = lockAmount
			lockAmountToInc = available
			needUpdate = true
		}
	}
	available = available.Sub(available, lockAmountToInc)
	lockedInBig := new(big.Int).SetBytes(account.Locked)
	lockedInBig = lockedInBig.Add(lockedInBig, lockAmountToInc)
	account.Available = available.Bytes()
	account.Locked = lockedInBig.Bytes()
	if !exists {
		dexFund.Accounts = append(dexFund.Accounts, account)
	}
	return needUpdate, nil
}

func DoSettleFund(db vmctxt_interface.VmDatabase, action *dexproto.UserFundSettle) error {
	address := types.Address{}
	address.SetBytes([]byte(action.Address))
	if dexFund, err := GetUserFundFromStorage(db, address); err != nil {
		return err
	} else {
		for _, fundSettle := range action.FundSettles {
			if tokenId, err := types.BytesToTokenTypeId(fundSettle.Token); err != nil {
				return err
			} else {
				if tokenInfo, _ := GetTokenInfo(db, tokenId); tokenInfo == nil {
					return InvalidTokenErr
				}
				account, exists := GetAccountByTokeIdFromFund(dexFund, tokenId)
				//fmt.Printf("origin account for :address %s, tokenId %s, available %s, locked %s\n", address.String(), tokenId.String(), new(big.Int).SetBytes(account.Available).String(), new(big.Int).SetBytes(account.Locked).String())
				if CmpToBigZero(fundSettle.ReduceLocked) != 0 {
					if CmpForBigInt(fundSettle.ReduceLocked, account.Locked) > 0 {
						return fmt.Errorf("try reduce locked amount execeed locked")
					}
					account.Locked = SubBigIntAbs(account.Locked, fundSettle.ReduceLocked)
				}
				if CmpToBigZero(fundSettle.ReleaseLocked) != 0 {
					if CmpForBigInt(fundSettle.ReleaseLocked, account.Locked) > 0 {
						return fmt.Errorf("try release locked amount execeed locked")
					}
					account.Locked = SubBigIntAbs(account.Locked, fundSettle.ReleaseLocked)
					account.Available = AddBigInt(account.Available, fundSettle.ReleaseLocked)
				}
				if CmpToBigZero(fundSettle.IncAvailable) != 0 {
					account.Available = AddBigInt(account.Available, fundSettle.IncAvailable)
				}
				if !exists {
					dexFund.Accounts = append(dexFund.Accounts, account)
				}
				// must do after account updated by settle
				if bytes.Equal(fundSettle.Token, VxTokenBytes) {
					if err = OnSettleVx(db, action.Address, fundSettle, account); err != nil {
						return err
					}
				}
				//fmt.Printf("settle for :address %s, tokenId %s, ReduceLocked %s, ReleaseLocked %s, IncAvailable %s\n", address.String(), tokenId.String(), new(big.Int).SetBytes(action.ReduceLocked).String(), new(big.Int).SetBytes(action.ReleaseLocked).String(), new(big.Int).SetBytes(action.IncAvailable).String())
			}
		}
		if err = SaveUserFundToStorage(db, address, dexFund); err != nil {
			return err
		}
	}
	return nil
}

func PledgeRequest(db vmctxt_interface.VmDatabase, address types.Address, pledgeType int8, amount *big.Int) ([]byte, error) {
	if pledgeType == PledgeForVip {
		if pledgeVip, _ := GetPledgeForVip(db, address); pledgeVip != nil {
			return nil, PledgeForVipExistsErr
		}
	}
	if dexFund, err := GetUserFundFromStorage(db, address); err != nil {
		return nil, err
	} else {
		account, exists := GetAccountByTokeIdFromFund(dexFund, ledger.ViteTokenId)
		if !exists || CmpForBigInt(account.Available, amount.Bytes()) < 0 {
			return nil, ExceedFundAvailableErr
		} else {
			account.Available = SubBigInt(account.Available, amount.Bytes()).Bytes()
			if err = SaveUserFundToStorage(db, address, dexFund); err != nil {
				return nil, err
			}
			if pledgeData, err := abi.ABIPledge.PackMethod(abi.MethodNamePledge, address, types.AddressDexFund, pledgeType, amount); err != nil {
				return nil, err
			} else {
				return pledgeData, err
			}
		}
	}
}

func CancelPledgeRequest(db vmctxt_interface.VmDatabase, address types.Address, pledgeType int8, amount *big.Int) ([]byte, error) {
	if pledgeType == PledgeForVx {
		available := GetPledgeForVx(db, address)
		leave := new(big.Int).Sub(available, amount)
		if leave.Sign() < 0 || leave.Cmp(PledgeForVxMinAmount) < 0 {
			return nil, ExceedPledgeAvailableErr
		}
	} else {
		pledgeVip, _ := GetPledgeForVip(db, address)
		if pledgeVip == nil {
			return nil, PledgeForVipNotExistsErr
		} else {
			if pledgeVip.PledgeTimes == 1 && GetTimestampInt64(db)-pledgeVip.Timestamp < PledgeForVipDuration {
				return nil, PledgeForVipNotExpireErr
			}
		}
	}
	if cancelPledgeData, err := abi.ABIPledge.PackMethod(abi.MethodNameCancelPledge, address, types.AddressDexFund, pledgeType, amount); err != nil {
		return nil, err
	} else {
		return cancelPledgeData, err
	}
}

func ValidPrice(price string) bool {
	if len(price) == 0 {
		return false
	} else if pr, err := strconv.ParseFloat(price, 64); err != nil || pr <= 0 {
		return false
	} else if !checkPriceChar(price) {
		return false
	} else {
		idx := strings.Index(price, ".")
		if idx > priceIntMaxLen { // 10^12 < 2^40(5bytes)
			return false
		} else if idx >= 0 {
			if len(price)-(idx+1) > priceDecimalMaxLen { // // max price precision is 9 decimals 10^9 < 2^32(4bytes)
				return false
			}
		}
	}
	return true
}

func checkPriceChar(price string) bool {
	for _, v := range price {
		if v != '.' && (uint8(v) < '0' || uint8(v) > '9') {
			return false
		}
	}
	return true
}
