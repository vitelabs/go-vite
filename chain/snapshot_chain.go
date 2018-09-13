package chain

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

func (c *Chain) InsertSnapshotBlock(snapshotBlock *ledger.SnapshotBlock, needBroadCast bool) error {
	batch := new(leveldb.Batch)

	// Save snapshot block
	if err := c.chainDb.Sc.WriteSnapshotBlock(batch, snapshotBlock); err != nil {
		c.log.Error("WriteSnapshotBlock failed, error is "+err.Error(), "method", "InsertSnapshotBlock")
		return err
	}

	// Save snapshot content
	if err := c.chainDb.Sc.WriteSnapshotContent(batch, snapshotBlock.Height, snapshotBlock.SnapshotContent); err != nil {
		c.log.Error("WriteSnapshotContent failed, error is "+err.Error(), "method", "InsertSnapshotBlock")
		return err
	}

	// Save snapshot hash index
	c.chainDb.Sc.WriteSnapshotHash(batch, &snapshotBlock.Hash, snapshotBlock.Height)

	// Check and create account
	address := types.PubkeyToAddress(snapshotBlock.PublicKey)
	account, getErr := c.chainDb.Account.GetAccountByAddress(&address)

	if getErr != nil {
		c.log.Error("GetAccountByAddress failed, error is "+getErr.Error(), "method", "InsertSnapshotBlock")
		// Create account
		return getErr
	}

	if account == nil {
		c.createAccountLock.Lock()
		defer c.createAccountLock.Unlock()

		accountId, newAccountIdErr := c.newAccountId()
		if newAccountIdErr != nil {
			c.log.Error("newAccountId failed, error is "+newAccountIdErr.Error(), "method", "InsertSnapshotBlock")
		}

		if err := c.createAccount(batch, accountId, &address, snapshotBlock.PublicKey); err != nil {
			c.log.Error("createAccount failed, error is "+getErr.Error(), "method", "InsertSnapshotBlock")
			return err
		}
	}

	// Write db
	if err := c.chainDb.Commit(batch); err != nil {
		c.log.Error("c.chainDb.Commit(batch) failed, error is "+err.Error(), "method", "InsertSnapshotBlock")
		return err
	}

	return nil
}
func (c *Chain) GetSnapshotBlocks(containSnapshotContent bool) (blocks []*ledger.SnapshotBlock, returnErr error) {
	return nil, nil
}

func (c *Chain) GetSnapshotContent(snapshotHash types.Hash) (block *ledger.SnapshotContentItem, returnErr error) {
	return nil, nil
}

func (c *Chain) GetSbAndSc() {

}

func (c *Chain) GetSnapshotBlockByHash(hash *types.Hash) (block *ledger.SnapshotBlock, returnErr error) {
	return nil, nil
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
