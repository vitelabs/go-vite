package chain

import "github.com/vitelabs/go-vite/common/types"

func (c *chain) GetLatestBlockEventId() (uint64, error) {
	return c.ChainDb().Be.LatestEventId()
}

func (c *chain) GetEvent(eventId uint64) (byte, []types.Hash, error) {
	return c.ChainDb().Be.GetEvent(eventId)
}
