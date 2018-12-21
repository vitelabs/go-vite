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
	"github.com/vitelabs/go-vite/vm_context/vmctxt_interface"
	"math/big"
	"sort"
	"strings"
)

const (
	jsonDexTrade = `
	[
		{"type":"function","name":"DexTradeNewOrder", "inputs":[{"name":"data","type":"bytes"}]},
		{"type":"function","name":"DexTradeCancelOrder", "inputs":[{"name":"orderId","type":"uint256"}, {"name":"tradeToken","type":"tokenId"}, {"name":"quoteToken","type":"tokenId"}, {"name":"side", "type":"bool"}]}
	]`

	MethodNameDexTradeNewOrder    = "DexTradeNewOrder"
	MethodNameDexTradeCancelOrder = "DexTradeCancelOrder"
)

var (
	ABIDexTrade, _ = abi.JSONToABIContract(strings.NewReader(jsonDexTrade))
)

type ParamDexCancelOrder struct {
	OrderId    *big.Int
	QuoteToken types.TokenTypeId
	TradeToken types.TokenTypeId
	Side       bool
}

type MethodDexTradeNewOrder struct {
}

func (md *MethodDexTradeNewOrder) GetFee(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexTradeNewOrder) GetRefundData() []byte {
	return []byte{}
}

func (md *MethodDexTradeNewOrder) DoSend(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, quotaLeft uint64) (uint64, error) {
	if quotaLeft, err := util.UseQuota(quotaLeft, dexTradeNewOrderGas); err != nil {
		return quotaLeft, err
	}
	if !bytes.Equal(block.AccountAddress.Bytes(), types.AddressDexFund.Bytes()) {
		return quotaLeft, fmt.Errorf("invalid block source")
	}
	return util.UseQuotaForData(block.Data, quotaLeft)
}

func (md *MethodDexTradeNewOrder) DoReceive(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock) ([]*SendBlock, error) {
	var err error
	if !bytes.Equal(sendBlock.AccountAddress.Bytes(), types.AddressDexFund.Bytes()) {
		return []*SendBlock{}, fmt.Errorf("invalid block source")
	}
	param := new(ParamDexSerializedData)
	if err = ABIDexTrade.UnpackMethod(param, MethodNameDexTradeNewOrder, sendBlock.Data); err != nil {
		return []*SendBlock{}, err
	}
	order := &dexproto.Order{}
	if err = proto.Unmarshal(param.Data, order); err != nil {
		return []*SendBlock{}, err
	}
	storage, _ := db.(dex.BaseStorage)
	matcher := dex.NewMatcher(&types.AddressDexTrade, &storage)
	matcher.MatchOrder(dex.Order{*order})
	address := &types.Address{}
	address.SetBytes(order.Address)
	return handleSettleActions(block, matcher.GetSettleActions())
}

type MethodDexTradeCancelOrder struct {
}

func (md *MethodDexTradeCancelOrder) GetFee(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexTradeCancelOrder) GetRefundData() []byte {
	return []byte{}
}

func (md *MethodDexTradeCancelOrder) DoSend(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, quotaLeft uint64) (uint64, error) {
	var err error
	if quotaLeft, err = util.UseQuota(quotaLeft, dexTradeCancelOrderGas); err != nil {
		return quotaLeft, err
	}
	param := new(ParamDexCancelOrder)
	if err = ABIDexTrade.UnpackMethod(param, MethodNameDexTradeCancelOrder, block.Data); err != nil {
		return quotaLeft, err
	}
	makerBookName := dex.GetBookNameToMake(param.TradeToken.Bytes(), param.QuoteToken.Bytes(), param.Side)
	storage, _ := db.(dex.BaseStorage)
	matcher := dex.NewMatcher(&types.AddressDexTrade, &storage)
	var order *dex.Order
	if order, err = matcher.GetOrderByIdAndBookName(param.OrderId.Uint64(), makerBookName); err != nil {
		return quotaLeft, err
	}
	if !bytes.Equal(block.AccountAddress.Bytes(), []byte(order.Address)) {
		return quotaLeft, fmt.Errorf("cancel order not own to initiator")
	}
	if order.Status != dex.Pending && order.Status != dex.PartialExecuted {
		return quotaLeft, fmt.Errorf("order status is invalid to cancel")
	}
	return util.UseQuotaForData(block.Data, quotaLeft)
}

func (md MethodDexTradeCancelOrder) DoReceive(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock) ([]*SendBlock, error) {
	param := new(ParamDexCancelOrder)
	ABIDexTrade.UnpackMethod(param, MethodNameDexTradeCancelOrder, sendBlock.Data)
	makerBookName := dex.GetBookNameToMake(param.TradeToken.Bytes(), param.QuoteToken.Bytes(), param.Side)
	storage, _ := db.(dex.BaseStorage)
	matcher := dex.NewMatcher(&types.AddressDexTrade, &storage)
	var (
		order *dex.Order
		err error
	)
	if order, err = matcher.GetOrderByIdAndBookName(param.OrderId.Uint64(), makerBookName); err != nil {
		return []*SendBlock{}, err
	}
	if !bytes.Equal(sendBlock.AccountAddress.Bytes(), []byte(order.Address)) {
		return []*SendBlock{}, fmt.Errorf("cancel order not own to initiator")
	}
	if order.Status != dex.Pending && order.Status != dex.PartialExecuted {
		return []*SendBlock{}, fmt.Errorf("order status is invalid to cancel")
	}
	if err = matcher.CancelOrderByIdAndBookName(order, makerBookName); err != nil {
		return []*SendBlock{}, err
	}
	return handleSettleActions(block, matcher.GetSettleActions())
}

func handleSettleActions(block *ledger.AccountBlock, actionMaps map[string]map[string]*dexproto.SettleAction) ([]*SendBlock, error) {
	//fmt.Printf("actionMaps.size %d\n", len(actionMaps))
	if len(actionMaps) == 0 {
		return []*SendBlock{}, nil
	}
	actions := make([]*dexproto.SettleAction, 0, 100)
	for _, actionMap := range actionMaps {
		for _, action := range actionMap {
			actions = append(actions, action)
		}
	}
	//sort actions for stable marsh result
	sort.Sort(settleActions(actions))
	//fmt.Printf("actions.size %d\n", len(actions))
	settleOrders := &dexproto.SettleActions{Actions: actions}
	var (
		settleData []byte
		err error
	)
	if settleData, err = proto.Marshal(settleOrders); err != nil {
		return []*SendBlock{}, err
	}

	if len(actions) > 0 {
		dexFundBlockData, _ := ABIDexFund.PackMethod(MethodNameDexFundSettleOrders, settleData)
		return []*SendBlock{
			{block,
				types.AddressDexFund,
				ledger.BlockTypeSendCall,
				big.NewInt(0),
				ledger.ViteTokenId, // no need send token
				dexFundBlockData,
			},
		} ,nil
	}
	return []*SendBlock{}, nil
}

type settleActions []*dexproto.SettleAction

func (sa settleActions) Len() int {
	return len(sa)
}

func (sa settleActions) Swap(i, j int) {
	sa[i], sa[j] = sa[j], sa[i]
}

func (sa settleActions) Less(i, j int) bool {
	addCmp := bytes.Compare(sa[i].Address, sa[j].Address)
	tokenCmp := bytes.Compare(sa[i].Address, sa[j].Address)
	if addCmp < 0 && tokenCmp < 0 {
		return true
	} else {
		return false
	}
}