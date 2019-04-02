package chain_block

import (
	"encoding/binary"
	"fmt"
	"github.com/golang/snappy"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/chain/file_manager"
	"github.com/vitelabs/go-vite/chain/utils"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto"
	"github.com/vitelabs/go-vite/ledger"
	"io"
	"path"
	"sync"
)

const (
	optFlush    = 1
	optRollback = 2
)

type BlockDB struct {
	fm *chain_file_manager.FileManager

	snappyWriteBuffer []byte
	wg                sync.WaitGroup

	fileSize int64
	id       types.Hash

	flushStartLocation  *chain_file_manager.Location
	targetFlushLocation *chain_file_manager.Location

	prepareRollbackLocation *chain_file_manager.Location
}

type SnapshotSegment struct {
	SnapshotBlock *ledger.SnapshotBlock

	AccountBlocks []*ledger.AccountBlock
}

func NewBlockDB(chainDir string) (*BlockDB, error) {
	id, _ := types.BytesToHash(crypto.Hash256([]byte("indexDb")))

	fileSize := int64(10 * 1024 * 1024)
	fm, err := chain_file_manager.NewFileManager(path.Join(chainDir, "blocks"), fileSize, 10)
	if err != nil {
		return nil, err
	}

	return &BlockDB{
		fm:                fm,
		fileSize:          fileSize,
		snappyWriteBuffer: make([]byte, fileSize),
		id:                id,
	}, nil
}

func (bDB *BlockDB) Id() types.Hash {
	return bDB.id
}

func (bDB *BlockDB) Prepare() {
	if bDB.prepareRollbackLocation == nil {
		bDB.targetFlushLocation = bDB.fm.LatestLocation()
	}
}

func (bDB *BlockDB) CancelPrepare() {
	bDB.prepareRollbackLocation = nil
	bDB.targetFlushLocation = nil
}

func (bDB *BlockDB) RedoLog() ([]byte, error) {
	var redoLog []byte
	if bDB.prepareRollbackLocation != nil {
		redoLog = make([]byte, 0, 13)
		redoLog = append(redoLog, optRollback)
		redoLog = append(redoLog, chain_utils.SerializeLocation(bDB.prepareRollbackLocation)...)

	} else if bDB.targetFlushLocation != nil && bDB.targetFlushLocation.Compare(bDB.fm.FlushLocation()) > 0 {
		bfp := newBlockFileParser()
		startLocation := bDB.fm.FlushLocation()
		targetLocation := bDB.targetFlushLocation
		bDB.wg.Add(1)
		go func() {
			defer bDB.wg.Done()
			bDB.fm.ReadRange(startLocation, targetLocation, bfp)
			bfp.Close()
		}()
		iterator := bfp.Iterator()

		redoLog = make([]byte, 0, startLocation.Distance(bDB.fileSize, targetLocation)+13)
		redoLog = append(redoLog, optFlush)
		redoLog = append(redoLog, chain_utils.SerializeLocation(startLocation)...)

		for buf := range iterator {
			redoLog = append(redoLog, buf.Buffer...)
		}
		if err := bfp.Error(); err != nil {
			return nil, err
		}
	}
	return redoLog, nil

}

func (bDB *BlockDB) Commit() error {
	defer func() {
		bDB.targetFlushLocation = nil
		bDB.prepareRollbackLocation = nil
	}()

	if bDB.prepareRollbackLocation != nil {
		return bDB.DeleteTo(bDB.prepareRollbackLocation)

	}
	return bDB.fm.Flush(bDB.targetFlushLocation)

}

func (bDB *BlockDB) PatchRedoLog(redoLog []byte) error {
	switch redoLog[0] {
	case optFlush:
		startLocation := chain_utils.DeserializeLocation(redoLog[1:13])

		if err := bDB.DeleteTo(startLocation); err != nil {
			return err
		}

		toLocation, err := bDB.writeRaw(redoLog[13:])
		if err != nil {
			return err
		}

		if err := bDB.fm.Flush(toLocation); err != nil {
			return err
		}

		bDB.targetFlushLocation = nil

	case optRollback:
		rollbackTargetLocation := chain_utils.DeserializeLocation(redoLog[1:13])
		if err := bDB.DeleteTo(rollbackTargetLocation); err != nil {
			return err
		}

		bDB.prepareRollbackLocation = nil
	}

	return nil
}

func (bDB *BlockDB) Stop() {
	bDB.wg.Wait()
}

func (bDB *BlockDB) FileSize() int64 {
	return bDB.fileSize
}

