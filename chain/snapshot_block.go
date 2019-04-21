package chain

import (
	"fmt"
	"github.com/vitelabs/go-vite/common/helper"
	"math/big"
	"sort"
	"time"

	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/chain/file_manager"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

func (c *chain) IsGenesisSnapshotBlock(hash types.Hash) bool {
	return c.genesisSnapshotBlock.Hash == hash

}
func (c *chain) IsSnapshotBlockExisted(hash types.Hash) (bool, error) {
	// cache
	if ok := c.cache.IsSnapshotBlockExisted(hash); ok {
		return ok, nil
	}

	// query index
	ok, err := c.indexDB.IsSnapshotBlockExisted(&hash)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.indexDB.IsSnapshotBlockExisted failed, error is %s, hash is %s", err, hash))
		return false, cErr
	}

	return ok, nil
}

func (c *chain) GetGenesisSnapshotBlock() *ledger.SnapshotBlock {
	return c.genesisSnapshotBlock
}

func (c *chain) GetLatestSnapshotBlock() *ledger.SnapshotBlock {
	return c.cache.GetLatestSnapshotBlock()
}

func (c *chain) GetSnapshotHeightByHash(hash types.Hash) (uint64, error) {
	// cache
	if header := c.cache.GetSnapshotHeaderByHash(hash); header != nil {
		return header.Height, nil
	}

	height, err := c.indexDB.GetSnapshotBlockHeight(&hash)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.indexDB.GetSnapshotBlockHeight failed,  hash is %s. Error: %s,",
			hash, err.Error()))
		c.log.Error(cErr.Error(), "method", "GetSnapshotHeightByHash")
		return height, cErr
	}

	return height, nil
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

	//snapshotBlock.SnapshotContent = nil
	return snapshotBlock, nil
}

// block contains header、snapshot content
func (c *chain) GetSnapshotBlockByHeight(height uint64) (*ledger.SnapshotBlock, error) {
	// cache
	if block := c.cache.GetSnapshotBlockByHeight(height); block != nil {
		return block, nil
	}

	// query location
	return c.QuerySnapshotBlockByHeight(height)
}

func (c *chain) GetSnapshotHeaderByHash(hash types.Hash) (*ledger.SnapshotBlock, error) {
	// cache
	if block := c.cache.GetSnapshotHeaderByHash(hash); block != nil {
		return block, nil
	}

	// query location
	location, err := c.indexDB.GetSnapshotBlockLocationByHash(&hash)
	if err != nil {
		c.log.Error(fmt.Sprintf("c.indexDB.GetSnapshotBlockLocationByHash failed, error is %s, hash is %s",
			err.Error(), hash), "method", "GetSnapshotHeaderByHash")
		return nil, err
	}

	if location == nil {
		return nil, nil
	}

	// query block
	snapshotBlock, err := c.blockDB.GetSnapshotHeader(location)
	if err != nil {
		c.log.Error(fmt.Sprintf("c.blockDB.GetSnapshotHeader failed, error is %s, hash is %s, location is %+v\n",
			err.Error(), hash, location), "method", "GetSnapshotHeaderByHash")
		return nil, err
	}

	//snapshotBlock.SnapshotContent = nil

	return snapshotBlock, nil
}

func (c *chain) GetSnapshotBlockByHash(hash types.Hash) (*ledger.SnapshotBlock, error) {
	// cache
	if header := c.cache.GetSnapshotBlockByHash(hash); header != nil {
		return header, nil
	}

	// query location
	location, err := c.indexDB.GetSnapshotBlockLocationByHash(&hash)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.indexDB.GetSnapshotBlockLocation failed, error is %s, hash is %s",
			err.Error(), hash))
		c.log.Error(cErr.Error(), "method", "GetSnapshotBlockByHash")
		return nil, cErr
	}
	if location == nil {
		return nil, nil
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
func (c *chain) GetRangeSnapshotHeaders(startHash types.Hash, endHash types.Hash) ([]*ledger.SnapshotBlock, error) {
	blocks, err := c.getSnapshotBlockList(func() ([]*chain_file_manager.Location, [2]uint64, error) {
		return c.indexDB.GetRangeSnapshotBlockLocations(&startHash, &endHash)
	}, true, true)

	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.getSnapshotBlockList failed, error is %s, startHash is %s, endHash is %s",
			err.Error(), startHash, endHash))
		c.log.Error(cErr.Error(), "method", "GetRangeSnapshotHeaders")
		return nil, cErr
	}
	return blocks, nil
}

