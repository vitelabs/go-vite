package unconfirmed

import (
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/wallet"
	"github.com/vitelabs/go-vite/vm_context"
)

type Vite interface {
	WalletManager() *wallet.Manager
	Chain() *chain.Chain
	VmContext() *vm_context.Chain
}

