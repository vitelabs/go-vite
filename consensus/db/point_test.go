package consensus_db

import (
	"testing"

	"github.com/magiconair/properties/assert"
)

func TestContent_Copy(t *testing.T) {
	point := &Content{ExpectedNum: 13, FactualNum: 11}

	copy := point.Copy()

	copy.FactualNum = 10

	t.Log(copy.FactualNum, point.FactualNum)

	assert.Equal(t, copy.FactualNum, uint32(10))
	assert.Equal(t, point.FactualNum, uint32(11))
}
