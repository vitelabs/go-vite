package handler

import (
	protoTypes "github.com/vitelabs/go-vite/protocols/types"
	"github.com/vitelabs/go-vite/wallet"
)

type Vite interface {
	Pm () *protoTypes.ProtocolManager
	WalletManager() *wallet.Manager
}