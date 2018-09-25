package verifier

import (
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

type Chain interface {
	//AccountReader
	//SnapshotReader
	Chain() *chain.Chain
}

type Consensus interface {
	VerifyAccountProducer(block *ledger.AccountBlock) error
}

type Signer interface {
	SignData(a types.Address, data []byte) (signedData, pubkey []byte, err error)
	SignDataWithPassphrase(a types.Address, passphrase string, data []byte) (signedData, pubkey []byte, err error)
}

//type SnapshotReader interface {
//	GetSnapshotBlockByHash(hash *types.Hash) (*ledger.SnapshotBlock, error)
//	GetConfirmBlock(accountBlock *ledger.AccountBlock) *ledger.SnapshotBlock
//	GetConfirmTimes(accountBlock *ledger.AccountBlock) uint64
//}
//
//type AccountReader interface {
//	GetLatestAccountBlock(addr *types.Address) (*ledger.AccountBlock, error)
//	GetAccountBlockByHash(blockHash *types.Hash) (*ledger.AccountBlock, error)
//	GetContractGid(addr *types.Address) (*types.Gid, error)
//}
