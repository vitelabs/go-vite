package chain

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/ledger"
)

// 0 means error, 1 means not exist, 2 means general account, 3 means contract account.
func (c *Chain) AccountType(address *types.Address) (uint64, error) {
	account, err := c.GetAccount(address)
	if err != nil {
		return ledger.AccountTypeError, err
	}

	if account == nil {
		return ledger.AccountTypeNotExist, nil
	}

	gid, getGidErr := c.GetContractGid(address)
	if getGidErr != nil {
		return ledger.AccountTypeError, getGidErr
	}

	if gid == nil {
		return ledger.AccountTypeGeneral, nil
	}
	return ledger.AccountTypeContract, nil
}

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
	account := &ledger.Account{
		AccountId: accountId,
		PublicKey: publicKey,
	}

	c.chainDb.Account.WriteAccountIndex(batch, accountId, address)
	if err := c.chainDb.Account.WriteAccount(batch, account); err != nil {
		return err
	}
	return nil
}
