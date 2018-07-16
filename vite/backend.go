package vite

import (
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/p2p"
	"github.com/vitelabs/go-vite/protocols"
	"github.com/micro/go-config"
	"github.com/vitelabs/go-vite/config"
)

type Vite struct {
	ledger *ledger.Ledger
	p2p *p2p.Server
	pm *protocols.ProtocolManager
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

	vite.ledger = ledger.NewLedger(vite)

	vite.pm = protocols.NewProtocolManager(vite.ledger)

	vite.p2p = &p2p.Server{
		Config: NewP2pConfig(),
	}

	vite.p2p.Start()
	vite.pm.Start()

}