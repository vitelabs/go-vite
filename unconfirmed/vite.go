package unconfirmed

import (
	"github.com/vitelabs/go-vite/wallet"
	"github.com/vitelabs/go-vite/ledger/handler_interface"
)

type Vite interface {
	WalletManager() *wallet.Manager
	Ledger() handler_interface.Manager
}
