package chain_block

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/ledger"
	"path"
)

type BlockDB struct {
	fm *fileManager
}

type SnapshotSegment struct {
	SnapshotBlock *ledger.SnapshotBlock
	AccountBlocks []*ledger.AccountBlock
}

func NewBlockDB(chainDir string) (*BlockDB, error) {
	fm, err := newFileManager(path.Join(chainDir, "blocks"))
	if err != nil {
		return nil, err
	}

	return &BlockDB{
		fm: fm,
	}, nil
}

func (bDB *BlockDB) Destroy() {}

func (bDB *BlockDB) Write(ss *SnapshotSegment) ([]*Location, *Location, error) {
	accountBlocksLocation := make([]*Location, 0, len(ss.AccountBlocks))

	for _, accountBlock := range ss.AccountBlocks {
		buf, err := accountBlock.Serialize()
		if err != nil {
			return nil, nil, errors.New(fmt.Sprintf("ss.AccountBlocks.Serialize failed, error is %s, accountBlock is %+v", err.Error(), accountBlock))
		}
		if location, err := bDB.fm.Write(buf); err != nil {
			return nil, nil, errors.New(fmt.Sprintf("bDB.fm.Write failed, error is %s, accountBlock is %+v", err.Error(), accountBlock))
		} else {
			accountBlocksLocation = append(accountBlocksLocation, location)
		}
	}

	buf, err := ss.SnapshotBlock.Serialize()
	if err != nil {
		return nil, nil, errors.New(fmt.Sprintf("ss.SnapshotBlock.Serialize failed, error is %s, snapshotBlock is %+v", err.Error(), ss.SnapshotBlock))
	}
	snapshotBlockLocation, err := bDB.fm.Write(buf)
	if err != nil {
		return nil, nil, errors.New(fmt.Sprintf("bDB.fm.Write failed, error is %s, snapshotBlock is %+v", err.Error(), ss.SnapshotBlock))
	}
	return accountBlocksLocation, snapshotBlockLocation, nil
}

func (bDB *BlockDB) DeleteTo(location *Location) ([]*SnapshotSegment, []*ledger.AccountBlock, error) {
	// bDB.fm.DeleteTo(location)
	return nil, nil, nil
}
