package chain

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/chain/block"
)

func (c *chain) checkAndRepair() error {
	// repair block db
	if err := c.blockDB.CheckAndRepair(); err != nil {
		return errors.New(fmt.Sprintf("c.blockDB.CheckAndRepair failed. Error: %s", err))
	}

	// repair index db
	if err := c.checkAndRepairIndexDb(c.blockDB.LatestLocation()); err != nil {
		return err
	}

	//repair state db
	stateDbLatestLocation, err := c.stateDB.QueryLatestLocation()

	if err != nil {
		return errors.New(fmt.Sprintf("c.stateDB.QueryLatestLocation failed. Error: %s", err))
	}

	blockDbLatestLocation := c.blockDB.LatestLocation()

	compareResult := stateDbLatestLocation.Compare(blockDbLatestLocation)

	if compareResult > 0 {
		if err := c.stateDB.Rollback(blockDbLatestLocation); err != nil {
			return errors.New(fmt.Sprintf("c.stateDB.Rollback failed. Error: %s", err))
		}
	} else if compareResult < 0 {
		if err := c.blockDB.Rollback(stateDbLatestLocation); err != nil {
			return errors.New(fmt.Sprintf("c.blockDB.Rollback failed. Error: %s", err))
		}

		return c.checkAndRepairIndexDb(stateDbLatestLocation)
	}

	return nil
}

func (c *chain) checkAndRepairIndexDb(latestLocation *chain_block.Location) error {
	indexDbLatestLocation, err := c.indexDB.QueryLatestLocation()
	if err != nil {
		return errors.New(fmt.Sprintf("c.indexDB.QueryLatestLocation failed. Error: %s", err))
	}
	if indexDbLatestLocation == nil {
		return errors.New(fmt.Sprintf("latestLocation is nil, Error: %s", err))
	}

	compareResult := indexDbLatestLocation.Compare(latestLocation)

	if compareResult < 0 {
		segs, err := c.blockDB.ReadRange(indexDbLatestLocation, latestLocation)
		if err != nil {
			return errors.New(fmt.Sprintf("c.blockDB.ReadRange failed, startLocation is %+v, endLocation is %+v. Error: %s",
				indexDbLatestLocation, latestLocation, err))
		}
		for _, seg := range segs {
			for _, block := range seg.AccountBlocks {
				if err := c.indexDB.InsertAccountBlock(block); err != nil {
					return errors.New(fmt.Sprintf("c.indexDB.InsertAccountBlock failed, block is %+v. Error: %s",
						block, err))
				}
			}
			if seg.SnapshotBlock != nil {
				if err := c.indexDB.InsertSnapshotBlock(
					seg.SnapshotBlock,
					seg.AccountBlocks,
					seg.SnapshotBlockLocation,
					seg.AccountBlocksLocation,
					nil,
					seg.RightBoundary,
				); err != nil {
					return errors.New(fmt.Sprintf("c.indexDB.InsertSnapshotBlock failed. Error: %s",
						err))
				}
			}
		}
	} else if compareResult > 0 {
		if err := c.indexDB.Rollback(latestLocation); err != nil {
			return errors.New(fmt.Sprintf("c.indexDB.Rollback failed, location is %+v. Error: %s",
				latestLocation, err))
		}
	}
	return nil
}
