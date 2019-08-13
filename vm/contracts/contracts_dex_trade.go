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

type MethodDexTradeNewOrder struct {
}

func (md *MethodDexTradeNewOrder) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexTradeNewOrder) GetRefundData(sendBlock *ledger.AccountBlock) ([]byte, bool) {
	return []byte{}, false
}

func (md *MethodDexTradeNewOrder) GetSendQuota(data []byte, gasTable *util.GasTable) (uint64, error) {
	return util.TxGasCost(data, gasTable)
}

func (md *MethodDexTradeNewOrder) GetReceiveQuota(gasTable *util.GasTable) uint64 {
	return 0
}

func (md *MethodDexTradeNewOrder) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) (err error) {
	if !bytes.Equal(block.AccountAddress.Bytes(), types.AddressDexFund.Bytes()) {
		return dex.InvalidSourceAddressErr
	}
	err = cabi.ABIDexTrade.UnpackMethod(new(dex.ParamDexSerializedData), cabi.MethodNameDexTradeNewOrder, block.Data)
	return
}

func (md *MethodDexTradeNewOrder) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	var (
		err     error
		blocks  []*ledger.AccountBlock
		matcher *dex.Matcher
	)
	param := new(dex.ParamDexSerializedData)
	cabi.ABIDexTrade.UnpackMethod(param, cabi.MethodNameDexTradeNewOrder, sendBlock.Data)
	order := &dex.Order{}
	if err = order.DeSerialize(param.Data); err != nil {
		panic(err)
	}
	if matcher, err = dex.NewMatcher(db, order.MarketId); err != nil {
		return handleDexReceiveErr(tradeLogger, cabi.MethodNameDexTradeNewOrder, err, sendBlock)
	}
	if err = matcher.MatchOrder(order, block.PrevHash); err != nil {
		return OnNewOrderFailed(order, matcher.MarketInfo)
	}
	if blocks, err = handleSettleActions(block, matcher.GetFundSettles(), matcher.GetFees(), matcher.MarketInfo); err != nil {
		return OnNewOrderFailed(order, matcher.MarketInfo)
	}
	return blocks, err
}

type MethodDexTradeCancelOrder struct {
}

func (md *MethodDexTradeCancelOrder) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexTradeCancelOrder) GetRefundData(sendBlock *ledger.AccountBlock) ([]byte, bool) {
	return []byte{}, false
}

func (md *MethodDexTradeCancelOrder) GetSendQuota(data []byte, gasTable *util.GasTable) (uint64, error) {
	return util.TxGasCost(data, gasTable)
}

func (md *MethodDexTradeCancelOrder) GetReceiveQuota(gasTable *util.GasTable) uint64 {
	return 0
}

func (md *MethodDexTradeCancelOrder) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) (err error) {
	err = cabi.ABIDexTrade.UnpackMethod(new(dex.ParamDexCancelOrder), cabi.MethodNameDexTradeCancelOrder, block.Data)
	return
}

func (md MethodDexTradeCancelOrder) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	param := new(dex.ParamDexCancelOrder)
	cabi.ABIDexTrade.UnpackMethod(param, cabi.MethodNameDexTradeCancelOrder, sendBlock.Data)
	return handleCancelOrderById(db, param.OrderId, cabi.MethodNameDexTradeCancelOrder, block, sendBlock)
}

type MethodDexTradeNotifyNewMarket struct {
}

func (md *MethodDexTradeNotifyNewMarket) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexTradeNotifyNewMarket) GetRefundData(sendBlock *ledger.AccountBlock) ([]byte, bool) {
	return []byte{}, false
}

func (md *MethodDexTradeNotifyNewMarket) GetSendQuota(data []byte, gasTable *util.GasTable) (uint64, error) {
	return util.TxGasCost(data, gasTable)
}

func (md *MethodDexTradeNotifyNewMarket) GetReceiveQuota(gasTable *util.GasTable) uint64 {
	return 0
}

