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
	chunks, err := c.GetSubLedgerAfterHeight(height)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.GetSubLedgerAfterHeight failed, Height is %d. Error: %s", height, err.Error()))
		c.log.Error(cErr.Error(), "method", "recoverUnconfirmedCache")
		return cErr
	}
	if len(chunks) <= 0 || len(chunks[0].AccountBlocks) <= 0 {
		return nil
	}

	c.flusher.Pause()

	// record rollback location
	if _, err := c.blockDB.Rollback(nextLocation); err != nil {
		cErr := errors.New(fmt.Sprintf("c.blockDB.Rollback failed, location is %+v. Error: %s", nextLocation, err.Error()))

		c.log.Error(cErr.Error(), "method", "recoverUnconfirmedCache")
		return cErr
	}

	// rebuild unconfirmed cache
	for _, chunk := range chunks {
		if chunk.SnapshotBlock != nil {
			continue
		}

		// recover unconfirmed pool
		c.cache.RecoverUnconfirmedPool(chunk.AccountBlocks)

	}

	fmt.Println("recover")
	for _, ab := range chunks[0].AccountBlocks {
		fmt.Printf("%+v\n", ab)
	}

	fmt.Println("recover end")

	return nil

}

//// TODO
//func (c *chain) recoverUnconfirmedCache() error {
//	c.flusher.Pause()
//
//	// rebuild unconfirmed cache
//	height := c.GetLatestSnapshotBlock().Height
//
//	location, err := c.indexDB.GetSnapshotBlockLocation(height)
//	if err != nil {
//		cErr := errors.New(fmt.Sprintf("c.indexDB.GetSnapshotBlockLocation failed, latestHeight is %d. Error: %s", height, err.Error()))
//		c.log.Error(cErr.Error(), "method", "recoverUnconfirmedCache")
//		return cErr
//	}
//
//	nextLocation, err := c.blockDB.GetNextLocation(location)
//	if err != nil {
//		cErr := errors.New(fmt.Sprintf("c.blockDB.GetNextLocation failed. Error: %s", err.Error()))
//		c.log.Error(cErr.Error(), "method", "recoverUnconfirmedCache")
//		return cErr
//	}
//
//	if nextLocation == nil {
//		return nil
//	}
//
//	// rollback blockDb
//	chunks, err := c.blockDB.Rollback(nextLocation)
//	if err != nil {
//		cErr := errors.New(fmt.Sprintf("c.blockDB.Rollback failed. Error: %s", err.Error()))
//		c.log.Error(cErr.Error(), "method", "recoverUnconfirmedCache")
//		return cErr
//	}
//
//	if len(chunks) <= 0 || len(chunks[0].AccountBlocks) <= 0 {
//		return nil
//	}
//
//	fmt.Println("recover")
//	for _, ab := range chunks[0].AccountBlocks {
//		fmt.Printf("%+v\n", ab)
//	}
//
//	fmt.Println("recover end")
//
//	if len(chunks) <= 0 {
//		return nil
//	}
//
//	// recover cache
//	c.cache.RecoverUnconfirmedPool(chunks[0].AccountBlocks)
//
//	return nil
//}
