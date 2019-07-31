package dex

import (
	"bytes"
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/contracts/abi"
	cabi "github.com/vitelabs/go-vite/vm/contracts/abi"
	dexproto "github.com/vitelabs/go-vite/vm/contracts/dex/proto"
	"github.com/vitelabs/go-vite/vm/util"
	"github.com/vitelabs/go-vite/vm_db"
	"math/big"
	"strconv"
	"strings"
)

func CheckMarketParam(marketParam *ParamDexFundNewMarket) (err error) {
	if marketParam.TradeToken == marketParam.QuoteToken {
		return TradeMarketInvalidTokenPairErr
	}
	return nil
}

func RenderMarketInfo(db vm_db.VmDb, marketInfo *MarketInfo, tradeToken, quoteToken types.TokenTypeId, tradeTokenInfo *TokenInfo, creator *types.Address) error {
	quoteTokenInfo, ok := GetTokenInfo(db, quoteToken)
	if !ok || quoteTokenInfo.QuoteTokenType <= 0 {
		return TradeMarketInvalidQuoteTokenErr
	}
	marketInfo.MarketSymbol = getDexTokenSymbol(quoteTokenInfo)
	marketInfo.TradeToken = tradeToken.Bytes()
	marketInfo.QuoteToken = quoteToken.Bytes()
	marketInfo.QuoteTokenType = quoteTokenInfo.QuoteTokenType
	marketInfo.QuoteTokenDecimals = quoteTokenInfo.Decimals

	var tradeTokenExists bool
	if tradeTokenInfo == nil {
		tradeTokenInfo, tradeTokenExists = GetTokenInfo(db, tradeToken)
	} else {
		tradeTokenExists = true
	}
	if tradeTokenExists {
		if tradeTokenInfo.QuoteTokenType > quoteTokenInfo.QuoteTokenType {
			return TradeMarketInvalidTokenPairErr
		}
		if !bytes.Equal(tradeTokenInfo.Owner, creator.Bytes()) {
			return OnlyOwnerAllowErr
		}
		renderMarketInfoWithTradeTokenInfo(db, marketInfo, tradeTokenInfo)
	}

	if marketInfo.Creator == nil {
		marketInfo.Creator = creator.Bytes()
	}
	if marketInfo.Timestamp == 0 {
		marketInfo.Timestamp = GetTimestampInt64(db)
	}
	return nil
}

func renderMarketInfoWithTradeTokenInfo(db vm_db.VmDb, marketInfo *MarketInfo, tradeTokenInfo *TokenInfo) {
	marketInfo.MarketSymbol = fmt.Sprintf("%s_%s", getDexTokenSymbol(tradeTokenInfo), marketInfo.MarketSymbol)
	marketInfo.TradeTokenDecimals = tradeTokenInfo.Decimals
	marketInfo.Valid = true
	marketInfo.Owner = tradeTokenInfo.Owner
}

func OnNewMarketValid(db vm_db.VmDb, reader util.ConsensusReader, marketInfo *MarketInfo, tradeToken, quoteToken types.TokenTypeId, address *types.Address) (block []*ledger.AccountBlock, err error) {
	if _, err = SubUserFund(db, *address, ledger.ViteTokenId.Bytes(), NewMarketFeeAmount); err != nil {
		DeleteMarketInfo(db, tradeToken, quoteToken)
		AddErrEvent(db, err)
		//WARN: when not enough fund, take receive as true, but not notify dexTrade
		return nil, nil
	}
	userFee := &dexproto.UserFeeSettle{}
	userFee.Address = address.Bytes()
	userFee.BaseFee = NewMarketFeeMineAmount.Bytes()
	SettleFeesWithTokenId(db, reader, true, ledger.ViteTokenId, ViteTokenDecimals, ViteTokenType, []*dexproto.UserFeeSettle{userFee}, NewMarketFeeDonateAmount, nil)
	marketInfo.MarketId = NewAndSaveMarketId(db)
	SaveMarketInfo(db, marketInfo, tradeToken, quoteToken)
	AddMarketEvent(db, marketInfo)
	var marketBytes, syncData, burnData []byte
	if marketBytes, err = marketInfo.Serialize(); err != nil {
		panic(err)
	} else {
		var syncNewMarketBlock, newMarketFeeBurnBlock *ledger.AccountBlock
		if syncData, err = cabi.ABIDexTrade.PackMethod(cabi.MethodNameDexTradeNotifyNewMarket, marketBytes); err != nil {
			panic(err)
		} else {
			syncNewMarketBlock = &ledger.AccountBlock{
				AccountAddress: types.AddressDexFund,
				ToAddress:      types.AddressDexTrade,
				BlockType:      ledger.BlockTypeSendCall,
				TokenId:        ledger.ViteTokenId,
				Amount:         big.NewInt(0),
				Data:           syncData,
			}
		}
		if burnData, err = cabi.ABIMintage.PackMethod(cabi.MethodNameBurn); err != nil {
			panic(err)
		} else {
			newMarketFeeBurnBlock = &ledger.AccountBlock{
				AccountAddress: types.AddressDexFund,
				ToAddress:      types.AddressMintage,
				BlockType:      ledger.BlockTypeSendCall,
				TokenId:        ledger.ViteTokenId,
				Amount:         NewMarketFeeBurnAmount,
				Data:           burnData,
			}
		}
		return []*ledger.AccountBlock{syncNewMarketBlock, newMarketFeeBurnBlock}, nil
	}
}

