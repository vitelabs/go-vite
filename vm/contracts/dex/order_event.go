package dex

import (
	"fmt"
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
const errEventName = "errEventName"
const cancelOrderFailEventName = "cancelOrderFailEvent"


const (
	NewOrderGetFundFail = iota
	NewOrderLockFundFail
	NewOrderSaveFundFail
	NewOrderInternalErr
	TradeMarketNotExistsFail
	OrderAmountTooSmallFail
)

type OrderEvent interface {
	GetTopicId() types.Hash
	toDataBytes() []byte
	FromBytes([]byte) interface{}
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

type CancelOrderFailEvent struct {
	dexproto.CancelOrderFail
}

type ErrEvent struct {
	error
}

func (od NewOrderEvent) GetTopicId() types.Hash {
	return fromNameToHash(newOrderEventName)
}

func (od NewOrderEvent) toDataBytes() []byte {
	data, _ := proto.Marshal(&od.OrderInfo)
	return data
}

func (od NewOrderEvent) FromBytes(data []byte) interface{} {
	event := NewOrderEvent{}
	if err := proto.Unmarshal(data, &event.OrderInfo); err != nil {
		return nil
	} else {
		return event
	}
}

func (od OrderUpdateEvent) GetTopicId() types.Hash {
	return fromNameToHash(orderUpdateEventName)
}

func (od OrderUpdateEvent) toDataBytes() []byte {
	data, _ := proto.Marshal(&od.OrderUpdateInfo)
	return data
}

func (od OrderUpdateEvent) FromBytes(data []byte) interface{} {
	event := OrderUpdateEvent{}
	if err := proto.Unmarshal(data, &event.OrderUpdateInfo); err != nil {
		return nil
	} else {
		return event
	}
}

func (tx TransactionEvent) GetTopicId() types.Hash {
	return fromNameToHash(txEventName)
}

func (tx TransactionEvent) toDataBytes() []byte {
	data, _ := proto.Marshal(&tx.Transaction)
	return data
}

func (tx TransactionEvent) FromBytes(data []byte) interface{} {
	event := TransactionEvent{}
	if err := proto.Unmarshal(data, &event.Transaction); err != nil {
		return nil
	} else {
		return event
	}
}

func (of NewOrderFailEvent) GetTopicId() types.Hash {
	return fromNameToHash(newOrderFailEventName)
}

func (of NewOrderFailEvent) toDataBytes() []byte {
	data, _ := proto.Marshal(&of.OrderFail)
	return data
}

func (of NewOrderFailEvent) FromBytes(data []byte) interface{} {
	event := NewOrderFailEvent{}
	if err := proto.Unmarshal(data, &event.OrderFail); err != nil {
		return nil
	} else {
		return event
	}
}

func (me NewMarketEvent) GetTopicId() types.Hash {
	return fromNameToHash(newMarketEventName)
}

func (me NewMarketEvent) toDataBytes() []byte {
	data, _ := proto.Marshal(&me.NewMarket)
	return data
}

func (me NewMarketEvent) FromBytes(data []byte) interface{} {
	event := NewMarketEvent{}
	if err := proto.Unmarshal(data, &event.NewMarket); err != nil {
		return nil
	} else {
		return event
	}
}

func (err ErrEvent) GetTopicId() types.Hash {
	return fromNameToHash(errEventName)
}

func (err ErrEvent) toDataBytes() []byte {
	return []byte(err.Error())
}

func (err ErrEvent) FromBytes(data []byte) interface{} {
	return ErrEvent{fmt.Errorf(string(data))}
}

func (ce CancelOrderFailEvent) GetTopicId() types.Hash {
	return fromNameToHash(cancelOrderFailEventName)
}

func (ce CancelOrderFailEvent) toDataBytes() []byte {
	data, _ := proto.Marshal(&ce.CancelOrderFail)
	return data
}

func (ce CancelOrderFailEvent) FromBytes(data []byte) interface{} {
	event := CancelOrderFailEvent{}
	if err := proto.Unmarshal(data, &event.CancelOrderFail); err != nil {
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