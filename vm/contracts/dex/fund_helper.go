package dex

import (
	"bytes"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/vitelabs/go-vite/v2/common/types"
	"github.com/vitelabs/go-vite/v2/common/upgrade"
	"github.com/vitelabs/go-vite/v2/interfaces"
	ledger "github.com/vitelabs/go-vite/v2/interfaces/core"
	"github.com/vitelabs/go-vite/v2/vm/contracts/abi"
	cabi "github.com/vitelabs/go-vite/v2/vm/contracts/abi"
	dexproto "github.com/vitelabs/go-vite/v2/vm/contracts/dex/proto"
	"github.com/vitelabs/go-vite/v2/vm/util"
)

func CheckMarketParam(marketParam *ParamOpenNewMarket) (err error) {
	if marketParam.TradeToken == marketParam.QuoteToken {
		return TradeMarketInvalidTokenPairErr
	}
	return nil
}

func RenderMarketInfo(db interfaces.VmDb, marketInfo *MarketInfo, tradeToken, quoteToken types.TokenTypeId, tradeTokenInfo *TokenInfo, creator *types.Address) error {
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

func renderMarketInfoWithTradeTokenInfo(db interfaces.VmDb, marketInfo *MarketInfo, tradeTokenInfo *TokenInfo) {
	marketInfo.MarketSymbol = fmt.Sprintf("%s_%s", getDexTokenSymbol(tradeTokenInfo), marketInfo.MarketSymbol)
	marketInfo.TradeTokenDecimals = tradeTokenInfo.Decimals
	marketInfo.Valid = true
	marketInfo.Owner = tradeTokenInfo.Owner
}

func OnNewMarketValid(db interfaces.VmDb, reader util.ConsensusReader, marketInfo *MarketInfo, tradeToken, quoteToken types.TokenTypeId, address *types.Address) (blocks []*ledger.AccountBlock, err error) {
	if _, err = ReduceAccount(db, *address, ledger.ViteTokenId.Bytes(), NewMarketFeeAmount); err != nil {
		DeleteMarketInfo(db, tradeToken, quoteToken)
		AddErrEvent(db, err)
		//WARN: when not enough fund, take receive as true, but not notify dexTrade
		return nil, nil
	}
	userFee := &dexproto.FeeSettle{}
	userFee.Address = address.Bytes()
	userFee.BaseFee = NewMarketFeeMineAmount.Bytes()
	var donateAmount = new(big.Int).Set(NewMarketFeeDonateAmount)
	if IsEarthFork(db) {
		donateAmount.Add(donateAmount, NewMarketFeeBurnAmount)
	}
	SettleFeesWithTokenId(db, reader, true, ledger.ViteTokenId, ViteTokenDecimals, ViteTokenType, []*dexproto.FeeSettle{userFee}, donateAmount, nil)
	marketInfo.MarketId = NewAndSaveMarketId(db)
	SaveMarketInfo(db, marketInfo, tradeToken, quoteToken)
	AddMarketEvent(db, marketInfo)
	var marketBytes, syncData, burnData []byte
	if marketBytes, err = marketInfo.Serialize(); err != nil {
		panic(err)
	} else {
		var syncNewMarketMethod = cabi.MethodNameDexTradeSyncNewMarket
		if !IsLeafFork(db) {
			syncNewMarketMethod = cabi.MethodNameDexTradeNotifyNewMarket
		}
		if syncData, err = cabi.ABIDexTrade.PackMethod(syncNewMarketMethod, marketBytes); err != nil {
			panic(err)
		} else {
			blocks = append(blocks, &ledger.AccountBlock{
				AccountAddress: types.AddressDexFund,
				ToAddress:      types.AddressDexTrade,
				BlockType:      ledger.BlockTypeSendCall,
				TokenId:        ledger.ViteTokenId,
				Amount:         big.NewInt(0),
				Data:           syncData,
			})
		}
		if !IsEarthFork(db) { // burn on fee dividend, not on new market
			if burnData, err = cabi.ABIAsset.PackMethod(cabi.MethodNameBurn); err != nil {
				panic(err)
			} else {
				blocks = append(blocks, &ledger.AccountBlock{
					AccountAddress: types.AddressDexFund,
					ToAddress:      types.AddressAsset,
					BlockType:      ledger.BlockTypeSendCall,
					TokenId:        ledger.ViteTokenId,
					Amount:         NewMarketFeeBurnAmount,
					Data:           burnData,
				})
			}
		}
		return
	}
}

func OnNewMarketPending(db interfaces.VmDb, param *ParamOpenNewMarket, marketInfo *MarketInfo) (data []byte, err error) {
	SaveMarketInfo(db, marketInfo, param.TradeToken, param.QuoteToken)
	if err = AddToPendingNewMarkets(db, param.TradeToken, param.QuoteToken); err != nil {
		return
	}
	if data, err = abi.ABIAsset.PackMethod(abi.MethodNameGetTokenInfo, param.TradeToken, uint8(GetTokenForNewMarket)); err != nil {
		panic(err)
	} else {
		return
	}
}

func OnNewMarketGetTokenInfoSuccess(db interfaces.VmDb, reader util.ConsensusReader, tradeTokenId types.TokenTypeId, tokenInfoRes *ParamGetTokenInfoCallback) (appendBlocks []*ledger.AccountBlock, err error) {
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

func OnNewMarketGetTokenInfoFailed(db interfaces.VmDb, tradeTokenId types.TokenTypeId) (err error) {
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

func OnSetQuoteTokenPending(db interfaces.VmDb, token types.TokenTypeId, quoteTokenType uint8) []byte {
	AddToPendingSetQuotes(db, token, quoteTokenType)
	if data, err := abi.ABIAsset.PackMethod(abi.MethodNameGetTokenInfo, token, uint8(GetTokenForSetQuote)); err != nil {
		panic(err)
	} else {
		return data
	}
}

func OnSetQuoteGetTokenInfoSuccess(db interfaces.VmDb, tokenInfoRes *ParamGetTokenInfoCallback) error {
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

func OnSetQuoteGetTokenInfoFailed(db interfaces.VmDb, tokenId types.TokenTypeId) (err error) {
	_, err = FilterPendingSetQuotes(db, tokenId)
	return
}

func OnTransferTokenOwnerPending(db interfaces.VmDb, token types.TokenTypeId, origin, new types.Address) []byte {
	AddToPendingTransferTokenOwners(db, token, origin, new)
	if data, err := abi.ABIAsset.PackMethod(abi.MethodNameGetTokenInfo, token, uint8(GetTokenForTransferOwner)); err != nil {
		panic(err)
	} else {
		return data
	}
}

func OnTransferOwnerGetTokenInfoSuccess(db interfaces.VmDb, param *ParamGetTokenInfoCallback) error {
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

func OnTransferOwnerGetTokenInfoFailed(db interfaces.VmDb, tradeTokenId types.TokenTypeId) (err error) {
	_, err = FilterPendingTransferTokenOwners(db, tradeTokenId)
	return
}

func PreCheckOrderParam(orderParam *ParamPlaceOrder, isStemFork bool) error {
	if orderParam.Quantity.Sign() <= 0 {
		return InvalidOrderQuantityErr
	}
	if orderParam.OrderType < Limited || orderParam.OrderType > ImmediateOrCancel {
		return InvalidOrderTypeErr
	}
	if orderParam.OrderType == Limited || orderParam.OrderType == PostOnly || orderParam.OrderType == FillOrKill || orderParam.OrderType == ImmediateOrCancel {
		if !ValidPrice(orderParam.Price, isStemFork) {
			return InvalidOrderPriceErr
		}
	}
	return nil
}

func DoPlaceOrder(db interfaces.VmDb, param *ParamPlaceOrder, accountAddress, agent *types.Address, sendHash types.Hash) ([]*ledger.AccountBlock, error) {
	var (
		dexFund         *Fund
		tradeBlockData  []byte
		err             error
		orderInfoBytes  []byte
		marketInfo      *MarketInfo
		ok              bool
		enrichOrderFork bool
	)
	order := &Order{}
	if param.OrderType != Limited {
		enrichOrderFork = IsVersion11EnrichOrderFork(db)
		if !enrichOrderFork {
			return nil, InvalidOrderTypeErr
		}
	}
	if order.Type == Market {
		param.Price = "0"
	}
	if marketInfo, err = RenderOrder(order, param, db, accountAddress, agent, sendHash, enrichOrderFork); err != nil {
		return nil, err
	}
	if dexFund, ok = GetFund(db, *accountAddress); !ok {
		return nil, ExceedFundAvailableErr
	}
	if err = CheckAndLockFundForNewOrder(db, dexFund, order, marketInfo, enrichOrderFork); err != nil {
		return nil, err
	}
	SaveFund(db, *accountAddress, dexFund)
	if !order.Side && IsMarketOrder(order, enrichOrderFork) {
		RenderMarketBuyOrderAmountAndFee(order)
	}
	if orderInfoBytes, err = order.Serialize(); err != nil {
		panic(err)
	}
	var placeOrderMethod = cabi.MethodNameDexTradePlaceOrder
	if !IsLeafFork(db) {
		placeOrderMethod = cabi.MethodNameDexTradeNewOrder
	}
	if tradeBlockData, err = cabi.ABIDexTrade.PackMethod(placeOrderMethod, orderInfoBytes); err != nil {
		panic(err)
	}
	return []*ledger.AccountBlock{
		{
			AccountAddress: types.AddressDexFund,
			ToAddress:      types.AddressDexTrade,
			BlockType:      ledger.BlockTypeSendCall,
			TokenId:        ledger.ViteTokenId,
			Amount:         big.NewInt(0),
			Data:           tradeBlockData,
		},
	}, nil
}

func RenderOrder(order *Order, param *ParamPlaceOrder, db interfaces.VmDb, accountAddress, agent *types.Address, sendHash types.Hash, enrichOrderFork bool) (*MarketInfo, error) {
	var (
		marketInfo *MarketInfo
		ok         bool
	)
	if IsDexStopped(db) {
		return nil, DexStoppedErr
	}
	if marketInfo, ok = GetMarketInfo(db, param.TradeToken, param.QuoteToken); !ok || !marketInfo.Valid {
		return nil, TradeMarketNotExistsErr
	} else if marketInfo.Stopped {
		return nil, TradeMarketStoppedErr
	} else if agent != nil && !IsMarketGrantedToAgent(db, *accountAddress, *agent, marketInfo.MarketId) {
		return nil, TradeMarketNotGrantedErr
	}
	order.Id = ComposeOrderId(db, marketInfo.MarketId, param.Side, param.Price)
	order.MarketId = marketInfo.MarketId
	order.Address = accountAddress.Bytes()
	order.Side = param.Side
	order.Type = int32(param.OrderType)
	order.Price = PriceToBytes(param.Price)
	order.Quantity = param.Quantity.Bytes()
	RenderFeeRate(*accountAddress, order, marketInfo, db)
	if order.Type == Limited || IsAdvancedLimitOrder(order, enrichOrderFork) {
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
	if IsStemFork(db) {
		if agent != nil && !IsDexRobotFork(db) { // not fill agent anymore after dex robot fork
			order.Agent = agent.Bytes()
		}
		order.SendHash = sendHash.Bytes()
	}
	return marketInfo, nil
}

func isAmountTooSmall(db interfaces.VmDb, amount []byte, marketInfo *MarketInfo) bool {
	typeInfo := QuoteTokenTypeInfos[marketInfo.QuoteTokenType]
	tradeThreshold := GetTradeThreshold(db, marketInfo.QuoteTokenType)
	if typeInfo.Decimals == marketInfo.QuoteTokenDecimals {
		return new(big.Int).SetBytes(amount).Cmp(tradeThreshold) < 0
	} else {
		return AdjustAmountForDecimalsDiff(amount, marketInfo.QuoteTokenDecimals-typeInfo.Decimals).Cmp(tradeThreshold) < 0
	}
}

func RenderFeeRate(address types.Address, order *Order, marketInfo *MarketInfo, db interfaces.VmDb) {
	if IsDexStableMarketFork(db) && marketInfo.GetStableMarket() { //stable pair with zero fees
		return
	}
	var vipReduceFeeRate int32 = 0
	if _, ok := GetSuperVIPStaking(db, address); ok {
		vipReduceFeeRate = BaseFeeRate
	} else if _, ok := GetVIPStaking(db, address); ok {
		vipReduceFeeRate = VipReduceFeeRate
	}
	order.TakerFeeRate = BaseFeeRate - vipReduceFeeRate
	order.TakerOperatorFeeRate = marketInfo.TakerOperatorFeeRate
	order.MakerFeeRate = BaseFeeRate - vipReduceFeeRate
	order.MakerOperatorFeeRate = marketInfo.MakerOperatorFeeRate
	if _, err := GetInviterByInvitee(db, address); err == nil { // invited
		order.TakerFeeRate = order.TakerFeeRate * 9 / 10
		order.TakerOperatorFeeRate = order.TakerOperatorFeeRate * 9 / 10
		order.MakerFeeRate = order.MakerFeeRate * 9 / 10
		order.MakerOperatorFeeRate = order.MakerOperatorFeeRate * 9 / 10
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
		if len(fund.AccountSettles) == 0 {
			return fmt.Errorf("no user funds to settle")
		}
	}
	return nil
}

func CheckAndLockFundForNewOrder(db interfaces.VmDb, dexFund *Fund, order *Order, marketInfo *MarketInfo, enrichOrderFork bool) (err error) {
	var (
		lockToken, lockAmount []byte
		lockTokenId           types.TokenTypeId
		account               *dexproto.Account
		exists                bool
	)
	switch order.Side {
	case false: //buy
		lockToken = marketInfo.QuoteToken
		if order.Type == Limited || IsAdvancedLimitOrder(order, enrichOrderFork) {
			lockAmount = AddBigInt(order.Amount, order.LockedBuyFee)
		}
	case true: // sell
		lockToken = marketInfo.TradeToken
		lockAmount = order.Quantity
	}
	if lockTokenId, err = types.BytesToTokenTypeId(lockToken); err != nil {
		panic(err)
	}
	if account, exists = GetAccountByToken(dexFund, lockTokenId); !exists {
		return ExceedFundAvailableErr
	}
	available := new(big.Int).SetBytes(account.Available)
	var lockAmountToInc *big.Int
	if order.Side == true || order.Type == Limited || IsAdvancedLimitOrder(order, enrichOrderFork) {
		lockAmountToInc = new(big.Int).SetBytes(lockAmount)
	} else { //market buy order (order.Side == false && IsMarketOrder(order, enrichOrderFork))
		lockAmountToInc = new(big.Int).SetBytes(available.Bytes())
		if isAmountTooSmall(db, lockAmountToInc.Bytes(), marketInfo) {
			return OrderAmountTooSmallErr
		}
		order.Amount = lockAmountToInc.Bytes()
	}
	available = available.Sub(available, lockAmountToInc)
	if available.Sign() < 0 {
		return ExceedFundAvailableErr
	}
	if order.Type == Market {
		marketOrderAmtThreshold := GetMarketOrderAmtThreshold(db, marketInfo.QuoteTokenType)
		quoteTokenAdjustedAmt := AdjustAmountForDecimalsDiff(marketOrderAmtThreshold.Bytes(), QuoteTokenTypeInfos[marketInfo.QuoteTokenType].Decimals-marketInfo.QuoteTokenDecimals)
		order.MarketOrderAmtThreshold = quoteTokenAdjustedAmt.Bytes()
	}
	account.Available = available.Bytes()
	account.Locked = AddBigInt(account.Locked, lockAmountToInc.Bytes())
	return
}

//marketOrder lockByFee will be simply set to maxFeeRate, regardless sell or buy feeRate difference, this will simplify the matcher logic
func RenderMarketBuyOrderAmountAndFee(order *Order) { //divide buy amount to amount + feeAmount
	var feeRate = MaxTotalFeeRate(*order)
	if feeRate > 0 {
		adjustedFeeRate := int32(float32(feeRate)/(float32(RateCardinalNum)+float32(feeRate))*float32(RateCardinalNum)) + 1 // add 1 for ceil feeRate
		order.LockedBuyFee = CalculateAmountForRate(order.Amount, adjustedFeeRate)
		order.Amount = SubBigIntAbs(order.Amount, order.LockedBuyFee)
	}
}

func newTokenInfoFromCallback(db interfaces.VmDb, param *ParamGetTokenInfoCallback) *TokenInfo {
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

func CheckCancelAgentOrder(db interfaces.VmDb, sender types.Address, param *ParamCancelOrderByHash) (owner types.Address, err error) {
	if param.Principal != types.ZERO_ADDRESS && param.Principal != sender {
		owner = param.Principal
		if marketInfo, ok := GetMarketInfo(db, param.TradeToken, param.QuoteToken); !ok {
			return owner, TradeMarketNotExistsErr
		} else {
			if !IsMarketGrantedToAgent(db, param.Principal, sender, marketInfo.MarketId) {
				return owner, TradeMarketNotGrantedErr
			}
		}
	} else {
		owner = sender
	}
	return
}

func DoCancelOrder(sendHash types.Hash, owner types.Address) ([]*ledger.AccountBlock, error) {
	if tradeBlockData, err := cabi.ABIDexTrade.PackMethod(abi.MethodNameDexTradeInnerCancelOrderBySendHash, sendHash, owner); err != nil {
		panic(err)
	} else {
		return []*ledger.AccountBlock{
			{
				AccountAddress: types.AddressDexFund,
				ToAddress:      types.AddressDexTrade,
				BlockType:      ledger.BlockTypeSendCall,
				TokenId:        ledger.ViteTokenId,
				Amount:         big.NewInt(0),
				Data:           tradeBlockData,
			},
		}, nil
	}
}

func ValidPrice(price string, isFork bool) bool {
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
		} else if idx >= 0 && len(price)-(idx+1) > priceDecimalMaxLen { // max price precision is 12 decimals 10^12 < 2^40(5bytes)
			return false
		} else if idx == -1 && isFork && len(price) > priceIntMaxLen {
			return false
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
		param := new(ParamPlaceOrder)
		if err := abi.ABIDexFund.UnpackMethod(param, cabi.MethodNameDexFundNewOrder, data); err == nil {
			return ValidPrice(param.Price, true) || param.OrderType == Market && len(param.Price) <= 25
		} else {
			valid = false
		}
	}
	return
}

func MaxTotalFeeRate(order Order) int32 {
	takerRate := order.TakerFeeRate + order.TakerOperatorFeeRate
	makerRate := order.MakerFeeRate + order.MakerOperatorFeeRate
	if takerRate > makerRate {
		return takerRate
	} else {
		return makerRate
	}
}

func GenerateHeightPoint(db interfaces.VmDb) upgrade.HeightPoint {
	if latestSb, err := db.LatestSnapshotBlock(); err != nil {
		panic(err)
	} else {
		return upgrade.NewHeightPoint(latestSb.Height)
	}
}

func IsDexFeeFork(db interfaces.VmDb) bool {
	if latestSb, err := db.LatestSnapshotBlock(); err != nil {
		panic(err)
	} else {
		return upgrade.IsDexFeeUpgrade(latestSb.Height)
	}
}

func IsStemFork(db interfaces.VmDb) bool {
	if latestSb, err := db.LatestSnapshotBlock(); err != nil {
		panic(err)
	} else {
		return upgrade.IsStemUpgrade(latestSb.Height)
	}
}

func IsLeafFork(db interfaces.VmDb) bool {
	if latestSb, err := db.LatestSnapshotBlock(); err != nil {
		panic(err)
	} else {
		return upgrade.IsLeafUpgrade(latestSb.Height)
	}
}

func IsEarthFork(db interfaces.VmDb) bool {
	if latestSb, err := db.LatestSnapshotBlock(); err != nil {
		panic(err)
	} else {
		return upgrade.IsEarthUpgrade(latestSb.Height)
	}
}

func IsDexMiningFork(db interfaces.VmDb) bool {
	if latestSb, err := db.LatestSnapshotBlock(); err != nil {
		panic(err)
	} else {
		return upgrade.IsDexMiningUpgrade(latestSb.Height)
	}
}

func IsDexRobotFork(db interfaces.VmDb) bool {
	if latestSb, err := db.LatestSnapshotBlock(); err != nil {
		panic(err)
	} else {
		return upgrade.IsDexRobotUpgrade(latestSb.Height)
	}
}

func IsDexStableMarketFork(db interfaces.VmDb) bool {
	if latestSb, err := db.LatestSnapshotBlock(); err != nil {
		panic(err)
	} else {
		return upgrade.IsDexStableMarketUpgrade(latestSb.Height)
	}
}

func IsVersion10Upgrade(db interfaces.VmDb) bool {
	return util.CheckFork(db, upgrade.IsVersion10Upgrade)
}

func IsVersion11AddTransferAssetEvent(db interfaces.VmDb) bool {
	return util.CheckFork(db, upgrade.IsVersion11Upgrade)
}

func IsVersion11EnrichOrderFork(db interfaces.VmDb) bool {
	return util.CheckFork(db, upgrade.IsVersion11Upgrade)
}

func IsVersion11DeprecateClearingExpiredOrder(db interfaces.VmDb) bool {
	return util.CheckFork(db, upgrade.IsVersion11Upgrade)
}

func IsVersion12Upgrade(db interfaces.VmDb) bool {
	return util.CheckFork(db, upgrade.IsVersion12Upgrade)
}

func ValidOperatorFeeRate(feeRate int32) bool {
	return feeRate >= 0 && feeRate <= MaxOperatorFeeRate
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
		for len(indexStr) < 3 {
			indexStr = "0" + indexStr
		}
		return fmt.Sprintf("%s-%s", tokenInfo.Symbol, indexStr)
	}
}

func IsAdvancedLimitOrder(order *Order, enrichOrderFork bool) bool {
	return enrichOrderFork && (order.Type == PostOnly || order.Type == FillOrKill || order.Type == ImmediateOrCancel)
}

func IsMarketOrder(order *Order, enrichOrderFork bool) bool {
	return enrichOrderFork && order.Type == Market
}

type AmountWithToken struct {
	Token   types.TokenTypeId
	Amount  *big.Int
	Deleted bool
}
