package handler

import (
	"github.com/vitelabs/go-vite/wallet"
	protoInterface "github.com/vitelabs/go-vite/protocols/interfaces"
)

type Vite interface {
	Pm () protoInterface.ProtocolManager
	WalletManager() *wallet.Manager
}