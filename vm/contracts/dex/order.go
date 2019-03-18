package dex

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"github.com/golang/protobuf/proto"
	orderproto "github.com/vitelabs/go-vite/vm/contracts/dex/proto"
	"math/big"
)

const OrderIdLength = 20
const (
	Pending = iota
	PartialExecuted
	FullyExecuted
	Cancelled
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

var orderStorageSalt = []byte("order:")

const (
	Limited = iota
	Market
)

type Order struct {
	orderproto.Order
}

type TakerOrder struct {
	orderproto.OrderInfo
}

var nilOrderIdValue = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
var maxOrderIdValue = []byte{255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255}

type OrderId [OrderIdLength]byte

func NewOrderId(value []byte) (OrderId, error) {
	key := &OrderId{}
	if err := key.setBytes(value); err != nil {
		return *key, err
	}
	return *key, nil
}

func (id OrderId) getStorageKey() []byte {
	return append(orderStorageSalt, id[:]...)
}

func (id OrderId) isNilKey() bool {
	return bytes.Compare(id[:], nilOrderIdValue) == 0
}

func (id OrderId) isBarrierKey() bool {
	return bytes.Compare(id[:], maxOrderIdValue) == 0
}

func (id OrderId) equals(counter nodeKeyType) bool {
	ct := counter.(OrderId)
	return bytes.Compare(id[:], ct.bytes()) == 0
}

func (id OrderId) toString() string {
	return base64.StdEncoding.EncodeToString(id.bytes())
}

func (id OrderId) IsNormal() bool {
	return !id.isNilKey() && !id.isBarrierKey()
}

func (id OrderId) bytes() []byte {
	return id[:]
}

func (id *OrderId) setBytes(value []byte) error {
	if len(value) != OrderIdLength {
		return fmt.Errorf("invalid OrderId length error %d", len(value))
	}
	copy(id[:], value)
	return nil
}

type OrderNodeProtocol struct{}

func (protocol *OrderNodeProtocol) getNilKey() nodeKeyType {
	key, _ := NewOrderId(nilOrderIdValue)
	return nodeKeyType(key)
}

func (protocol *OrderNodeProtocol) getBarrierKey() nodeKeyType {
	key, _ := NewOrderId(maxOrderIdValue)
	return nodeKeyType(key)
}

func (protocol *OrderNodeProtocol) serialize(node *skiplistNode) ([]byte, error) {
	protoNode := &orderproto.OrderNode{}
	protoNode.ForwardOnLevel = convertKeyOnLevelToProto(node.forwardOnLevel)
	protoNode.BackwardOnLevel = convertKeyOnLevelToProto(node.backwardOnLevel)
	pl := *node.payload
	order, _ := pl.(Order)
	protoNode.Order = &order.Order
	return proto.Marshal(protoNode)
}

func (protocol *OrderNodeProtocol) deSerialize(nodeData []byte) (*skiplistNode, error) {
	orderNode := &orderproto.OrderNode{}
	if err := proto.Unmarshal(nodeData, orderNode); err != nil {
		return nil, err
	} else {
		node := &skiplistNode{}
		node.nodeKey, _ = NewOrderId(orderNode.Order.Id)
		order := Order{}
		order.Order = *orderNode.Order
		od := nodePayload(order)
		node.payload = &od
		node.forwardOnLevel = convertKeyOnLevelFromProto(orderNode.ForwardOnLevel)
		node.backwardOnLevel = convertKeyOnLevelFromProto(orderNode.BackwardOnLevel)
		return node, nil
	}
}

func (protocol *OrderNodeProtocol) serializeMeta(meta *skiplistMeta) ([]byte, error) {
	protoMeta := &orderproto.OrderListMeta{}
	protoMeta.Header = meta.header.(OrderId).bytes()
	protoMeta.Tail = meta.tail.(OrderId).bytes()
	protoMeta.Length = meta.length
	protoMeta.Level = int32(meta.level)
	protoMeta.ForwardOnLevel = make([][]byte, len(meta.forwardOnLevel))
	for i, v := range meta.forwardOnLevel {
		protoMeta.ForwardOnLevel[i] = v.(OrderId).bytes()
	}
	return proto.Marshal(protoMeta)
}

func (protocol *OrderNodeProtocol) deSerializeMeta(nodeData []byte) (*skiplistMeta, error) {
	orderMeta := &orderproto.OrderListMeta{}
	if err := proto.Unmarshal(nodeData, orderMeta); err != nil {
		return nil, err
	} else {
		meta := &skiplistMeta{}
		meta.header, _ = NewOrderId(orderMeta.Header)
		meta.tail, _ = NewOrderId(orderMeta.Tail)
		meta.length = orderMeta.Length
		meta.level = int8(orderMeta.Level)
		meta.forwardOnLevel = make([]nodeKeyType, len(orderMeta.ForwardOnLevel))
		for i, v := range orderMeta.ForwardOnLevel {
			meta.forwardOnLevel[i], _ = NewOrderId(v)
		}
		return meta, nil
	}
}

func priceEqual(a string, b string) bool {
	af, _ := new(big.Float).SetString(a)
	bf, _ := new(big.Float).SetString(b)
	return af.Cmp(bf) == 0
}

func convertKeyOnLevelToProto(from []nodeKeyType) [][]byte {
	to := make([][]byte, len(from))
	for i, v := range from {
		to[i] = v.(OrderId).bytes()
	}
	return to
}

func convertKeyOnLevelFromProto(from [][]byte) []nodeKeyType {
	to := make([]nodeKeyType, len(from))
	for i, v := range from {
		orderKey, _ := NewOrderId(v)
		to[i] = nodeKeyType(orderKey)
	}
	return to
}

// orders should sort as desc by price and timestamp
func (order Order) compareTo(toPayload *nodePayload) int8 {
	target, _ := (*toPayload).(Order)
	return CompareOrderPrice(order, target)
}

// orders should sort as desc by price and timestamp
func (order Order) randSeed() int64 {
	return int64(binary.BigEndian.Uint64(order.Id[12:20]))
}

func CompareOrderPrice(order Order, target Order) int8 {
	var result int8
	if priceEqual(order.GetPrice(), target.GetPrice()) {
		//if order.GetTimestamp() == target.GetTimestamp() {
		//	result = 0
		//} else if order.GetTimestamp() > target.GetTimestamp() {
		//	result = -1
		//} else {
		//	result = 1
		//}
		return 0
	} else {
		cp, _ := new(big.Float).SetString(order.Price)
		tp, _ := new(big.Float).SetString(target.Price)
		if cp.Cmp(tp) > 0 {
			switch order.GetSide() {
			case false: // bid/buy
				result = 1
			case true: // ask/sell
				result = -1
			}
		} else {
			switch target.GetSide() {
			case false:
				result = -1
			case true:
				result = 1
			}
		}
	}
	return result
}