func (c *chain) GetRangeSnapshotBlocks(startHash types.Hash, endHash types.Hash) ([]*ledger.SnapshotBlock, error) {

	blocks, err := c.getSnapshotBlockList(func() ([]*chain_file_manager.Location, [2]uint64, error) {
		return c.indexDB.GetRangeSnapshotBlockLocations(&startHash, &endHash)
	}, true, false)

	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.getSnapshotBlockList failed, error is %s, startHash is %s, endHash is %s",
			err.Error(), startHash, endHash))
		c.log.Error(cErr.Error(), "method", "GetRangeSnapshotBlocks")
		return nil, cErr
	}
	return blocks, nil
}

// contains the snapshot header that has the blockHash
func (c *chain) GetSnapshotHeaders(blockHash types.Hash, higher bool, count uint64) ([]*ledger.SnapshotBlock, error) {
	blocks, err := c.getSnapshotBlockList(func() ([]*chain_file_manager.Location, [2]uint64, error) {
		return c.indexDB.GetSnapshotBlockLocationList(&blockHash, higher, count)
	}, higher, true)

	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.getSnapshotBlockList failed, error is %s, blockHash is %s, higher is %+v, count is %d",
			err.Error(), blockHash, higher, count))
		c.log.Error(cErr.Error(), "method", "GetSnapshotHeaders")
		return nil, cErr
	}
	return blocks, nil
}

// contains the snapshot block that has the blockHash
func (c *chain) GetSnapshotBlocks(blockHash types.Hash, higher bool, count uint64) ([]*ledger.SnapshotBlock, error) {
	blocks, err := c.getSnapshotBlockList(func() ([]*chain_file_manager.Location, [2]uint64, error) {
		return c.indexDB.GetSnapshotBlockLocationList(&blockHash, higher, count)
	}, higher, false)

	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.getSnapshotBlockList failed, error is %s, blockHash is %s, higher is %+v, count is %d",
			err.Error(), blockHash, higher, count))
		c.log.Error(cErr.Error(), "method", "GetSnapshotBlocks")
		return nil, cErr
	}
	return blocks, nil
}

// contains the snapshot header that has the blockHash
func (c *chain) GetSnapshotHeadersByHeight(height uint64, higher bool, count uint64) ([]*ledger.SnapshotBlock, error) {
	blocks, err := c.getSnapshotBlockList(func() ([]*chain_file_manager.Location, [2]uint64, error) {
		return c.indexDB.GetSnapshotBlockLocationListByHeight(height, higher, count)
	}, higher, true)

	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.getSnapshotBlockList failed, height is %d, higher is %+v, count is %d. Error: %s",
			height, higher, count, err.Error()))
		c.log.Error(cErr.Error(), "method", "GetSnapshotHeadersByHeight")
		return nil, cErr
	}
	return blocks, nil
}

// contains the snapshot block that has the blockHash
func (c *chain) GetSnapshotBlocksByHeight(height uint64, higher bool, count uint64) ([]*ledger.SnapshotBlock, error) {
	blocks, err := c.getSnapshotBlockList(func() ([]*chain_file_manager.Location, [2]uint64, error) {
		return c.indexDB.GetSnapshotBlockLocationListByHeight(height, higher, count)
	}, higher, false)

	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.getSnapshotBlockList failed, height is %d, higher is %+v, count is %d. Error: %s, ",
			height, higher, count, err.Error()))
		c.log.Error(cErr.Error(), "method", "GetSnapshotBlocksByHeight")
		return nil, cErr
	}
	return blocks, nil
}

