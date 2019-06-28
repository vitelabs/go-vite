package net

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/vitelabs/go-vite/net/discovery"

	"github.com/vitelabs/go-vite/crypto/ed25519"

	"github.com/golang/protobuf/proto"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/net/p2p"
	"github.com/vitelabs/go-vite/net/protos"
	"github.com/vitelabs/go-vite/net/vnode"
)

var netLog = log15.New("module", "net")

var errNetIsRunning = errors.New("network is already running")
var errNetIsNotRunning = errors.New("network is not running")

const maxNeighbors = 100

type net struct {
	config  *config.Net
	chain   Chain
	peerKey ed25519.PrivateKey
	nodeID  vnode.NodeID

	peers    *peerSet
	*syncer  // use pointer but not interface, because syncer can be start/stop, but interface has no start/stop method
	*fetcher // use pointer but not interface, because fetcher can be start/stop, but interface has no start/stop method
	*broadcaster
	reader     *cacheReader
	downloader syncDownloader
	BlockSubscriber
	server    *syncServer
	handlers  *msgHandlers
	query     *queryHandler
	hb        *heartBeater
	running   int32
	log       log15.Logger
	tracer    Tracer
	sn        *sbpn
	consensus Consensus
	p2p       p2p.P2P
}

func (n *net) ProtoData() (height uint64, head types.Hash, genesis types.Hash) {
	genesis = n.chain.GetGenesisSnapshotBlock().Hash
	current := n.chain.GetLatestSnapshotBlock()

	height = current.Height
	head = current.Hash

	return
}

