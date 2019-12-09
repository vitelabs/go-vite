package dex

import (
	"bytes"
	"fmt"
	"github.com/vitelabs/go-vite/common/fork"
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

func CheckMarketParam(marketParam *ParamOpenNewMarket) (err error) {
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

func OnNewMarketValid(db vm_db.VmDb, reader util.ConsensusReader, marketInfo *MarketInfo, tradeToken, quoteToken types.TokenTypeId, address *types.Address) (blocks []*ledger.AccountBlock, err error) {
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

func OnNewMarketPending(db vm_db.VmDb, param *ParamOpenNewMarket, marketInfo *MarketInfo) (data []byte, err error) {
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

func OnNewMarketGetTokenInfoSuccess(db vm_db.VmDb, reader util.ConsensusReader, tradeTokenId types.TokenTypeId, tokenInfoRes *ParamGetTokenInfoCallback) (appendBlocks []*ledger.AccountBlock, err error) {
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
	if data, err := abi.ABIAsset.PackMethod(abi.MethodNameGetTokenInfo, token, uint8(GetTokenForSetQuote)); err != nil {
		panic(err)
	} else {
		return data
	}
}

func OnSetQuoteGetTokenInfoSuccess(db vm_db.VmDb, tokenInfoRes *ParamGetTokenInfoCallback) error {
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
	if data, err := abi.ABIAsset.PackMethod(abi.MethodNameGetTokenInfo, token, uint8(GetTokenForTransferOwner)); err != nil {
		panic(err)
	} else {
		return data
	}
}

func OnTransferOwnerGetTokenInfoSuccess(db vm_db.VmDb, param *ParamGetTokenInfoCallback) error {
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

func PreCheckOrderParam(orderParam *ParamPlaceOrder, isStemFork bool) error {
	if orderParam.Quantity.Sign() <= 0 {
		return InvalidOrderQuantityErr
	}
	// TODO add market order support
	if orderParam.OrderType != Limited {
		return InvalidOrderTypeErr
	}
	if orderParam.OrderType == Limited {
		if !ValidPrice(orderParam.Price, isStemFork) {
			return InvalidOrderPriceErr
		}
	}
	return nil
}

func DoPlaceOrder(db vm_db.VmDb, param *ParamPlaceOrder, accountAddress, agent *types.Address, sendHash types.Hash) ([]*ledger.AccountBlock, error) {
	var (
		dexFund        *Fund
		tradeBlockData []byte
		err            error
		orderInfoBytes []byte
		marketInfo     *MarketInfo
		ok             bool
	)
	order := &Order{}
	if marketInfo, err = RenderOrder(order, param, db, accountAddress, agent, sendHash); err != nil {
		return nil, err
	}
	if dexFund, ok = GetFund(db, *accountAddress); !ok {
		return nil, ExceedFundAvailableErr
	}
	if err = CheckAndLockFundForNewOrder(dexFund, order, marketInfo); err != nil {
		return nil, err
	}
	SaveFund(db, *accountAddress, dexFund)
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

func RenderOrder(order *Order, param *ParamPlaceOrder, db vm_db.VmDb, accountAddress, agent *types.Address, sendHash types.Hash) (*MarketInfo, error) {
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
	if IsStemFork(db) {
		if agent != nil {
			order.Agent = agent.Bytes()
		}
		order.SendHash = sendHash.Bytes()
	}
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

func CheckAndLockFundForNewOrder(dexFund *Fund, order *Order, marketInfo *MarketInfo) (err error) {
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
	if account, exists = GetAccountByToken(dexFund, lockTokenId); !exists {
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

func newTokenInfoFromCallback(db vm_db.VmDb, param *ParamGetTokenInfoCallback) *TokenInfo {
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
			return ValidPrice(param.Price, true)
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

func IsDexFeeFork(db vm_db.VmDb) bool {
	if latestSb, err := db.LatestSnapshotBlock(); err != nil {
		panic(err)
	} else {
		return fork.IsDexFeeFork(latestSb.Height)
	}
}

func IsStemFork(db vm_db.VmDb) bool {
	if latestSb, err := db.LatestSnapshotBlock(); err != nil {
		panic(err)
	} else {
		return fork.IsStemFork(latestSb.Height)
	}
}

func IsLeafFork(db vm_db.VmDb) bool {
	if latestSb, err := db.LatestSnapshotBlock(); err != nil {
		panic(err)
	} else {
		return fork.IsLeafFork(latestSb.Height)
	}
}

func IsEarthFork(db vm_db.VmDb) bool {
	if latestSb, err := db.LatestSnapshotBlock(); err != nil {
		panic(err)
	} else {
		return fork.IsEarthFork(latestSb.Height)
	}
}

func IsDexMiningFork(db vm_db.VmDb) bool {
	if latestSb, err := db.LatestSnapshotBlock(); err != nil {
		panic(err)
	} else {
		return fork.IsDexMiningFork(latestSb.Height)
	}
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

type AmountWithToken struct {
	Token   types.TokenTypeId
	Amount  *big.Int
	Deleted bool
}
