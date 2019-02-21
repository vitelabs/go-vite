package sbpn

import (
	"net"
	"sync"

	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/consensus"
	"github.com/vitelabs/go-vite/p2p"
	"github.com/vitelabs/go-vite/p2p/discovery"
)

/**
network between super block producers
*/

const subId = "enhance"

// Informer tell me who are super block producers
type Informer interface {
	SubscribeProducers(gid types.Gid, id string, fn func(event consensus.ProducersEvent))
	UnSubscribe(gid types.Gid, id string)
}

type target struct {
	id      discovery.NodeID
	address types.Address
	tcp     *net.TCPAddr
}

// Finder to find snapshot block producers, and connect to them
type Finder interface {
	Start(p2p p2p.Server) error
	Stop()
	Info() interface{}
	SetListener(listener Listener)
}

type Info struct {
	Addr   string
	SBPS   []string
	SPeers Peers
}

type Peers struct {
	Total  int
	Sbps   int
	Detail []Peer
}

type Peer struct {
	ID      string
	NetAddr string
	Address string
	SBP     bool
}

type Listener interface {
	GotNodeCallback(*discovery.Node)
	GotSBPSCallback([]types.Address)
	ConnectCallback(addr types.Address, id discovery.NodeID)
}

type finder struct {
	self     types.Address
	informer Informer

	nodes      sync.Map // addr: target  all nodes
	sbpTargets sync.Map // id: target  sbps

	nodeChan chan *discovery.Node
	p2p      p2p.Server
	term     chan struct{}
	wg       sync.WaitGroup
	listener Listener
}

func (f *finder) SetListener(listener Listener) {
	f.listener = listener
}

func (f *finder) Info() interface{} {
	var p2pPeers = f.p2p.Peers()

	total := len(p2pPeers)
	var speers = Peers{
		Total:  total,
		Detail: make([]Peer, 0, total),
	}

	for _, peer := range p2pPeers {
		// is sbp
		if v, ok := f.sbpTargets.Load(peer.ID); ok {
			if tgt, ok := v.(target); ok {
				speers.Detail = append(speers.Detail, Peer{
					ID:      tgt.id.String(),
					NetAddr: tgt.tcp.String(),
					Address: tgt.address.String(),
					SBP:     true,
				})
			}
		} else {
			speers.Detail = append(speers.Detail, Peer{
				ID:      peer.ID,
				NetAddr: peer.Address,
				SBP:     false,
			})
		}
	}

	return Info{
		Addr:   f.self.String(),
		SBPS:   []string{},
		SPeers: speers,
	}
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

func (f *finder) Start(svr p2p.Server) error {
	if svr == nil {
		return errors.New("p2p server is invalid")
	}

	f.term = make(chan struct{})
	svr.SubNodes(f.nodeChan)
	f.p2p = svr

	f.informer.SubscribeProducers(types.SNAPSHOT_GID, subId, f.receive)

	f.wg.Add(1)
	go f.parseLoop()

	return nil
}

func (f *finder) receive(event consensus.ProducersEvent) {
	go f.connect(event.Addrs)
}

func (f *finder) connect(addrs []types.Address) {
	if f.listener != nil {
		f.listener.GotSBPSCallback(addrs)
	}

	iAmSBP := false
	for _, addr := range addrs {
		if addr == f.self {
			iAmSBP = true
			break
		}
	}

	if !iAmSBP {
		return
	}

	var node target
	for _, addr := range addrs {
		if addr == f.self {
			continue
		}

		if v, ok := f.nodes.Load(addr); ok {
			if node, ok = v.(target); ok {
				f.sbpTargets.Store(node.id.String(), node)

				f.p2p.Connect(node.id, node.tcp)
				if f.listener != nil {
					f.listener.ConnectCallback(addr, node.id)
				}
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
			if f.listener != nil {
				f.listener.GotNodeCallback(node)
			}
		}
	}
}
