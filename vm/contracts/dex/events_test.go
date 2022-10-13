package dex

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDecodeEvent(t *testing.T) {
	event := &NewOrderEvent{}

	buf, err := base64.RawStdEncoding.DecodeString("CgIYBBIDdsf8")
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	err = event.FromBytes(buf)
	if err != nil {
		t.Error(t, err)
		t.FailNow()
	}
	assert.Equal(t, event.Order.MarketId, int32(4))
	assert.Equal(t, base64.RawStdEncoding.EncodeToString(event.TradeToken), "dsf8")
}
