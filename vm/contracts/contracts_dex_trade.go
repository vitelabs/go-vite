package contracts

import (
	"bytes"
	"github.com/golang/protobuf/proto"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	cabi "github.com/vitelabs/go-vite/vm/contracts/abi"
	"github.com/vitelabs/go-vite/vm/contracts/dex"
	dexproto "github.com/vitelabs/go-vite/vm/contracts/dex/proto"
	"github.com/vitelabs/go-vite/vm/util"
	"github.com/vitelabs/go-vite/vm_db"
	"math/big"
	"sort"
)

var tradeLogger = log15.New("module", "dex_trade")

type MethodDexTradePlaceOrder struct {
	MethodName string
}

func (md *MethodDexTradePlaceOrder) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexTradePlaceOrder) GetRefundData(sendBlock *ledger.AccountBlock, sbHeight uint64) ([]byte, bool) {
	return []byte{}, false
}

func (md *MethodDexTradePlaceOrder) GetSendQuota(data []byte, gasTable *util.QuotaTable) (uint64, error) {
	return util.RequestQuotaCost(data, gasTable)
}

func (md *MethodDexTradePlaceOrder) GetReceiveQuota(gasTable *util.QuotaTable) uint64 {
	return 0
}

func (md *MethodDexTradePlaceOrder) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) (err error) {
	if !bytes.Equal(block.AccountAddress.Bytes(), types.AddressDexFund.Bytes()) {
		return dex.InvalidSourceAddressErr
	}
	err = cabi.ABIDexTrade.UnpackMethod(new(dex.ParamSerializedData), md.MethodName, block.Data)
	return
}

func (md *MethodDexTradePlaceOrder) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	var (
		err     error
		blocks  []*ledger.AccountBlock
		matcher *dex.Matcher
	)
	param := new(dex.ParamSerializedData)
	cabi.ABIDexTrade.UnpackMethod(param, md.MethodName, sendBlock.Data)
	order := &dex.Order{}
	if err = order.DeSerialize(param.Data); err != nil {
		panic(err)
	}
	if matcher, err = dex.NewMatcher(db, order.MarketId); err != nil {
		return handleDexReceiveErr(tradeLogger, md.MethodName, err, sendBlock)
	}
	if err = matcher.MatchOrder(order, block.PrevHash); err != nil {
		return OnPlaceOrderFailed(db, order, matcher.MarketInfo)
	}
	if blocks, err = handleSettleActions(db, block, matcher.GetFundSettles(), matcher.GetFees(), matcher.MarketInfo); err != nil {
		return OnPlaceOrderFailed(db, order, matcher.MarketInfo)
	}
	return blocks, err
}

type MethodDexTradeCancelOrder struct {
	MethodName string
}

func (md *MethodDexTradeCancelOrder) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexTradeCancelOrder) GetRefundData(sendBlock *ledger.AccountBlock, sbHeight uint64) ([]byte, bool) {
	return []byte{}, false
}

func (md *MethodDexTradeCancelOrder) GetSendQuota(data []byte, gasTable *util.QuotaTable) (uint64, error) {
	return util.RequestQuotaCost(data, gasTable)
}

func (md *MethodDexTradeCancelOrder) GetReceiveQuota(gasTable *util.QuotaTable) uint64 {
	return 0
}

func (md *MethodDexTradeCancelOrder) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) (err error) {
	err = cabi.ABIDexTrade.UnpackMethod(new(dex.ParamDexCancelOrder), md.MethodName, block.Data)
	return
}

func (md MethodDexTradeCancelOrder) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	param := new(dex.ParamDexCancelOrder)
	cabi.ABIDexTrade.UnpackMethod(param, md.MethodName, sendBlock.Data)
	return handleCancelOrderById(db, param.OrderId, md.MethodName, block, sendBlock)
}

type MethodDexTradeSyncNewMarket struct {
	MethodName string
}

func (md *MethodDexTradeSyncNewMarket) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexTradeSyncNewMarket) GetRefundData(sendBlock *ledger.AccountBlock, sbHeight uint64) ([]byte, bool) {
	return []byte{}, false
}

func (md *MethodDexTradeSyncNewMarket) GetSendQuota(data []byte, gasTable *util.QuotaTable) (uint64, error) {
	return util.RequestQuotaCost(data, gasTable)
}

func (md *MethodDexTradeSyncNewMarket) GetReceiveQuota(gasTable *util.QuotaTable) uint64 {
	return 0
}

func (md *MethodDexTradeSyncNewMarket) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) (err error) {
	if !bytes.Equal(block.AccountAddress.Bytes(), types.AddressDexFund.Bytes()) {
		return dex.InvalidSourceAddressErr
	}
	err = cabi.ABIDexTrade.UnpackMethod(new(dex.ParamSerializedData), md.MethodName, block.Data)
	return
}

