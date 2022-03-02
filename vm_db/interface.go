package vm_db

import (
	"math/big"

	"github.com/vitelabs/go-vite/v2/common/types"
	"github.com/vitelabs/go-vite/v2/interfaces"
	"github.com/vitelabs/go-vite/v2/interfaces/core"
)

type Chain interface {
	GetQuotaUsedList(address types.Address) []types.QuotaInfo

	GetGlobalQuota() types.QuotaInfo

	GetBalance(addr types.Address, tokenId types.TokenTypeId) (*big.Int, error)

	GetContractCode(contractAddr types.Address) ([]byte, error)

	GetContractMeta(contractAddress types.Address) (meta *core.ContractMeta, err error)

	GetConfirmSnapshotHeaderByAbHash(abHash types.Hash) (*core.SnapshotBlock, error)

	GetConfirmedTimes(blockHash types.Hash) (uint64, error)

	GetContractMetaInSnapshot(contractAddress types.Address, snapshotHeight uint64) (meta *core.ContractMeta, err error)

	GetSnapshotHeaderByHash(hash types.Hash) (*core.SnapshotBlock, error)

	GetSnapshotBlockByHeight(height uint64) (*core.SnapshotBlock, error)

	GetAccountBlockByHash(blockHash types.Hash) (*core.AccountBlock, error)

	GetLatestAccountBlock(addr types.Address) (*core.AccountBlock, error)

	GetVmLogList(logHash *types.Hash) (core.VmLogList, error)

	GetUnconfirmedBlocks(addr types.Address) []*core.AccountBlock

	GetGenesisSnapshotBlock() *core.SnapshotBlock

	GetStakeBeneficialAmount(addr types.Address) (*big.Int, error)

	GetStorageIterator(address types.Address, prefix []byte) (interfaces.StorageIterator, error)

	GetValue(addr types.Address, key []byte) ([]byte, error)

	GetCallDepth(sendBlockHash types.Hash) (uint16, error)

	GetSnapshotBlockByContractMeta(addr types.Address, fromHash types.Hash) (*core.SnapshotBlock, error)

	GetSeedConfirmedSnapshotBlock(addr types.Address, fromHash types.Hash) (*core.SnapshotBlock, error)

	GetSeed(limitSb *core.SnapshotBlock, fromHash types.Hash) (uint64, error)
}