func OnNewMarketPending(db vm_db.VmDb, param *ParamDexFundNewMarket, marketInfo *MarketInfo) []byte {
	SaveMarketInfo(db, marketInfo, param.TradeToken, param.QuoteToken)
	AddToPendingNewMarkets(db, param.TradeToken, param.QuoteToken)
	if data, err := abi.ABIMintage.PackMethod(abi.MethodNameGetTokenInfo, param.TradeToken, uint8(GetTokenForNewMarket)); err != nil {
		panic(err)
	} else {
		return data
	}
}

func OnNewMarketGetTokenInfoSuccess(db vm_db.VmDb, reader util.ConsensusReader, tradeTokenId types.TokenTypeId, tokenInfoRes *ParamDexFundGetTokenInfoCallback) (appendBlocks []*ledger.AccountBlock, err error) {
	tradeTokenInfo := newTokenInfoFromCallback(db, tokenInfoRes)
	SaveTokenInfo(db, tradeTokenId, tradeTokenInfo)
	AddTokenEvent(db, tradeTokenInfo)
	var quoteTokens [][]byte
	if quoteTokens, err = FilterPendingNewMarkets(db, tradeTokenId); err != nil {
		return
	} else {
		for _, qt := range quoteTokens {
			var (
				quoteTokenId types.TokenTypeId
				marketInfo   *MarketInfo
				ok           bool
			)
			if quoteTokenId, err = types.BytesToTokenTypeId(qt); err != nil {
				panic(err)
			} else {
				if marketInfo, ok = GetMarketInfo(db, tradeTokenId, quoteTokenId); ok && !marketInfo.Valid {
					var creator types.Address
					if creator, err = types.BytesToAddress(marketInfo.Creator); err != nil {
						panic(err)
					}
					if bizErr := RenderMarketInfo(db, marketInfo, tradeTokenId, quoteTokenId, tradeTokenInfo, &creator); bizErr != nil {
						DeleteMarketInfo(db, tradeTokenId, quoteTokenId)
						AddErrEvent(db, bizErr)
					} else {
						var blocks []*ledger.AccountBlock
						if blocks, err = OnNewMarketValid(db, reader, marketInfo, tradeTokenId, quoteTokenId, &creator); err == nil {
							if len(blocks) > 0 {
								appendBlocks = append(appendBlocks, blocks...)
							}
						} else {
							return
						}
					}
				}
			}
		}
		return
	}
}

func OnNewMarketGetTokenInfoFailed(db vm_db.VmDb, tradeTokenId types.TokenTypeId) (err error) {
	var quoteTokens [][]byte
	if quoteTokens, err = FilterPendingNewMarkets(db, tradeTokenId); err != nil {
		return
	} else {
		for _, qt := range quoteTokens {
			if quoteTokenId, err := types.BytesToTokenTypeId(qt); err != nil {
				panic(err)
			} else {
				if marketInfo, ok := GetMarketInfo(db, tradeTokenId, quoteTokenId); ok && !marketInfo.Valid {
					DeleteMarketInfo(db, tradeTokenId, quoteTokenId)
				}
			}
		}
		return
	}
}

func OnSetQuoteTokenPending(db vm_db.VmDb, token types.TokenTypeId, quoteTokenType uint8) []byte {
	AddToPendingSetQuotes(db, token, quoteTokenType)
	if data, err := abi.ABIMintage.PackMethod(abi.MethodNameGetTokenInfo, token, uint8(GetTokenForSetQuote)); err != nil {
		panic(err)
	} else {
		return data
	}
}

func OnSetQuoteGetTokenInfoSuccess(db vm_db.VmDb, tokenInfoRes *ParamDexFundGetTokenInfoCallback) error {
	if action, err := FilterPendingSetQuotes(db, tokenInfoRes.TokenId); err != nil {
		return err
	} else {
		tokenInfo := newTokenInfoFromCallback(db, tokenInfoRes)
		tokenInfo.QuoteTokenType = int32(action.QuoteTokenType)
		SaveTokenInfo(db, tokenInfoRes.TokenId, tokenInfo)
		AddTokenEvent(db, tokenInfo)
		return nil
	}
}