func (md MethodDexTradeSyncNewMarket) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	param := new(dex.ParamSerializedData)
	cabi.ABIDexTrade.UnpackMethod(param, md.MethodName, sendBlock.Data)
	marketInfo := &dex.MarketInfo{}
	if err := marketInfo.DeSerialize(param.Data); err != nil {
		return handleDexReceiveErr(tradeLogger, md.MethodName, err, sendBlock)
	} else {
		dex.SaveMarketInfoById(db, marketInfo)
		return nil, nil
	}
}

type MethodDexTradeClearExpiredOrders struct {
	MethodName string
}

func (md *MethodDexTradeClearExpiredOrders) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexTradeClearExpiredOrders) GetRefundData(sendBlock *ledger.AccountBlock, sbHeight uint64) ([]byte, bool) {
	return []byte{}, false
}

func (md *MethodDexTradeClearExpiredOrders) GetSendQuota(data []byte, gasTable *util.QuotaTable) (uint64, error) {
	return util.RequestQuotaCost(data, gasTable)
}

func (md *MethodDexTradeClearExpiredOrders) GetReceiveQuota(gasTable *util.QuotaTable) uint64 {
	return 0
}

func (md *MethodDexTradeClearExpiredOrders) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) (err error) {
	err = cabi.ABIDexTrade.UnpackMethod(new(dex.ParamSerializedData), md.MethodName, block.Data)
	return
}

func (md MethodDexTradeClearExpiredOrders) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	param := new(dex.ParamSerializedData)
	cabi.ABIDexTrade.UnpackMethod(param, md.MethodName, sendBlock.Data)
	if len(param.Data) == 0 || len(param.Data)%dex.OrderIdBytesLength != 0 || len(param.Data)/dex.OrderIdBytesLength > dex.CleanExpireOrdersMaxCount {
		return handleDexReceiveErr(tradeLogger, md.MethodName, dex.InvalidInputParamErr, sendBlock)
	}
	if fundSettles, markerInfo, err := dex.CleanExpireOrders(db, param.Data); err != nil {
		return handleDexReceiveErr(tradeLogger, md.MethodName, err, sendBlock)
	} else if len(fundSettles) > 0 {
		if appendBlocks, err := handleSettleActions(db, block, fundSettles, nil, markerInfo); err != nil {
			return handleDexReceiveErr(tradeLogger, md.MethodName, err, sendBlock)
		} else {
			return appendBlocks, nil
		}
	} else {
		return nil, nil
	}
}

type MethodDexTradeCancelOrderByTransactionHash struct {
	MethodName string
}

func (md *MethodDexTradeCancelOrderByTransactionHash) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexTradeCancelOrderByTransactionHash) GetRefundData(sendBlock *ledger.AccountBlock, sbHeight uint64) ([]byte, bool) {
	return []byte{}, false
}

func (md *MethodDexTradeCancelOrderByTransactionHash) GetSendQuota(data []byte, gasTable *util.QuotaTable) (uint64, error) {
	return util.RequestQuotaCost(data, gasTable)
}

func (md *MethodDexTradeCancelOrderByTransactionHash) GetReceiveQuota(gasTable *util.QuotaTable) uint64 {
	return 0
}

func (md *MethodDexTradeCancelOrderByTransactionHash) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) (err error) {
	err = cabi.ABIDexTrade.UnpackMethod(new(types.Hash), md.MethodName, block.Data)
	return
}

func (md MethodDexTradeCancelOrderByTransactionHash) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	sendHash := new(types.Hash)
	cabi.ABIDexTrade.UnpackMethod(sendHash, md.MethodName, sendBlock.Data)
	if orderId, ok := dex.GetOrderIdByHash(db, sendHash.Bytes()); !ok {
		return handleDexReceiveErr(tradeLogger, md.MethodName, dex.InvalidOrderHashErr, sendBlock)
	} else {
		return handleCancelOrderById(db, orderId, md.MethodName, block, sendBlock)
	}
}

func OnPlaceOrderFailed(db vm_db.VmDb, order *dex.Order, marketInfo *dex.MarketInfo) ([]*ledger.AccountBlock, error) {
	accountSettle := &dexproto.AccountSettle{}
	switch order.Side {
	case false: // buy
		accountSettle.IsTradeToken = false
		accountSettle.ReleaseLocked = dex.AddBigInt(order.Amount, order.LockedBuyFee)
	case true: // sell
		accountSettle.IsTradeToken = true
		accountSettle.ReleaseLocked = order.Quantity
	}
	fundSettle := &dexproto.FundSettle{}
	fundSettle.Address = order.Address
	fundSettle.AccountSettles = append(fundSettle.AccountSettles, accountSettle)
	settleActions := &dexproto.SettleActions{}
	settleActions.FundActions = append(settleActions.FundActions, fundSettle)
	settleActions.TradeToken = marketInfo.TradeToken
	settleActions.QuoteToken = marketInfo.QuoteToken
	var (
		settleData, dexSettleBlockData []byte
		newErr                         error
	)
	if settleData, newErr = proto.Marshal(settleActions); newErr != nil {
		panic(newErr)
	}
	var settleMethod = cabi.MethodNameDexFundSettleOrdersV2
	if !dex.IsLeafFork(db) {
		settleMethod = cabi.MethodNameDexFundSettleOrders
	}
	if dexSettleBlockData, newErr = cabi.ABIDexFund.PackMethod(settleMethod, settleData); newErr != nil {
		panic(newErr)
	}
	return []*ledger.AccountBlock{
		{
			AccountAddress: types.AddressDexTrade,
			ToAddress:      types.AddressDexFund,
			BlockType:      ledger.BlockTypeSendCall,
			TokenId:        ledger.ViteTokenId,
			Amount:         big.NewInt(0),
			Data:           dexSettleBlockData,
		},
	}, nil
}

