package chain_block

import (
	"fmt"
	"github.com/golang/snappy"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/chain/file_manager"
	"github.com/vitelabs/go-vite/ledger"
	"path"
	"sync"
)

type BlockDB struct {
	fm *chain_file_manager.FileManager

	snappyWriteBuffer []byte
	wg                sync.WaitGroup

	fileSize int64
}

type SnapshotSegment struct {
	SnapshotBlock         *ledger.SnapshotBlock
	SnapshotBlockLocation *chain_file_manager.Location // optional

	AccountBlocks         []*ledger.AccountBlock
	AccountBlocksLocation []*chain_file_manager.Location // optional

	RightBoundary *chain_file_manager.Location // optional
}

func NewBlockDB(chainDir string) (*BlockDB, error) {
	fileSize := int64(20 * 1024 * 1024)
	fm, err := chain_file_manager.NewFileManager(path.Join(chainDir, "blocks"), fileSize)
	if err != nil {
		return nil, err
	}

	return &BlockDB{
		fm:                fm,
		fileSize:          fileSize,
		snappyWriteBuffer: make([]byte, fileSize),
	}, nil
}

func (bDB *BlockDB) Stop() {
	bDB.wg.Wait()
}
func (bDB *BlockDB) FileSize() int64 {
	return bDB.fileSize
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

func (bDB *BlockDB) LatestLocation() *chain_file_manager.Location {
	return bDB.fm.LatestLocation()
}

func (bDB *BlockDB) Write(ss *SnapshotSegment) ([]*chain_file_manager.Location, *chain_file_manager.Location, error) {
	accountBlocksLocation := make([]*chain_file_manager.Location, 0, len(ss.AccountBlocks))

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

func (bDB *BlockDB) Read(location *chain_file_manager.Location) ([]byte, error) {
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

func (bDB *BlockDB) ReadRaw(startLocation *chain_file_manager.Location, buf []byte) (*chain_file_manager.Location, int, error) {
	return bDB.fm.ReadRaw(startLocation, buf)
}

func (bDB *BlockDB) ReadRange(startLocation *chain_file_manager.Location, endLocation *chain_file_manager.Location) ([]*SnapshotSegment, error) {
	bfp := newBlockFileParser()
	bDB.wg.Add(1)
	go func() {
		defer bDB.wg.Done()
		bDB.fm.ReadRange(startLocation, endLocation, bfp)
		if endLocation != nil {
			buf, err := bDB.fm.Read(endLocation)
			if err != nil {
				bfp.WriteError(err)
				return
			}
			bfp.Write(buf, endLocation)
		}
		bfp.Close()
	}()

	var segList []*SnapshotSegment
	var seg *SnapshotSegment

	var snappyReadBuffer = make([]byte, 0, 8*1024) // 8KB
	iterator := bfp.Iterator()

	currentFileId := uint64(0)
	currentOffset := int64(0)

	for buf := range iterator {
		if seg == nil {
			seg = &SnapshotSegment{}
		}

		if buf.FileId != currentFileId {
			currentFileId = buf.FileId
			if buf.FileId == startLocation.FileId {
				currentOffset = startLocation.Offset
			} else {
				currentOffset = 0
			}
		}

		sBuf, err := snappy.Decode(snappyReadBuffer, buf.Buffer)
		if err != nil {
			return nil, err
		}

		location := chain_file_manager.NewLocation(currentFileId, currentOffset)
		if buf.BlockType == BlockTypeSnapshotBlock {

			sb := &ledger.SnapshotBlock{}
			if err := sb.Deserialize(sBuf); err != nil {
				return nil, err
			}
			seg.SnapshotBlock = sb
			seg.SnapshotBlockLocation = location
			seg.RightBoundary = chain_file_manager.NewLocation(currentFileId, currentOffset+buf.Size)

			segList = append(segList, seg)

			seg = nil
		} else if buf.BlockType == BlockTypeAccountBlock {
			ab := &ledger.AccountBlock{}
			if err := ab.Deserialize(sBuf); err != nil {
				return nil, err
			}
			seg.AccountBlocks = append(seg.AccountBlocks, ab)
			seg.AccountBlocksLocation = append(seg.AccountBlocksLocation, location)
		}

		currentOffset += buf.Size
	}

	if err := bfp.Error(); err != nil {
		return nil, err
	}

	if seg != nil {
		seg.RightBoundary = chain_file_manager.NewLocation(currentFileId, currentOffset)
		segList = append(segList, seg)
	}

	return segList, nil
}

func (bDB *BlockDB) Rollback(prevLocation *chain_file_manager.Location) ([]*SnapshotSegment, error) {

	// bDB.fm.DeleteAndReadTo(location)
	bfp := newBlockFileParser()
	//bfp2 := newBlockFileParser()

	bDB.wg.Add(1)
	go func() {
		defer bDB.wg.Done()
		bDB.fm.ReadRange(prevLocation, nil, bfp)
		bfp.Close()

		bDB.fm.DeleteTo(prevLocation)
	}()

	var segList []*SnapshotSegment
	var seg *SnapshotSegment
	var snappyReadBuffer = make([]byte, 0, 1024*1024) // 1M

	iterator := bfp.Iterator()

	for buf := range iterator {
		if seg == nil {
			seg = &SnapshotSegment{}
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
			segList = append(segList, seg)
			seg = nil
		} else if buf.BlockType == BlockTypeAccountBlock {

			ab := &ledger.AccountBlock{}
			if err := ab.Deserialize(sBuf); err != nil {
				return nil, err
			}
			seg.AccountBlocks = append(seg.AccountBlocks, ab)
		}
	}

	if seg != nil {
		segList = append(segList, seg)
	}

	if err := bfp.Error(); err != nil {
		return nil, err
	}

	return segList, nil
}
