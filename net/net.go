package net

import (
	"encoding/binary"
	"errors"
	"fmt"
	_net "net"
	"path"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vitelabs/go-vite/ledger"

	"github.com/vitelabs/go-vite/net/netool"

	"github.com/vitelabs/go-vite/vitepb"

	"github.com/golang/protobuf/proto"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/net/database"
	"github.com/vitelabs/go-vite/net/discovery"
	"github.com/vitelabs/go-vite/net/vnode"
)

var netLog = log15.New("module", "net")

var errNetIsRunning = errors.New("network is already running")
var errNetIsNotRunning = errors.New("network is not running")

const maxNeighbors = 100
const DBDirName = "db"

type net struct {
	config  *config.Net
	peerKey ed25519.PrivateKey
	node    *vnode.Node

	finder *finder

	discover *discovery.Discovery

	db *database.DB

	dialer       _net.Dialer
	listener     _net.Listener
	hkr          *handshaker
	receiveSlots chan struct{}

	confirmedHashHeightList []*ledger.HashHeight

	syncServer *syncServer

	peers *peerSet

	chain Chain

	*syncer  // use pointer but not interface, because syncer can be start/stop, but interface has no start/stop method
	*fetcher // use pointer but not interface, because fetcher can be start/stop, but interface has no start/stop method
	*broadcaster
	reader     *cacheReader
	downloader syncDownloader
	BlockSubscriber
	handlers *msgHandlers
	query    *queryHandler
	hb       *heartBeater

	blackList netool.BlackList

	running int32

	log log15.Logger

	wg sync.WaitGroup
}

func (n *net) listenLoop() {
	defer n.wg.Done()

	n.receiveSlots = make(chan struct{}, n.config.MaxPendingPeers)

	var tempDelay time.Duration
	var maxDelay = time.Second

	for {
		var err error
		var conn _net.Conn
		conn, err = n.listener.Accept()
		if err != nil {
			if ne, ok := err.(_net.Error); ok && ne.Temporary() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}

				if tempDelay > maxDelay {
					tempDelay = maxDelay
				}

				time.Sleep(tempDelay)

				continue
			}
			break
		}

		n.receiveSlots <- struct{}{}
		go n.onConnection(conn, peerId{}, true)
	}
}

func (n *net) ConnectNode(node *vnode.Node) (err error) {
	if n.peers.has(node.ID) {
		return PeerAlreadyConnected
	}

	if node.ID == n.node.ID {
		err = PeerConnectSelf
		return
	}

	if n.blackList.Banned(node.ID.Bytes()) {
		return fmt.Errorf("node %s has been banned", node.ID)
	}

	conn, err := _net.Dial("tcp", node.Address())
	if err != nil {
		n.blackList.Ban(node.ID.Bytes(), 10)
		return
	}

	go n.onConnection(conn, node.ID, false)

	return nil
}

func (n *net) onConnection(conn _net.Conn, id peerId, inbound bool) {
	var c Codec
	var their *HandshakeMsg
	var superior bool
	var err error
	var flag PeerFlag
	if inbound {
		flag = PeerFlagInbound
		c, their, superior, err = n.hkr.ReceiveHandshake(conn)
		<-n.receiveSlots
	} else {
		flag = PeerFlagOutbound
		c, their, superior, err = n.hkr.InitiateHandshake(conn, id)
	}

	if err != nil {
		_ = Disconnect(c, err)
		n.log.Warn(fmt.Sprintf("failed to handshake with %s: %v", conn.RemoteAddr(), err))
		return
	}

	var fileAddress, publicAddress string
	addr := conn.RemoteAddr()
	if tcpAddr, ok := addr.(*_net.TCPAddr); ok {
		publicAddress = extractAddress(tcpAddr, their.FileAddress, 8483)
		fileAddress = extractAddress(tcpAddr, their.FileAddress, 8484)
	}

	peer := newPeer(c, their, publicAddress, fileAddress, superior, flag, n.peers, n.handlers)

	// run peer
	_ = n.onPeerAdded(peer)

	n.onPeerRemoved(peer)
}

