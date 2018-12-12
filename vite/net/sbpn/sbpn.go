package sbpn

import (
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/consensus"
	"github.com/vitelabs/go-vite/p2p/discovery"
	"net"
	"sync"
)

/**
network between super block producers
*/

const subId = "enhance"

// tell me who are super block producers
type Informer interface {
	SubscribeProducers(gid types.Gid, id string, fn func(event consensus.ProducersEvent))
	UnSubscribe(gid types.Gid, id string)
}

type P2P interface {
	SubNodes(ch chan<- *discovery.Node)
	UnSubNodes(ch chan<- *discovery.Node)
	Connect(id discovery.NodeID, addr *net.TCPAddr)
}

type target struct {
	id      discovery.NodeID
	address types.Address
	tcp     *net.TCPAddr
}

type Finder interface {
	Start(p2p P2P) error
	Stop()
}

type finder struct {
	self     types.Address
	informer Informer
	nodes    sync.Map
	nodeChan chan *discovery.Node
	p2p      P2P
	term     chan struct{}
	wg       sync.WaitGroup
}

func (f *finder) Stop() {
	if f.term == nil {
		return
	}

	select {
	case <-f.term:
	default:
		close(f.term)
		f.wg.Wait()
		f.p2p.UnSubNodes(f.nodeChan)
		f.informer.UnSubscribe(types.SNAPSHOT_GID, subId)
	}
}

func New(addr types.Address, informer Informer) Finder {
	return &finder{
		self:     addr,
		informer: informer,
		nodes:    sync.Map{},
		nodeChan: make(chan *discovery.Node, 16),
	}
}

func (f *finder) Start(p2p P2P) error {
	if p2p == nil {
		return errors.New("p2p server is invalid")
	}

	f.term = make(chan struct{})
	p2p.SubNodes(f.nodeChan)

	f.informer.SubscribeProducers(types.SNAPSHOT_GID, subId, f.receive)

	f.wg.Add(1)
	go f.parseLoop()

	return nil
}

func (f *finder) receive(event consensus.ProducersEvent) {
	go f.connect(event.Addrs)
}

func (f *finder) connect(addrs []types.Address) {
	var node *target
	for _, addr := range addrs {
		if v, ok := f.nodes.Load(addr); ok {
			if node, ok = v.(*target); ok {
				f.p2p.Connect(node.id, node.tcp)
			}
		}
	}
}

func (f *finder) parseLoop() {
	defer f.wg.Done()
loop:
	for {
		select {
		case <-f.term:
			break loop
		case node := <-f.nodeChan:
			t := nodeParser(node)
			f.nodes.Store(t.address, t)
		}
	}
}
