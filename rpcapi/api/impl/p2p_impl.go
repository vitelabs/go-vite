package impl

import (
	"github.com/vitelabs/go-vite/p2p"
	"github.com/vitelabs/go-vite/rpcapi/api"
)

func NewP2PApi(p2p *p2p.Server) api.P2PApi {
	return P2PApiImpl{
		p2p: p2p,
	}
}

type P2PApiImpl struct {
	p2p *p2p.Server
}

func (p P2PApiImpl) String() string {
	return "P2PApiImpl"
}

func (p P2PApiImpl) NetworkAvailable() bool {
	log.Info("called NetworkAvailable ")
	return p.p2p.Available()

}

func (p P2PApiImpl) PeersCount() int {
	log.Info("called PeersCount ")
	return p.p2p.PeersCount()
}
