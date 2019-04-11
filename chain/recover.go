package chain

import (
	"errors"
	"fmt"
)

func (c *chain) recoverUnconfirmedCache() error {

	// rebuild unconfirmed cache
	height := c.GetLatestSnapshotBlock().Height
	location, err := c.indexDB.GetSnapshotBlockLocation(height)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.indexDB.GetSnapshotBlockLocation failed, GetLatestHeight is %d. Error: %s", height, err.Error()))
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
		cErr := errors.New(fmt.Sprintf("c.blockDB.RollbackAccountBlocks failed. Error: %s", err.Error()))
		c.log.Error(cErr.Error(), "method", "recoverUnconfirmedCache")
		return cErr
	}

	if len(chunks) <= 0 || len(chunks[0].AccountBlocks) <= 0 {
		return nil
	}

	// rebuild unconfirmed cache
	for _, chunk := range chunks {
		if chunk.SnapshotBlock != nil {
			continue
		}

		// recover unconfirmed pool
		c.cache.RecoverUnconfirmedPool(chunk.AccountBlocks)
	}

	return nil

}