func OnSetQuoteGetTokenInfoFailed(db vm_db.VmDb, tokenId types.TokenTypeId) (err error) {
	_, err = FilterPendingSetQuotes(db, tokenId)
	return
}

func OnTransferTokenOwnerPending(db vm_db.VmDb, token types.TokenTypeId, origin, new types.Address) []byte {
	AddToPendingTransferTokenOwners(db, token, origin, new)
	if data, err := abi.ABIMintage.PackMethod(abi.MethodNameGetTokenInfo, token, uint8(GetTokenForTransferOwner)); err != nil {
		panic(err)
	} else {
		return data
	}
}

func OnTransferOwnerGetTokenInfoSuccess(db vm_db.VmDb, param *ParamDexFundGetTokenInfoCallback) error {
	if action, err := FilterPendingTransferTokenOwners(db, param.TokenId); err != nil {
		return err
	} else {
		if bytes.Equal(action.Origin, param.Owner.Bytes()) ||
			param.TokenId == ledger.ViteTokenId && bytes.Equal(action.Origin, initViteTokenOwner.Bytes()) && len(getValueFromDb(db, viteOwnerInitiated)) == 0 {
			tokenInfo := newTokenInfoFromCallback(db, param)
			tokenInfo.Owner = action.New
			SaveTokenInfo(db, param.TokenId, tokenInfo)
			AddTokenEvent(db, tokenInfo)
			return nil
		} else {
			return OnlyOwnerAllowErr
		}
	}
}

func OnTransferOwnerGetTokenInfoFailed(db vm_db.VmDb, tradeTokenId types.TokenTypeId) (err error) {
	_, err = FilterPendingTransferTokenOwners(db, tradeTokenId)
	return
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

func RenderOrder(order *Order, param *ParamDexFundNewOrder, db vm_db.VmDb, address types.Address) (*MarketInfo, error) {
	var (
		marketInfo *MarketInfo
		ok         bool
	)
	if IsViteXStopped(db) {
		return nil, ViteXStoppedErr
	}
	if marketInfo, ok = GetMarketInfo(db, param.TradeToken, param.QuoteToken); !ok || !marketInfo.Valid {
		return nil, TradeMarketNotExistsErr
	} else if marketInfo.Stopped {
		return nil, TradeMarketStoppedErr
	}
	order.Id = ComposeOrderId(db, marketInfo.MarketId, param.Side, param.Price)
	order.MarketId = marketInfo.MarketId
	order.Address = address.Bytes()
	order.Side = param.Side
	order.Type = int32(param.OrderType)
	order.Price = PriceToBytes(param.Price)
	order.Quantity = param.Quantity.Bytes()
	RenderFeeRate(address, order, marketInfo, db)
	if order.Type == Limited {
		order.Amount = CalculateRawAmount(order.Quantity, order.Price, marketInfo.TradeTokenDecimals-marketInfo.QuoteTokenDecimals)
		if !order.Side { //buy
			order.LockedBuyFee = CalculateAmountForRate(order.Amount, MaxTotalFeeRate(*order))
		}
		totalAmount := AddBigInt(order.Amount, order.LockedBuyFee)
		if isAmountTooSmall(db, totalAmount, marketInfo) {
			return marketInfo, OrderAmountTooSmallErr
		}
	}
	order.Status = Pending
	order.ExecutedQuantity = big.NewInt(0).Bytes()
	order.ExecutedAmount = big.NewInt(0).Bytes()
	order.RefundToken = []byte{}
	order.RefundQuantity = big.NewInt(0).Bytes()
	order.Timestamp = GetTimestampInt64(db)
	return marketInfo, nil
}

func isAmountTooSmall(db vm_db.VmDb, amount []byte, marketInfo *MarketInfo) bool {
	typeInfo, _ := QuoteTokenTypeInfos[marketInfo.QuoteTokenType]
	tradeThreshold := GetTradeThreshold(db, marketInfo.QuoteTokenType)
	if typeInfo.Decimals == marketInfo.QuoteTokenDecimals {
		return new(big.Int).SetBytes(amount).Cmp(tradeThreshold) < 0
	} else {
		return AdjustAmountForDecimalsDiff(amount, marketInfo.QuoteTokenDecimals-typeInfo.Decimals).Cmp(tradeThreshold) < 0
	}
}

func RenderFeeRate(address types.Address, order *Order, marketInfo *MarketInfo, db vm_db.VmDb) {
	var vipReduceFeeRate int32 = 0
	if _, ok := GetPledgeForVip(db, address); ok {
		vipReduceFeeRate = VipReduceFeeRate
	}
	order.TakerFeeRate = BaseFeeRate - vipReduceFeeRate
	order.TakerBrokerFeeRate = marketInfo.TakerBrokerFeeRate
	order.MakerFeeRate = BaseFeeRate - vipReduceFeeRate
	order.MakerBrokerFeeRate = marketInfo.MakerBrokerFeeRate
	if _, err := GetInviterByInvitee(db, address); err == nil { // invited
		order.TakerFeeRate = order.TakerFeeRate * 9 / 10
		order.TakerBrokerFeeRate = order.TakerBrokerFeeRate * 9 / 10
		order.MakerFeeRate = order.MakerFeeRate * 9 / 10
		order.MakerBrokerFeeRate = order.MakerBrokerFeeRate * 9 / 10
	}
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
	return nil
}

func CheckAndLockFundForNewOrder(dexFund *UserFund, order *Order, marketInfo *MarketInfo) (err error) {
	var (
		lockToken, lockAmount []byte
		lockTokenId           types.TokenTypeId
		account               *dexproto.Account
		exists                bool
	)
	switch order.Side {
	case false: //buy
		lockToken = marketInfo.QuoteToken
		if order.Type == Limited {
			lockAmount = AddBigInt(order.Amount, order.LockedBuyFee)
		}
	case true: // sell
		lockToken = marketInfo.TradeToken
		lockAmount = order.Quantity
	}
	if lockTokenId, err = types.BytesToTokenTypeId(lockToken); err != nil {
		panic(err)
	}
	if account, exists = GetAccountByTokeIdFromFund(dexFund, lockTokenId); !exists {
		return ExceedFundAvailableErr
	}
	available := new(big.Int).SetBytes(account.Available)
	lockAmountToInc := new(big.Int).SetBytes(lockAmount)
	available = available.Sub(available, lockAmountToInc)
	if available.Sign() < 0 {
		return ExceedFundAvailableErr
	}
	account.Available = available.Bytes()
	account.Locked = AddBigInt(account.Locked, lockAmountToInc.Bytes())
	return
}

func newTokenInfoFromCallback(db vm_db.VmDb, param *ParamDexFundGetTokenInfoCallback) *TokenInfo {
	tokenInfo := &TokenInfo{}
	tokenInfo.TokenId = param.TokenId.Bytes()
	tokenInfo.Decimals = int32(param.Decimals)
	tokenInfo.Symbol = param.TokenSymbol
	tokenInfo.Index = int32(param.Index)
	if param.TokenId == ledger.ViteTokenId {
		tokenInfo.Owner = initViteTokenOwner.Bytes()
		setValueToDb(db, viteOwnerInitiated, []byte{1})
	} else {
		tokenInfo.Owner = param.Owner.Bytes()
	}
	return tokenInfo
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
			if len(price)-(idx+1) > priceDecimalMaxLen { // // max price precision is 12 decimals 10^12 < 2^40(5bytes)
				return false
			}
		}
	}
	return true
}

