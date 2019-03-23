package net

import (
	"fmt"
	"sync"
	"time"

	"github.com/vitelabs/go-vite/crypto/ed25519"

	"github.com/vitelabs/go-vite/p2p/vnode"

	"github.com/vitelabs/go-vite/common/types"

	"github.com/vitelabs/go-vite/vite/net/protos"

	"github.com/golang/protobuf/proto"

	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/p2p"
)

var netLog = log15.New("module", "vite/net")

type Config struct {
	Single      bool // for test
	FileAddress string
	Chain       Chain
	Verifier    Verifier
	PeerKey     ed25519.PrivateKey
}

const DefaultFilePort uint16 = 8484
const ID = 1

type net struct {
	Config
	peers    *peerSet
	*syncer  // use pointer but not interface, because syncer can be start/stop, but interface has no start/stop method
	*fetcher // use pointer but not interface, because fetcher can be start/stop, but interface has no start/stop method
	*broadcaster
	BlockSubscriber
	query    *queryHandler // handle query message (eg. getAccountBlocks, getSnapshotblocks, getChunk, getSubLedger)
	term     chan struct{}
	log      log15.Logger
	wg       sync.WaitGroup
	fs       *fileServer
	handlers map[code]MsgHandler
	hb       *heartBeater
}

func (n *net) ProtoData() []byte {

}

func (n *net) ReceiveHandshake(msg p2p.HandshakeMsg, protoData []byte) (state interface{}, level p2p.Level, err error) {
	panic("implement me")
}

func (n *net) Name() string {
	return "vite"
}

func (n *net) ID() p2p.ProtocolID {
	return ID
}

func (n *net) Handle(msg p2p.Msg) error {
	if handler, ok := n.handlers[code(msg.Code)]; ok {
		p := n.peers.get(msg.Sender.ID())
		if p != nil {
			return handler.Handle(msg, p)
		} else {
			return errPeerNotExist
		}
	}

	return p2p.PeerUnknownMessage
}

func (n *net) SetState(state []byte, peer p2p.Peer) {
	var heartbeat = &protos.ProtocolState{}

	err := proto.Unmarshal(state, heartbeat)
	if err != nil {
		n.log.Error(fmt.Sprintf("Failed to unmarshal heartbeat message: %v", err))
		return
	}

	p := n.peers.get(peer.ID())
	if p != nil {
		var head types.Hash
		head, err = types.BytesToHash(heartbeat.Head)
		if err != nil {
			return
		}

		p.setHead(head, heartbeat.Height)

		var pl = make([]peerConn, len(heartbeat.Peers))
		for i, hp := range heartbeat.Peers {
			pl[i] = peerConn{
				id:  hp.ID,
				add: hp.Status != protos.ProtocolState_Disconnected,
			}
		}

		p.setPeers(pl, heartbeat.Patch)
	}
}

func (n *net) OnPeerAdded(peer p2p.Peer) (err error) {
	p := newPeer(peer, n.log.New("peer", peer.ID()))

	err = n.peers.add(p)
	if err != nil {
		return
	}

	// todo sync

	return nil
}

func (n *net) OnPeerRemoved(peer p2p.Peer) (err error) {
	_, err = n.peers.remove(peer.ID())

	return
}

func New(cfg Config) Net {
	// wraper_verifier
	cfg.Verifier = newVerifier(cfg.Verifier)

	// for test
	if cfg.Single {
		return mock(cfg)
	}

	g := new(gid)
	peers := newPeerSet()

	feed := newBlockFeeder()

	forward := newRedForwardStrategy(peers, 3, 30)
	broadcaster := newBroadcaster(peers, cfg.Verifier, feed, newMemBlockStore(1000), forward, nil, netLog.New("module", "broadcast"))
	syncer := newSyncer(cfg.Chain, peers, cfg.Verifier, g, feed)
	fetcher := newFetcher(peers, g, cfg.Verifier, feed)

	syncer.SubscribeSyncStatus(fetcher.subSyncState)     // subscribe sync status
	syncer.SubscribeSyncStatus(broadcaster.subSyncState) // subscribe sync status

	n := &net{
		Config:          cfg,
		BlockSubscriber: feed,
		peers:           peers,
		syncer:          syncer,
		fetcher:         fetcher,
		broadcaster:     broadcaster,
		fs:              newFileServer(cfg.FileAddress, cfg.Chain),
		handlers:        make(map[code]MsgHandler),
		log:             netLog,
		hb:              newHeartBeater(peers, netLog.New("module", "heartbeat")),
	}

	n.addHandler(_statusHandler(statusHandler))
	n.query = newQueryHandler(cfg.Chain)
	n.addHandler(n.query) // GetSubLedgerCode, GetSnapshotBlocksCode, GetAccountBlocksCode, GetChunkCode
	// n.addHandler(broadcaster) // NewSnapshotBlockCode, NewAccountBlockCode
	n.addHandler(fetcher) // SnapshotBlocksCode, AccountBlocksCode

	return n
}

type heartBeater struct {
	last      time.Time
	lastPeers map[vnode.NodeID]struct{}
	ps        *peerSet
	log       log15.Logger
}

func newHeartBeater(ps *peerSet, log log15.Logger) *heartBeater {
	return &heartBeater{
		ps:  ps,
		log: log,
	}
}

func (h *heartBeater) state() []byte {
	var heartBeat = &protos.ProtocolState{
		Peers:     nil,
		Patch:     true,
		Timestamp: time.Now().Unix(),
	}

	idMap := h.ps.idMap()

	if time.Now().Sub(h.last) > time.Minute {
		heartBeat.Patch = false
		h.lastPeers = make(map[vnode.NodeID]struct{})
	}

	var id vnode.NodeID
	var ok bool
	for id = range h.lastPeers {
		if _, ok = idMap[id]; ok {
			continue
		}
		heartBeat.Peers = append(heartBeat.Peers, &protos.ProtocolState_Peer{
			ID:     id[:],
			Status: protos.ProtocolState_Disconnected,
		})
	}

	for id = range idMap {
		if _, ok = h.lastPeers[id]; ok {
			continue
		}
		heartBeat.Peers = append(heartBeat.Peers, &protos.ProtocolState_Peer{
			ID:     id[:],
			Status: protos.ProtocolState_Connected,
		})
	}

	data, err := proto.Marshal(heartBeat)
	if err != nil {

		return nil
	}

	return data
}

func (n *net) State() []byte {
	return n.hb.state()
}

func (n *net) addHandler(handler MsgHandler) {
	for _, cmd := range handler.Cmds() {
		n.handlers[cmd] = handler
	}
}

func (n *net) Start(svr p2p.Server) (err error) {
	n.term = make(chan struct{})

	if err = n.fs.start(); err != nil {
		return
	}

	n.query.start()

	n.fetcher.start()

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

		n.query.stop()

		n.fetcher.stop()

		n.wg.Wait()
	}
}

func (n *net) Info() NodeInfo {
	return NodeInfo{
		Latency: n.broadcaster.Statistic(),
	}
}

type NodeInfo struct {
	PeerCount int     `json:"peerCount"`
	Latency   []int64 `json:"latency"` // [0,1,12,24]
}