func (n *net) ReceiveHandshake(msg *p2p.HandshakeMsg) (level p2p.Level, err error) {
	var id = msg.ID.String()
	var key string

	if msg.Key != nil {
		addr := types.PubkeyToAddress(msg.Key)
		if n.sn != nil && n.sn.isSbp(addr) {
			level = p2p.Superior
		}
	}

	for _, key2 := range n.config.AccessDenyKeys {
		if key2 == id || key2 == key {
			err = p2p.PeerNoPermission
			return
		}
	}

	if n.config.AccessControl == "any" {
		return
	}

	if strings.Contains(n.config.AccessControl, "producer") && level == p2p.Superior {
		return
	}

	for _, key2 := range n.config.AccessAllowKeys {
		if key2 == id || key2 == key {
			return
		}
	}

	err = p2p.PeerNoPermission
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

		p.SetHead(head, heartbeat.Height)

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

func (n *net) OnPeerAdded(peer *p2p.Peer) (err error) {
	p := newPeer(peer, netLog.New("peer", peer.ID()))

	err = n.peers.add(p)
	if err != nil {
		return
	}

	go n.syncer.start()

	return nil
}

func (n *net) OnPeerRemoved(peer *p2p.Peer) (err error) {
	_, err = n.peers.remove(peer.ID())

	return
}

func New(cfg *config.Net, chain Chain, verifier Verifier, consensus Consensus, irreader IrreversibleReader) (Net, error) {
	var err error

	// for test
	if cfg.Single {
		return mock(chain), nil
	}

	var peerKey ed25519.PrivateKey
	peerKey, err = cfg.Init()
	if err != nil {
		return nil, err
	}

	peers := newPeerSet()

	feed := newBlockFeeder()
	feed.setBlackHashList(cfg.BlackBlockHashList)

	forward := createForardStrategy(cfg.ForwardStrategy, peers)
	broadcaster := newBroadcaster(peers, verifier, feed, newMemBlockStore(1000), forward, nil, chain)

	receiver := &safeBlockNotifier{
		blockFeeder: feed,
		Verifier:    verifier,
	}

	var id peerId
	id, _ = vnode.Bytes2NodeID(peerKey.PubByte())
	syncConnFac := &defaultSyncConnectionFactory{
		chain:   chain,
		peers:   peers,
		id:      id,
		peerKey: peerKey,
		mineKey: cfg.MineKey,
	}
	downloader := newExecutor(50, 10, peers, syncConnFac)

	syncServer := newSyncServer(cfg.ListenInterface+":"+strconv.Itoa(cfg.FilePort), chain, syncConnFac)

	reader := newCacheReader(chain, verifier, downloader, irreader)
	reader.setBlackHashList(cfg.BlackBlockHashList)

	syncer := newSyncer(chain, peers, reader, downloader, irreader, 10*time.Minute)
	syncer.setBlackHashList(cfg.BlackBlockHashList)

	fetcher := newFetcher(peers, receiver)
	fetcher.setBlackHashList(cfg.BlackBlockHashList)

	syncer.SubscribeSyncStatus(fetcher.subSyncState)
	syncer.SubscribeSyncStatus(broadcaster.subSyncState)

	n := &net{
		config:          cfg,
		chain:           chain,
		consensus:       consensus,
		nodeID:          id,
		peerKey:         peerKey,
		BlockSubscriber: feed,
		peers:           peers,
		syncer:          syncer,
		reader:          reader,
		fetcher:         fetcher,
		broadcaster:     broadcaster,
		downloader:      downloader,
		server:          syncServer,
		handlers:        newHandlers("vite"),
		hb:              newHeartBeater(peers, chain),
		log:             netLog,
	}

	p2pConfig := &p2p.Config{
		Config: &discovery.Config{
			ListenAddress: cfg.ListenInterface + ":" + strconv.Itoa(cfg.Port),
			PublicAddress: cfg.PublicAddress,
			DataDir:       cfg.DataDir,
			PeerKey:       cfg.PeerKey,
			BootNodes:     cfg.BootNodes,
			BootSeeds:     cfg.BootSeeds,
			NetID:         cfg.NetID,
		},
		Discover:          cfg.Discover,
		Name:              cfg.Name,
		MaxPeers:          cfg.MaxPeers,
		MaxInboundRatio:   cfg.MaxInboundRatio,
		MinPeers:          cfg.MinPeers,
		MaxPendingPeers:   cfg.MaxPendingPeers,
		StaticNodes:       cfg.StaticNodes,
		FilePublicAddress: cfg.PublicAddress,
		FilePort:          cfg.FilePort,
		MineKey:           cfg.MineKey,
	}
	err = p2pConfig.Ensure()
	if err != nil {
		return nil, err
	}
	n.p2p = p2p.New(p2pConfig)
	err = n.p2p.Register(n)
	if err != nil {
		return nil, err
	}

	err = n.handlers.register(&stateHandler{
		maxNeighbors: 100,
		peers:        peers,
	})
	if err != nil {
		panic(fmt.Errorf("cannot register handler: broadcaster: %v", err))
	}

	n.query, err = newQueryHandler(chain)
	if err != nil {
		panic(fmt.Errorf("cannot construct query handler: %v", err))
	}

	// GetSubLedgerCode, CodeGetSnapshotBlocks, CodeGetAccountBlocks, GetChunkCode
	if err = n.handlers.register(n.query); err != nil {
		panic(fmt.Errorf("cannot register handler: query: %v", err))
	}

	// CodeNewSnapshotBlock, CodeNewAccountBlock
	if err = n.handlers.register(broadcaster); err != nil {
		panic(fmt.Errorf("cannot register handler: broadcaster: %v", err))
	}

	// CodeSnapshotBlocks, CodeAccountBlocks
	if err = n.handlers.register(fetcher); err != nil {
		panic(fmt.Errorf("cannot register handler: fetcher: %v", err))
	}

	// CodeSnapshotBlocks, CodeAccountBlocks
	if err = n.handlers.register(syncer); err != nil {
		panic(fmt.Errorf("cannot register handler: syncer: %v", err))
	}

	return n, nil
}

type heartBeater struct {
	chain     chainReader
	last      time.Time
	lastPeers map[peerId]struct{}
	ps        *peerSet
}

func newHeartBeater(ps *peerSet, chain chainReader) *heartBeater {
	return &heartBeater{
		chain:     chain,
		lastPeers: make(map[peerId]struct{}),
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

func (n *net) Init(consensus Consensus, reader IrreversibleReader) {
	// todo move to constructor
	n.consensus = consensus
	n.syncer.irreader = reader
	n.reader.irreader = reader
}

func (n *net) State() []byte {
	return n.hb.state()
}

func (n *net) Start() (err error) {
	if atomic.CompareAndSwapInt32(&n.running, 0, 1) {
		err = n.p2p.Start()
		if err != nil {
			return
		}

		if err = n.server.start(); err != nil {
			return
		}

		n.downloader.start()

		n.reader.start()

		n.query.start()

		n.fetcher.start()

		var addr types.Address
		if len(n.config.MineKey) != 0 {
			addr = types.PubkeyToAddress(n.config.MineKey.PubByte())
			setNodeExt(n.config.MineKey, n.p2p.Node())
			n.fetcher.setSBP()
		}
		n.sn = newSbpn(addr, n.peers, n.p2p, n.consensus)
		if discv := n.p2p.Discovery(); discv != nil {
			discv.SubscribeNode(n.sn.receiveNode)
		}

		return
	}

	return errNetIsRunning
}

func (n *net) Stop() error {
	if atomic.CompareAndSwapInt32(&n.running, 1, 0) {
		_ = n.p2p.Stop()

		n.reader.stop()

		n.syncer.stop()

		n.downloader.stop()

		_ = n.server.stop()

		n.query.stop()

		n.fetcher.stop()

		n.sn.clean()

		return nil
	}

	return errNetIsNotRunning
}

func (n *net) Nodes() []*vnode.Node {
	// todo
	return nil
}

func (n *net) PeerKey() ed25519.PrivateKey {
	return n.peerKey
}

func (n *net) Info() NodeInfo {
	info := NodeInfo{
		NodeInfo: n.p2p.Info(),
		Latency:  n.broadcaster.Statistic(),
		Peers:    n.peers.info(),
		Height:   n.chain.GetLatestSnapshotBlock().Height,
	}

	if n.server != nil {
		info.Server = n.server.status()
	}

	return info
}

type NodeInfo struct {
	p2p.NodeInfo
	Height  uint64           `json:"height"`
	Peers   []PeerInfo       `json:"peers"`
	Latency []int64          `json:"latency"` // [0,1,12,24]
	Server  FileServerStatus `json:"server"`
}
