package vite

import (
	"github.com/vitelabs/go-vite/p2p"
	"github.com/vitelabs/go-vite/protocols"
	"github.com/vitelabs/go-vite/wallet"
	ledgerHandler "github.com/vitelabs/go-vite/ledger/handler"

	"github.com/vitelabs/go-vite/ledger/handler_interface"
	protoInterface "github.com/vitelabs/go-vite/protocols/interfaces"

)

type Vite struct {
	ledger *ledgerHandler.Manager
	p2p *p2p.Server
	pm *protocols.ProtocolManager
	walletManager *wallet.Manager
}

func NewP2pConfig () *p2p.Config {
	return &p2p.Config{}
}

func New (cfg *p2p.Config) (*Vite, error){
	//viteconfig.LoadConfig("gvite")
	//fmt.Printf("%+v\n", config.Map())


	vite := &Vite{}

	vite.ledger = ledgerHandler.NewManager(vite)
	vite.walletManager = wallet.NewManager("fromConfig")

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

func (v *Vite) Ledger () (handler_interface.Manager){
	return v.ledger
}

func (v *Vite) P2p () (*p2p.Server){
	return v.p2p
}

func (v *Vite) Pm () (protoInterface.ProtocolManager)  {
	return v.pm
}

func (v *Vite) WalletManager () (*wallet.Manager)  {
	return v.walletManager
}