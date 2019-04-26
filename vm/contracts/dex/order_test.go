package dex

import (
	"encoding/binary"
	"encoding/hex"
	"github.com/stretchr/testify/assert"
	"strconv"
	"testing"
)

func TestSerialize(t *testing.T) {
	order := Order{}
	order.Id = orderIdBytesFromInt(100)
	order.Address = []byte("123")
	order.Price = PriceToBytes("10.0")
	//order.Timestamp = 10000
	order.Side = false
	node := &skiplistNode{}
	pl := nodePayload(order)
	node.payload = &pl

	protocol := &OrderNodeProtocol{}
	data, _ := protocol.serialize(node)
	res, _ := protocol.deSerialize(data)
	od := (*res.payload).(Order)
	assert.Equal(t, od.Address, order.Address)
	assert.Equal(t, od.Price, order.Price)
	//assert.Equal(t, od.Timestamp, order.Timestamp)
	assert.Equal(t, od.Side, order.Side)

	nilOrderIdBytes, err := hex.DecodeString("0000000000000000000000000000000000000000")
	assert.Equal(t, nil, err)
	nilOrderId, err := NewOrderId(nilOrderIdBytes)
	assert.False(t, nilOrderId.IsNormal())
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
	assert.Equal(t, int8(0), (*payload29).(Order).compareTo(payload2))
}

func newNodeInfo(value int) (nodeKeyType, *nodePayload) {
	return newNodeInfoWithPriceAndTs(value, strconv.Itoa(int(value)), uint64(value))
}

func newNodeInfoWithPrice(value int, price string) (nodeKeyType, *nodePayload) {
	return newNodeInfoWithPriceAndTs(value, price, uint64(value))
}

func newNodeInfoWithTs(value int, ts uint64) (nodeKeyType, *nodePayload) {
	return newNodeInfoWithPriceAndTs(value, strconv.Itoa(int(value)), ts)
}

func newNodeInfoWithPriceAndTs(value int, price string, ts uint64) (nodeKeyType, *nodePayload) {
	order := Order{}
	order.Id = orderIdBytesFromInt(value)
	order.Price = PriceToBytes(price)
	//order.Timestamp = int64(ts)
	order.Side = false // buy
	pl := nodePayload(order)
	orderId, _ := NewOrderId(order.Id)
	return orderId, &pl
}

func orderIdBytesFromInt(v int) []byte {
	bs := make([]byte, 4)
	binary.BigEndian.PutUint32(bs, uint32(v))
	res := append([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, bs...)
	return res
}

