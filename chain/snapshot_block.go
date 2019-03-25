package chain

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/chain/block"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"sort"
	"time"
)

func (c *chain) IsSnapshotBlockExisted(hash *types.Hash) (bool, error) {
	// cache
	if ok := c.cache.IsSnapshotBlockExisted(hash); ok {
		return ok, nil
	}

	// query index
	ok, err := c.indexDB.IsSnapshotBlockExisted(hash)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.indexDB.IsSnapshotBlockExisted failed, error is %s, hash is %s", err, hash))
		return false, cErr
	}

	return ok, nil
}

// is valid
func (c *chain) IsSnapshotContentValid(snapshotContent ledger.SnapshotContent) (invalidMap map[types.Address]*ledger.HashHeight, err error) {
	//for addr, hashHeight := range snapshotContent {
	//
	//}
	return nil, nil
}

func (c *chain) GetGenesisSnapshotBlock() *ledger.SnapshotBlock {
	return c.cache.GetGenesisSnapshotBlock()
}

func (c *chain) GetLatestSnapshotBlock() *ledger.SnapshotBlock {
	return c.cache.GetLatestSnapshotBlock()
}

// header without snapshot content
func (c *chain) GetSnapshotHeaderByHeight(height uint64) (*ledger.SnapshotBlock, error) {
	// cache
	if header := c.cache.GetSnapshotHeaderByHeight(height); header != nil {
		return header, nil
	}

	// query location
	location, err := c.indexDB.GetSnapshotBlockLocation(height)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.indexDB.GetSnapshotBlockLocation failed,  height is %d. Error: %s,",
			height, err.Error()))
		c.log.Error(cErr.Error(), "method", "GetSnapshotHeaderByHeight")
		return nil, cErr
	}
	if location == nil {
		return nil, nil
	}

	// query block
	snapshotBlock, err := c.blockDB.GetSnapshotHeader(location)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.blockDB.GetSnapshotHeader failed, error is %s, height is %d, location is %+v",
			err.Error(), height, location))
		c.log.Error(cErr.Error(), "method", "GetSnapshotHeaderByHeight")
		return nil, cErr
	}

	if snapshotBlock == nil {
		return nil, nil
	}

	snapshotBlock.SnapshotContent = nil
	return snapshotBlock, nil
}

// block contains header、snapshot content
func (c *chain) GetSnapshotBlockByHeight(height uint64) (*ledger.SnapshotBlock, error) {
	// cache
	if block := c.cache.GetSnapshotBlockByHeight(height); block != nil {
		return block, nil
	}

	// query location
	location, err := c.indexDB.GetSnapshotBlockLocation(height)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.indexDB.GetSnapshotBlockLocation failed, error is %s, height is %d",
			err.Error(), height))
		c.log.Error(cErr.Error(), "method", "GetSnapshotBlockByHeight")
		return nil, cErr
	}

	// query block
	snapshotBlock, err := c.blockDB.GetSnapshotBlock(location)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.blockDB.GetSnapshotBlock failed, error is %s, height is %d, location is %+v",
			err.Error(), height, location))
		c.log.Error(cErr.Error(), "method", "GetSnapshotBlockByHeight")
		return nil, cErr
	}

	return snapshotBlock, nil
}

func (c *chain) GetSnapshotHeaderByHash(hash *types.Hash) (*ledger.SnapshotBlock, error) {
	// cache
	if block := c.cache.GetSnapshotHeaderByHash(hash); block != nil {
		return block, nil
	}

	// query location
	location, err := c.indexDB.GetSnapshotBlockLocationByHash(hash)
	if err != nil {
		c.log.Error(fmt.Sprintf("c.indexDB.GetSnapshotBlockLocationByHash failed, error is %s, hash is %s",
			err.Error(), hash), "method", "GetSnapshotHeaderByHash")
		return nil, err
	}

	// query block
	snapshotBlock, err := c.blockDB.GetSnapshotHeader(location)
	if err != nil {
		c.log.Error(fmt.Sprintf("c.blockDB.GetSnapshotHeader failed, error is %s, hash is %s, location is %+v\n",
			err.Error(), hash, location), "method", "GetSnapshotHeaderByHash")
		return nil, err
	}

	snapshotBlock.SnapshotContent = nil

	return snapshotBlock, nil
}