func (bDB *BlockDB) Close() error {
	if err := bDB.fm.Close(); err != nil {
		return errors.New(fmt.Sprintf("bDB.fm.Close failed, error is %s", err))
	}

	bDB.fm = nil
	return nil
}

func (bDB *BlockDB) LatestLocation() *chain_file_manager.Location {
	latestLocation := bDB.fm.LatestLocation()
	if bDB.prepareRollbackLocation != nil && latestLocation.Compare(bDB.prepareRollbackLocation) > 0 {
		return bDB.prepareRollbackLocation
	}

	return latestLocation
}

func (bDB *BlockDB) Write(ss *SnapshotSegment) ([]*chain_file_manager.Location, *chain_file_manager.Location, error) {
	if bDB.prepareRollbackLocation != nil {
		return nil, nil, errors.New("prepare rollback, can't write")
	}

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
	if bDB.prepareRollbackLocation != nil && bDB.prepareRollbackLocation.Compare(location) <= 0 {
		return nil, nil
	}
	buf, _, err := bDB.fm.Read(location)
	if err != nil {
		return nil, err
	}
	if len(buf) <= 0 {
		return nil, nil
	}

	sBuf, err := snappy.Decode(nil, buf[1:])
	if err != nil {
		return nil, err
	}
	return sBuf, nil
}

func (bDB *BlockDB) ReadRaw(startLocation *chain_file_manager.Location, buf []byte) (*chain_file_manager.Location, int, error) {
	if bDB.prepareRollbackLocation != nil && bDB.prepareRollbackLocation.Compare(startLocation) <= 0 {
		return nil, 0, nil
	}

	nextLocation, count, err := bDB.fm.ReadRaw(startLocation, buf)
	if bDB.prepareRollbackLocation != nil && bDB.prepareRollbackLocation.Compare(nextLocation) <= 0 {
		size := bDB.prepareRollbackLocation.Distance(bDB.fileSize, startLocation)

		return bDB.prepareRollbackLocation, int(size), err
	}
	return nextLocation, count, err
}

func (bDB *BlockDB) ReadRange(startLocation *chain_file_manager.Location, endLocation *chain_file_manager.Location) ([]*SnapshotSegment, error) {
	bfp := newBlockFileParser()

	endLocation = bDB.maxLocation(endLocation)

	bDB.wg.Add(1)
	go func() {
		defer bDB.wg.Done()
		bDB.fm.ReadRange(startLocation, endLocation, bfp)
		if endLocation != nil {
			buf, _, err := bDB.fm.Read(endLocation)

			if len(buf) >= 0 {
				bufSizeBytes := make([]byte, 4)
				binary.BigEndian.PutUint32(bufSizeBytes, uint32(len(buf)))
				bfp.Write(bufSizeBytes)
				bfp.Write(buf)
			}

			if err != nil && err != io.EOF {
				bfp.WriteError(err)
				return
			}

		}
		bfp.Close()
	}()

	var segList []*SnapshotSegment
	var seg *SnapshotSegment

	var snappyReadBuffer = make([]byte, 0, 8*1024) // 8kb
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

	if err := bfp.Error(); err != nil {
		return nil, err
	}

	if seg != nil {
		segList = append(segList, seg)
	}

	return segList, nil
}

func (bDB *BlockDB) GetNextLocation(location *chain_file_manager.Location) (*chain_file_manager.Location, error) {
	nextLocation, err := bDB.fm.GetNextLocation(location)
	if err != nil {
		if err != io.EOF {
			return nil, err
		}
		return nil, nil
	}
	return nextLocation, nil
}

func (bDB *BlockDB) Rollback(location *chain_file_manager.Location) ([]*SnapshotSegment, error) {
	bfp := newBlockFileParser()

	bDB.wg.Add(1)
	go func() {
		defer bDB.wg.Done()
		bDB.fm.ReadRange(location, bDB.fm.LatestLocation(), bfp)
		bfp.Close()
	}()

	var segList []*SnapshotSegment
	var seg *SnapshotSegment
	var snappyReadBuffer = make([]byte, 0, 512*1024) // 512KB

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

	if len(segList) > 0 {
		bDB.prepareRollbackLocation = location
	}

	return segList, nil

}

func (bDB *BlockDB) writeRaw(buf []byte) (*chain_file_manager.Location, error) {

	return bDB.fm.Write(buf)
}

func (bDB *BlockDB) maxLocation(location *chain_file_manager.Location) *chain_file_manager.Location {

	latestLocation := bDB.LatestLocation()
	if location == nil || (latestLocation != nil && location.Compare(latestLocation) > 0) {
		return latestLocation
	}
	return location
}
