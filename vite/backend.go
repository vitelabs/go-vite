package vite

import (
	ledgerHandler "github.com/vitelabs/go-vite/ledger/handler"
	"github.com/vitelabs/go-vite/p2p"
	"github.com/vitelabs/go-vite/protocols"
	"github.com/vitelabs/go-vite/wallet"

	"github.com/vitelabs/go-vite/ledger/handler_interface"
	protoInterface "github.com/vitelabs/go-vite/protocols/interfaces"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/consensus"
	"github.com/vitelabs/go-vite/miner"
	"time"
)

type Vite struct {
	ledger        *ledgerHandler.Manager
	p2p           *p2p.Server
	pm            *protocols.ProtocolManager
	walletManager *wallet.Manager
	miner         *miner.Miner
	verifier      *consensus.Verifier
}

func NewP2pConfig() *p2p.Config {
	return &p2p.Config{}
}

func New(cfg *p2p.Config) (*Vite, error) {
	//viteconfig.LoadConfig("gvite")
	//fmt.Printf("%+v\n", config.Map())

	vite := &Vite{}

	vite.ledger = ledgerHandler.NewManager(vite)
	vite.walletManager = wallet.NewManager("fromConfig")

	pwd := "123"
	coinbase, _ := types.HexToAddress("vite_2ad1b8f936f015fc80a2a5857dffb84b39f7675ab69ae31fc8")
	vite.miner = miner.NewMiner(vite.ledger.Sc(), coinbase)
	vite.walletManager.KeystoreManager.Unlock(coinbase, pwd, time.Second*10)
	vite.miner.Init()
	vite.miner.Start()
	//vite.pm = protocols.NewProtocolManager(vite)
	//
	//var initP2pErr error
	//vite.p2p, initP2pErr = p2p.NewServer(cfg, vite.pm.HandlePeer)
	//if initP2pErr != nil {
	//	log.Fatal(initP2pErr)
	//}
	//
	//vite.p2p.Start()
	return vite, nil
}

func (v *Vite) Ledger() handler_interface.Manager {
	return v.ledger
}

func (v *Vite) P2p() *p2p.Server {
	return v.p2p
}

func (v *Vite) Pm() protoInterface.ProtocolManager {
	return v.pm
}

func (v *Vite) WalletManager() *wallet.Manager {
	return v.walletManager
}

func (v *Vite) Miner() *miner.Miner {
	return v.miner
}

func (v *Vite) Verifier() consensus.Verifier {
	return v.verifier
}