func (c *chain) GetSnapshotBlockByHash(hash *types.Hash) (*ledger.SnapshotBlock, error) {
	// cache
	if header := c.cache.GetSnapshotBlockByHash(hash); header != nil {
		return header, nil
	}

	// query location
	location, err := c.indexDB.GetSnapshotBlockLocationByHash(hash)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.indexDB.GetSnapshotBlockLocation failed, error is %s, hash is %s",
			err.Error(), hash))
		c.log.Error(cErr.Error(), "method", "GetSnapshotBlockByHash")
		return nil, cErr
	}

	// query block
	snapshotBlock, err := c.blockDB.GetSnapshotBlock(location)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.blockDB.GetSnapshotBlock failed, error is %s, hash is %s, location is %+v",
			err.Error(), hash, location))
		c.log.Error(cErr.Error(), "method", "GetSnapshotBlockByHash")
		return nil, cErr
	}

	return snapshotBlock, nil
}

// contains start snapshot block and end snapshot block
func (c *chain) GetRangeSnapshotHeaders(startHash *types.Hash, endHash *types.Hash) ([]*ledger.SnapshotBlock, error) {
	blocks, err := c.getSnapshotBlockList(func() ([]*chain_block.Location, [2]uint64, error) {
		return c.indexDB.GetRangeSnapshotBlockLocations(startHash, endHash)
	}, true)

	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.getSnapshotBlockList failed, error is %s, startHash is %s, endHash is %s",
			err.Error(), startHash, endHash))
		c.log.Error(cErr.Error(), "method", "GetRangeSnapshotHeaders")
		return nil, cErr
	}
	return blocks, nil
}

func (c *chain) GetRangeSnapshotBlocks(startHash *types.Hash, endHash *types.Hash) ([]*ledger.SnapshotBlock, error) {

	blocks, err := c.getSnapshotBlockList(func() ([]*chain_block.Location, [2]uint64, error) {
		return c.indexDB.GetRangeSnapshotBlockLocations(startHash, endHash)
	}, false)

	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.getSnapshotBlockList failed, error is %s, startHash is %s, endHash is %s",
			err.Error(), startHash, endHash))
		c.log.Error(cErr.Error(), "method", "GetRangeSnapshotBlocks")
		return nil, cErr
	}
	return blocks, nil
}

// contains the snapshot header that has the blockHash
func (c *chain) GetSnapshotHeaders(blockHash *types.Hash, higher bool, count uint64) ([]*ledger.SnapshotBlock, error) {
	blocks, err := c.getSnapshotBlockList(func() ([]*chain_block.Location, [2]uint64, error) {
		return c.indexDB.GetSnapshotBlockLocationList(blockHash, higher, count)
	}, true)

	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.getSnapshotBlockList failed, error is %s, blockHash is %s, higher is %+v, count is %d",
			err.Error(), blockHash, higher, count))
		c.log.Error(cErr.Error(), "method", "GetSnapshotHeaders")
		return nil, cErr
	}
	return blocks, nil
}

// contains the snapshot block that has the blockHash
func (c *chain) GetSnapshotBlocks(blockHash *types.Hash, higher bool, count uint64) ([]*ledger.SnapshotBlock, error) {
	blocks, err := c.getSnapshotBlockList(func() ([]*chain_block.Location, [2]uint64, error) {
		return c.indexDB.GetSnapshotBlockLocationList(blockHash, higher, count)
	}, false)

	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.getSnapshotBlockList failed, error is %s, blockHash is %s, higher is %+v, count is %d",
			err.Error(), blockHash, higher, count))
		c.log.Error(cErr.Error(), "method", "GetSnapshotBlocks")
		return nil, cErr
	}
	return blocks, nil
}

func (c *chain) GetConfirmSnapshotHeaderByAbHash(abHash *types.Hash) (*ledger.SnapshotBlock, error) {
	confirmHeight, err := c.indexDB.GetConfirmHeightByHash(abHash)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.indexDB.GetConfirmHeightByHash failed, error is %s, blockHash is %s",
			err.Error(), abHash))
		c.log.Error(cErr.Error(), "method", "GetConfirmSnapshotHeaderByAbHash")
		return nil, cErr
	}
	if confirmHeight <= 0 {
		return nil, err
	}

	return c.GetSnapshotHeaderByHeight(confirmHeight)
}

func (c *chain) GetConfirmSnapshotBlockByAbHash(abHash *types.Hash) (*ledger.SnapshotBlock, error) {
	confirmHeight, err := c.indexDB.GetConfirmHeightByHash(abHash)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.indexDB.GetConfirmHeightByHash failed, error is %s, blockHash is %s",
			err.Error(), abHash))
		c.log.Error(cErr.Error(), "method", "GetConfirmSnapshotBlockByAbHash")
		return nil, cErr
	}
	if confirmHeight <= 0 {
		return nil, err
	}

	return c.GetSnapshotBlockByHeight(confirmHeight)
}

