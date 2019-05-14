package dex

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/interfaces"
	"github.com/vitelabs/go-vite/ledger"
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

type BaseStorage interface {
	GetValue(key []byte) ([]byte, error)
	SetValue(key []byte, value []byte) error
	AddLog(log *ledger.VmLog)
	GetLogListHash() *types.Hash
	NewStorageIterator(prefix []byte) (interfaces.StorageIterator, error)
}

type OrderId [OrderIdBytesLength]byte

func NewOrderId(value []byte) (OrderId, error) {
	key := &OrderId{}
	if err := key.setBytes(value); err != nil {
		return *key, err
	}
	return *key, nil
}

func (id OrderId) toString() string {
	return base64.StdEncoding.EncodeToString(id.bytes())
}

func (id OrderId) bytes() []byte {
	return id[:]
}

func (id *OrderId) setBytes(value []byte) error {
	if len(value) != OrderIdBytesLength {
		return fmt.Errorf("invalid OrderId length error %d", len(value))
	}
	copy(id[:], value)
	return nil
}

type Order struct {
	dexproto.Order
}

func (od *Order) Serialize() ([]byte, error) {
	od.MarketId = 0
	od.Side = false
	od.Price = nil
	od.Timestamp = 0
	return proto.Marshal(&od.Order)
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
