package handler

import (
	protoInterface "github.com/vitelabs/go-vite/protocols/interfaces"
	"github.com/vitelabs/go-vite/wallet"
	"github.com/vitelabs/go-vite/consensus"
	"github.com/vitelabs/go-vite/miner"
)

type Vite interface {
	Pm() protoInterface.ProtocolManager
	WalletManager() *wallet.Manager
	Miner() *miner.Miner
	Verifier() consensus.Verifier
}
