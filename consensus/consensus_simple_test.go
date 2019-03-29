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
	assert.True(t, stime.Add(time.Duration(info.PlanInterval)).Equal(etime))

	result, err := cs.ElectionIndex(0)
	assert.Nil(t, err)

	t.Log(result)

}