func (n *net) authorize(c Codec, flag PeerFlag, msg *HandshakeMsg) (superior bool, err error) {
	if msg.ID == n.node.ID {
		err = PeerConnectSelf
		return
	}

	if n.peers.has(msg.ID) {
		err = PeerAlreadyConnected
		return
	}

	// is deny
	var id = msg.ID.String()
	var key string
	if msg.Key != nil {
		key = msg.Key.Hex()
	}
	for _, key2 := range n.config.AccessDenyKeys {
		if key2 == id || key2 == key {
			err = PeerNoPermission
			return
		}
	}

	// superior
	if msg.Key != nil {
		addr := types.PubkeyToAddress(msg.Key)
		if n.finder.isSBP(addr) {
			superior = true
		}
	}

	// whitelist
	for _, key2 := range n.config.AccessAllowKeys {
		if key2 == id || key2 == key {
			return
		}
	}

	// producer
	if superior {
		return
	}

	// no space
	if n.peers.countWithoutSBP() >= n.config.MaxPeers {
		err = PeerTooManyPeers
		return
	}

	if n.peers.inboundWithoutSBP() >= (n.config.MaxPeers / n.config.MaxInboundRatio) {
		err = PeerTooManyInboundPeers
		return
	}

	if n.config.AccessControl == "any" {
		return
	}

	err = PeerNoPermission
	return
}

func (n *net) checkPeer(peer *Peer) {
	if len(n.confirmedHashHeightList) == 0 {
		// default is reliable
		peer.setReliable(true)
		return
	}

	if false == n.peers.has(peer.Id) {
		return
	}

	highest := n.confirmedHashHeightList[0].Height
	// peer is too high
	if peer.isReliable() && peer.Height > highest+3600*24 {
		return
	}

	var shouldCheck bool
	for _, hashheight := range n.confirmedHashHeightList {
		if hashheight.Height <= peer.Height {
			shouldCheck = true
			n.fetcher.fetchSnapshotBlock(hashheight.Hash, peer, func(msg Msg, err error) {
				var interval time.Duration
				if err != nil {
					peer.setReliable(false)
					interval = time.Minute
				} else {
					peer.setReliable(true)
					interval = 10 * time.Minute
				}

				time.AfterFunc(interval, func() {
					n.checkPeer(peer)
				})
			})
			break
		}
	}

	if false == shouldCheck {
		peer.setReliable(true)
		lowest := n.confirmedHashHeightList[len(n.confirmedHashHeightList)-1].Height
		if lowest > peer.Height {
			const maxDuration = 10 * time.Minute
			duration := time.Duration(lowest-peer.Height) * time.Second
			if duration > maxDuration {
				duration = maxDuration
			}
			time.AfterFunc(duration, func() {
				n.checkPeer(peer)
			})
		}
	}
}

func (n *net) onPeerAdded(peer *Peer) (err error) {
	err = n.peers.add(peer)
	if err != nil {
		_ = Disconnect(peer.codec, err)
		return
	}

	go n.checkPeer(peer)

	if err = peer.run(); err != nil {
		n.blackList.Ban(peer.Id.Bytes(), 10)
		n.log.Warn(fmt.Sprintf("peer %s run done: %v", peer, err))
	} else {
		n.log.Info(fmt.Sprintf("peer %s run done", peer))
	}

	_ = peer.Close(err)

	return
}

func (n *net) onPeerRemoved(peer *Peer) {
	_, _ = n.peers.remove(peer.Id)

	return
}

func retrieveAddressBytesFromConfig(address string, port int) (data []byte, err error) {
	if address != "" {
		var ep vnode.EndPoint
		ep, err = vnode.ParseEndPoint(address)
		if err != nil {
			return nil, err
		}
		data, err = ep.Serialize()
		if err != nil {
			return nil, err
		}
	} else {
		data = make([]byte, 2)
		binary.BigEndian.PutUint16(data, uint16(port))
	}

	return
}