func (c *chain) GetConfirmSnapshotHeaderByAbHash(abHash types.Hash) (*ledger.SnapshotBlock, error) {
	confirmHeight, err := c.indexDB.GetConfirmHeightByHash(&abHash)
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

func (c *chain) GetConfirmSnapshotBlockByAbHash(abHash types.Hash) (*ledger.SnapshotBlock, error) {
	confirmHeight, err := c.indexDB.GetConfirmHeightByHash(&abHash)
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
	if genesis.Timestamp.UnixNano() >= timeNanosecond {
		return nil, nil
	}
	if latest.Timestamp.UnixNano() < timeNanosecond {
		return latest, nil
	}

	endSec := latest.Timestamp.Unix()
	timeSec := timestamp.Unix()

	gap := uint64(endSec - timeSec)
	estimateHeight := uint64(1)
	if latest.Height > gap {
		estimateHeight = latest.Height - gap
	}

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
			gap := uint64(blockHeader.Timestamp.Unix() - timeSec)
			if gap <= 0 {
				gap = 1
			}

			if blockHeader.Height <= gap {
				lowBoundary = genesis
				break
			} else {
				estimateHeight = blockHeader.Height - gap
			}

		} else {
			lowBoundary = blockHeader
			estimateHeight = blockHeader.Height + uint64(timeSec-blockHeader.Timestamp.Unix())
			if estimateHeight > latest.Height {
				highBoundary = latest
			}
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

	snapshotHeaders, err := c.GetSnapshotHeaders(endHashHeight.Hash, false, endHashHeight.Height-startHeader.Height)
	if err != nil {
		return nil, err
	}
	if producer != nil {
		var result = make([]*ledger.SnapshotBlock, 0, len(snapshotHeaders)/75+3)
		for _, snapshotHeader := range snapshotHeaders {
			if snapshotHeader.Producer() == *producer {
				result = append(result)
			}
		}
		return result, nil
	}

	return snapshotHeaders, nil
}

func (c *chain) QuerySnapshotBlockByHeight(height uint64) (*ledger.SnapshotBlock, error) {
	// query location
	location, err := c.indexDB.GetSnapshotBlockLocation(height)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.indexDB.GetSnapshotBlockLocation failed, height is %d. Error:  %s, ",
			height, err.Error()))
		c.log.Error(cErr.Error(), "method", "QuerySnapshotBlockByHeight")
		return nil, cErr
	}
	if location == nil {
		return nil, nil
	}

	// query block
	snapshotBlock, err := c.blockDB.GetSnapshotBlock(location)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.blockDB.GetSnapshotBlock failed, height is %d, location is %+v. Error: %s",
			height, location, err.Error()))
		c.log.Error(cErr.Error(), "method", "QuerySnapshotBlockByHeight")
		return nil, cErr
	}

	return snapshotBlock, nil
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
		return nil, nil
	}

	sb, err := c.blockDB.GetSnapshotBlock(location)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.blockDB.GetSnapshotBlock failed, error is %s", err.Error()))

		c.log.Error(cErr.Error(), "method", "getLatestSnapshotBlock")
		return nil, cErr
	}

	return sb, nil
}

func (c *chain) GetRandomSeed(snapshotHash types.Hash, n int) uint64 {
	count := uint64(10 * 60)

	headHeight, err := c.GetSnapshotHeightByHash(snapshotHash)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.GetSnapshotHeightByHash failed, snapshotHash is %s. Error: %s,",
			snapshotHash, err.Error()))
		c.log.Error(cErr.Error(), "method", "GetRandomSeed")
		return 0
	}

	tailHeight := uint64(1)
	if headHeight > count {
		tailHeight = headHeight - count
	}

	producerMap := make(map[types.Address]struct{})

	seedCount := 0
	randomSeed := uint64(0)

	for h := headHeight; h >= tailHeight && seedCount < n; h-- {
		snapshotHeader, err := c.GetSnapshotHeaderByHeight(h)
		if err != nil {
			cErr := errors.New(fmt.Sprintf("c.GetSnapshotHeaderByHeight failed, error is %s", err.Error()))
			c.log.Error(cErr.Error(), "method", "GetRandomSeed")
			return 0
		}

		if snapshotHeader == nil {
			cErr := errors.New(fmt.Sprintf("snapshotHeader is nil. height is %d", h))

			c.log.Error(cErr.Error(), "method", "GetRandomSeed")
			return 0
		}
		if snapshotHeader.Seed <= 0 {
			continue
		}

		producer := snapshotHeader.Producer()
		if _, ok := producerMap[producer]; !ok {
			randomSeed += snapshotHeader.Seed

			seedCount++
			producerMap[producer] = struct{}{}
		}
	}

	return randomSeed
}

const DefaultSeedRangeCount = 25

func (c *chain) GetSnapshotBlockByContractMeta(addr *types.Address, fromHash *types.Hash) (*ledger.SnapshotBlock, error) {
	meta, err := c.GetContractMeta(*addr)
	if err != nil {
		return nil, err
	}
	if meta == nil || meta.SendConfirmedTimes == 0 {
		return nil, nil
	}
	firstConfirmedSb, err := c.GetConfirmSnapshotHeaderByAbHash(*fromHash)
	if err != nil {
		return nil, err
	}
	if firstConfirmedSb == nil {
		return nil, errors.New("failed to find referred sendBlock' confirmSnapshotBlock")
	}
	limitSb, err := c.GetSnapshotBlockByHeight(firstConfirmedSb.Height + uint64(meta.SendConfirmedTimes) - 1)
	if err != nil {
		return nil, err
	}
	if limitSb == nil {
		return nil, errors.New("fromBlock confirmed times not enough")
	}
	return limitSb, nil
}

