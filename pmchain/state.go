package pmchain

import (
	"errors"
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
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
	ss, err := c.getStateSnapshot(addr, sbHash)
	if err != nil {
		c.log.Error(err.Error(), "method", "GetConfirmedBalance")
		return nil, err
	}
	defer ss.Release()

	balance, err := ss.GetBalance(tokenId)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("ss.GetBalance failed, addr is %s, tokenId is %s, sbHash is %s", addr, tokenId, sbHash))
		c.log.Error(cErr.Error(), "method", "GetConfirmedBalance")
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

func (c *chain) GetQuotaUnused(address *types.Address) uint64 {
	return 0
}

func (c *chain) GetQuotaUsed(address *types.Address) (quotaUsed uint64, blockCount uint64) {
	return 0, 0
}
