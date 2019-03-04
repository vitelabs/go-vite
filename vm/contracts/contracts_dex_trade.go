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

func (md *MethodDexTradeNewOrder) GetSendQuota(data []byte) (uint64, error) {
	return util.TotalGasCost(dexTradeNewOrderGas, data)
}

func (p *MethodDexTradeNewOrder) GetReceiveQuota() uint64 {
	return 0
}

func (md *MethodDexTradeNewOrder) DoSend(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock) error {
	if !bytes.Equal(block.AccountAddress.Bytes(), types.AddressDexFund.Bytes()) {
		return fmt.Errorf("invalid block source")
	}
	return nil
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
	blocks, err := handleSettleActions(block, matcher.GetFundSettles(), matcher.GetFees())
	if err != nil {
		fmt.Printf("MethodDexTradeNewOrder doReceive err %v\n", err)
	}
	return blocks, err
}

type MethodDexTradeCancelOrder struct {
}

func (md *MethodDexTradeCancelOrder) GetFee(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexTradeCancelOrder) GetRefundData() []byte {
	return []byte{}
}

func (md *MethodDexTradeCancelOrder) GetSendQuota(data []byte) (uint64, error) {
	return util.TotalGasCost(dexTradeCancelOrderGas, data)
}

func (p *MethodDexTradeCancelOrder) GetReceiveQuota() uint64 {
	return 0
}

func (md *MethodDexTradeCancelOrder) DoSend(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock) error {
	var err error
	param := new(ParamDexCancelOrder)
	if err = ABIDexTrade.UnpackMethod(param, MethodNameDexTradeCancelOrder, block.Data); err != nil {
		return err
	}
	makerBookId := dex.GetBookIdToMake(param.TradeToken.Bytes(), param.QuoteToken.Bytes(), param.Side)
	storage, _ := db.(dex.BaseStorage)
	matcher := dex.NewMatcher(&types.AddressDexTrade, &storage)
	var order *dex.Order
	if order, err = matcher.GetOrderByIdAndBookId(param.OrderId, makerBookId); err != nil {
		return err
	}
	if !bytes.Equal(block.AccountAddress.Bytes(), []byte(order.Address)) {
		return fmt.Errorf("cancel order not own to initiator")
	}
	if order.Status != dex.Pending && order.Status != dex.PartialExecuted {
		return fmt.Errorf("order status is invalid to cancel")
	}
	return nil
}

func (md MethodDexTradeCancelOrder) DoReceive(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock) ([]*SendBlock, error) {
	param := new(ParamDexCancelOrder)
	ABIDexTrade.UnpackMethod(param, MethodNameDexTradeCancelOrder, sendBlock.Data)
	makerBookId := dex.GetBookIdToMake(param.TradeToken.Bytes(), param.QuoteToken.Bytes(), param.Side)
	storage, _ := db.(dex.BaseStorage)
	matcher := dex.NewMatcher(&types.AddressDexTrade, &storage)
	var (
		order *dex.Order
		err   error
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

func handleSettleActions(block *ledger.AccountBlock, fundSettles map[types.Address]map[types.TokenTypeId]*dexproto.FundSettle, feeSettles map[types.TokenTypeId]map[types.Address]*dexproto.UserFeeSettle) ([]*SendBlock, error) {
	//fmt.Printf("fundSettles.size %d\n", len(fundSettles))
	if len(fundSettles) == 0 && len(feeSettles) == 0 {
		return []*SendBlock{}, nil
	}
	settleActions := &dexproto.SettleActions{}
	if len(fundSettles) > 0 {
		fundActions := make([]*dexproto.UserFundSettle, 0, len(fundSettles))
		for address, fundSettleMap := range fundSettles {
			settles := make([]*dexproto.FundSettle, 0, len(fundSettleMap))
			for _, fundAction := range fundSettleMap {
				settles = append(settles, fundAction)
			}
			sort.Sort(FundSettleSorter(settles))

			userFundSettle := &dexproto.UserFundSettle{}
			userFundSettle.Address = address.Bytes()
			userFundSettle.FundSettles = settles
			fundActions = append(fundActions, userFundSettle)
		}
		//sort fundActions for stable marsh result
		sort.Sort(UserFundSettleSorter(fundActions))
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
			sort.Sort(UserFeeSettleSorter(userFeeSettles))

			feeSettle := &dexproto.FeeSettle{}
			feeSettle.Token = token.Bytes()
			feeSettle.UserFeeSettles = userFeeSettles
			feeActions = append(feeActions, feeSettle)
		}
		sort.Sort(FeeSettleSorter(feeActions))
		settleActions.FeeActions = feeActions
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
	}, nil
}

type FundSettleSorter []*dexproto.FundSettle

func (st FundSettleSorter) Len() int {
	return len(st)
}

func (st FundSettleSorter) Swap(i, j int) {
	st[i], st[j] = st[j], st[i]
}

func (st FundSettleSorter) Less(i, j int) bool {
	tkCmp := bytes.Compare(st[i].Token, st[j].Token)
	if tkCmp < 0 {
		return true
	} else {
		return false
	}
}

type UserFundSettleSorter []*dexproto.UserFundSettle

func (st UserFundSettleSorter) Len() int {
	return len(st)
}

func (st UserFundSettleSorter) Swap(i, j int) {
	st[i], st[j] = st[j], st[i]
}

func (st UserFundSettleSorter) Less(i, j int) bool {
	addCmp := bytes.Compare(st[i].Address, st[j].Address)
	if addCmp < 0 {
		return true
	} else {
		return false
	}
}

type UserFeeSettleSorter []*dexproto.UserFeeSettle

func (st UserFeeSettleSorter) Len() int {
	return len(st)
}

func (st UserFeeSettleSorter) Swap(i, j int) {
	st[i], st[j] = st[j], st[i]
}

func (st UserFeeSettleSorter) Less(i, j int) bool {
	addCmp := bytes.Compare(st[i].Address, st[j].Address)
	if addCmp < 0 {
		return true
	} else {
		return false
	}
}

type FeeSettleSorter []*dexproto.FeeSettle

func (st FeeSettleSorter) Len() int {
	return len(st)
}

func (st FeeSettleSorter) Swap(i, j int) {
	st[i], st[j] = st[j], st[i]
}

func (st FeeSettleSorter) Less(i, j int) bool {
	tkCmp := bytes.Compare(st[i].Token, st[j].Token)
	if tkCmp < 0 {
		return true
	} else {
		return false
	}
}
