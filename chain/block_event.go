package chain

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/monitor"
	"time"
)

func (c *chain) GetLatestBlockEventId() (uint64, error) {
	monitorTags := []string{"chain", "GetLatestBlockEventId"}
	defer monitor.LogTimerConsuming(monitorTags, time.Now())

	return c.ChainDb().Be.LatestEventId()
}

func (c *chain) GetEvent(eventId uint64) (byte, []types.Hash, error) {
	monitorTags := []string{"chain", "GetEvent"}
	defer monitor.LogTimerConsuming(monitorTags, time.Now())

	return c.ChainDb().Be.GetEvent(eventId)
}
