package dex

import (
	"github.com/golang/protobuf/proto"
	orderproto "github.com/vitelabs/go-vite/vm/contracts/dex/proto"
	"math"
	"math/big"
	"strconv"
	)

const orderStorageSalt = "order:"
const orderHeaderValue = math.MaxUint64
//const MinPricePermit = 0.000000009
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

const (
	Limited = iota
	Market
)

type Order struct {
	orderproto.Order
}

type orderKey struct {
	value uint64
}

func newOrderKey(value uint64) orderKey {
	return orderKey{value}
}

func (key orderKey) getStorageKey() []byte {
	return []byte(orderStorageSalt + strconv.Itoa(int(key.value)))
}

func (key orderKey) isNil() bool {
	return key.value == 0
}

func (key orderKey) isHeader() bool {
	return key.value == orderHeaderValue
}

func (key orderKey) equals(counter nodeKeyType) bool {
	ct, _ := counter.(orderKey)
	return ct.value == key.value
}

func (key orderKey) toString() string {
	return strconv.Itoa(int(key.value))
}

type OrderNodeProtocol struct {}

func (protocol *OrderNodeProtocol) getNilKey() nodeKeyType {
	key := newOrderKey(0)
	return nodeKeyType(key)
}

func (protocol *OrderNodeProtocol) getHeaderKey() nodeKeyType {
	key := newOrderKey(orderHeaderValue)
	return nodeKeyType(key)
}

func (protocol *OrderNodeProtocol) serialize(node *skiplistNode) []byte  {
	protoNode := &orderproto.OrderNode{}
	protoNode.ForwardOnLevel = convertKeyOnLevelToProto(node.forwardOnLevel)
	protoNode.BackwardOnLevel = convertKeyOnLevelToProto(node.backwardOnLevel)
	pl := *node.payload
	order, _ := pl.(Order)
	protoNode.Order = &order.Order
	nodeData, _ := proto.Marshal(protoNode)
	return nodeData
}

func (protocol *OrderNodeProtocol) deSerialize(nodeData []byte) *skiplistNode {
	orderNode := &orderproto.OrderNode{}
	if err := proto.Unmarshal(nodeData, orderNode); err != nil {
		return nil
	} else {
		node := &skiplistNode{}
		node.nodeKey = newOrderKey(orderNode.Order.Id)
		order := Order{}
		order.Order = *orderNode.Order
		od := nodePayload(order)
		node.payload = &od
		node.forwardOnLevel = convertKeyOnLevelFromProto(orderNode.ForwardOnLevel)
		node.backwardOnLevel = convertKeyOnLevelFromProto(orderNode.BackwardOnLevel)
		return node
	}
}

func (protocol *OrderNodeProtocol) serializeMeta(meta *skiplistMeta) []byte {
	protoMeta := &orderproto.OrderListMeta{}
	protoMeta.Header = meta.header.(orderKey).value
	protoMeta.Tail = meta.tail.(orderKey).value
	protoMeta.Length = uint32(meta.length)
	protoMeta.Level = uint32(meta.level)
	for _, v := range meta.forwardOnLevel {
		protoMeta.ForwardOnLevel = append(protoMeta.ForwardOnLevel, v.(orderKey).value)
	}
	metaData, _ := proto.Marshal(protoMeta)
	return metaData
}

func (protocol *OrderNodeProtocol) deSerializeMeta(nodeData []byte) *skiplistMeta {
	orderMeta := &orderproto.OrderListMeta{}
	if err := proto.Unmarshal(nodeData, orderMeta); err != nil {
		return nil
	} else {
		meta := &skiplistMeta{}
		meta.header = newOrderKey(orderMeta.Header)
		meta.tail = newOrderKey(orderMeta.Tail)
		meta.length = int(orderMeta.Length)
		meta.level = int8(orderMeta.Level)
		for _, v := range orderMeta.ForwardOnLevel {
			meta.forwardOnLevel = append(meta.forwardOnLevel, newOrderKey(v))
		}
		return meta
	}
}

func priceEqual(a string, b string) bool {
	af, _ := new(big.Float).SetString(a)
	bf, _ := new(big.Float).SetString(b)
	return  af.Cmp(bf) == 0
}

func convertKeyOnLevelToProto(from []nodeKeyType) []uint64 {
	to := make([]uint64, len(from), len(from))
	for i, v := range from {
		to[i] = v.(orderKey).value
	}
	return to
}

func convertKeyOnLevelFromProto(from []uint64) []nodeKeyType {
	to := make([]nodeKeyType, len(from), len(from))
	for i, v := range from {
		to[i] = newOrderKey(v)
	}
	return to
}

// orders should sort as desc by price and timestamp
func (order Order) compareTo(toPayload *nodePayload) int8 {
	target, _:= (*toPayload).(Order)
	return CompareOrderPrice(order, target)

}

func CompareOrderPrice(order Order, target Order) int8 {
	var result int8
	if priceEqual(order.GetPrice(), target.GetPrice()) {
		if order.GetTimestamp() == target.GetTimestamp() {
			result = 0
		} else if order.GetTimestamp() > target.GetTimestamp() {
			result = -1
		} else {
			result = 1
		}
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

