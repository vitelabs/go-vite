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
		cErr := errors.New(fmt.Sprintf("c.GetSubLedgerAfterHeight failed, Height is %d. Error: %s", height, err.Error()))
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
	//chunks, err := c.GetSubLedgerAfterHeight(height)
	//if err != nil {
	//	cErr := errors.New(fmt.Sprintf("c.GetSubLedgerAfterHeight failed, Height is %d. Error: %s", height, err.Error()))
	//	c.log.Error(cErr.Error(), "method", "recoverUnconfirmedCache")
	//	return cErr
	//}
	if len(chunks) <= 0 || len(chunks[0].AccountBlocks) <= 0 {
		return nil
	}

	//startLocation, err := c.indexDB.GetAccountBlockLocationByHash(&chunks[0].AccountBlocks[0].Hash)
	//if err != nil {
	//	cErr := errors.New(fmt.Sprintf("c.indexDB.GetAccountBlockLocationByHash failed. Error: %s", err.Error()))
	//	c.log.Error(cErr.Error(), "method", "recoverUnconfirmedCache")
	//	return cErr
	//}

	// rollback index db
	if err := c.indexDB.Rollback(chunks); err != nil {
		cErr := errors.New(fmt.Sprintf("c.indexDB.Rollback failed. Error: %s", err.Error()))
		c.log.Error(cErr.Error(), "method", "recoverUnconfirmedCache")
		return cErr
	}

	// rollback state db

	if err := c.stateDB.Rollback(chunks); err != nil {
		cErr := errors.New(fmt.Sprintf("c.stateDB.Rollback failed. Error: %s", err.Error()))
		c.log.Error(cErr.Error(), "method", "recoverUnconfirmedCache")
		return cErr
	}
	for _, chunk := range chunks {
		if chunk.SnapshotBlock != nil {
			continue
		}
		if len(chunk.AccountBlocks) > 0 {
			// recover unconfirmed pool
			c.cache.RecoverAccountBlocks(chunk.AccountBlocks)

			// recover index db
			for _, accountBlock := range chunk.AccountBlocks {
				c.indexDB.InsertAccountBlock(accountBlock)

			}
		}
	}

	// rollback blockDb
	//if _, err := c.blockDB.Rollback(startLocation); err != nil {
	//
	//}

	c.flusher.Flush()
	return nil
}

//func (c *chain) checkAndRepair() error {
//	// repair block db
//	if err := c.blockDB.CheckAndRepair(); err != nil {
//		return errors.New(fmt.Sprintf("c.blockDB.CheckAndRepair failed. Error: %s", err))
//	}
//
//	// repair index db
//	if err := c.checkAndRepairIndexDb(c.blockDB.LatestLocation()); err != nil {
//		return err
//	}
//
//	//repair state db
//	if err := c.stateDB.CheckAndRepair(); err != nil {
//		return errors.New(fmt.Sprintf("c.stateDB.CheckAndRepair failed. Error: %s", err))
//	}
//
//	stateDbLatestLocation, err := c.stateDB.QueryLatestLocation()
//
//	if err != nil {
//		return errors.New(fmt.Sprintf("c.stateDB.QueryLatestLocation failed. Error: %s", err))
//	}
//
//	blockDbLatestLocation := c.blockDB.LatestLocation()
//
//	compareResult := stateDbLatestLocation.Compare(blockDbLatestLocation)
//
//	if compareResult > 0 {
//		if err := c.stateDB.CheckAndDelete(blockDbLatestLocation); err != nil {
//			return errors.New(fmt.Sprintf("c.stateDB.DeleteTo failed. Error: %s", err))
//		}
//	} else if compareResult < 0 {
//		if err := c.blockDB.DeleteTo(stateDbLatestLocation); err != nil {
//			return errors.New(fmt.Sprintf("c.blockDB.DeleteTo failed. Error: %s", err))
//		}
//
//		return c.checkAndRepairIndexDb(stateDbLatestLocation)
//	}
//
//	return nil
//}
//
//func (c *chain) checkAndRepairIndexDb(latestLocation *chain_file_manager.Location) error {
//	indexDbLatestLocation, err := c.indexDB.QueryLatestLocation()
//	if err != nil {
//		return errors.New(fmt.Sprintf("c.indexDB.QueryLatestLocation failed. Error: %s", err))
//	}
//	if indexDbLatestLocation == nil {
//		return errors.New(fmt.Sprintf("latestLocation is nil, Error: %s", err))
//	}
//
//	compareResult := indexDbLatestLocation.Compare(latestLocation)
//
//	if compareResult < 0 {
//		segs, err := c.blockDB.ReadRange(indexDbLatestLocation, latestLocation)
//		if err != nil {
//			return errors.New(fmt.Sprintf("c.blockDB.ReadRange failed, startLocation is %+v, endLocation is %+v. Error: %s",
//				indexDbLatestLocation, latestLocation, err))
//		}
//		for _, seg := range segs {
//			for _, block := range seg.AccountBlocks {
//				if err := c.indexDB.InsertAccountBlock(block); err != nil {
//					return errors.New(fmt.Sprintf("c.indexDB.InsertAccountBlock failed, block is %+v. Error: %s",
//						block, err))
//				}
//			}
//			if seg.SnapshotBlock != nil {
//				if err := c.indexDB.InsertSnapshotBlock(
//					seg.SnapshotBlock,
//					seg.AccountBlocks,
//					seg.SnapshotBlockLocation,
//					seg.AccountBlocksLocation,
//					nil,
//					seg.RightBoundary,
//				); err != nil {
//					return errors.New(fmt.Sprintf("c.indexDB.InsertSnapshotBlock failed. Error: %s",
//						err))
//				}
//			}
//		}
//	} else if compareResult > 0 {
//		if err := c.indexDB.DeleteTo(latestLocation); err != nil {
//			return errors.New(fmt.Sprintf("c.indexDB.DeleteTo failed, location is %+v. Error: %s",
//				latestLocation, err))
//		}
//	}
//	return nil
//}
