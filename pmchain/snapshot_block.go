package pmchain

import (
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"time"
)

func (c *chain) IsSnapshotBlockExisted(hash *types.Hash) (bool, error) {
	return false, nil
}

// is valid
func (c *chain) IsSnapshotContentValid(snapshotContent *ledger.SnapshotContent) (invalidMap map[types.Address]*ledger.HashHeight, err error) {
	return nil, nil
}
func (c *chain) GetGenesisSnapshotHeader() *ledger.SnapshotBlock {
	return nil
}

func (c *chain) GetGenesisSnapshotBlock() *ledger.SnapshotBlock {
	return nil
}

func (c *chain) GetLatestSnapshotHeader() *ledger.SnapshotBlock {
	return nil
}

func (c *chain) GetLatestSnapshotBlock() *ledger.SnapshotBlock {
	return nil
}

// header without snapshot content
func (c *chain) GetSnapshotHeaderByHeight(height uint64) (*ledger.SnapshotBlock, error) {
	return nil, nil
}

// block contains header、snapshot content
func (c *chain) GetSnapshotBlockByHeight(height uint64) (*ledger.SnapshotBlock, error) {
	return nil, nil
}

func (c *chain) GetSnapshotHeaderByHash(hash *types.Hash) (*ledger.SnapshotBlock, error) {
	// cache
	if block := c.cache.GetSnapshotHeaderByHash(hash); block != nil {
		return block, nil
	}

	// query location
	location, err := c.indexDB.GetSnapshotBlockLocationByHash(hash)
	if err != nil {
		c.log.Error(fmt.Sprintf("c.indexDB.GetSnapshotBlockLocationByHash failed, error is %s, hash is %s, location is %s",
			err.Error(), hash, location), "method", "GetSnapshotHeaderByHash")
		return nil, err
	}

	// query block
	sb, err := c.blockDB.GetSnapshotBlock(location)
	if err != nil {
		c.log.Error(fmt.Sprintf("c.blockDB.GetSnapshotBlock failed, error is %s, hash is %s, location is %s",
			err.Error(), hash, location), "method", "GetSnapshotHeaderByHash")
		return nil, err
	}

	return sb, nil
}

func (c *chain) GetSnapshotBlockByHash(hash *types.Hash) (*ledger.SnapshotBlock, error) {
	return nil, nil
}

// contains start snapshot block and end snapshot block
func (c *chain) GetRangeSnapshotHeaders(startHash *types.Hash, endHash *types.Hash) ([]*ledger.SnapshotBlock, error) {
	return nil, nil
}

func (c *chain) GetRangeSnapshotBlocks(startHash *types.Hash, endHash *types.Hash) ([]*ledger.SnapshotBlock, error) {
	return nil, nil
}

// contains the snapshot header that has the blockHash
func (c *chain) GetSnapshotHeaders(blockHash *types.Hash, higher bool, count uint64) ([]*ledger.SnapshotBlock, error) {
	return nil, nil
}

// contains the snapshot block that has the blockHash
func (c *chain) GetSnapshotBlocks(blockHash *types.Hash, higher bool, count uint64) ([]*ledger.SnapshotBlock, error) {
	return nil, nil
}

func (c *chain) GetConfirmSnapshotHeaderByAbHash(abHash *types.Hash) (*ledger.SnapshotBlock, error) {
	return nil, nil
}

func (c *chain) GetConfirmSnapshotBlockByAbHash(abHash *types.Hash) (*ledger.SnapshotBlock, error) {
	return nil, nil
}

func (c *chain) GetSnapshotHeaderBeforeTime(timestamp *time.Time) (*ledger.SnapshotBlock, error) {
	return nil, nil
}

func (c *chain) GetSnapshotHeadersAfterOrEqualTime(endHashHeight *ledger.HashHeight, startTime *time.Time, producer *types.Address) ([]*ledger.SnapshotBlock, error) {
	return nil, nil
}

func (c *chain) GetSnapshotContentBySbHash(hash *types.Hash) (ledger.SnapshotContent, error) {
	return nil, nil
}
