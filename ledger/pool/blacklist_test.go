package pool

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/vitelabs/go-vite/common"
)

func TestBlacklist_AddAddTimeout(t *testing.T) {

	bl, err := NewBlacklist()
	if err != nil {
		assert.Fail(t, err.Error())
	}
	hash := common.MockHash(10)
	bl.AddAddTimeout(hash, time.Second*5)

	assert.True(t, bl.Exists(hash))

	time.Sleep(7 * time.Second)
	assert.False(t, bl.Exists(hash))

	bl.AddAddTimeout(hash, time.Second*5)
	assert.True(t, bl.Exists(hash))
}
