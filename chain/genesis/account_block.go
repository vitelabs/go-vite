package chain_genesis

import (
	"encoding/hex"
	"math/big"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm_db"
)

func NewGenesisAccountBlocks(cfg *config.Genesis) []*vm_db.VmAccountBlock {
	list := make([]*vm_db.VmAccountBlock, 0)
	list = newGenesisContractAccountBlocks(cfg, list)
	list = newGenesisNormalAccountBlocks(cfg, list)
	return list
}

func newGenesisContractAccountBlocks(cfg *config.Genesis, list []*vm_db.VmAccountBlock) []*vm_db.VmAccountBlock {
	for addrStr, storageMap := range cfg.ContractStorageMap {
		addr, err := types.HexToAddress(addrStr)
		if err != nil {
			panic(err)
		}
		block := ledger.AccountBlock{
			BlockType:      ledger.BlockTypeGenesisReceive,
			Height:         1,
			AccountAddress: addr,
			Amount:         big.NewInt(0),
			Fee:            big.NewInt(0),
		}

		vmdb := vm_db.NewEmptyVmDB(&addr)
		for k, v := range storageMap {
			kBytes, err := hex.DecodeString(k)
			if err != nil {
				panic(err)
			}
			vBytes, err := hex.DecodeString(v)
			if err != nil {
				panic(err)
			}
			err = vmdb.SetValue(kBytes, vBytes)
			if err != nil {
				panic(err)
			}
		}
		if len(cfg.ContractLogsMap) > 0 {
			if logList, ok := cfg.ContractLogsMap[addrStr]; ok && len(logList) > 0 {
				for _, log := range logList {
					dataBytes, err := hex.DecodeString(log.Data)
					if err != nil {
						panic(err)
					}
					vmdb.AddLog(&ledger.VmLog{Data: dataBytes, Topics: log.Topics})
				}
			}
		}
		list = append(list, &vm_db.VmAccountBlock{&block, vmdb})
	}
	return list
}

func newGenesisNormalAccountBlocks(cfg *config.Genesis, list []*vm_db.VmAccountBlock) []*vm_db.VmAccountBlock {
	for addrStr, balanceMap := range cfg.AccountBalanceMap {
		addr, err := types.HexToAddress(addrStr)
		if err != nil {
			panic(err)
		}
		block := ledger.AccountBlock{
			BlockType:      ledger.BlockTypeGenesisReceive,
			Height:         1,
			AccountAddress: addr,
			Amount:         big.NewInt(0),
			Fee:            big.NewInt(0),
		}

		vmdb := vm_db.NewEmptyVmDB(&addr)
		for tokenIdStr, balance := range balanceMap {
			tokenId, err := types.HexToTokenTypeId(tokenIdStr)
			if err != nil {
				panic(err)
			}
			vmdb.SetBalance(&tokenId, balance)
		}
		list = append(list, &vm_db.VmAccountBlock{&block, vmdb})
	}
	return list
}
