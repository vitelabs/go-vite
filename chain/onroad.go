package chain

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/monitor"
	"time"
)

func (c *chain) IsSuccessReceived(addr *types.Address, hash *types.Hash) bool {
	monitorTags := []string{"chain", "IsSuccessReceived"}
	defer monitor.LogTimerConsuming(monitorTags, time.Now())

	if meta, _ := c.ChainDb().OnRoad.GetMeta(addr, hash); meta != nil {
		return false
	}
	return true
}