func New(cfg *config.Net, chain Chain, verifier Verifier, consensus Consensus, irreader IrreversibleReader) (Net, error) {
	// for test
	if cfg.Single {
		return mock(chain), nil
	}

	var err error

	var blackHashList = make(map[types.Hash]struct{}, len(cfg.BlackBlockHashList))
	for _, hexStr := range cfg.BlackBlockHashList {
		strs := strings.Split(hexStr, "/")
		var hash types.Hash
		hash, err = types.HexToHash(strs[0])
		if err != nil {
			return nil, err
		}
		blackHashList[hash] = struct{}{}
	}

	// from high to low
	var confirmedHashList = make([]*ledger.HashHeight, 0, len(cfg.WhiteBlockList))
	for _, hexStr := range cfg.WhiteBlockList {
		strs := strings.Split(hexStr, "/")
		var hash types.Hash
		hash, err = types.HexToHash(strs[0])
		if err != nil {
			return nil, err
		}
		var height uint64
		height, err = strconv.ParseUint(strs[1], 10, 64)
		if err != nil {
			return nil, err
		}
		confirmedHashList = append(confirmedHashList, &ledger.HashHeight{
			Height: height,
			Hash:   hash,
		})
	}

	var peerKey ed25519.PrivateKey
	peerKey, err = cfg.Init()
	if err != nil {
		return nil, err
	}

	peers := newPeerSet()

	feed := newBlockFeeder(blackHashList)

	forward := createForardStrategy(cfg.ForwardStrategy, peers)
	broadcaster := newBroadcaster(peers, verifier, feed, newMemBlockStore(1000), forward, chain)

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

	reader := newCacheReader(chain, verifier, downloader, irreader, blackHashList)

	syncer := newSyncer(chain, peers, reader, downloader, irreader, 10*time.Minute, blackHashList)

	fetcher := newFetcher(peers, receiver, blackHashList)

	fetcher.st = syncer.state.state()
	broadcaster.st = syncer.state.state()
	syncer.SubscribeSyncStatus(fetcher.subSyncState)
	syncer.SubscribeSyncStatus(broadcaster.subSyncState)

	n := &net{
		config: cfg,
		chain:  chain,
		node: &vnode.Node{
			ID:  id,
			Net: cfg.NetID,
		},
		peerKey:         peerKey,
		BlockSubscriber: feed,
		peers:           peers,
		syncer:          syncer,
		reader:          reader,
		fetcher:         fetcher,
		broadcaster:     broadcaster,
		downloader:      downloader,
		syncServer:      newSyncServer(cfg.ListenInterface+":"+strconv.Itoa(cfg.FilePort), chain, syncConnFac),
		handlers:        newHandlers("vite"),
		hb:              newHeartBeater(peers, chain),
		blackList: netool.NewBlackList(func(t int64, count int) bool {
			now := time.Now().Unix()

			if now < t {
				return true
			}

			return false
		}),
		log:                     netLog,
		confirmedHashHeightList: confirmedHashList,
	}

	fileAddress, err := retrieveAddressBytesFromConfig(cfg.FilePublicAddress, cfg.FilePort)
	if err != nil {
		return nil, err
	}
	publicAddress, err := retrieveAddressBytesFromConfig(cfg.PublicAddress, cfg.Port)
	if err != nil {
		return nil, err
	}

	n.hkr = &handshaker{
		version:       version,
		netId:         cfg.NetID,
		name:          cfg.Name,
		id:            id,
		genesis:       chain.GetGenesisSnapshotBlock().Hash,
		fileAddress:   fileAddress,
		publicAddress: publicAddress,
		peerKey:       peerKey,
		key:           cfg.MineKey,
		codecFactory: &transportFactory{
			minCompressLength: 100,
			readTimeout:       readMsgTimeout,
			writeTimeout:      writeMsgTimeout,
		},
		chain:        chain,
		blackList:    n.blackList,
		onHandshaker: n.authorize,
	}

	n.db, err = database.New(path.Join(cfg.DataDir, DBDirName), 1, n.node.ID)
	if err != nil {
		return nil, err
	}

	if cfg.Discover {
		n.discover = discovery.New(peerKey, n.node, cfg.BootNodes, cfg.BootSeeds, cfg.ListenInterface+":"+strconv.Itoa(cfg.Port), n.db)
	}

	var addr types.Address
	if len(n.config.MineKey) != 0 {
		addr = types.PubkeyToAddress(n.config.MineKey.PubByte())
	}

	n.finder, err = newFinder(addr, n.peers, cfg.MinPeers, cfg.StaticNodes, n.db, n, consensus)
	if err != nil {
		return nil, err
	}

	if n.finder._selfIsSBP {
		n.fetcher.sbp = true
		n.syncer.sbp = true
	}

	if n.discover != nil {
		n.discover.SetFinder(n.finder)
		if len(n.config.MineKey) != 0 {
			setNodeExt(n.config.MineKey, n.node)
		}
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

func (n *net) beatLoop() {
	defer n.wg.Done()

	beatTicker := time.NewTicker(10 * time.Second)
	defer beatTicker.Stop()
	storeTicker := time.NewTicker(time.Minute)
	defer storeTicker.Stop()

	for {
		select {
		case <-beatTicker.C:
			if n.running == 0 {
				return
			}

			for _, pe := range n.peers.peers() {
				_ = pe.WriteMsg(Msg{
					Code:    CodeHeartBeat,
					Payload: n.hb.state(),
				})
			}

		case <-storeTicker.C:
			if n.running == 0 {
				return
			}

			for _, pe := range n.peers.peers() {
				ep, err := vnode.ParseEndPoint(pe.publicAddress)
				if err != nil {
					continue
				}
				_ = n.db.StoreNode(&vnode.Node{
					ID:       pe.Id,
					EndPoint: ep,
					Net:      n.node.Net,
				})

				var weight int64
				if pe.Superior {
					weight = 1000
				} else if pe.reliable == 1 {
					weight = 100
				}

				n.db.StoreMark(pe.Id, weight)
			}
		}
	}
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

	var heartBeat = &vitepb.State{
		Peers:     nil,
		Patch:     true,
		Head:      current.Hash.Bytes(),
		Height:    current.Height,
		Timestamp: time.Now().Unix(),
	}

	idMap := h.ps.idMap()

	if time.Now().Sub(h.last) > time.Minute {
		heartBeat.Patch = false
		h.lastPeers = make(map[peerId]struct{})
	}

	var id peerId
	var ok bool
	for id = range h.lastPeers {
		if _, ok = idMap[id]; ok {
			delete(idMap, id)
			continue
		}
		heartBeat.Peers = append(heartBeat.Peers, &vitepb.State_Peer{
			ID:     id.Bytes(),
			Status: vitepb.State_Disconnected,
		})
	}

	for id = range idMap {
		heartBeat.Peers = append(heartBeat.Peers, &vitepb.State_Peer{
			ID:     id.Bytes(),
			Status: vitepb.State_Connected,
		})
	}

	data, err := proto.Marshal(heartBeat)
	if err != nil {
		return nil
	}

	h.lastPeers = idMap

	return data
}

func (n *net) Start() (err error) {
	if atomic.CompareAndSwapInt32(&n.running, 0, 1) {
		n.listener, err = _net.Listen("tcp", n.config.ListenInterface+":"+strconv.Itoa(n.config.Port))
		if err != nil {
			return
		}

		if n.discover != nil {
			err = n.discover.Start()
			if err != nil {
				return
			}
		}

		n.wg.Add(1)
		go n.listenLoop()

		if err = n.syncServer.start(); err != nil {
			return
		}

		n.finder.start()

		n.downloader.start()

		n.reader.start()

		n.query.start()

		n.fetcher.start()

		go n.syncer.checkLoop(&n.running)

		n.wg.Add(1)
		go n.beatLoop()

		return
	}

	return errNetIsRunning
}

func (n *net) Stop() error {
	if atomic.CompareAndSwapInt32(&n.running, 1, 0) {
		if n.discover != nil {
			_ = n.discover.Stop()
		}

		_ = n.listener.Close()

		n.reader.stop()

		n.syncer.stop()

		n.downloader.stop()

		_ = n.syncServer.stop()

		n.finder.stop()

		n.query.stop()

		n.fetcher.stop()

		n.finder.clean()

		n.wg.Wait()
		return nil
	}

	return errNetIsNotRunning
}

func (n *net) Nodes() []*vnode.Node {
	return n.discover.Nodes()
}

func (n *net) PeerKey() ed25519.PrivateKey {
	return n.peerKey
}

func (n *net) PeerCount() int {
	return n.peers.count()
}

func (n *net) Info() NodeInfo {
	ps := n.peers.info()
	info := NodeInfo{
		ID:        n.node.ID,
		Name:      n.config.Name,
		NetID:     n.config.NetID,
		Version:   version,
		Address:   "",
		PeerCount: len(ps),
		Peers:     ps,
		Height:    n.chain.GetLatestSnapshotBlock().Height,
		//Nodes:     n.discover.NodesCount(),
		Latency:               n.broadcaster.Statistic(),
		BroadCheckFailedRatio: n.broadcaster.rings.failedRatio(),
		Server:                FileServerStatus{},
	}

	if n.syncServer != nil {
		info.Server = n.syncServer.status()
	}

	return info
}

type NodeInfo struct {
	ID                    vnode.NodeID     `json:"id"`
	Name                  string           `json:"name"`
	NetID                 int              `json:"netId"`
	Version               int              `json:"version"`
	Address               string           `json:"address"`
	PeerCount             int              `json:"peerCount"`
	Peers                 []PeerInfo       `json:"peers"`
	Height                uint64           `json:"height"`
	Nodes                 int              `json:"nodes"`
	Latency               []int64          `json:"latency"` // [0,1,12,24]
	BroadCheckFailedRatio float32          `json:"broadCheckFailedRatio"`
	Server                FileServerStatus `json:"server"`
}
