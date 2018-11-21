package contracts

import (
	"bytes"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/abi"
	"github.com/vitelabs/go-vite/vm/contracts/dex"
	dexproto "github.com/vitelabs/go-vite/vm/contracts/dex/proto"
	"github.com/vitelabs/go-vite/vm/util"
	"github.com/vitelabs/go-vite/vm_context"
	"math/big"
	"strings"
)

const (
	jsonDexTrade = `
	[
		{"type":"function","name":"DexTradeNewOrder", "inputs":[{"name":"data","type":"bytes"}]},
		{"type":"function","name":"DexTradeCancelOrder", "inputs":[{"name":"orderId","type":"uint64"}, {"name":"quoteAsset","type":"tokenId"}, {"name":"tradeAsset","type":"tokenId"}, {"name":"side", "type":"bool"}]}
	]`

	MethodNameDexTradeNewOrder    = "DexTradeNewOrder"
	MethodNameDexTradeCancelOrder = "DexTradeCancelOrder"
)

var (
	ABIDexTrade, _ = abi.JSONToABIContract(strings.NewReader(jsonDexTrade))
)

type ParamDexCancelOrder struct {
	orderId     uint64
	quoteAsset  types.TokenTypeId
	tradeAsset 	types.TokenTypeId
	side bool
}


type MethodDexTradeNewOrder struct {
}

func (md *MethodDexTradeNewOrder) GetFee(context contractsContext, block *vm_context.VmAccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexTradeNewOrder) DoSend(context contractsContext, block *vm_context.VmAccountBlock, quotaLeft uint64) (uint64, error) {
	var err error
	if quotaLeft, err = util.UseQuota(quotaLeft, dexTradeNewOrderGas); err != nil {
		return quotaLeft, err
	}
	if !bytes.Equal(block.AccountBlock.AccountAddress.Bytes(), AddressDexFund.Bytes()) {
		return quotaLeft, fmt.Errorf("invalid block source")
	}
	quotaLeft, err = util.UseQuotaForData(block.AccountBlock.Data, quotaLeft)
	return quotaLeft, nil
}

func (md *MethodDexTradeNewOrder) DoReceive(context contractsContext, block *vm_context.VmAccountBlock, sendBlock *ledger.AccountBlock) error {
	var err error
	if !bytes.Equal(block.AccountBlock.AccountAddress.Bytes(), AddressDexFund.Bytes()) {
		return fmt.Errorf("invalid block source")
	}
	param := new(ParamDexSerializedData)
	if err = ABIDexTrade.UnpackMethod(param, MethodNameDexTradeNewOrder, block.AccountBlock.Data); err != nil {
		return err
	}
	order := &dexproto.Order{}
	if err = proto.Unmarshal(param.Data, order); err != nil {
		return err
	}
	storage, _ := block.VmContext.(dex.BaseStorage)
	matcher := dex.NewMatcher(&AddressDexTrade, &storage)
	matcher.MatchOrder(dex.Order{*order})
	return handleSettleActions(context, block, matcher.GetSettleActions())
}

type MethodDexTradeCancelOrder struct {
}

func (md *MethodDexTradeCancelOrder) GetFee(context contractsContext, block *vm_context.VmAccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexTradeCancelOrder) DoSend(context contractsContext, block *vm_context.VmAccountBlock, quotaLeft uint64) (uint64, error) {
	var err error
	if quotaLeft, err = util.UseQuota(quotaLeft, dexTradeCancelOrderGas); err != nil {
		return quotaLeft, err
	}
	param := new(ParamDexCancelOrder)
	if err = ABIDexTrade.UnpackMethod(param, MethodNameDexTradeCancelOrder, block.AccountBlock.Data); err != nil {
		return quotaLeft, err
	}
	makerBookName := dex.GetBookNameToMake(param.tradeAsset.Bytes(), param.quoteAsset.Bytes(), param.side)
	storage, _ := block.VmContext.(dex.BaseStorage)
	matcher := dex.NewMatcher(&AddressDexTrade, &storage)
	var order *dex.Order
	if order, err = matcher.GetOrderByIdAndBookName(param.orderId, makerBookName); err != nil {
		return quotaLeft, err
	}
	if !bytes.Equal(block.AccountBlock.AccountAddress.Bytes(), []byte(order.Address)) {
		return quotaLeft, fmt.Errorf("cancel order not own to initiator")
	}
	if order.Status != dex.Pending && order.Status != dex.PartialExecuted {
		return quotaLeft, fmt.Errorf("order status is invalid to cancel")
	}
	return quotaLeft, nil
}

func (md MethodDexTradeCancelOrder) DoReceive(context contractsContext, block *vm_context.VmAccountBlock, sendBlock *ledger.AccountBlock) error {
	param := new(ParamDexCancelOrder)
	ABIDexTrade.UnpackMethod(param, MethodNameDexTradeCancelOrder, block.AccountBlock.Data)
	makerBookName := dex.GetBookNameToMake(param.tradeAsset.Bytes(), param.quoteAsset.Bytes(), param.side)
	storage, _ := block.VmContext.(dex.BaseStorage)
	matcher := dex.NewMatcher(&AddressDexTrade, &storage)
	var (
		order *dex.Order
		err error
	)
	if order, err = matcher.GetOrderByIdAndBookName(param.orderId, makerBookName); err != nil {
		return err
	}
	if !bytes.Equal(block.AccountBlock.AccountAddress.Bytes(), []byte(order.Address)) {
		return fmt.Errorf("cancel order not own to initiator")
	}
	if order.Status != dex.Pending && order.Status != dex.PartialExecuted {
		return fmt.Errorf("order status is invalid to cancel")
	}
	if err = matcher.CancelOrderByIdAndBookName(order, makerBookName); err != nil {
		return err
	}
	return handleSettleActions(context, block, matcher.GetSettleActions())
}

func handleSettleActions(context contractsContext, block *vm_context.VmAccountBlock, actionMaps map[string]map[string]*dexproto.SettleAction) (err error) {
	if len(actionMaps) == 0 {
		return nil
	}
	actions := make([]*dexproto.SettleAction, 0, 100)
	for _, actionMap := range actionMaps {
		for _, action := range actionMap {
			actions = append(actions, action)
		}
	}
	settleOrders := &dexproto.SettleOrders{Actions: actions}
	var settleData []byte
	if settleData, err = proto.Marshal(settleOrders); err != nil {
		return err
	}
	if len(actions) > 0 {
		dexFundBlockData, _ := ABIDexFund.PackMethod(MethodNameDexFundSettleOrders, settleData)
		context.AppendBlock(
			&vm_context.VmAccountBlock{
				util.MakeSendBlock(
					block.AccountBlock,
					AddressDexFund,
					ledger.BlockTypeSendCall,
					big.NewInt(0),
					ledger.ViteTokenId, // no need send token
					context.GetNewBlockHeight(block),
					dexFundBlockData),
				nil})
	}
	return nil
}