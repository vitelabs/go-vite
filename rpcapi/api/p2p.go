package api

import (
	"github.com/vitelabs/go-vite/p2p"
)

func NewP2PApi(p2p *p2p.Server) P2PApi {
	return P2PApi{
		p2p: p2p,
	}
}

type P2PApi struct {
	p2p *p2p.Server
}

func (p P2PApi) String() string {
	return "P2PApi"
}

func (p P2PApi) NetworkAvailable() bool {
	log.Info("called NetworkAvailable ")
	return p.p2p.Available()

}

func (p P2PApi) PeersCount() int {
	log.Info("called PeersCount ")
	return p.p2p.PeersCount()
}
