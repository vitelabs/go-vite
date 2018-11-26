package dex

import (
	"github.com/stretchr/testify/assert"
	"strconv"
	"testing"
)

func TestSerialize(t *testing.T) {
	order := Order{}
	order.Address = []byte("123")
	order.Price = "10.0"
	order.Timestamp = 10000
	order.Side = false
	node := &skiplistNode{}
	pl := nodePayload(order)
	node.payload = &pl

	protocol := &OrderNodeProtocol{}
	data := protocol.serialize(node)
	res := protocol.deSerialize(data)
	od := (*res.payload).(Order)
	assert.Equal(t, od.Address, order.Address)
	assert.Equal(t, od.Price, order.Price)
	assert.Equal(t, od.Timestamp, order.Timestamp)
	assert.Equal(t, od.Side, order.Side)
}

func TestCompare(t *testing.T) {
	_, payload1 := newNodeInfo(1)
	_, payload2 := newNodeInfo(2)
	assert.Equal(t, int8(-1), (*payload1).(Order).compareTo(payload2))
	assert.Equal(t, int8(1), (*payload2).(Order).compareTo(payload1))

	_, payload21 := newNodeInfoWithPrice(2,"2.000000009")
	assert.Equal(t, int8(1), (*payload21).(Order).compareTo(payload2))

	_, payload22 := newNodeInfoWithPrice(2,"2.00000001")
	assert.Equal(t, int8(1), (*payload22).(Order).compareTo(payload2))

	_, payload29 := newNodeInfoWithTs(2, 3)
	assert.Equal(t, int8(-1), (*payload29).(Order).compareTo(payload2))
}

func newNodeInfo(value int) (nodeKeyType, *nodePayload) {
	return newNodeInfoWithPriceAndTs(value, strconv.Itoa(value), uint64(value))
}

func newNodeInfoWithPrice(value int, price string) (nodeKeyType, *nodePayload) {
	return newNodeInfoWithPriceAndTs(value, price, uint64(value))
}

func newNodeInfoWithTs(value int, ts uint64) (nodeKeyType, *nodePayload) {
	return newNodeInfoWithPriceAndTs(value, strconv.Itoa(value), ts)
}

func newNodeInfoWithPriceAndTs(value int, price string, ts uint64) (nodeKeyType, *nodePayload) {
	order := Order{}
	order.Id = uint64(value)
	order.Price = price
	order.Timestamp = int64(ts)
	order.Side = false // buy
	pl := nodePayload(order)
	return newOrderKey(order.Id), &pl
}