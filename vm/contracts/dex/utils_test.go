package dex

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeComposeOrderId(t *testing.T) {
	orderByt, err := hex.DecodeString("00000300ffffffff4fed5fa0dfff005d94f2bd000014")
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	marketId, side, price, timestamp, err := DeComposeOrderId(orderByt)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	t.Log(marketId, side, BytesToPrice(price), timestamp)

	assert.Equal(t, marketId, int32(3), "assert market")
	assert.Equal(t, side, false, "assert side")
	assert.Equal(t, BytesToPrice(price), "176.08", "assert price")
	assert.Equal(t, timestamp, int64(1570042557), "assert timestamp")
}