func (c *chain) GetSeed(limitSb *ledger.SnapshotBlock, fromHash types.Hash) (uint64, error) {
	seed := c.GetRandomSeed(limitSb.Hash, DefaultSeedRangeCount)
	seedByte := helper.LeftPadBytes(new(big.Int).SetUint64(seed).Bytes(), types.HashSize)
	var resultSeed types.Hash
	for i := 0; i < types.HashSize; i++ {
		resultSeed[i] = seedByte[i] ^ fromHash[i]
	}
	return helper.BytesToU64(resultSeed.Bytes()), nil
}

func (c *chain) GetLastSeedSnapshotHeader(producer types.Address) (*ledger.SnapshotBlock, error) {
	headHeight := c.GetLatestSnapshotBlock().Height

	tailHeight := uint64(1)
	count := uint64(10 * 60)
	if headHeight > count {
		tailHeight = headHeight - count
	}

	for h := headHeight; h >= tailHeight; h-- {

		snapshotHeader, err := c.GetSnapshotHeaderByHeight(h)
		if err != nil {
			cErr := errors.New(fmt.Sprintf("c.GetSnapshotHeaderByHeight failed, error is %s", err.Error()))
			c.log.Error(cErr.Error(), "method", "GetLastSeedSnapshotHeader")
			return nil, nil
		}

		if snapshotHeader == nil {
			cErr := errors.New(fmt.Sprintf("snapshotHeader is nil. height is %d", h))

			c.log.Error(cErr.Error(), "method", "GetRandomSeed")
			return nil, nil
		}

		if snapshotHeader.Producer() == producer && snapshotHeader.SeedHash != nil {
			return snapshotHeader, nil
		}
	}

	return nil, nil
}

// [snapshotBlock(startHeight), ...blocks... , snapshotBlock(endHeight)]
func (c *chain) GetSubLedger(startHeight, endHeight uint64) ([]*ledger.SnapshotChunk, error) {
	if startHeight <= 0 {
		startHeight = 1
	}
	latestSb, err := c.QueryLatestSnapshotBlock()
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.QueryLatestSnapshotBlock failed. Error: %s,",
			err.Error()))
		c.log.Error(cErr.Error(), "method", "GetSubLedger")
		return nil, cErr
	}
	if endHeight > latestSb.Height {
		endHeight = latestSb.Height
	}
	// query location
	startLocation, err := c.indexDB.GetSnapshotBlockLocation(startHeight)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.indexDB.GetSnapshotBlockLocation failed,  height is %d. Error: %s,",
			startHeight, err.Error()))
		c.log.Error(cErr.Error(), "method", "GetSubLedger")
		return nil, cErr
	}
	if startLocation == nil {
		return nil, nil
	}

	endLocation, err := c.indexDB.GetSnapshotBlockLocation(endHeight)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.indexDB.GetSnapshotBlockLocation failed,  height is %d. Error: %s,",
			endHeight, err.Error()))
		c.log.Error(cErr.Error(), "method", "GetSubLedger")
		return nil, cErr
	}
	if endLocation == nil {
		return nil, nil
	}

	segList, err := c.blockDB.ReadRange(startLocation, endLocation)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.blockDB.ReadRange failed, startLocation is %+v, endLocation is %+v, . Error: %s,",
			startLocation, endLocation, err.Error()))
		c.log.Error(cErr.Error(), "method", "GetSubLedger")
		return nil, cErr
	}
	return segList, nil
}

// [startHeight, GetLatestHeight]
func (c *chain) GetSubLedgerAfterHeight(height uint64) ([]*ledger.SnapshotChunk, error) {
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

type getSnapshotListFunc func() ([]*chain_file_manager.Location, [2]uint64, error)

func (c *chain) getSnapshotBlockList(getList getSnapshotListFunc, higher bool, onlyHeader bool) ([]*ledger.SnapshotBlock, error) {
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

	currentHeight := endHeight
	if higher {
		currentHeight = startHeight
	}
	index := 0

	for {
		if (higher && currentHeight > endHeight) || (!higher && currentHeight < startHeight) {
			break
		}
		var block *ledger.SnapshotBlock
		var err error
		if onlyHeader {
			block = c.cache.GetSnapshotHeaderByHeight(currentHeight)

			if block == nil {
				block, err = c.blockDB.GetSnapshotHeader(locations[index])
			}
		} else {
			block = c.cache.GetSnapshotBlockByHeight(currentHeight)
			if block == nil {
				block, err = c.blockDB.GetSnapshotBlock(locations[index])
			}
		}

		if err != nil {
			return nil, err
		}

		blocks[index] = block
		index++
		if higher {
			currentHeight++
		} else {
			currentHeight--
		}
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
		if blockMap[i] == nil {
			err = errors.New("Rolling back")
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
