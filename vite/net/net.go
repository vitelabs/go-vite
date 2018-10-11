package net

import (
	"fmt"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/p2p"
	"github.com/vitelabs/go-vite/vite/net/message"
	"github.com/vitelabs/go-vite/vite/net/topo"
	"sync"
	"time"
)

type Config struct {
	Single bool // for test

	Port     uint16
	Chain    Chain
	Verifier Verifier

	// for topo
	Topology []string
	Topic    string
	Interval int64 // second
}

const DefaultPort uint16 = 8484

type net struct {
	*Config
	peers *peerSet
	*syncer
	*fetcher
	*broadcaster
	*receiver
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
	// todo for test
	if cfg.Single {
		return mockNet()
	}

	port := cfg.Port
	if port == 0 {
		port = DefaultPort
	}

	fc := newFileClient(cfg.Chain)

	peers := newPeerSet()
	pool := newRequestPool()

	broadcaster := newBroadcaster(peers)
	filter := newFilter()
	receiver := newReceiver(cfg.Verifier, broadcaster, filter)
	syncer := newSyncer(cfg.Chain, peers, pool, receiver, fc)
	fetcher := newFetcher(filter, peers, receiver, pool)

	syncer.feed.Sub(receiver.listen) // subscribe sync status
	syncer.feed.Sub(fetcher.listen)  // subscribe sync status

	n := &net{
		Config:      cfg,
		peers:       peers,
		syncer:      syncer,
		fetcher:     fetcher,
		broadcaster: broadcaster,
		receiver:    receiver,
		fs:          newFileServer(port, cfg.Chain),
		fc:          fc,
		handlers:    make(map[cmd]MsgHandler),
		log:         log15.New("module", "vite/net"),
	}

	pool.ctx = &context{
		syncer: syncer,
		peers:  peers,
		pool:   pool,
		fc:     fc,
	}

	n.addHandler(_statusHandler(statusHandler))
	n.addHandler(&getSubLedgerHandler{cfg.Chain})
	n.addHandler(&getSnapshotBlocksHandler{cfg.Chain})
	n.addHandler(&getAccountBlocksHandler{cfg.Chain})
	n.addHandler(&getChunkHandler{cfg.Chain})
	n.addHandler(pool)     // receive all response except NewSnapshotBlockCode
	n.addHandler(receiver) // receive newBlocks
	n.addHandler(fetcher)

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
	n.topo = topo.New(&topo.Config{
		Addrs: cfg.Topology,
	})
	n.protocols = append(n.protocols, n.topo.Protocol())

	return n
}

func (n *net) Protocols() []*p2p.Protocol {
	return n.protocols
}

func (n *net) addHandler(handler MsgHandler) {
	cmds := handler.Cmds()
	for _, cmd := range cmds {
		n.handlers[cmd] = handler
	}
}

func (n *net) Start(svr *p2p.Server) (err error) {
	// todo more safe
	if n.Single {
		return nil
	}

	n.term = make(chan struct{})

	err = n.fs.start()
	if err != nil {
		return
	}

	n.fc.start()

	err = n.topo.Start(svr)

	return
}

func (n *net) Stop() {
	// todo more safe
	if n.Single {
		return
	}

	select {
	case <-n.term:
	default:
		close(n.term)

		n.syncer.Stop()

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

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	n.log.Info(fmt.Sprintf("startPeer %s", p))

	go n.syncer.Start()

	for {
		select {
		case <-n.term:
			return p2p.DiscQuitting
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

func (n *net) handleMsg(p *Peer) (err error) {
	msg, err := p.mrw.ReadMsg()
	if err != nil {
		n.log.Error(fmt.Sprintf("read message from %s error: %v", p, err))
		return
	}
	defer msg.Discard()

	code := cmd(msg.Cmd)
	n.log.Info(fmt.Sprintf("receive %s from %s", code, p))

	if code == HandshakeCode {
		n.log.Error(fmt.Sprintf("handshake twice with %s", p))
		return errHandshakeTwice
	}

	handler := n.handlers[code]
	if handler != nil {
		return handler.Handle(msg, p)
	}

	n.log.Error(fmt.Sprintf("missing handler for message %s", code))

	return fmt.Errorf("unknown message cmd %d", msg.Cmd)
}