func handleCancelOrderById(db vm_db.VmDb, orderId []byte, method string, block, sendBlock *ledger.AccountBlock) ([]*ledger.AccountBlock, error) {
	var (
		order        *dex.Order
		marketId     int32
		matcher      *dex.Matcher
		appendBlocks []*ledger.AccountBlock
		err          error
	)
	if marketId, _, _, _, err = dex.DeComposeOrderId(orderId); err != nil {
		return handleDexReceiveErr(tradeLogger, method, dex.InvalidOrderIdErr, sendBlock)
	}
	if matcher, err = dex.NewMatcher(db, marketId); err != nil {
		return handleDexReceiveErr(tradeLogger, method, err, sendBlock)
	}
	if order, err = matcher.GetOrderById(orderId); err != nil {
		return handleDexReceiveErr(tradeLogger, method, err, sendBlock)
	}
	if !bytes.Equal(sendBlock.AccountAddress.Bytes(), order.Address) && !bytes.Equal(sendBlock.AccountAddress.Bytes(), order.Agent) {
		return handleDexReceiveErr(tradeLogger, method, dex.CancelOrderOwnerInvalidErr, sendBlock)
	}
	if order.Status != dex.Pending && order.Status != dex.PartialExecuted {
		return handleDexReceiveErr(tradeLogger, method, dex.CancelOrderInvalidStatusErr, sendBlock)
	}
	matcher.CancelOrderById(order)
	if appendBlocks, err = handleSettleActions(db, block, matcher.GetFundSettles(), nil, matcher.MarketInfo); err != nil {
		return handleDexReceiveErr(tradeLogger, method, err, sendBlock)
	} else {
		return appendBlocks, nil
	}
}

func handleSettleActions(db vm_db.VmDb, block *ledger.AccountBlock, fundSettles map[types.Address]map[bool]*dexproto.AccountSettle, feeSettles map[types.Address]*dexproto.FeeSettle, marketInfo *dex.MarketInfo) ([]*ledger.AccountBlock, error) {
	//fmt.Printf("fundSettles.size %d\n", len(fundSettles))
	if len(fundSettles) == 0 && len(feeSettles) == 0 {
		return nil, nil
	}
	settleActions := &dexproto.SettleActions{}
	if len(fundSettles) > 0 {
		fundActions := make([]*dexproto.FundSettle, 0, len(fundSettles))
		for address, accountSettleMap := range fundSettles {
			accountSettles := make([]*dexproto.AccountSettle, 0, len(accountSettleMap))
			for _, accountSettle := range accountSettleMap {
				accountSettles = append(accountSettles, accountSettle)
			}
			sort.Sort(dex.AccountSettleSorter(accountSettles))

			userFundSettle := &dexproto.FundSettle{}
			userFundSettle.Address = address.Bytes()
			userFundSettle.AccountSettles = accountSettles
			fundActions = append(fundActions, userFundSettle)
		}
		//sort fundActions for stable marsh result
		sort.Sort(dex.FundSettleSorter(fundActions))
		//fmt.Printf("fundActions.size %d\n", len(fundActions))
		settleActions.FundActions = fundActions
	}
	//every block will trigger exactly one market, fee token type should also be single
	if len(feeSettles) > 0 {
		feeActions := make([]*dexproto.FeeSettle, 0, len(feeSettles))
		for _, feeSettle := range feeSettles {
			feeActions = append(feeActions, feeSettle)
		}
		sort.Sort(dex.FeeSettleSorter(feeActions))
		settleActions.FeeActions = feeActions
	}
	settleActions.TradeToken = marketInfo.TradeToken
	settleActions.QuoteToken = marketInfo.QuoteToken
	var (
		settleData, dexSettleBlockData []byte
		err                            error
	)
	if settleData, err = proto.Marshal(settleActions); err != nil {
		panic(err)
	}
	var settleMethod = cabi.MethodNameDexFundSettleOrdersV2
	if !dex.IsLeafFork(db) {
		settleMethod = cabi.MethodNameDexFundSettleOrders
	}
	if dexSettleBlockData, err = cabi.ABIDexFund.PackMethod(settleMethod, settleData); err != nil {
		panic(err)
	}
	return []*ledger.AccountBlock{
		{
			AccountAddress: block.AccountAddress,
			ToAddress:      types.AddressDexFund,
			BlockType:      ledger.BlockTypeSendCall,
			TokenId:        ledger.ViteTokenId,
			Amount:         big.NewInt(0),
			Data:           dexSettleBlockData,
		},
	}, nil
}
