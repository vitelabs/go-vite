package chain_state

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/chain/block"
	"github.com/vitelabs/go-vite/chain/utils"
)

func (sDB *StateDB) CheckAndDelete(toLocation *chain_block.Location) error {

	snapshotBlock, err := sDB.chain.QueryLatestSnapshotBlock()
	if err != nil {
		return err
	}

	batch := new(leveldb.Batch)

	location, err := sDB.undo(batch, snapshotBlock)
	if err != nil {
		return err
	}
	if location != nil {
		sDB.updateUndoLocation(batch, location)
	}
	sDB.updateStateDbLocation(batch, toLocation)

	if err := sDB.db.Write(batch, nil); err != nil {
		return err
	}

	if location != nil {
		if err := sDB.undoLogger.DeleteTo(location); err != nil {
			return err
		}
	}

	return nil
}

func (sDB *StateDB) CheckAndRepair() error {
	value, err := sDB.db.Get(chain_utils.CreateUndoLocationKey(), nil)
	if err != nil {
		return err
	}

	if len(value) <= 0 {
		return nil
	}

	location := chain_utils.DeserializeLocation(value)
	if sDB.undoLogger.CompareLocation(location) > 0 {
		if err := sDB.undoLogger.DeleteTo(location); err != nil {
			return err
		}
	}
	return nil
}
