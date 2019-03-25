package chain_block

import (
	"github.com/golang/snappy"
	"github.com/vitelabs/go-vite/ledger"
)

func (bDB *BlockDB) CheckAndRepair() error {
	latestLocation := bDB.fm.LatestLocation()
	startLocation := NewLocation(latestLocation.FileId(), 0)
	bfp := newBlockFileParser()

	bDB.wg.Add(1)
	go func() {
		defer bDB.wg.Done()
		bDB.fm.ReadRange(startLocation, latestLocation, bfp)
	}()

	snappyReadBuffer := make([]byte, 0, 1024*1024) // 1M
	iterator := bfp.Iterator()

	currentOffset := int64(0)
	for buf := range iterator {
		sBuf, err := snappy.Decode(snappyReadBuffer, buf.Buffer)
		if err != nil {
			break
		}
		sb := &ledger.SnapshotBlock{}
		if err := sb.Deserialize(sBuf); err != nil {
			break
		}

		if buf.BlockType == BlockTypeSnapshotBlock {

			sb := &ledger.SnapshotBlock{}
			if err := sb.Deserialize(sBuf); err != nil {
				break
			}

		} else if buf.BlockType == BlockTypeAccountBlock {
			ab := &ledger.AccountBlock{}
			if err := ab.Deserialize(sBuf); err != nil {
				break
			}
		}
		currentOffset += buf.Size
	}

	if currentOffset < latestLocation.Offset() {
		if err := bDB.fm.DeleteTo(NewLocation(startLocation.FileId(), currentOffset)); err != nil {
			return err
		}
	}
	return nil

}

func (bDB *BlockDB) DeleteTo(location *Location) error {
	return bDB.fm.DeleteTo(location)
}
