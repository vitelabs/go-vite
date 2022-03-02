package chain

import (
	"fmt"
	"math/big"

	"github.com/vitelabs/go-vite/v2/common/types"
	"github.com/vitelabs/go-vite/v2/interfaces"
	ledger "github.com/vitelabs/go-vite/v2/interfaces/core"
	"github.com/vitelabs/go-vite/v2/vm/util"
)

func (c *chain) GetBalance(addr types.Address, tokenId types.TokenTypeId) (*big.Int, error) {
	result, err := c.stateDB.GetBalance(addr, tokenId)
	if err != nil {
		cErr := fmt.Errorf("c.stateDB.GetBalance failed, Addr is %s, tokenId is %s. Error: %s", addr, tokenId, err)
		c.log.Error(cErr.Error(), "method", "GetBalance")
		return nil, cErr
	}
	return result, nil
}
func (c *chain) GetBalanceMap(addr types.Address) (map[types.TokenTypeId]*big.Int, error) {
	result, err := c.stateDB.GetBalanceMap(addr)
	if err != nil {
		cErr := fmt.Errorf("c.stateDB.GetBalanceMap failed, Addr is %s. Error: %s,", addr, err)
		c.log.Error(cErr.Error(), "method", "GetBalance")
		return nil, cErr
	}
	return result, nil
}

// get confirmed snapshot Balance, if history is too old, failed
func (c *chain) GetConfirmedBalanceList(addrList []types.Address, tokenId types.TokenTypeId, sbHash types.Hash) (map[types.Address]*big.Int, error) {
	balanceMap := make(map[types.Address]*big.Int, len(addrList))

	if err := c.stateDB.GetSnapshotBalanceList(balanceMap, sbHash, addrList, tokenId); err != nil {
		c.log.Error(err.Error(), "method", "GetConfirmedBalance")
		return nil, err
	}

	return balanceMap, nil
}

// get contract code
func (c *chain) GetContractCode(contractAddress types.Address) ([]byte, error) {
	code, err := c.stateDB.GetCode(contractAddress)
	if err != nil {
		cErr := fmt.Errorf("c.stateDB.GetCode failed, error is %s, Addr is %s", err, contractAddress)
		c.log.Error(cErr.Error(), "method", "GetBalance")
		return nil, cErr
	}
	return code, nil
}

func (c *chain) GetContractMeta(contractAddress types.Address) (*ledger.ContractMeta, error) {
	if meta := ledger.GetBuiltinContractMeta(contractAddress); meta != nil {
		return meta, nil
	}
	meta, err := c.stateDB.GetContractMeta(contractAddress)
	if err != nil {
		cErr := fmt.Errorf("c.stateDB.GetContractMeta failed, error is %s, Addr is %s", err, contractAddress)
		c.log.Error(cErr.Error(), "method", "GetBalance")
		return nil, cErr
	}
	return meta, nil
}

func (c *chain) GetContractMetaInSnapshot(contractAddress types.Address, snapshotHeight uint64) (*ledger.ContractMeta, error) {
	if meta := ledger.GetBuiltinContractMeta(contractAddress); meta != nil {
		return meta, nil
	}

	meta, err := c.stateDB.GetContractMeta(contractAddress)
	if err != nil {
		cErr := fmt.Errorf("c.stateDB.GetContractMeta failed, error is %s, Addr is %s", err, contractAddress)
		c.log.Error(cErr.Error(), "method", "GetBalance")
		return nil, cErr
	}

	if meta == nil {
		return nil, nil
	}

	createBlockHash := meta.CreateBlockHash
	confirmedHeight, err := c.indexDB.GetConfirmHeightByHash(&createBlockHash)
	if err != nil {
		return nil, err
	}

	if confirmedHeight <= 0 || confirmedHeight > snapshotHeight {
		return nil, nil
	}

	return meta, nil
}

func (c *chain) GetContractList(gid types.Gid) ([]types.Address, error) {
	addrList, err := c.stateDB.GetContractList(&gid)
	if err != nil {
		cErr := fmt.Errorf("c.stateDB.GetContractList failed, gid is %s. Error: %s", gid, err)
		c.log.Error(cErr.Error(), "method", "GetContractList")
		return nil, cErr
	}
	if util.IsDelegateGid(gid) {
		addrList = append(addrList, types.BuiltinContracts...)
	}
	return addrList, nil
}

func (c *chain) GetVmLogList(logListHash *types.Hash) (ledger.VmLogList, error) {
	if logListHash == nil {
		return nil, nil
	}

	logList, err := c.stateDB.GetVmLogList(logListHash)
	if err != nil {
		cErr := fmt.Errorf("c.stateDB.GetVmLogList failed, error is %s, logListHash is %s", err, logListHash)
		c.log.Error(cErr.Error(), "method", "GetVmLogList")
		return nil, cErr
	}
	return logList, nil
}

// GetVmLogListByAddress query && vmLogs filter  [start,end]
func (c *chain) GetVMLogListByAddress(address types.Address, start uint64, end uint64, id *types.Hash) (ledger.VmLogList, error) {
	if !types.IsContractAddr(address) {
		return nil, nil
	}
	blocks, err := c.GetAccountBlocksByRange(address, start, end)
	if err != nil {
		return nil, err
	}
	var result ledger.VmLogList
	for _, block := range blocks {
		logs, err := c.GetVmLogList(block.LogHash)
		if err != nil {
			return nil, err
		}
		for _, log := range logs {
			if id == nil {
				result = append(result, log)
			} else if id != nil && len(log.Topics) > 0 && log.Topics[0] == *id {
				result = append(result, log)
			}
		}
	}
	return result, nil
}

func (c *chain) GetQuotaUnused(address types.Address) (uint64, error) {
	_, quotaInfo, err := c.GetStakeQuota(address)
	if err != nil {
		cErr := fmt.Errorf("c.GetStakeQuota failed, address is %s. Error: %s", address, err)
		c.log.Error(cErr.Error(), "method", "GetQuotaUnused")
		return 0, cErr
	}

	return quotaInfo.Current(), nil
}

func (c *chain) GetGlobalQuota() types.QuotaInfo {
	return c.cache.GetGlobalQuota()
}

func (c *chain) GetQuotaUsedList(address types.Address) []types.QuotaInfo {
	//return c.cache.GetQuotaUsedList(&address)
	return c.cache.GetQuotaUsedList(address)
}

func (c *chain) GetStorageIterator(address types.Address, prefix []byte) (interfaces.StorageIterator, error) {
	ss := c.stateDB.NewStorageIterator(address, prefix)
	return ss, nil
}

func (c *chain) GetValue(address types.Address, key []byte) ([]byte, error) {
	value, err := c.stateDB.GetStorageValue(&address, key)
	if err != nil {
		cErr := fmt.Errorf("c.stateDB.GetStorageValue failed, address is %s. key is %s", address, key)
		c.log.Error(cErr.Error(), "method", "GetStorageValue")
		return nil, cErr
	}
	return value, err
}
