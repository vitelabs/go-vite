package dex

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	dexproto "github.com/vitelabs/go-vite/vm/contracts/dex/proto"
	"github.com/vitelabs/go-vite/vm_context/vmctxt_interface"
)

type ParamDexCancelOrder struct {
	OrderId    []byte
	TradeToken types.TokenTypeId
	QuoteToken types.TokenTypeId
	Side       bool
}

func EmitCancelOrderFailLog(db vmctxt_interface.VmDatabase, param *ParamDexCancelOrder, address []byte, err error) {
	cancelFail := dexproto.CancelOrderFail{}
	cancelFail.Id = param.OrderId
	cancelFail.Address = address
	cancelFail.TradeToken = param.TradeToken.Bytes()
	cancelFail.QuoteToken = param.QuoteToken.Bytes()
	cancelFail.Side = param.Side
	cancelFail.ErrMsg = err.Error()
	event := CancelOrderFailEvent{cancelFail}

	log := &ledger.VmLog{}
	log.Topics = append(log.Topics, event.GetTopicId())
	log.Data = event.toDataBytes()
	db.AddLog(log)
}
