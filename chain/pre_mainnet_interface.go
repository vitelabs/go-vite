package chain

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm_context"
)

type PreMainNetChain interface {
	/*
	 *	Lifecycle
	 */

	/*
	 *	C(Create)
	 */

	// vmAccountBlocks must have the same address
	InsertAccountBlocks(vmAccountBlocks []*vm_context.VmAccountBlock) error

	InsertSnapshotBlock(snapshotBlock *ledger.SnapshotBlock) error

	/*
	 *	D(Delete)
	 */

	// contain the account block of toHashHeight
	DeleteAccountBlocks(addr *types.Address, toHashHeight *ledger.HashHeight) (map[types.Address][]*ledger.AccountBlock, error)

	// contain the account block of toHeight
	DeleteAccountBlocksToHeight(addr *types.Address, toHeight uint64) (map[types.Address][]*ledger.AccountBlock, error)

	// contain the snapshot block of toHashHeight
	DeleteSnapshotBlocks(toHashHeight *ledger.HashHeight) (map[types.Address][]*ledger.AccountBlock, error)

	// contain the snapshot block of toHeight
	DeleteSnapshotBlocksToHeight(toHeight uint64) (map[types.Address][]*ledger.AccountBlock, error)

	/*
	 *	R(Retrieve)
	 */

	// ====== Query account block ======
	IsAccountBlockExisted(hash *types.Hash) (bool, error)

	GetAccountBlockByHeight(addr *types.Address, height uint64) (*ledger.AccountBlock, error)

	GetAccountBlockByHash(blockHash *types.Hash) (*ledger.AccountBlock, error)

	// query receive blocks of send block
	GetReceiveAbsBySendAb(sendBlockHash *types.Hash) ([]*ledger.AccountBlock, error)

	// query send block of receive block
	GetSendAbByReceiveAb(receiveBlockHash *types.Hash) ([]*ledger.AccountBlock, error)

	// is snapped
	IsConfirmed(abHashHeight *ledger.HashHeight) (uint64, error)

	// ====== Query snapshot block ======
	IsSnapshotBlockExisted(hash *types.Hash) (bool, error)

	GetLatestSnapshotBlock() (*ledger.SnapshotBlock, error)

	GetSnapshotHeadByHeight(height uint64) (*ledger.SnapshotBlock, error)

	GetSnapshotBlockByHeight(height uint64) (*ledger.SnapshotBlock, error)

	GetSnapshotHeadByHash(hash *types.Hash) (*ledger.SnapshotBlock, error)

	GetSnapshotBlockByHash(hash *types.Hash) (*ledger.SnapshotBlock, error)

	GetConfirmSnapshotHeadByAbHashHeight(abHashHeight *ledger.HashHeight) (*ledger.SnapshotBlock, error)

	GetConfirmSnapshotBlockByAbHashHeight(abHashHeight *ledger.HashHeight) (*ledger.SnapshotBlock, error)

	GetSnapshotBodyBySbHash(hash *types.Hash) (*ledger.SnapshotContent, error)

	// ====== Query account ======
	AccountType(address *types.Address) (byte, error)

	// ====== Query quota ======
	GetQuotaOwned(address *types.Address, sbHashHeight *ledger.HashHeight) uint64

	GetQuotaUsed(address *types.Address, sbHashHeight *ledger.HashHeight) uint64

	// ====== Query consensus info ======
	GetConsensusInfo(address *types.Address, sbHashHeight *ledger.HashHeight) (*types.ConsensusGroupInfo, error)
}
