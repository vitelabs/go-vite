package rocket

import (
	"encoding/binary"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm_context"
)

var blockId = uint64(0)

func (c *chain) InsertAccountBlocks(vmAccountBlocks []*vm_context.VmAccountBlock) error {
	blockId += 1
	blockIdBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(blockIdBytes, blockId)

	batch := new(leveldb.Batch)
	for _, vmBlock := range vmAccountBlocks {
		accountBlock := vmBlock.AccountBlock
		blockBytes, err := vmBlock.AccountBlock.DbSerialize()
		if err != nil {
			return err
		}
		batch.Put(blockIdBytes, blockBytes)
		if err := c.createAccountIfNotExists(batch, &vmBlock.AccountBlock.AccountAddress, vmBlock.AccountBlock.PublicKey); err != nil {
			return err
		}

		// Save block meta
		blockMeta := &ledger.AccountBlockMeta{
			AccountId:         2,
			Height:            accountBlock.Height,
			RefSnapshotHeight: 2,
		}

		saveBlockMetaErr := c.chainDb.Ac.WriteBlockMeta(batch, &accountBlock.Hash, blockMeta)
		if saveBlockMetaErr != nil {
			c.log.Error("WriteBlockMeta failed, error is "+saveBlockMetaErr.Error(), "method", "InsertAccountBlocks")
			return saveBlockMetaErr
		}
	}
	if err := c.chainDb.Db().Write(batch, nil); err != nil {
		return err
	}
	return nil
}

func (c *chain) newAccountId() (uint64, error) {
	lastAccountId, err := c.chainDb.Account.GetLastAccountId()
	if err != nil {
		return 0, err
	}
	return lastAccountId + 1, nil
}

func (c *chain) createAccount(batch *leveldb.Batch, accountId uint64, address *types.Address, publicKey ed25519.PublicKey) (*ledger.Account, error) {
	account := &ledger.Account{
		AccountAddress: *address,
		AccountId:      accountId,
		PublicKey:      publicKey,
	}

	c.chainDb.Account.WriteAccountIndex(batch, accountId, address)
	if err := c.chainDb.Account.WriteAccount(batch, account); err != nil {
		return nil, err
	}
	return account, nil
}

func (c *chain) createAccountIfNotExists(batch *leveldb.Batch, addr *types.Address, pbkey ed25519.PublicKey) error {
	var getErr error
	isExisted := false
	if isExisted, getErr = c.chainDb.Account.IsAccountExisted(addr); getErr != nil {
		c.log.Error("GetAccountByAddress failed, error is "+getErr.Error(), "method", "InsertAccountBlocks")
		return getErr
	}

	if !isExisted {
		// Create account
		c.createAccountLock.Lock()
		defer c.createAccountLock.Unlock()

		accountId, newAccountIdErr := c.newAccountId()
		if newAccountIdErr != nil {
			c.log.Error("newAccountId failed, error is "+newAccountIdErr.Error(), "method", "InsertAccountBlocks")
			return newAccountIdErr
		}

		var caErr error
		if _, caErr = c.createAccount(batch, accountId, addr, pbkey); caErr != nil {
			c.log.Error("createAccount failed, error is "+caErr.Error(), "method", "InsertAccountBlocks")
			return caErr
		}
	}
	return nil
}
