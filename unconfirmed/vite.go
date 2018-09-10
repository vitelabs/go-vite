package unconfirmed

import (
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/wallet"
)

type Vite interface {
	WalletManager() *wallet.Manager
	Chain() *chain.Chain
}