func (c *chain) GetSnapshotHeaderBeforeTime(timestamp *time.Time) (*ledger.SnapshotBlock, error) {
	// normal logic
	genesis := c.GetGenesisSnapshotBlock()
	latest := c.GetLatestSnapshotBlock()

	timeNanosecond := timestamp.UnixNano()
	if genesis.Timestamp.UnixNano() < timeNanosecond {
		return genesis, nil
	}
	if latest.Timestamp.UnixNano() > timeNanosecond {
		return nil, nil
	}

	endSec := latest.Timestamp.Second()
	timeSec := timestamp.Second()

	estimateHeight := latest.Height - uint64(endSec-timeSec)
	var highBoundary, lowBoundary *ledger.SnapshotBlock

	for highBoundary == nil || lowBoundary == nil {
		blockHeader, err := c.GetSnapshotHeaderByHeight(estimateHeight)
		if err != nil {
			cErr := errors.New(fmt.Sprintf("c.GetSnapshotHeaderByHeight failed, error is %s, height is %d",
				err.Error(), estimateHeight))
			c.log.Error(cErr.Error(), "method", "GetSnapshotHeaderBeforeTime")
			return nil, cErr
		}

		if blockHeader == nil {
			cErr := errors.New(fmt.Sprintf("blockHeader is nil,  height is %d",
				estimateHeight))
			c.log.Error(cErr.Error(), "method", "GetSnapshotHeaderBeforeTime")
			return nil, cErr
		}

		if blockHeader.Timestamp.UnixNano() >= timeNanosecond {
			highBoundary = blockHeader
			gap := uint64(blockHeader.Timestamp.Second() - timeSec)
			if blockHeader.Height <= gap {
				break
			}

			if gap <= 0 {
				gap = 1
			}
			estimateHeight = blockHeader.Height - gap

		} else {
			lowBoundary = blockHeader
			estimateHeight = blockHeader.Height + uint64(timeSec-blockHeader.Timestamp.Second())
		}
	}

	if highBoundary.Height == lowBoundary.Height+1 {
		return lowBoundary, nil
	}
	block, err := c.binarySearchBeforeTime(lowBoundary, highBoundary, timeNanosecond)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.binarySearchBeforeTime failed,  lowBoundary is %+v, highBoundary is %+v, timeNanosecond is %d",
			lowBoundary, highBoundary, timeNanosecond))
		c.log.Error(cErr.Error(), "method", "GetSnapshotHeaderBeforeTime")
		return nil, cErr
	}
	return block, nil
}

func (c *chain) GetSnapshotHeadersAfterOrEqualTime(endHashHeight *ledger.HashHeight, startTime *time.Time, producer *types.Address) ([]*ledger.SnapshotBlock, error) {
	startHeader, err := c.GetSnapshotHeaderBeforeTime(startTime)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.GetSnapshotHeaderBeforeTime failed,  error is %s, startTime is %s",
			err.Error(), startTime))
		c.log.Error(cErr.Error(), "method", "GetSnapshotHeaderBeforeTime")
		return nil, cErr
	}

	if startHeader == nil {
		startHeader = c.GetGenesisSnapshotBlock()
	}

	snapshotHeaders, err := c.GetSnapshotHeaders(&endHashHeight.Hash, false, endHashHeight.Height-startHeader.Height)
	if err != nil {
		return nil, err
	}
	if producer != nil {
		var result = make([]*ledger.SnapshotBlock, 0, 9)
		for _, snapshotHeader := range snapshotHeaders {
			if snapshotHeader.Producer() == *producer {
				result = append(result)
			}
		}
		return result, nil
	}

	return snapshotHeaders, nil
}

func (c *chain) QueryLatestSnapshotBlock() (*ledger.SnapshotBlock, error) {
	var err error
	location, err := c.indexDB.GetLatestSnapshotBlockLocation()
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.indexDB.GetLatestSnapshotBlockLocation failed, error is %s", err.Error()))
		c.log.Error(cErr.Error(), "method", "getLatestSnapshotBlock")
		return nil, cErr
	}

	if location == nil {
		cErr := errors.New("location is nil")
		c.log.Error(cErr.Error(), "method", "getLatestSnapshotBlock")
		return nil, cErr
	}

	sb, err := c.blockDB.GetSnapshotBlock(location)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.blockDB.GetSnapshotBlock failed, error is %s", err.Error()))

		c.log.Error(cErr.Error(), "method", "getLatestSnapshotBlock")
		return nil, cErr
	}

	return sb, nil
}

