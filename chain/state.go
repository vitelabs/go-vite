package chain

import (
	"errors"
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/interfaces"
	"github.com/vitelabs/go-vite/ledger"
	"math/big"
)

func (c *chain) GetBalance(addr types.Address, tokenId types.TokenTypeId) (*big.Int, error) {
	result, err := c.stateDB.GetBalance(&addr, &tokenId)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.stateDB.GetBalance failed, error is %s, addr is %s, tokenId is %s", err, addr, tokenId))
		c.log.Error(cErr.Error(), "method", "GetBalance")
		return nil, cErr
	}
	return result, nil
}
func (c *chain) GetBalanceMap(addr types.Address) (map[types.TokenTypeId]*big.Int, error) {
	return nil, nil
}

// get confirmed snapshot balance, if history is too old, failed
func (c *chain) GetConfirmedBalanceList(addrList []types.Address, tokenId types.TokenTypeId, sbHash types.Hash) (map[types.Address]*big.Int, error) {
	balances, err := c.stateDB.GetSnapshotBalanceList(sbHash, addrList, tokenId)
	if err != nil {
		c.log.Error(err.Error(), "method", "GetConfirmedBalance")
		return nil, err
	}

	return balances, nil
}

// get contract code
func (c *chain) GetContractCode(contractAddress types.Address) ([]byte, error) {
	code, err := c.stateDB.GetCode(&contractAddress)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.stateDB.GetCode failed, error is %s, addr is %s", err, contractAddress))
		c.log.Error(cErr.Error(), "method", "GetBalance")
		return nil, cErr
	}
	return code, nil
}

func (c *chain) GetContractMeta(contractAddress types.Address) (*ledger.ContractMeta, error) {
	meta, err := c.stateDB.GetContractMeta(&contractAddress)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.stateDB.GetContractMeta failed, error is %s, addr is %s", err, contractAddress))
		c.log.Error(cErr.Error(), "method", "GetBalance")
		return nil, cErr
	}
	return meta, nil
}

func (c *chain) GetContractList(gid types.Gid) ([]types.Address, error) {

	addrList, err := c.stateDB.GetContractList(&gid)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.stateDB.GetContractList failed, gid is %s. Error: %s", gid, err))
		c.log.Error(cErr.Error(), "method", "GetContractList")
		return nil, cErr
	}
	return addrList, nil
}

func (c *chain) GetQuotaUnused(address types.Address) (uint64, error) {
	quotaInfo, err := c.GetPledgeQuota(address)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.GetPledgeQuota failed, address is %s. Error: %s", address, err))
		c.log.Error(cErr.Error(), "method", "GetQuotaUnused")
		return 0, cErr
	}

	return quotaInfo.Total() - quotaInfo.Used(), nil
}

func (c *chain) GetQuotaUsed(address types.Address) (quotaUsed uint64, blockCount uint64) {
	return c.cache.GetQuotaUsed(&address)
}

func (c *chain) GetStorageIterator(address types.Address, prefix []byte) (interfaces.StorageIterator, error) {
	ss := c.stateDB.NewStorageIterator(&address, prefix)
	return ss, nil
}

func (c *chain) GetValue(address types.Address, key []byte) ([]byte, error) {
	value, err := c.stateDB.GetValue(&address, key)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.stateDB.GetValue failed, address is %s. key is %s", address, key))
		c.log.Error(cErr.Error(), "method", "GetValue")
		return nil, cErr
	}
	return value, err
}
