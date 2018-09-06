package chain

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

func (c *Chain) GetSnapshotBlocks() {

}

func (c *Chain) GetSnapshotContent() {

}

func (c *Chain) GetSbAndSc() {

}

func (c *Chain) GetLatestSnapshotBlock() (block *ledger.SnapshotBlock, returnErr error) {
	defer func() {
		if returnErr != nil {
			c.log.Error(returnErr.Error(), "method", "GetLatestSnapshotBlock")
		}
	}()

	block, err := c.chainDb.Sc.GetLatestBlock()
	if err != nil {
		return nil, &types.GetError{
			Code: 1,
			Err:  err,
		}
	}

	return block, nil
}

func (c *Chain) GetGenesesSnapshotBlock() (block *ledger.SnapshotBlock, returnErr error) {
	defer func() {
		if returnErr != nil {
			c.log.Error(returnErr.Error(), "method", "GetGenesesSnapshotBlock")
		}
	}()

	block, err := c.chainDb.Sc.GetGenesesBlock()
	if err != nil {
		return nil, &types.GetError{
			Code: 1,
			Err:  err,
		}
	}

	return block, nil
}

func (c *Chain) GetSbHashList() {

}

func (c *Chain) GetConfirmBlock() {

}

func (c *Chain) GetConfirmTimes() {

}
