package handler

import (
	"github.com/vitelabs/go-vite/protocols"
	"github.com/vitelabs/go-vite/wallet"
)

type Vite interface {
	Pm () *protocols.ProtocolManager
	WalletManager() *wallet.Manager
}