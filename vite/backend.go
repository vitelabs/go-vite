package vite

import (
	"github.com/vitelabs/go-vite/p2p"
	"github.com/vitelabs/go-vite/protocols"
	"github.com/micro/go-config"
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/wallet"
	ledgerHandler "github.com/vitelabs/go-vite/ledger/handler"
)

type Vite struct {
	ledger *ledgerHandler.Manager
	p2p *p2p.Server
	pm *protocols.ProtocolManager
	walletManager *wallet.Manager
}

func NewP2pConfig () (p2p.Config){
	peerServer := config.Get("peerServer").StringMap(map[string]string{"name": "test-go-vite"})
	return p2p.Config{
		Name: peerServer["name"],
	}
}


func New () (*Vite, error){
	viteconfig.LoadConfig("gvite")

	vite := &Vite{}

	vite.ledger = ledgerHandler.NewManager(vite)
	vite.walletManager = wallet.NewManager("fromConfig")

	vite.pm = protocols.NewProtocolManager(vite.ledger)

	vite.p2p = &p2p.Server{
		Config: NewP2pConfig(),
	}

	vite.p2p.Start()
	return vite, nil
}

func (v *Vite) Ledger () (*ledgerHandler.Manager){
	return v.ledger
}

func (v *Vite) P2p () (*p2p.Server){
	return v.p2p
}

func (v *Vite) Pm () (*protocols.ProtocolManager)  {
	return v.pm
}

func (v *Vite) WalletManager () (*wallet.Manager)  {
	return v.walletManager
}