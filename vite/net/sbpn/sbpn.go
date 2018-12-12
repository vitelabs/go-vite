package sbpn

import (
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/p2p/discovery"
	"net"
	"sync"
	"time"
)

/**
network between super block producers
*/

// tell me who are super block producers
type Informer interface {
	SubscribeProducers()
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
	}
}

func New(informer Informer) Finder {
	return &finder{}
}

func (f *finder) Start(p2p P2P) error {
	if p2p == nil {
		return errors.New("p2p server is invalid")
	}

	f.term = make(chan struct{})
	p2p.SubNodes(f.nodeChan)

	f.wg.Add(1)
	go f.loop()

	f.wg.Add(1)
	go f.parseLoop()

	return nil
}

func (f *finder) loop() {
	defer f.wg.Done()

	interval := 75 * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

loop:
	for {
		select {
		case <-f.term:
			break loop
		case <-ticker.C:
			// todo get fellows and connect
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
