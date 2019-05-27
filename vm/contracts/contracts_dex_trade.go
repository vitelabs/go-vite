package contracts

import (
	"bytes"
	"github.com/golang/protobuf/proto"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	cabi "github.com/vitelabs/go-vite/vm/contracts/abi"
	"github.com/vitelabs/go-vite/vm/contracts/dex"
	dexproto "github.com/vitelabs/go-vite/vm/contracts/dex/proto"
	"github.com/vitelabs/go-vite/vm/util"
	"github.com/vitelabs/go-vite/vm_db"
	"math/big"
	"sort"
)

type MethodDexTradeNewOrder struct {
}

func (md *MethodDexTradeNewOrder) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexTradeNewOrder) GetRefundData(sendBlock *ledger.AccountBlock) ([]byte, bool) {
	return []byte{}, false
}

func (md *MethodDexTradeNewOrder) GetSendQuota(data []byte) (uint64, error) {
	return util.TotalGasCost(dexTradeNewOrderGas, data)
}

func (md *MethodDexTradeNewOrder) GetReceiveQuota() uint64 {
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
		return handleReceiveErr(err)
	}
	if err = matcher.MatchOrder(order); err != nil {
		return OnNewOrderFailed(order, matcher.MarketInfo)
	}
	if blocks, err = handleSettleActions(block, matcher.GetFundSettles(), matcher.GetFees()); err != nil {
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

func (md *MethodDexTradeCancelOrder) GetSendQuota(data []byte) (uint64, error) {
	return util.TotalGasCost(dexTradeCancelOrderGas, data)
}

func (md *MethodDexTradeCancelOrder) GetReceiveQuota() uint64 {
	return 0
}

func (md *MethodDexTradeCancelOrder) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) (err error) {
	err = cabi.ABIDexTrade.UnpackMethod(new(dex.ParamDexCancelOrder), cabi.MethodNameDexTradeCancelOrder, block.Data)
	return
}

func (md MethodDexTradeCancelOrder) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	param := new(dex.ParamDexCancelOrder)
	cabi.ABIDexTrade.UnpackMethod(param, cabi.MethodNameDexTradeCancelOrder, sendBlock.Data)
	var (
		order        *dex.Order
		err          error
		marketId     int32
		matcher      *dex.Matcher
		appendBlocks []*ledger.AccountBlock
	)
	if marketId, _, _, _, err = dex.DeComposeOrderId(param.OrderId); err != nil {
		return handleReceiveErr(dex.InvalidOrderIdErr)
	}
	if matcher, err = dex.NewMatcher(db, marketId); err != nil {
		return handleReceiveErr(err)
	}
	if order, err = matcher.GetOrderById(param.OrderId); err != nil {
		return handleReceiveErr(err)
	}
	if !bytes.Equal(sendBlock.AccountAddress.Bytes(), []byte(order.Address)) {
		return handleReceiveErr(dex.CancelOrderOwnerInvalidErr)
	}
	if order.Status != dex.Pending && order.Status != dex.PartialExecuted {
		return handleReceiveErr(dex.CancelOrderInvalidStatusErr)
	}
	matcher.CancelOrderById(order)
	if appendBlocks, err = handleSettleActions(block, matcher.GetFundSettles(), nil); err != nil {
		return handleReceiveErr(err)
	} else {
		return appendBlocks, nil
	}
}

type MethodDexTradeNotifyNewMarket struct {
}

func (md *MethodDexTradeNotifyNewMarket) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexTradeNotifyNewMarket) GetRefundData(sendBlock *ledger.AccountBlock) ([]byte, bool) {
	return []byte{}, false
}

func (md *MethodDexTradeNotifyNewMarket) GetSendQuota(data []byte) (uint64, error) {
	return util.TotalGasCost(dexTradeNotifyNewMarketGas, data)
}

func (md *MethodDexTradeNotifyNewMarket) GetReceiveQuota() uint64 {
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
		return handleReceiveErr(err)
	} else {
		dex.SaveMarketInfoById(db, marketInfo)
		return nil, nil
	}
}

func OnNewOrderFailed(order *dex.Order, marketInfo *dex.MarketInfo) ([]*ledger.AccountBlock, error) {
	fundSettle := &dexproto.FundSettle{}
	switch order.Side {
	case false: // buy
		fundSettle.Token = marketInfo.QuoteToken
		fundSettle.ReleaseLocked = dex.AddBigInt(order.Amount, order.LockedBuyFee)
	case true: // sell
		fundSettle.Token = marketInfo.TradeToken
		fundSettle.ReleaseLocked = order.Quantity
	}
	userFundSettle := &dexproto.UserFundSettle{}
	userFundSettle.Address = order.Address
	userFundSettle.FundSettles = append(userFundSettle.FundSettles, fundSettle)
	settleActions := &dexproto.SettleActions{}
	settleActions.FundActions = append(settleActions.FundActions, userFundSettle)
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

func handleSettleActions(block *ledger.AccountBlock, fundSettles map[types.Address]map[types.TokenTypeId]*dexproto.FundSettle, feeSettles map[types.TokenTypeId]map[types.Address]*dexproto.UserFeeSettle) ([]*ledger.AccountBlock, error) {
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
		feeActions := make([]*dexproto.FeeSettle, 0, 10)
		for token, userFeeSettleMap := range feeSettles {
			userFeeSettles := make([]*dexproto.UserFeeSettle, 0, len(userFeeSettleMap))
			for _, userFeeSettle := range userFeeSettleMap {
				userFeeSettles = append(userFeeSettles, userFeeSettle)
			}
			sort.Sort(dex.UserFeeSettleSorter(userFeeSettles))

			feeSettle := &dexproto.FeeSettle{}
			feeSettle.Token = token.Bytes()
			feeSettle.UserFeeSettles = userFeeSettles
			feeActions = append(feeActions, feeSettle)
		}
		sort.Sort(dex.FeeSettleSorter(feeActions))
		settleActions.FeeActions = feeActions
	}

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
