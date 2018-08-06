package apis

import (
	"github.com/vitelabs/go-vite/p2p"
	"strconv"
	"github.com/vitelabs/go-vite/rpc/api_interface"
)

func NewP2PApi(p2p *p2p.Server) api_interface.P2PApi {
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

func (p P2PApiImpl) NetworkAvailable(noop interface{}, reply *string) error {
	log.Debug("called NetworkAvailable ")
	*reply = strconv.FormatBool(p.p2p.Available())
	return nil

}

func (p P2PApiImpl) PeersCount(noop interface{}, reply *string) error {
	log.Debug("called PeersCount ")
	*reply = strconv.Itoa(p.p2p.PeersCount())
	return nil
}
