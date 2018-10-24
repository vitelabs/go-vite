package net

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/monitor"
	"sync"
	"time"

	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/p2p"
	"github.com/vitelabs/go-vite/vite/net/message"
	"github.com/vitelabs/go-vite/vite/net/topo"
)

var netLog = log15.New("module", "vite/net")

type Config struct {
	Single bool // for test

	Port     uint16
	Chain    Chain
	Verifier Verifier

	// for topo
	Topology     []string
	Topic        string
	Interval     int64 // second
	TopoDisabled bool
}

const DefaultPort uint16 = 8484

type net struct {
	*Config
	peers *peerSet
	*syncer
	*fetcher
	*broadcaster
	*receiver
	pool      *requestPool
	term      chan struct{}
	log       log15.Logger
	protocols []*p2p.Protocol // mount to p2p.Server
	wg        sync.WaitGroup
	fs        *fileServer
	fc        *fileClient
	handlers  map[cmd]MsgHandler
	topo      *topo.Topology
}

// auto from
func New(cfg *Config) Net {
	// for test
	if cfg.Single {
		return mock()
	}

	if cfg.Port == 0 {
		cfg.Port = DefaultPort
	}

	fc := newFileClient(cfg.Chain)

	peers := newPeerSet()
	pool := newRequestPool(peers, fc)

	broadcaster := newBroadcaster(peers)
	filter := newFilter()
	receiver := newReceiver(cfg.Verifier, broadcaster, filter)
	syncer := newSyncer(cfg.Chain, peers, pool, receiver)
	fetcher := newFetcher(filter, peers, pool)

	syncer.feed.Sub(receiver.listen) // subscribe sync status
	syncer.feed.Sub(fetcher.listen)  // subscribe sync status

	n := &net{
		Config:      cfg,
		peers:       peers,
		syncer:      syncer,
		fetcher:     fetcher,
		broadcaster: broadcaster,
		receiver:    receiver,
		fs:          newFileServer(cfg.Port, cfg.Chain),
		fc:          fc,
		handlers:    make(map[cmd]MsgHandler),
		log:         netLog,
		pool:        pool,
	}

	n.addHandler(_statusHandler(statusHandler))
	n.addHandler(&getSubLedgerHandler{cfg.Chain})
	n.addHandler(&getSnapshotBlocksHandler{cfg.Chain})
	n.addHandler(&getAccountBlocksHandler{cfg.Chain})
	n.addHandler(&getChunkHandler{cfg.Chain})
	n.addHandler(pool)     // FileListCode, SubLedgerCode, ExceptionCode
	n.addHandler(receiver) // NewSnapshotBlockCode, NewAccountBlockCode, SnapshotBlocksCode, AccountBlocksCode

	n.protocols = append(n.protocols, &p2p.Protocol{
		Name: CmdSetName,
		ID:   CmdSet,
		Handle: func(p *p2p.Peer, rw p2p.MsgReadWriter) error {
			// will be called by p2p.Peer.runProtocols use goroutine
			peer := newPeer(p, rw, CmdSet)
			return n.handlePeer(peer)
		},
	})

	// topo
	if !cfg.TopoDisabled {
		n.topo = topo.New(&topo.Config{
			Addrs:    cfg.Topology,
			Interval: cfg.Interval,
			Topic:    cfg.Topic,
		})
		n.protocols = append(n.protocols, n.topo.Protocol())
	}

	return n
}

func (n *net) Protocols() []*p2p.Protocol {
	return n.protocols
}

func (n *net) addHandler(handler MsgHandler) {
	for _, cmd := range handler.Cmds() {
		n.handlers[cmd] = handler
	}
}

func (n *net) Start(svr *p2p.Server) (err error) {
	n.term = make(chan struct{})

	if err = n.fs.start(); err != nil {
		return
	}

	n.fc.start()

	if n.topo != nil {
		if err = n.topo.Start(svr); err != nil {
			return
		}
	}

	n.pool.start()

	return
}

func (n *net) Stop() {
	if n.term == nil {
		return
	}

	select {
	case <-n.term:
	default:
		close(n.term)

		n.syncer.Stop()

		n.pool.start()

		n.fs.stop()

		n.fc.stop()

		if n.topo != nil {
			n.topo.Stop()
		}

		n.wg.Wait()
	}
}

// will be called by p2p.Server, run as goroutine
func (n *net) handlePeer(p *Peer) error {
	current := n.Chain.GetLatestSnapshotBlock()
	genesis := n.Chain.GetGenesisSnapshotBlock()

	n.log.Info(fmt.Sprintf("handshake with %s", p))
	err := p.Handshake(&message.HandShake{
		CmdSet:  p.CmdSet,
		Height:  current.Height,
		Port:    n.Port,
		Current: current.Hash,
		Genesis: genesis.Hash,
	})

	if err != nil {
		n.log.Error(fmt.Sprintf("handshake with %s error: %v", p, err))
		return err
	}
	n.log.Info(fmt.Sprintf("handshake with %s done", p))

	return n.startPeer(p)
}

func (n *net) startPeer(p *Peer) error {
	n.peers.Add(p)
	defer n.peers.Del(p)

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	n.log.Info(fmt.Sprintf("startPeer %s", p))

	common.Go(n.syncer.Start)

	for {
		select {
		case <-n.term:
			return p2p.DiscQuitting

		//case err := <-p.errch:
		//	n.log.Error(fmt.Sprintf("peer error: %v", err))
		//	return p2p.DiscProtocolError

		case <-ticker.C:
			current := n.Chain.GetLatestSnapshotBlock()
			p.Send(StatusCode, 0, &ledger.HashHeight{
				Hash:   current.Hash,
				Height: current.Height,
			})
		default:
			if err := n.handleMsg(p); err != nil {
				return err
			}
		}
	}
}

var errMissHandler = errors.New("missing message handler")

func (n *net) handleMsg(p *Peer) (err error) {
	msg, err := p.mrw.ReadMsg()
	if err != nil {
		n.log.Error(fmt.Sprintf("read message from %s error: %v", p, err))
		return
	}

	code := cmd(msg.Cmd)

	if handler, ok := n.handlers[code]; ok && handler != nil {
		n.log.Info(fmt.Sprintf("begin handle message %s", code))

		begin := time.Now()
		err = handler.Handle(msg, p)
		monitor.LogDuration("net", "handle_"+code.String(), time.Now().Sub(begin).Nanoseconds())

		n.log.Info(fmt.Sprintf("handle message %s done", code))
		return err
	}

	n.log.Error(fmt.Sprintf("missing handler for message %d", msg.Cmd))

	return errMissHandler
}

func (n *net) Info() *NodeInfo {
	return &NodeInfo{
		Peers: n.peers.Info(),
	}
}

type NodeInfo struct {
	Peers []*PeerInfo `json:"peers"`
}
