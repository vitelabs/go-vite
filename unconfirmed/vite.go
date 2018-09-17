package unconfirmed

import (
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/net"
	"github.com/vitelabs/go-vite/producer"
	"github.com/vitelabs/go-vite/vm_context"
	"github.com/vitelabs/go-vite/wallet"
)

type Vite interface {
	Net() *net.Net
	Chain() *chain.Chain
	WalletManager() *wallet.Manager
	Producer() producer.Producer
	PoolReader
	ConsensusVerifier
}

type PoolReader interface {
	ExistInPool(address types.Address, fromBlockHash types.Hash) bool
	AddDirectAccountBlock(address types.Address, vmAccountBlock *vm_context.VmAccountBlock) error
	AddDirectAccountBlocks(address types.Address, received *vm_context.VmAccountBlock, sendBlocks []*vm_context.VmAccountBlock) error
}

type ConsensusVerifier interface {
	VerifyAccountProducer(block *ledger.AccountBlock) error
}
