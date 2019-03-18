package dex

import (
	"github.com/golang/protobuf/proto"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/common/types"
	dexproto "github.com/vitelabs/go-vite/vm/contracts/dex/proto"
)

const newOrderEventName = "newOrderEvent"
const orderUpdateEventName = "orderUpdateEvent"
const newOrderFailEventName = "newOrderFailEvent"
const txEventName = "txEvent"
const newMarketEventName = "newMarketEvent"


const (
	NewOrderGetFundFail = iota
	NewOrderLockFundFail
	NewOrderSaveFundFail
	NewOrderInternalErr
)

type OrderEvent interface {
	getTopicId() types.Hash
	toDataBytes() []byte
	fromBytes([]byte) interface{}
}

type NewOrderEvent struct {
	dexproto.OrderInfo
}

type OrderUpdateEvent struct {
	dexproto.OrderUpdateInfo
}

type TransactionEvent struct {
	dexproto.Transaction
}

type NewOrderFailEvent struct {
	dexproto.OrderFail
}

type NewMarketEvent struct {
	dexproto.NewMarket
}

func (od NewOrderEvent) getTopicId() types.Hash {
	return fromNameToHash(newOrderEventName)
}

func (od NewOrderEvent) toDataBytes() []byte {
	data, _ := proto.Marshal(&od.OrderInfo)
	return data
}

func (od NewOrderEvent) fromBytes(data []byte) interface{} {
	event := NewOrderEvent{}
	if err := proto.Unmarshal(data, &event.OrderInfo); err != nil {
		return nil
	} else {
		return event
	}
}

func (od OrderUpdateEvent) getTopicId() types.Hash {
	return fromNameToHash(orderUpdateEventName)
}

func (od OrderUpdateEvent) toDataBytes() []byte {
	data, _ := proto.Marshal(&od.OrderUpdateInfo)
	return data
}

func (od OrderUpdateEvent) fromBytes(data []byte) interface{} {
	event := OrderUpdateEvent{}
	if err := proto.Unmarshal(data, &event.OrderUpdateInfo); err != nil {
		return nil
	} else {
		return event
	}
}

func (tx TransactionEvent) getTopicId() types.Hash {
	return fromNameToHash(txEventName)
}

func (tx TransactionEvent) toDataBytes() []byte {
	data, _ := proto.Marshal(&tx.Transaction)
	return data
}

func (tx TransactionEvent) fromBytes(data []byte) interface{} {
	event := TransactionEvent{}
	if err := proto.Unmarshal(data, &event.Transaction); err != nil {
		return nil
	} else {
		return event
	}
}

func (of NewOrderFailEvent) getTopicId() types.Hash {
	return fromNameToHash(newOrderFailEventName)
}

func (of NewOrderFailEvent) toDataBytes() []byte {
	data, _ := proto.Marshal(&of.OrderFail)
	return data
}

func (of NewOrderFailEvent) fromBytes(data []byte) interface{} {
	event := NewOrderFailEvent{}
	if err := proto.Unmarshal(data, &event.OrderFail); err != nil {
		return nil
	} else {
		return event
	}
}

func (me NewMarketEvent) getTopicId() types.Hash {
	return fromNameToHash(newMarketEventName)
}

func (me NewMarketEvent) toDataBytes() []byte {
	data, _ := proto.Marshal(&me.NewMarket)
	return data
}

func (me NewMarketEvent) fromBytes(data []byte) interface{} {
	event := NewMarketEvent{}
	if err := proto.Unmarshal(data, &event.NewMarket); err != nil {
		return nil
	} else {
		return event
	}
}

func fromNameToHash(name string) types.Hash {
	hs := types.Hash{}
	hs.SetBytes(common.RightPadBytes([]byte(name), types.HashSize))
	return hs
}