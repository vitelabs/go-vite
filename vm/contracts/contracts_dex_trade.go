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
		{"type":"function","name":"DexTradeCancelOrder", "inputs":[{"name":"orderId","type":"bytes"}, {"name":"tradeToken","type":"tokenId"}, {"name":"quoteToken","type":"tokenId"}, {"name":"side", "type":"bool"}]}
	]`

	MethodNameDexTradeNewOrder    = "DexTradeNewOrder"
	MethodNameDexTradeCancelOrder = "DexTradeCancelOrder"
)

var (
	ABIDexTrade, _ = abi.JSONToABIContract(strings.NewReader(jsonDexTrade))
)

type ParamDexCancelOrder struct {
	OrderId    []byte
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

func (md *MethodDexTradeNewOrder) GetQuota() uint64 {
	return 1000
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
	param := new(dex.ParamDexSerializedData)
	if err = ABIDexTrade.UnpackMethod(param, MethodNameDexTradeNewOrder, sendBlock.Data); err != nil {
		return []*SendBlock{}, err
	}
	order := &dexproto.Order{}
	if err = proto.Unmarshal(param.Data, order); err != nil {
		return []*SendBlock{}, err
	}
	storage, _ := db.(dex.BaseStorage)
	matcher := dex.NewMatcher(&types.AddressDexTrade, &storage)
	if err = matcher.MatchOrder(dex.Order{*order}); err != nil {
		return []*SendBlock{}, err
	}
	return handleSettleActions(block, matcher.GetFundSettles(), matcher.GetFees())
}

type MethodDexTradeCancelOrder struct {
}

func (md *MethodDexTradeCancelOrder) GetFee(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexTradeCancelOrder) GetRefundData() []byte {
	return []byte{}
}

func (md *MethodDexTradeCancelOrder) GetQuota() uint64 {
	return 1000
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
	makerBookId := dex.GetBookIdToMake(param.TradeToken.Bytes(), param.QuoteToken.Bytes(), param.Side)
	storage, _ := db.(dex.BaseStorage)
	matcher := dex.NewMatcher(&types.AddressDexTrade, &storage)
	var order *dex.Order
	if order, err = matcher.GetOrderByIdAndBookId(param.OrderId, makerBookId); err != nil {
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
	makerBookId := dex.GetBookIdToMake(param.TradeToken.Bytes(), param.QuoteToken.Bytes(), param.Side)
	storage, _ := db.(dex.BaseStorage)
	matcher := dex.NewMatcher(&types.AddressDexTrade, &storage)
	var (
		order *dex.Order
		err error
	)
	if order, err = matcher.GetOrderByIdAndBookId(param.OrderId, makerBookId); err != nil {
		return []*SendBlock{}, err
	}
	if order.Status != dex.Pending && order.Status != dex.PartialExecuted {
		return []*SendBlock{}, fmt.Errorf("order status is invalid to cancel")
	}
	if err = matcher.CancelOrderByIdAndBookId(order, makerBookId); err != nil {
		return []*SendBlock{}, err
	}
	return handleSettleActions(block, matcher.GetFundSettles(), nil)
}

func handleSettleActions(block *ledger.AccountBlock, fundSettles map[types.Address]map[types.TokenTypeId]*dexproto.FundSettle, feeSettles map[types.TokenTypeId]*dexproto.FeeSettle) ([]*SendBlock, error) {
	//fmt.Printf("fundSettles.size %d\n", len(fundSettles))
	if len(fundSettles) == 0 && len(feeSettles) == 0 {
		return []*SendBlock{}, nil
	}
	settleActions := &dexproto.SettleActions{}
	if len(fundSettles) > 0 {
		fundActions := make([]*dexproto.FundSettle, 0, 100)
		for _, fundSettleMap := range fundSettles {
			for _, fundAction := range fundSettleMap {
				fundActions = append(fundActions, fundAction)
			}
		}
		//sort fundActions for stable marsh result
		sort.Sort(fundActionsWrapper(fundActions))
		//fmt.Printf("fundActions.size %d\n", len(fundActions))
		settleActions.FundActions = fundActions
	}
	//every block will trigger exactly one market, fee token type should also be single
	for _, feeAction := range feeSettles {
		settleActions.FeeActions = append(settleActions.FeeActions, feeAction)
	}

	var (
		settleData, dexSettleBlockData []byte
		err                            error
	)
	if settleData, err = proto.Marshal(settleActions); err != nil {
		return []*SendBlock{}, err
	}
	if dexSettleBlockData, err = ABIDexFund.PackMethod(MethodNameDexFundSettleOrders, settleData); err != nil {
		return []*SendBlock{}, err
	}
	return []*SendBlock{
		{block,
			types.AddressDexFund,
			ledger.BlockTypeSendCall,
			big.NewInt(0),
			ledger.ViteTokenId, // no need send token
			dexSettleBlockData,
		},
	} ,nil
}

type fundActionsWrapper []*dexproto.FundSettle

func (sa fundActionsWrapper) Len() int {
	return len(sa)
}

func (sa fundActionsWrapper) Swap(i, j int) {
	sa[i], sa[j] = sa[j], sa[i]
}

func (sa fundActionsWrapper) Less(i, j int) bool {
	addCmp := bytes.Compare(sa[i].Address, sa[j].Address)
	tokenCmp := bytes.Compare(sa[i].Address, sa[j].Address)
	if addCmp < 0 && tokenCmp < 0 {
		return true
	} else {
		return false
	}
}