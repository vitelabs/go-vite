package chain_block

import (
	"fmt"
	"github.com/golang/snappy"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/ledger"
	"path"
	"sync"
)

type BlockDB struct {
	fm *fileManager

	snappyWriteBuffer []byte
	wg                sync.WaitGroup
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
		fm:                fm,
		snappyWriteBuffer: make([]byte, 10*1024*1024),
	}, nil
}

func (bDB *BlockDB) Stop() {
	bDB.wg.Wait()
}

func (bDB *BlockDB) CleanAllData() error {
	return bDB.CleanAllData()
}

func (bDB *BlockDB) Destroy() error {
	if err := bDB.fm.Close(); err != nil {
		return errors.New(fmt.Sprintf("bDB.fm.Close failed, error is %s", err))
	}

	bDB.fm = nil
	return nil
}

func (bDB *BlockDB) Write(ss *SnapshotSegment) ([]*Location, *Location, error) {
	accountBlocksLocation := make([]*Location, 0, len(ss.AccountBlocks))

	for _, accountBlock := range ss.AccountBlocks {
		buf, err := accountBlock.Serialize()
		if err != nil {
			return nil, nil, errors.New(fmt.Sprintf("ss.AccountBlocks.Serialize failed, error is %s, accountBlock is %+v", err.Error(), accountBlock))
		}
		sBuf := snappy.Encode(bDB.snappyWriteBuffer, buf)
		if location, err := bDB.fm.Write(sBuf); err != nil {
			return nil, nil, errors.New(fmt.Sprintf("bDB.fm.Write failed, error is %s, accountBlock is %+v", err.Error(), accountBlock))
		} else {
			accountBlocksLocation = append(accountBlocksLocation, location)
		}
	}

	buf, err := ss.SnapshotBlock.Serialize()
	if err != nil {
		return nil, nil, errors.New(fmt.Sprintf("ss.SnapshotBlock.Serialize failed, error is %s, snapshotBlock is %+v", err.Error(), ss.SnapshotBlock))
	}
	sBuf := snappy.Encode(bDB.snappyWriteBuffer, buf)
	snapshotBlockLocation, err := bDB.fm.Write(sBuf)

	if err != nil {
		return nil, nil, errors.New(fmt.Sprintf("bDB.fm.Write failed, error is %s, snapshotBlock is %+v", err.Error(), ss.SnapshotBlock))
	}
	return accountBlocksLocation, snapshotBlockLocation, nil
}

func (bDB *BlockDB) Read(location *Location) ([]byte, error) {
	buf, err := bDB.fm.Read(location)
	if err != nil {
		return nil, err
	}
	sBuf, err := snappy.Decode(nil, buf)
	if err != nil {
		return nil, err
	}
	return sBuf, nil
}

func (bDB *BlockDB) DeleteTo(location *Location, prevLocation *Location) ([]*SnapshotSegment, []*ledger.AccountBlock, error) {
	// bDB.fm.DeleteTo(location)
	bfp := newBlockFileParser()
	readBfp := newBlockFileParser()

	bDB.wg.Add(1)
	go func() {
		defer bDB.wg.Wait()
		bDB.fm.DeleteTo(location, bfp)
		bDB.fm.ReadRange(prevLocation, location, readBfp)
	}()

	var segList []*SnapshotSegment
	var seg *SnapshotSegment
	var snappyReadBuffer = make([]byte, 0, 1024*1024) // 1M
LOOP:
	for {
		select {
		case buf, ok := <-bfp.Next():
			if !ok {
				break LOOP
			}
			if buf.BlockType == BlockTypeSnapshotBlock {
				if seg != nil {
					segList = append(segList, seg)
				}

				sBuf := snappy.Encode(snappyReadBuffer, buf.Buffer)
				sb := &ledger.SnapshotBlock{}
				if err := sb.Deserialize(sBuf); err != nil {
					return nil, nil, err
				}
				seg = &SnapshotSegment{
					SnapshotBlock: sb,
				}
			} else if buf.BlockType == BlockTypeAccountBlock {
				if seg == nil {
					seg = &SnapshotSegment{}
				}

				sBuf := snappy.Encode(snappyReadBuffer, buf.Buffer)
				ab := &ledger.AccountBlock{}
				if err := ab.Deserialize(sBuf); err != nil {
					return nil, nil, err
				}
				seg.AccountBlocks = append(seg.AccountBlocks, ab)
			}
		case err := <-bfp.Error():
			return nil, nil, err
		}

	}

	var accountBlockList []*ledger.AccountBlock
LOOP2:
	for {
		select {
		case buf, ok := <-bfp.Next():
			if !ok {
				break LOOP2
			}
			if buf.BlockType == BlockTypeAccountBlock {
				sBuf := snappy.Encode(snappyReadBuffer, buf.Buffer)
				ab := &ledger.AccountBlock{}
				if err := ab.Deserialize(sBuf); err != nil {
					return nil, nil, err
				}
				accountBlockList = append(accountBlockList, ab)
			}

		case err := <-bfp.Error():
			return nil, nil, err
		}
	}
	return segList, accountBlockList, nil
}
