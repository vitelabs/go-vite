package net

import (
	"encoding/hex"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/p2p"
	"github.com/vitelabs/go-vite/p2p/vnode"
	"github.com/vitelabs/go-vite/vite/net/protos"
)

var netLog = log15.New("module", "net")

var errNetIsRunning = errors.New("network is already running")
var errNetIsNotRunning = errors.New("network is not running")
var errInvalidSignature = errors.New("invalid signature")
var errDiffGenesisBlock = errors.New("different genesis block")
var errErrorHeadToHash = errors.New("error head to hash")

type Config struct {
	Single            bool
	FileListenAddress string
	MinePrivateKey    ed25519.PrivateKey
	P2PPrivateKey     ed25519.PrivateKey
	ForwardStrategy   string // default `cross`
	Chain
	Verifier
	Producer
}

const DefaultForwardStrategy = "cross"
const DefaultFilePort = 8484
const ID = 1
const maxNeighbors = 100

type net struct {
	Config
	nodeID   vnode.NodeID
	peers    *peerSet
	*syncer  // use pointer but not interface, because syncer can be start/stop, but interface has no start/stop method
	*fetcher // use pointer but not interface, because fetcher can be start/stop, but interface has no start/stop method
	*broadcaster
	reader     syncCacheReader
	downloader syncDownloader
	BlockSubscriber
	server   *syncServer
	handlers *msgHandlers
	query    *queryHandler
	hb       *heartBeater
	running  int32
	log      log15.Logger
	tracer   *tracer
}

func (n *net) ProtoData() (key []byte, height uint64, genesis types.Hash) {
	genesis = n.Chain.GetGenesisSnapshotBlock().Hash
	height = n.Chain.GetLatestSnapshotBlock().Height

	return
}

func (n *net) ReceiveHandshake(msg *p2p.HandshakeMsg) (level p2p.Level, err error) {
	//if msg.Key != nil {
	//	right := ed25519.Verify(msg.Key, msg.ID.Bytes(), msg.Signature)
	//	if !right {
	//		err = errInvalidSignature
	//		return
	//	}
	//
	//	addr := types.PubkeyToAddress(msg.Key)
	//	if n.Producer != nil && n.Producer.IsProducer(addr) {
	//		level = p2p.Superior
	//	}
	//}

	return
}

func (n *net) Handle(msg p2p.Msg) error {
	p := n.peers.get(msg.Sender.ID())
	if p != nil {
		return n.handlers.handle(msg, p)
	} else {
		return errPeerNotExist
	}
}

func (n *net) SetState(state []byte, peer p2p.Peer) {
	var heartbeat = &protos.State{}

	err := proto.Unmarshal(state, heartbeat)
	if err != nil {
		n.log.Error(fmt.Sprintf("failed to unmarshal heartbeat message from %s: %v", peer, err))
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

		// max 100 neighbors
		var count = len(heartbeat.Peers)
		if count > maxNeighbors {
			count = maxNeighbors
		}
		var pl = make([]peerConn, count)
		for i := 0; i < count; i++ {
			hp := heartbeat.Peers[i]
			pl[i] = peerConn{
				id:  hp.ID,
				add: hp.Status != protos.State_Disconnected,
			}
		}
		p.setPeers(pl, heartbeat.Patch)
	}
}

func (n *net) OnPeerAdded(peer p2p.Peer) (err error) {
	p := newPeer(peer, netLog.New("peer", peer.ID()))

	err = n.peers.add(p)
	if err != nil {
		return
	}

	go n.syncer.start()

	return nil
}

func (n *net) OnPeerRemoved(peer p2p.Peer) (err error) {
	_, err = n.peers.remove(peer.ID())

	return
}

