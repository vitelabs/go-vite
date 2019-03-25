package chain

import (
	"errors"
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/interfaces"
	"github.com/vitelabs/go-vite/ledger"
	"math/big"
)

func (c *chain) GetBalance(addr *types.Address, tokenId *types.TokenTypeId) (*big.Int, error) {
	result, err := c.stateDB.GetBalance(addr, tokenId)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.stateDB.GetBalance failed, error is %s, addr is %s, tokenId is %s", err, addr, tokenId))
		c.log.Error(cErr.Error(), "method", "GetBalance")
		return nil, cErr
	}
	return result, nil
}

// get confirmed snapshot balance, if history is too old, failed
func (c *chain) GetConfirmedBalance(addr *types.Address, tokenId *types.TokenTypeId, sbHash *types.Hash) (*big.Int, error) {
	balance, err := c.stateDB.GetSnapshotBalance(addr, tokenId, sbHash)
	if err != nil {
		c.log.Error(err.Error(), "method", "GetConfirmedBalance")
		return nil, err
	}

	return balance, nil
}

// get contract code
func (c *chain) GetContractCode(contractAddress *types.Address) ([]byte, error) {
	code, err := c.stateDB.GetCode(contractAddress)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.stateDB.GetCode failed, error is %s, addr is %s", err, contractAddress))
		c.log.Error(cErr.Error(), "method", "GetBalance")
		return nil, cErr
	}
	return code, nil
}

func (c *chain) GetContractMeta(contractAddress *types.Address) (*ledger.ContractMeta, error) {
	meta, err := c.stateDB.GetContractMeta(contractAddress)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.stateDB.GetContractMeta failed, error is %s, addr is %s", err, contractAddress))
		c.log.Error(cErr.Error(), "method", "GetBalance")
		return nil, cErr
	}
	return meta, nil
}

// TODO
func (c *chain) GetContractList(gid *types.Gid) (map[types.Address]*ledger.ContractMeta, error) {
	// do something
	c.stateDB.GetValue(&types.AddressConsensusGroup, gid.Bytes())
	return nil, nil
}

func (c *chain) GetQuotaUnused(address *types.Address) (uint64, error) {
	totalQuota, err := c.GetPledgeQuota(&c.GetLatestSnapshotBlock().Hash, address)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.GetPledgeQuota failed, address is %s. Error: %s", address, err))
		c.log.Error(cErr.Error(), "method", "GetQuotaUnused")
		return 0, cErr
	}

	quotaUsed, _ := c.GetQuotaUsed(address)
	return totalQuota - quotaUsed, nil
}

func (c *chain) GetQuotaUsed(address *types.Address) (quotaUsed uint64, blockCount uint64) {
	return c.cache.GetQuotaUsed(address)
}

func (c *chain) GetStateIterator(address *types.Address, prefix []byte) (interfaces.StorageIterator, error) {
	ss, err := c.stateDB.NewStorageIterator(address, prefix)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.stateDB.NewStorageIterator failed, address is %s. prefix is %s", address, prefix))
		c.log.Error(cErr.Error(), "method", "GetStorageIterator")
		return nil, cErr
	}
	return ss, nil
}

func (c *chain) GetValue(address *types.Address, key []byte) ([]byte, error) {
	value, err := c.stateDB.GetValue(address, key)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.stateDB.GetValue failed, address is %s. key is %s", address, key))
		c.log.Error(cErr.Error(), "method", "GetValue")
		return nil, cErr
	}
	return value, err
}
