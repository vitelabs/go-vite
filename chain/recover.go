package chain

import (
	"errors"
	"fmt"
)

// TODO
func (c *chain) recoverUnconfirmedCache() error {

	// rebuild unconfirmed cache
	height := c.GetLatestSnapshotBlock().Height

	location, err := c.indexDB.GetSnapshotBlockLocation(height)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.indexDB.GetSnapshotBlockLocation failed, latestHeight is %d. Error: %s", height, err.Error()))
		c.log.Error(cErr.Error(), "method", "recoverUnconfirmedCache")
		return cErr
	}

	nextLocation, err := c.blockDB.GetNextLocation(location)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.blockDB.GetNextLocation failed. Error: %s", err.Error()))
		c.log.Error(cErr.Error(), "method", "recoverUnconfirmedCache")
		return cErr
	}

	if nextLocation == nil {
		return nil
	}

	// rollback blockDb
	chunks, err := c.blockDB.Rollback(nextLocation)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.blockDB.Rollback failed. Error: %s", err.Error()))
		c.log.Error(cErr.Error(), "method", "recoverUnconfirmedCache")
		return cErr
	}

	if len(chunks) <= 0 {
		return nil
	}

	// recover cache
	c.cache.RecoverUnconfirmedPool(chunks[0].AccountBlocks)

	return nil
}
