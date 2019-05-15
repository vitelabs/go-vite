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
const txEventName = "txEvent"
const newMarketEventName = "newMarketEvent"
const errEventName = "errEvent"

type OrderEvent interface {
	GetTopicId() types.Hash
	toDataBytes() []byte
	FromBytes([]byte) interface{}
}

type NewOrderEvent struct {
	dexproto.NewOrderInfo
}

type OrderUpdateEvent struct {
	dexproto.OrderUpdateInfo
}

type TransactionEvent struct {
	dexproto.Transaction
}

type NewMarketEvent struct {
	dexproto.NewMarket
}
type ErrEvent struct {
	error
}

func (od NewOrderEvent) GetTopicId() types.Hash {
	return fromNameToHash(newOrderEventName)
}

func (od NewOrderEvent) toDataBytes() []byte {
	data, _ := proto.Marshal(&od.NewOrderInfo)
	return data
}

func (od NewOrderEvent) FromBytes(data []byte) interface{} {
	event := NewOrderEvent{}
	if err := proto.Unmarshal(data, &event.NewOrderInfo); err != nil {
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

func fromNameToHash(name string) types.Hash {
	hs := types.Hash{}
	hs.SetBytes(common.RightPadBytes([]byte(name), types.HashSize))
	return hs
}
