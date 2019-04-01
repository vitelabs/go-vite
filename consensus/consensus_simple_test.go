package consensus

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/vitelabs/go-vite/log15"
)

func TestSimpleCs(t *testing.T) {
	cs := newSimpleCs(log15.New("unitest", "simpleCs"))

	info := cs.GetInfo()
	stime, etime := cs.Index2Time(0)
	// genesis
	assert.True(t, stime.Unix() == simpleGenesis.Unix())
	// plan interval
	assert.Equal(t, stime.Add(time.Duration(info.PlanInterval)*time.Second), etime)

	result, err := cs.ElectionIndex(0)
	assert.Nil(t, err)

	for k, v := range result.Plans {
		assert.Equal(t, simpleAddrs[k/3], v.Member)
	}

	result, err = cs.ElectionIndex(1)
	assert.Nil(t, err)

	for k, v := range result.Plans {
		assert.Equal(t, simpleAddrs[k/3], v.Member)
	}

	stime, _ = cs.Index2Time(1)
	eleTime, err := cs.ElectionTime(stime)
	eleIndex, err := cs.ElectionIndex(1)

	for k, v := range eleIndex.Plans {
		assert.Equal(t, eleTime.Plans[k], v)
	}
}
