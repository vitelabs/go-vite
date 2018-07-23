package handler

import (
	"github.com/vitelabs/go-vite/wallet"
	protoInterface "github.com/vitelabs/go-vite/protocols/interfaces"
	"github.com/vitelabs/go-vite/miner"
	"github.com/vitelabs/go-vite/consensus"
)

type Vite interface {
	Pm () protoInterface.ProtocolManager
	WalletManager() *wallet.Manager
	Miner() *miner.Miner
	Verifier() consensus.Verifier
}