func (md *MethodDexTradeNotifyNewMarket) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) (err error) {
	if !bytes.Equal(block.AccountAddress.Bytes(), types.AddressDexFund.Bytes()) {
		return dex.InvalidSourceAddressErr
	}
	err = cabi.ABIDexTrade.UnpackMethod(new(dex.ParamDexSerializedData), cabi.MethodNameDexTradeNotifyNewMarket, block.Data)
	return
}

func (md MethodDexTradeNotifyNewMarket) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	param := new(dex.ParamDexSerializedData)
	cabi.ABIDexTrade.UnpackMethod(param, cabi.MethodNameDexTradeNotifyNewMarket, sendBlock.Data)
	marketInfo := &dex.MarketInfo{}
	if err := marketInfo.DeSerialize(param.Data); err != nil {
		return handleDexReceiveErr(tradeLogger, cabi.MethodNameDexTradeNotifyNewMarket, err, sendBlock)
	} else {
		dex.SaveMarketInfoById(db, marketInfo)
		return nil, nil
	}
}

type MethodDexTradeCleanExpireOrders struct {
}

func (md *MethodDexTradeCleanExpireOrders) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexTradeCleanExpireOrders) GetRefundData(sendBlock *ledger.AccountBlock) ([]byte, bool) {
	return []byte{}, false
}

func (md *MethodDexTradeCleanExpireOrders) GetSendQuota(data []byte, gasTable *util.GasTable) (uint64, error) {
	return util.TxGasCost(data, gasTable)
}

func (md *MethodDexTradeCleanExpireOrders) GetReceiveQuota(gasTable *util.GasTable) uint64 {
	return 0
}

func (md *MethodDexTradeCleanExpireOrders) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) (err error) {
	err = cabi.ABIDexTrade.UnpackMethod(new(dex.ParamDexSerializedData), cabi.MethodNameDexTradeCleanExpireOrders, block.Data)
	return
}

func (md MethodDexTradeCleanExpireOrders) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	param := new(dex.ParamDexSerializedData)
	cabi.ABIDexTrade.UnpackMethod(param, cabi.MethodNameDexTradeCleanExpireOrders, sendBlock.Data)
	if len(param.Data) == 0 || len(param.Data)%dex.OrderIdBytesLength != 0 || len(param.Data)/dex.OrderIdBytesLength > dex.CleanExpireOrdersMaxCount {
		return handleDexReceiveErr(tradeLogger, cabi.MethodNameDexTradeCleanExpireOrders, dex.InvalidInputParamErr, sendBlock)
	}
	if fundSettles, markerInfo, err := dex.CleanExpireOrders(db, param.Data); err != nil {
		return handleDexReceiveErr(tradeLogger, cabi.MethodNameDexTradeCleanExpireOrders, err, sendBlock)
	} else if len(fundSettles) > 0 {
		if appendBlocks, err := handleSettleActions(block, fundSettles, nil, markerInfo); err != nil {
			return handleDexReceiveErr(tradeLogger, cabi.MethodNameDexTradeCleanExpireOrders, err, sendBlock)
		} else {
			return appendBlocks, nil
		}
	} else {
		return nil, nil
	}
}

type MethodDexTradeCancelOrderByHash struct {
}

func (md *MethodDexTradeCancelOrderByHash) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexTradeCancelOrderByHash) GetRefundData(sendBlock *ledger.AccountBlock) ([]byte, bool) {
	return []byte{}, false
}

func (md *MethodDexTradeCancelOrderByHash) GetSendQuota(data []byte, gasTable *util.GasTable) (uint64, error) {
	return util.TxGasCost(data, gasTable)
}

func (md *MethodDexTradeCancelOrderByHash) GetReceiveQuota(gasTable *util.GasTable) uint64 {
	return 0
}

func (md *MethodDexTradeCancelOrderByHash) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) (err error) {
	err = cabi.ABIDexTrade.UnpackMethod(new(types.Hash), cabi.MethodNameDexTradeCancelOrderByHash, block.Data)
	return
}