// [startHeight, latestHeight]
func (c *chain) GetSubLedger(startHeight, endHeight uint64) ([]*chain_block.SnapshotSegment, error) {
	if startHeight <= 0 {
		startHeight = 1
	}
	latestSb := c.GetLatestSnapshotBlock()
	if endHeight > latestSb.Height {
		endHeight = latestSb.Height
	}
	// query location
	startLocation, err := c.indexDB.GetSnapshotBlockLocation(startHeight)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.indexDB.GetSnapshotBlockLocation failed,  height is %d. Error: %s,",
			startHeight, err.Error()))
		c.log.Error(cErr.Error(), "method", "GetSubLedgerAfterHeight")
		return nil, cErr
	}
	if startLocation == nil {
		return nil, nil
	}

	endLocation, err := c.indexDB.GetSnapshotBlockLocation(endHeight)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.indexDB.GetSnapshotBlockLocation failed,  height is %d. Error: %s,",
			endHeight, err.Error()))
		c.log.Error(cErr.Error(), "method", "GetSubLedgerAfterHeight")
		return nil, cErr
	}
	if endLocation == nil {
		return nil, nil
	}

	segList, err := c.blockDB.ReadRange(startLocation, endLocation)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.blockDB.ReadRange failed,  startLocation is %+v, endLocation is %+v, . Error: %s,",
			startLocation, endLocation, err.Error()))
		c.log.Error(cErr.Error(), "method", "GetSubLedgerAfterHeight")
		return nil, cErr
	}
	return segList, nil
}

func (c *chain) GetSeed(snapshotHash *types.Hash, n int) uint64 {
	return 0
}

// [startHeight, latestHeight]
func (c *chain) GetSubLedgerAfterHeight(height uint64) ([]*chain_block.SnapshotSegment, error) {
	// query location
	startLocation, err := c.indexDB.GetSnapshotBlockLocation(height)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.indexDB.GetSnapshotBlockLocation failed,  height is %d. Error: %s,",
			height, err.Error()))
		c.log.Error(cErr.Error(), "method", "GetSubLedgerAfterHeight")
		return nil, cErr
	}
	if startLocation == nil {
		return nil, nil
	}

	segList, err := c.blockDB.ReadRange(startLocation, nil)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.blockDB.ReadRange failed,  startLocation is %+v, endLocation is nil. Error: %s,",
			startLocation, err.Error()))
		c.log.Error(cErr.Error(), "method", "GetSubLedgerAfterHeight")
		return nil, cErr
	}
	return segList, nil
}

type getSnapshotListFunc func() ([]*chain_block.Location, [2]uint64, error)

func (c *chain) getSnapshotBlockList(getList getSnapshotListFunc, onlyHeader bool) ([]*ledger.SnapshotBlock, error) {
	locations, heightRange, err := getList()

	if err != nil {
		return nil, err
	}
	if len(locations) <= 0 {
		return nil, nil
	}

	blocks := make([]*ledger.SnapshotBlock, len(locations))

	startHeight := heightRange[0]
	endHeight := heightRange[1]

	currentHeight := startHeight
	index := 0

	for currentHeight <= endHeight {
		var block *ledger.SnapshotBlock
		var err error
		if onlyHeader {
			block := c.cache.GetSnapshotBlockByHeight(currentHeight)
			if block == nil {
				block, err = c.blockDB.GetSnapshotBlock(locations[index])
			}
		} else {
			block := c.cache.GetSnapshotHeaderByHeight(currentHeight)
			if block == nil {
				block, err = c.blockDB.GetSnapshotHeader(locations[index])
			}
		}

		if err != nil {
			return nil, err
		}

		blocks[index] = block
		index++
		currentHeight++
	}

	return blocks, nil
}

func (c *chain) binarySearchBeforeTime(start, end *ledger.SnapshotBlock, timeNanosecond int64) (*ledger.SnapshotBlock, error) {
	n := int(end.Height - start.Height + 1)

	var err error
	blockMap := make(map[int]*ledger.SnapshotBlock, n)

	i := sort.Search(n, func(i int) bool {
		if err != nil {
			return true
		}
		height := start.Height + uint64(i)

		blockMap[i], err = c.GetSnapshotHeaderByHeight(height)
		if err != nil {
			return true
		}

		return blockMap[i].Timestamp.UnixNano() >= timeNanosecond

	})

	if err != nil {
		return nil, err
	}
	if i >= n {
		return nil, nil
	}

	if block, ok := blockMap[i-1]; ok {
		return block, nil
	}
	return c.GetSnapshotHeaderByHeight(start.Height + uint64(i-1))

}