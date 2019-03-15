package pmchain

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
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
func (c *chain) IsSnapshotContentValid(snapshotContent *ledger.SnapshotContent) (invalidMap map[types.Address]*ledger.HashHeight, err error) {
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

	return nil, nil
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
		c.log.Error(cErr.Error(), "method", "GetSnapshotHeaderByHash")
		return nil, cErr
	}

	// query block
	snapshotBlock, err := c.blockDB.GetSnapshotBlock(location)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.blockDB.GetSnapshotBlock failed, error is %s, height is %d, location is %s",
			err.Error(), height, location))
		c.log.Error(cErr.Error(), "method", "GetSnapshotHeaderByHash")
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
	snapshotBlock, err := c.blockDB.GetSnapshotBlock(location)
	if err != nil {
		c.log.Error(fmt.Sprintf("c.blockDB.GetSnapshotBlock failed, error is %s, hash is %s, location is %s",
			err.Error(), hash, location), "method", "GetSnapshotHeaderByHash")
		return nil, err
	}

	return snapshotBlock, nil
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

func (c *chain) getLatestSnapshotBlock() (*ledger.SnapshotBlock, error) {
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