func (md MethodDexTradeCancelOrderByHash) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	sendHash := new(types.Hash)
	cabi.ABIDexTrade.UnpackMethod(sendHash, cabi.MethodNameDexTradeCancelOrderByHash, sendBlock.Data)

	if orderId, ok := dex.GetOrderIdByHash(db, sendHash.Bytes()); !ok {
		return handleDexReceiveErr(tradeLogger, cabi.MethodNameDexTradeCancelOrderByHash, dex.InvalidOrderHashErr, sendBlock)
	} else {
		return handleCancelOrderById(db, orderId, cabi.MethodNameDexTradeCancelOrderByHash, block, sendBlock)
	}
}

func OnNewOrderFailed(order *dex.Order, marketInfo *dex.MarketInfo) ([]*ledger.AccountBlock, error) {
	fundSettle := &dexproto.FundSettle{}
	switch order.Side {
	case false: // buy
		fundSettle.IsTradeToken = false
		fundSettle.ReleaseLocked = dex.AddBigInt(order.Amount, order.LockedBuyFee)
	case true: // sell
		fundSettle.IsTradeToken = true
		fundSettle.ReleaseLocked = order.Quantity
	}
	userFundSettle := &dexproto.UserFundSettle{}
	userFundSettle.Address = order.Address
	userFundSettle.FundSettles = append(userFundSettle.FundSettles, fundSettle)
	settleActions := &dexproto.SettleActions{}
	settleActions.FundActions = append(settleActions.FundActions, userFundSettle)
	settleActions.TradeToken = marketInfo.TradeToken
	settleActions.QuoteToken = marketInfo.QuoteToken
	var (
		settleData, dexSettleBlockData []byte
		newErr                         error
	)
	if settleData, newErr = proto.Marshal(settleActions); newErr != nil {
		panic(newErr)
	}
	if dexSettleBlockData, newErr = cabi.ABIDexFund.PackMethod(cabi.MethodNameDexFundSettleOrders, settleData); newErr != nil {
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
	if appendBlocks, err = handleSettleActions(block, matcher.GetFundSettles(), nil, matcher.MarketInfo); err != nil {
		return handleDexReceiveErr(tradeLogger, method, err, sendBlock)
	} else {
		return appendBlocks, nil
	}
}

func handleSettleActions(block *ledger.AccountBlock, fundSettles map[types.Address]map[bool]*dexproto.FundSettle, feeSettles map[types.Address]*dexproto.UserFeeSettle, marketInfo *dex.MarketInfo) ([]*ledger.AccountBlock, error) {
	//fmt.Printf("fundSettles.size %d\n", len(fundSettles))
	if len(fundSettles) == 0 && len(feeSettles) == 0 {
		return nil, nil
	}
	settleActions := &dexproto.SettleActions{}
	if len(fundSettles) > 0 {
		fundActions := make([]*dexproto.UserFundSettle, 0, len(fundSettles))
		for address, fundSettleMap := range fundSettles {
			settles := make([]*dexproto.FundSettle, 0, len(fundSettleMap))
			for _, fundAction := range fundSettleMap {
				settles = append(settles, fundAction)
			}
			sort.Sort(dex.FundSettleSorter(settles))

			userFundSettle := &dexproto.UserFundSettle{}
			userFundSettle.Address = address.Bytes()
			userFundSettle.FundSettles = settles
			fundActions = append(fundActions, userFundSettle)
		}
		//sort fundActions for stable marsh result
		sort.Sort(dex.UserFundSettleSorter(fundActions))
		//fmt.Printf("fundActions.size %d\n", len(fundActions))
		settleActions.FundActions = fundActions
	}
	//every block will trigger exactly one market, fee token type should also be single
	if len(feeSettles) > 0 {
		feeActions := make([]*dexproto.UserFeeSettle, 0, len(feeSettles))
		for _, feeSettle := range feeSettles {
			feeActions = append(feeActions, feeSettle)
		}
		sort.Sort(dex.UserFeeSettleSorter(feeActions))
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
	if dexSettleBlockData, err = cabi.ABIDexFund.PackMethod(cabi.MethodNameDexFundSettleOrders, settleData); err != nil {
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
