package unconfirmed

import (
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/net"
	"github.com/vitelabs/go-vite/producer"
	"github.com/vitelabs/go-vite/vm_context"
	"github.com/vitelabs/go-vite/wallet"
)

type Vite interface {
	Net() *net.Net
	Chain() *chain.Chain
	WalletManager() *wallet.Manager
	VmContext() *vm_context.Chain
	Producer() producer.Producer
}
