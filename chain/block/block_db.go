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
	SnapshotBlock         *ledger.SnapshotBlock
	SnapshotBlockLocation *Location // optional

	AccountBlocks         []*ledger.AccountBlock
	AccountBlocksLocation []*Location // optional

	RightBoundary *Location // optional
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
	return bDB.fm.RemoveAllFile()
}

func (bDB *BlockDB) Destroy() error {
	if err := bDB.fm.Close(); err != nil {
		return errors.New(fmt.Sprintf("bDB.fm.Close failed, error is %s", err))
	}

	bDB.fm = nil
	return nil
}

func (bDB *BlockDB) LatestLocation() *Location {
	return bDB.fm.LatestLocation()
}

func (bDB *BlockDB) Write(ss *SnapshotSegment) ([]*Location, *Location, error) {
	accountBlocksLocation := make([]*Location, 0, len(ss.AccountBlocks))

	for _, accountBlock := range ss.AccountBlocks {
		buf, err := accountBlock.Serialize()
		if err != nil {
			return nil, nil, errors.New(fmt.Sprintf("ss.AccountBlocks.Serialize failed, error is %s, accountBlock is %+v", err.Error(), accountBlock))
		}

		if location, err := bDB.fm.Write(makeWriteBytes(bDB.snappyWriteBuffer, BlockTypeAccountBlock, buf)); err != nil {
			return nil, nil, errors.New(fmt.Sprintf("bDB.fm.Write failed, error is %s, accountBlock is %+v", err.Error(), accountBlock))
		} else {
			accountBlocksLocation = append(accountBlocksLocation, location)
		}
	}

	buf, err := ss.SnapshotBlock.Serialize()
	if err != nil {
		return nil, nil, errors.New(fmt.Sprintf("ss.SnapshotBlock.Serialize failed, error is %s, snapshotBlock is %+v", err.Error(), ss.SnapshotBlock))
	}

	snapshotBlockLocation, err := bDB.fm.Write(makeWriteBytes(bDB.snappyWriteBuffer, BlockTypeSnapshotBlock, buf))

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
	sBuf, err := snappy.Decode(nil, buf[1:])
	if err != nil {
		return nil, err
	}
	return sBuf, nil
}

func (bDB *BlockDB) ReadRange(startLocation *Location, endLocation *Location) ([]*SnapshotSegment, error) {
	bfp := newBlockFileParser()
	bDB.wg.Add(1)
	go func() {
		defer bDB.wg.Done()
		bDB.fm.ReadRange(startLocation, endLocation, bfp)
	}()

	var segList []*SnapshotSegment
	var seg *SnapshotSegment
	var snappyReadBuffer = make([]byte, 0, 1024*1024) // 1M
	iterator := bfp.Iterator()

	currentFileId := uint64(0)
	currentOffset := int64(0)

	for buf := range iterator {
		if seg == nil {
			seg = &SnapshotSegment{}
		}

		if buf.FileId != currentFileId {
			currentFileId = buf.FileId
			currentOffset = 0
		}

		sBuf, err := snappy.Decode(snappyReadBuffer, buf.Buffer)
		if err != nil {
			return nil, err
		}
		if buf.BlockType == BlockTypeSnapshotBlock {

			sb := &ledger.SnapshotBlock{}
			if err := sb.Deserialize(sBuf); err != nil {
				return nil, err
			}
			seg.SnapshotBlock = sb
			seg.SnapshotBlockLocation = NewLocation(currentFileId, currentOffset)
			seg.RightBoundary = NewLocation(currentFileId, currentOffset+buf.Size)

			segList = append(segList, seg)

			seg = nil
		} else if buf.BlockType == BlockTypeAccountBlock {
			ab := &ledger.AccountBlock{}
			if err := ab.Deserialize(sBuf); err != nil {
				return nil, err
			}
			seg.AccountBlocks = append(seg.AccountBlocks, ab)
			seg.AccountBlocksLocation = append(seg.AccountBlocksLocation, NewLocation(currentFileId, currentOffset))
		}

		currentOffset += buf.Size
	}

	if seg != nil {
		seg.RightBoundary = NewLocation(currentFileId, currentOffset)
		segList = append(segList, seg)
	}

	if err := bfp.Error(); err != nil {
		return nil, err
	}

	return segList, nil
}

func (bDB *BlockDB) DeleteTo(location *Location, prevLocation *Location) ([]*SnapshotSegment, []*ledger.AccountBlock, error) {
	// bDB.fm.DeleteAndReadTo(location)
	bfp := newBlockFileParser()
	bfp2 := newBlockFileParser()

	bDB.wg.Add(1)
	go func() {
		defer bDB.wg.Done()
		bDB.fm.DeleteAndReadTo(location, bfp)
		bDB.fm.ReadRange(prevLocation, location, bfp2)
	}()

	var segList []*SnapshotSegment
	var seg *SnapshotSegment
	var snappyReadBuffer = make([]byte, 0, 1024*1024) // 1M

	iterator := bfp.Iterator()

	for buf := range iterator {

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

	}
	if err := bfp.Error(); err != nil {
		return nil, nil, err
	}

	var accountBlockList []*ledger.AccountBlock
	iterator2 := bfp2.Iterator()

	for buf := range iterator2 {
		if buf.BlockType == BlockTypeAccountBlock {
			sBuf := snappy.Encode(snappyReadBuffer, buf.Buffer)
			ab := &ledger.AccountBlock{}
			if err := ab.Deserialize(sBuf); err != nil {
				return nil, nil, err
			}
			accountBlockList = append(accountBlockList, ab)
		}
	}

	if err := bfp2.Error(); err != nil {
		return nil, nil, err
	}
	return segList, accountBlockList, nil
}