func New(cfg Config) Net {
	// wraper_verifier
	//cfg.Verifier = newVerifier(cfg.Verifier)

	// for test
	if cfg.Single {
		return mock(cfg)
	}

	peers := newPeerSet()

	feed := newBlockFeeder()

	forward := createForardStrategy(cfg.ForwardStrategy, peers)
	broadcaster := newBroadcaster(peers, cfg.Verifier, feed, newMemBlockStore(1000), forward, nil, cfg.Chain)

	receiver := &safeBlockNotifier{
		blockFeeder: feed,
		Verifier:    cfg.Verifier,
	}

	syncConnFac := &defaultSyncConnectionFactory{
		chain:      cfg.Chain,
		peers:      peers,
		privateKey: cfg.P2PPrivateKey,
	}
	downloader := newExecutor(100, 10, newPool(peers), syncConnFac)
	reader := newCacheReader(cfg.Chain, cfg.Verifier, downloader)
	syncer := newSyncer(cfg.Chain, peers, reader, downloader, 10*time.Minute)

	fetcher := newFetcher(peers, receiver)

	syncer.SubscribeSyncStatus(fetcher.subSyncState)
	syncer.SubscribeSyncStatus(broadcaster.subSyncState)

	n := &net{
		Config:          cfg,
		BlockSubscriber: feed,
		peers:           peers,
		syncer:          syncer,
		reader:          reader,
		fetcher:         fetcher,
		broadcaster:     broadcaster,
		downloader:      downloader,
		server:          newSyncServer(cfg.FileListenAddress, cfg.Chain, syncConnFac),
		handlers:        newHandlers("vite"),
		hb:              newHeartBeater(peers, cfg.Chain),
		log:             netLog,
	}

	var err error
	n.query, err = newQueryHandler(cfg.Chain)
	if err != nil {
		panic(fmt.Errorf("cannot construct query handler: %v", err))
	}

	// GetSubLedgerCode, GetSnapshotBlocksCode, GetAccountBlocksCode, GetChunkCode
	if err = n.handlers.register(n.query); err != nil {
		panic(fmt.Errorf("cannot register handler: query: %v", err))
	}

	// NewSnapshotBlockCode, NewAccountBlockCode
	if err = n.handlers.register(broadcaster); err != nil {
		panic(fmt.Errorf("cannot register handler: broadcaster: %v", err))
	}

	// SnapshotBlocksCode, AccountBlocksCode
	if err = n.handlers.register(fetcher); err != nil {
		panic(fmt.Errorf("cannot register handler: fetcher: %v", err))
	}

	// trace
	var p2pPub = cfg.P2PPrivateKey.PubByte()
	n.tracer = newTracer(hex.EncodeToString(p2pPub), peers, forward)
	if err = n.handlers.register(n.tracer); err != nil {
		panic(fmt.Errorf("cannot register handler: tracer: %v", err))
	}

	return n
}

type heartBeater struct {
	chain     chainReader
	last      time.Time
	lastPeers map[vnode.NodeID]struct{}
	ps        *peerSet
}

func newHeartBeater(ps *peerSet, chain chainReader) *heartBeater {
	return &heartBeater{
		chain:     chain,
		lastPeers: make(map[vnode.NodeID]struct{}),
		ps:        ps,
	}
}

func (h *heartBeater) state() []byte {
	current := h.chain.GetLatestSnapshotBlock()

	var heartBeat = &protos.State{
		Peers:     nil,
		Patch:     true,
		Head:      current.Hash.Bytes(),
		Height:    current.Height,
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
			delete(idMap, id)
			continue
		}
		heartBeat.Peers = append(heartBeat.Peers, &protos.State_Peer{
			ID:     id.Bytes(),
			Status: protos.State_Disconnected,
		})
	}

	for id = range idMap {
		heartBeat.Peers = append(heartBeat.Peers, &protos.State_Peer{
			ID:     id.Bytes(),
			Status: protos.State_Connected,
		})
	}

	data, err := proto.Marshal(heartBeat)
	if err != nil {
		return nil
	}

	h.lastPeers = idMap

	return data
}

func (n *net) State() []byte {
	return n.hb.state()
}

func (n *net) Start(svr p2p.P2P) (err error) {
	if atomic.CompareAndSwapInt32(&n.running, 0, 1) {
		n.nodeID = svr.Config().Node().ID

		if n.Producer != nil && n.MinePrivateKey != nil {
			addr := types.PubkeyToAddress(n.MinePrivateKey.PubByte())
			if n.Producer.IsProducer(addr) {
				// todo set finder
			}
		}

		fmt.Println(n.Chain.GetLatestSnapshotBlock().Height)

		if err = n.server.start(); err != nil {
			return
		}

		n.downloader.start()

		n.reader.start()

		n.query.start()

		n.fetcher.start()

		return
	}

	return errNetIsRunning
}

func (n *net) Stop() error {
	if atomic.CompareAndSwapInt32(&n.running, 1, 0) {
		n.reader.stop()

		n.syncer.stop()

		n.downloader.stop()

		_ = n.server.stop()

		n.query.stop()

		n.fetcher.stop()

		return nil
	}

	return errNetIsNotRunning
}

func (n *net) Trace() {
	if n.tracer != nil {
		n.tracer.Trace()
	}
}

func (n *net) Info() NodeInfo {
	info := NodeInfo{
		Latency: n.broadcaster.Statistic(),
		Peers:   n.peers.info(),
	}

	if n.server != nil {
		info.Server = n.server.status()
	}

	return info
}

type NodeInfo struct {
	Peers   []PeerInfo       `json:"peers"`
	Latency []int64          `json:"latency"` // [0,1,12,24]
	Server  FileServerStatus `json:"server"`
}
