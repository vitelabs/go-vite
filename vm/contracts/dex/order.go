package dex

import (
	"bytes"
	"github.com/golang/protobuf/proto"
	dexproto "github.com/vitelabs/go-vite/vm/contracts/dex/proto"
)

const (
	Pending = iota
	PartialExecuted
	FullyExecuted
	Cancelled
	NewFailed
)

const (
	cancelledByUser = iota
	cancelledByMarket
	cancelledOnTimeout
	partialExecutedUserCancelled
	partialExecutedCancelledByMarket
	partialExecutedCancelledOnTimeout
	unknownCancelledOnTimeout
)

const (
	Limited = iota
	Market
)

const OrderIdBytesLength = 22

type Order struct {
	dexproto.Order
}

func (od *Order) Serialize() ([]byte, error) {
	od.MarketId = 0
	od.Side = false
	od.Price = nil
	od.Timestamp = 0
	if data, err := proto.Marshal(&od.Order); err != nil {
		return nil, err
	} else {
		return data, err
	}
}

func (od *Order) DeSerialize(orderData []byte) error {
	order := dexproto.Order{}
	if err := proto.Unmarshal(orderData, &order); err != nil {
		return err
	}
	od.Order = order
	return od.RenderOrderById(order.Id)
}

func (od *Order) SerializeCompact() ([]byte, error) {
	od.Id = nil
	return od.Serialize()
}

func (od *Order) DeSerializeCompact(orderData []byte, orderId []byte) error {
	order := dexproto.Order{}
	if err := proto.Unmarshal(orderData, &order); err != nil {
		return err
	} else {
		od.Order = order
		od.Id = orderId
		return od.RenderOrderById(orderId)
	}
}

func (od *Order) RenderOrderById(orderId []byte) error {
	marketId, side, priceBytes, timestamp, err := DeComposeOrderId(orderId)
	if err != nil {
		return err
	}
	od.MarketId = marketId
	od.Side = side
	od.Price = priceBytes
	od.Timestamp = timestamp
	return nil
}

func priceCompare(a, b []byte) int {
	return bytes.Compare(a, b)
}
