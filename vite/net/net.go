package net

import (
	"errors"
	"fmt"
	"sync"
	"time"

	net2 "net"

	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/monitor"
	"github.com/vitelabs/go-vite/p2p"
	"github.com/vitelabs/go-vite/vite/net/message"
	"github.com/vitelabs/go-vite/vite/net/topo"
)

var netLog = log15.New("module", "vite/net")

type Config struct {
	Single bool // for test

	FileAddr string
	Chain    Chain
	Verifier Verifier

	// for topo
	Topology    []string
	Topic       string
	Interval    int64 // second
	TopoEnabled bool
}

const DefaultPort uint16 = 8484

type net struct {
	*Config
	peers *peerSet
	*syncer
	*fetcher
	*broadcaster
	*receiver
	*filter
	query     *queryHandler // handle query message (eg. getAccountBlocks, getSnapshotblocks, getChunk, getSubLedger)
	term      chan struct{}
	log       log15.Logger
	protocols []*p2p.Protocol // mount to p2p.server
	wg        sync.WaitGroup
	fs        *fileServer
	handlers  map[ViteCmd]MsgHandler
	topo      *topo.Topology
	plugins   []p2p.Plugin
}

// auto from
func New(cfg *Config) Net {
	// for test
	if cfg.Single {
		return mock()
	}

	g := new(gid)
	peers := newPeerSet()

	broadcaster := newBroadcaster(peers)
	filter := newFilter()
	receiver := newReceiver(cfg.Verifier, broadcaster, filter, nil)
	syncer := newSyncer(cfg.Chain, peers, g, receiver)
	fetcher := newFetcher(filter, peers, g)

	syncer.feed.Sub(receiver.listen) // subscribe sync status
	syncer.feed.Sub(fetcher.listen)  // subscribe sync status

	n := &net{
		Config:      cfg,
		peers:       peers,
		syncer:      syncer,
		fetcher:     fetcher,
		broadcaster: broadcaster,
		receiver:    receiver,
		filter:      filter,
		fs:          newFileServer(cfg.FileAddr, cfg.Chain),
		handlers:    make(map[ViteCmd]MsgHandler),
		log:         netLog,
	}

	n.addHandler(_statusHandler(statusHandler))
	n.query = newQueryHandler(cfg.Chain)
	n.addHandler(n.query)  // GetSubLedgerCode, GetSnapshotBlocksCode, GetAccountBlocksCode, GetChunkCode
	n.addHandler(syncer)   // FileListCode, SubLedgerCode
	n.addHandler(receiver) // NewSnapshotBlockCode, NewAccountBlockCode, SnapshotBlocksCode, AccountBlocksCode

	n.protocols = append(n.protocols, &p2p.Protocol{
		Name: Vite,
		ID:   CmdSet,
		Handle: func(p *p2p.Peer, rw *p2p.ProtoFrame) error {
			// will be called by p2p.Peer.runProtocols use goroutine
			peer := newPeer(p, rw, CmdSet)
			return n.handlePeer(peer)
		},
	})

	// topo
	if cfg.TopoEnabled {
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

func (n *net) AddPlugin(plugin p2p.Plugin) {
	n.plugins = append(n.plugins, plugin)
}

func (n *net) startPlugins(svr p2p.Server) (err error) {
	for _, plugin := range n.plugins {
		if err = plugin.Start(svr); err != nil {
			return
		}
	}
	return nil
}

func (n *net) addHandler(handler MsgHandler) {
	for _, cmd := range handler.Cmds() {
		n.handlers[cmd] = handler
	}
}

func (n *net) Start(svr p2p.Server) (err error) {
	n.term = make(chan struct{})

	n.receiver.p2p = svr

	if err = n.fs.start(); err != nil {
		return
	}

	if n.topo != nil {
		if err = n.topo.Start(svr); err != nil {
			return
		}
	}

	if err = n.startPlugins(svr); err != nil {
		return
	}

	n.wg.Add(1)
	common.Go(n.heartbeat)

	n.query.start()

	n.filter.start()

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

		n.fs.stop()

		if n.topo != nil {
			n.topo.Stop()
		}

		n.query.stop()

		n.filter.stop()

		n.wg.Wait()
	}
}

// will be called by p2p.server, run as goroutine
func (n *net) handlePeer(p *peer) error {
	current := n.Chain.GetLatestSnapshotBlock()
	genesis := n.Chain.GetGenesisSnapshotBlock()

	n.log.Debug(fmt.Sprintf("handshake with %s", p))

	var filePort uint16
	fileAddress, err := net2.ResolveTCPAddr("tcp", n.FileAddr)
	if err != nil {
		filePort = uint16(fileAddress.Port)
	} else {
		filePort = DefaultPort
	}

	err = p.Handshake(&message.HandShake{
		Height:  current.Height,
		Port:    filePort,
		Current: current.Hash,
		Genesis: genesis.Hash,
	})

	if err != nil {
		n.log.Error(fmt.Sprintf("handshake with %s error: %v", p, err))
		return err
	}

	n.log.Debug(fmt.Sprintf("handshake with %s done", p))

	return n.startPeer(p)
}

func (n *net) startPeer(p *peer) (err error) {
	n.peers.Add(p)
	defer n.peers.Del(p)

	n.log.Debug(fmt.Sprintf("startPeer %s", p))

	common.Go(n.syncer.Start)

loop:
	for {
		select {
		case <-n.term:
			err = p2p.DiscQuitting
			break loop

		case err = <-p.errChan:
			if err != nil {
				p.log.Error(fmt.Sprintf("peer %s error: %v", p.RemoteAddr(), err))
				break loop
			}

		default:
			if err := n.handleMsg(p); err != nil {
				return err
			}
		}
	}

	close(p.term)
	p.wg.Wait()

	return err
}

func (n *net) heartbeat() {
	defer n.wg.Done()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	var height uint64

	for {
		select {
		case <-n.term:
			return

		case <-ticker.C:
			l := n.peers.Peers()
			if len(l) == 0 {
				break
			}

			current := n.Chain.GetLatestSnapshotBlock()
			if height == current.Height {
				break
			}

			height = current.Height
			currentHeight = height

			for _, p := range l {
				p.Send(StatusCode, 0, &ledger.HashHeight{
					Hash:   current.Hash,
					Height: current.Height,
				})
			}

			monitor.LogEvent("net", "heartbeat")
		}
	}
}

var errMissHandler = errors.New("missing message handler")

func (n *net) handleMsg(p *peer) (err error) {
	msg, err := p.mrw.ReadMsg()
	if err != nil {
		n.log.Error(fmt.Sprintf("read message from %s error: %v", p, err))
		return
	}

	code := ViteCmd(msg.Cmd)

	// before syncDone, ignore GetAccountBlocksCode
	if n.syncer.SyncState() != Syncdone {
		if code == GetAccountBlocksCode {
			// ignore
			return
		}
	}

	if handler, ok := n.handlers[code]; ok && handler != nil {
		return handler.Handle(msg, p)
	}

	return nil
}

func (n *net) Info() *NodeInfo {
	peersInfo := n.peers.Info()

	return &NodeInfo{
		PeerCount: len(peersInfo),
		Peers:     peersInfo,
		Latency:   n.broadcaster.Statistic(),
	}
}

type NodeInfo struct {
	PeerCount int         `json:"peerCount"`
	Peers     []*PeerInfo `json:"peers"`
	Latency   []int64     `json:"latency"` // [0,1,12,24]
}

type Task struct {
	Msg    string `json:"Msg"`
	Sender string `json:"Sender"`
}

func (n *net) Tasks() (ts []*Task) {
	n.query.lock.RLock()
	defer n.query.lock.RUnlock()

	ts = make([]*Task, n.query.queue.Size())
	i := 0

	n.query.queue.Traverse(func(value interface{}) bool {
		t := value.(*queryTask)
		ts[i] = &Task{
			Msg:    ViteCmd(t.Msg.Cmd).String(),
			Sender: t.Sender.RemoteAddr().String(),
		}
		i++
		return true
	})

	return
}