func VerifyNewOrderPriceForRpc(data []byte) (valid bool) {
	valid = true
	if len(data) < 4 {
		return
	}
	if bytes.Equal(data[:4], newOrderMethodId) {
		param := new(ParamDexFundNewOrder)
		if err := abi.ABIDexFund.UnpackMethod(param, cabi.MethodNameDexFundNewOrder, data); err == nil {
			if !ValidPrice(param.Price) {
				valid = false
			} else if idx := strings.Index(param.Price, "."); idx == -1 && len(param.Price) > priceIntMaxLen {
				valid = false
			}
		} else {
			valid = false
		}
	}
	return
}

func MaxTotalFeeRate(order Order) int32 {
	takerRate := order.TakerFeeRate + order.TakerBrokerFeeRate
	makerRate := order.MakerFeeRate + order.MakerBrokerFeeRate
	if takerRate > makerRate {
		return takerRate
	} else {
		return makerRate
	}
}

// only for unit test
func SetFeeRate(baseRate int32) {
	BaseFeeRate = baseRate
}

func ValidBrokerFeeRate(feeRate int32) bool {
	return feeRate >= 0 && feeRate <= MaxBrokerFeeRate
}

func checkPriceChar(price string) bool {
	for _, v := range price {
		if v != '.' && (uint8(v) < '0' || uint8(v) > '9') {
			return false
		}
	}
	return true
}

func getDexTokenSymbol(tokenInfo *TokenInfo) string {
	if tokenInfo.Symbol == "VITE" || tokenInfo.Symbol == "VCP" || tokenInfo.Symbol == "VX" {
		return tokenInfo.Symbol
	} else {
		indexStr := strconv.Itoa(int(tokenInfo.Index))
		for ; len(indexStr) < 3; {
			indexStr = "0" + indexStr
		}
		return fmt.Sprintf("%s-%s", tokenInfo.Symbol, indexStr)
	}
}

type AmountWithToken struct {
	Token   types.TokenTypeId
	Amount  *big.Int
	Deleted bool
}
