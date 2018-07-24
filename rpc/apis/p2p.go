package apis

import (
	"github.com/vitelabs/go-vite/log"
	"github.com/vitelabs/go-vite/p2p"
	"strconv"
)

type P2PApi interface {
	// reply true or false
	NetworkAvailable(noop interface{}, reply *string) error
	// reply an int value represents PeersCount
	PeersCount(noop interface{}, reply *string) error
}

func NewP2PApi(p2p *p2p.Server) P2PApi {
	return P2PApiImpl{
		p2p: p2p,
	}
}

type P2PApiImpl struct {
	p2p *p2p.Server
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
