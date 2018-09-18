package chain

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/ledger"
)

func (c *Chain) GetAccount(address *types.Address) (*ledger.Account, error) {
	account, err := c.chainDb.Account.GetAccountByAddress(address)
	if err != nil {
		c.log.Error("Query account failed, error is "+err.Error(), "method", "GetAccount")
		return nil, err
	}
	return account, nil
}

func (c *Chain) newAccountId() (uint64, error) {
	lastAccountId, err := c.chainDb.Account.GetLastAccountId()
	if err != nil {
		return 0, err
	}
	return lastAccountId + 1, nil
}

func (c *Chain) createAccount(batch *leveldb.Batch, accountId uint64, address *types.Address, publicKey ed25519.PublicKey) error {
	// TODO create account
	account := &ledger.Account{
		AccountAddress: *address,
		AccountId:      accountId,
		PublicKey:      publicKey,
	}

	c.chainDb.Account.WriteAccountIndex(batch, accountId, address)
	if err := c.chainDb.Account.WriteAccount(batch, account); err != nil {
		return err
	}
	return nil
}
