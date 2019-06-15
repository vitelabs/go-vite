package core

import (
	"testing"
	"time"

	"github.com/magiconair/properties/assert"
)

func Test_Time2Index(t *testing.T) {
	genesis := time.Unix(1552708800, 0)
	ti := NewTimeIndex(genesis, time.Hour*24)
	t2 := time.Unix(1556251200, 0)
	index := ti.Time2Index(t2)
	assert.Equal(t, uint64(41), index)

}
