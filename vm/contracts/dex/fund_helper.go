package dex

import (
	"bytes"
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/contracts/abi"
	dexproto "github.com/vitelabs/go-vite/vm/contracts/dex/proto"
	"github.com/vitelabs/go-vite/vm/util"
	"github.com/vitelabs/go-vite/vm_db"
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
	if _, ok := QuoteTokenInfos[marketParam.QuoteToken]; !ok {
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

func RenderMarketInfo(db vm_db.VmDb, marketInfo *MarketInfo, newMarketEvent *NewMarketEvent, tradeToken, quoteToken types.TokenTypeId, tradeTokenInfo *TokenInfo, address *types.Address) (err error) {
	quoteTokenInfo := QuoteTokenInfos[quoteToken]
	marketInfo.MarketSymbol = quoteTokenInfo.Symbol
	marketInfo.QuoteTokenDecimals = quoteTokenInfo.Decimals
	if tradeTokenInfo == nil {
		if tradeTokenInfo, err = GetTokenInfo(db, tradeToken); err != nil {
			return err
		}
	}
	if tradeTokenInfo != nil {
		if err = renderMarketInfoWithTradeTokenInfo(db, marketInfo, tradeTokenInfo); err != nil {
			return err
		}
		renderNewMarketEvent(marketInfo, newMarketEvent, tradeToken, quoteToken, tradeTokenInfo, quoteTokenInfo)
	}

	if marketInfo.Creator == nil {
		marketInfo.Creator = address.Bytes()
	}
	if marketInfo.Timestamp == 0 {
		marketInfo.Timestamp = GetTimestampInt64(db)
	}
	return nil
}

func renderMarketInfoWithTradeTokenInfo(db vm_db.VmDb, marketInfo *MarketInfo, tradeTokenInfo *TokenInfo) (err error) {
	if marketInfo.MarketId, err = NewAndSaveMarketId(db); err != nil {
		return err
	}
	indexStr := strconv.Itoa(int(tradeTokenInfo.Index))
	for ; len(indexStr) < 3; {
		indexStr = "0" + indexStr
	}
	marketInfo.MarketSymbol = fmt.Sprintf("%s-%s_%s", tradeTokenInfo.Symbol, indexStr, marketInfo.MarketSymbol)
	marketInfo.TradeTokenDecimals = tradeTokenInfo.Decimals
	marketInfo.Valid = true
	return nil
}

func renderNewMarketEvent(marketInfo *MarketInfo, newMarketEvent *NewMarketEvent, tradeToken, quoteToken types.TokenTypeId, tradeTokenInfo, quoteTokenInfo *TokenInfo) {
	newMarketEvent.MarketId = marketInfo.MarketId
	newMarketEvent.MarketSymbol = marketInfo.MarketSymbol
	newMarketEvent.TradeToken = tradeToken.Bytes()
	newMarketEvent.TradeTokenDecimals = tradeTokenInfo.Decimals
	newMarketEvent.TradeTokenSymbol = tradeTokenInfo.Symbol
	newMarketEvent.QuoteToken = quoteToken.Bytes()
	newMarketEvent.QuoteTokenDecimals = quoteTokenInfo.Decimals
	newMarketEvent.QuoteTokenSymbol = quoteTokenInfo.Symbol
	newMarketEvent.Creator = marketInfo.Creator
}

func OnNewMarketValid(db vm_db.VmDb, reader util.ConsensusReader, marketInfo *MarketInfo, newMarketEvent *NewMarketEvent, tradeToken, quoteToken types.TokenTypeId, address *types.Address) (err error) {
	userFee := &dexproto.UserFeeSettle{}
	userFee.Address = address.Bytes()
	userFee.Amount = NewMarketFeeDividendAmount.Bytes()
	fee := &dexproto.FeeSettle{}
	fee.Token = ledger.ViteTokenId.Bytes()
	fee.UserFeeSettles = append(fee.UserFeeSettles, userFee)
	if err = SettleFeeSum(db, reader, []*dexproto.FeeSettle{fee}); err != nil {
		return
	}
	if err = SettleUserFees(db, reader, fee); err != nil {
		return
	}
	if err = AddDonateFeeSum(db, reader); err != nil {
		return
	}
	if err = SaveMarketInfo(db, marketInfo, tradeToken, quoteToken); err != nil {
		return
	}
	AddNewMarketEventLog(db, newMarketEvent)
	return
}

func OnNewMarketPending(db vm_db.VmDb, param *ParamDexFundNewMarket, marketInfo *MarketInfo) (data []byte, err error) {
	if err = SaveMarketInfo(db, marketInfo, param.TradeToken, param.QuoteToken); err != nil {
		return
	}
	if err = AddToPendingNewMarkets(db, param.TradeToken, param.QuoteToken); err != nil {
		return
	}
	if err = AddPendingNewMarketFeeSum(db); err != nil {
		return
	}
	if data, err = abi.ABIMintage.PackMethod(abi.MethodNameGetTokenInfo, param.TradeToken); err != nil {
		return
	} else {
		return
	}
}

func OnGetTokenInfoSuccess(db vm_db.VmDb, reader util.ConsensusReader, tradeTokenId types.TokenTypeId, tokenInfoRes *ParamDexFundGetTokenInfoCallback) (err error) {
	tradeTokenInfo := &TokenInfo{}
	tradeTokenInfo.Decimals = int32(tokenInfoRes.Decimals)
	tradeTokenInfo.Symbol = tokenInfoRes.TokenSymbol
	tradeTokenInfo.Index = int32(tokenInfoRes.Index)
	if err = SaveTokenInfo(db, tradeTokenId, tradeTokenInfo); err != nil {
		return err
	}
	var quoteTokens [][]byte
	if quoteTokens, err = FilterPendingNewMarkets(db, tradeTokenId); err != nil {
		return err
	} else {
		for _, qt := range quoteTokens {
			if quoteTokenId, err := types.BytesToTokenTypeId(qt); err != nil {
				return err
			} else {
				if marketInfo, err := GetMarketInfo(db, tradeTokenId, quoteTokenId); err != nil {
					return err
				} else {
					if marketInfo.Valid {
						return InvalidStatusForPendingMarketInfoErr
					}
					var address types.Address
					address, err = types.BytesToAddress(marketInfo.Creator)
					if err != nil {
						return err
					}
					newMarketEvent := &NewMarketEvent{}
					if err = RenderMarketInfo(db, marketInfo, newMarketEvent, tradeTokenId, quoteTokenId, tradeTokenInfo, nil); err != nil {
						return err
					}
					if err = SubPendingNewMarketFeeSum(db); err != nil {
						return err
					}
					if err = OnNewMarketValid(db, reader, marketInfo, newMarketEvent, tradeTokenId, quoteTokenId, &address); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

func OnGetTokenInfoFailed(db vm_db.VmDb, tradeTokenId types.TokenTypeId) (refundBlocks []*ledger.AccountBlock, err error) {
	var quoteTokens [][]byte
	if quoteTokens, err = FilterPendingNewMarkets(db, tradeTokenId); err != nil {
		return nil, err
	} else {
		refundBlocks := make([]*ledger.AccountBlock, len(quoteTokens))
		for index, qt := range quoteTokens {
			if quoteTokenId, err := types.BytesToTokenTypeId(qt); err != nil {
				return nil, err
			} else {
				if marketInfo, err := GetMarketInfo(db, tradeTokenId, quoteTokenId); err != nil {
					return nil, err
				} else {
					if marketInfo.Valid {
						return nil, InvalidStatusForPendingMarketInfoErr
					}
					if err = SubPendingNewMarketFeeSum(db); err != nil {
						return nil, err
					}
					if err = DeleteMarketInfo(db, tradeTokenId, quoteTokenId); err != nil {
						return nil, err
					}
					if address, err := types.BytesToAddress(marketInfo.Creator); err != nil {
						return nil, err
					} else {
						refundBlocks[index] = &ledger.AccountBlock{
							AccountAddress: types.AddressDexFund,
							ToAddress:      address,
							BlockType:      ledger.BlockTypeSendCall,
							Amount:         NewMarketFeeAmount,
							TokenId:        ledger.ViteTokenId,
						}
					}
				}
			}
		}
		return refundBlocks, nil
	}
}

func PreCheckOrderParam(orderParam *ParamDexFundNewOrder) error {
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

func RenderOrder(orderInfo *dexproto.OrderInfo, param *ParamDexFundNewOrder, db vm_db.VmDb, address types.Address) *dexError {
	order := &dexproto.Order{}
	orderInfo.Order = order
	order.Address = address.Bytes()
	order.Side = param.Side
	order.Type = int32(param.OrderType)
	order.Price = PriceToBytes(param.Price)
	order.Quantity = param.Quantity.Bytes()
	var (
		marketInfo *MarketInfo
		err        error
	)
	if marketInfo, err = GetMarketInfo(db, param.TradeToken, param.QuoteToken); err != nil || !marketInfo.Valid {
		return TradeMarketNotExistsError
	}
	if order.Id, err = CompositeOrderId(db, marketInfo.MarketId, param); err != nil {
		return CompositeOrderIdFailErr
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
		if len(fund.Address) != types.AddressSize {
			return fmt.Errorf("invalid address format for settle")
		}
		if len(fund.FundSettles) == 0 {
			return fmt.Errorf("no user funds to settle")
		}
	}

	for _, fee := range actions.FeeActions {
		if len(fee.Token) != types.TokenTypeIdSize {
			return fmt.Errorf("invalid tokenId format for fee settle")
		}
		if len(fee.UserFeeSettles) == 0 {
			return fmt.Errorf("no userFees to settle")
		}
	}
	return nil
}

func DepositAccount(db vm_db.VmDb, address types.Address, tokenId types.TokenTypeId, amount *big.Int) (*dexproto.Account, error) {
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

func DoSettleFund(db vm_db.VmDb, reader util.ConsensusReader, action *dexproto.UserFundSettle) error {
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
					if err = OnSettleVx(db, reader, action.Address, fundSettle, account); err != nil {
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

func PledgeRequest(db vm_db.VmDb, address types.Address, pledgeType int8, amount *big.Int) ([]byte, error) {
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
			if pledgeData, err := abi.ABIPledge.PackMethod(abi.MethodNameAgentPledge, address, types.AddressDexFund, pledgeType); err != nil {
				return nil, err
			} else {
				return pledgeData, err
			}
		}
	}
}

func CancelPledgeRequest(db vm_db.VmDb, address types.Address, pledgeType int8, amount *big.Int) ([]byte, error) {
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
	if cancelPledgeData, err := abi.ABIPledge.PackMethod(abi.MethodNameAgentCancelPledge, address, types.AddressDexFund, amount, pledgeType); err != nil {
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
