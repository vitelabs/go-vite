package net

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	net2 "net"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/vitelabs/go-vite/p2p/netool"

	"github.com/golang/protobuf/proto"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/p2p"
	"github.com/vitelabs/go-vite/p2p/vnode"
	"github.com/vitelabs/go-vite/vite/net/protos"
)

var netLog = log15.New("module", "vite/net")
var errNetIsRunning = errors.New("network is already running")
var errNetIsNotRunning = errors.New("network is not running")
var errInvalidSignature = errors.New("invalid signature")
var errDiffGenesisBlock = errors.New("different genesis block")
var errErrorHeadToHash = errors.New("error head to hash")

type Config struct {
	Single            bool
	FileListenAddress string
	FilePublicAddress string
	FilePort          int
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
	running  int32
	log      log15.Logger
	fs       *fileServer
	handlers *msgHandlers
	query    *queryHandler
	hb       *heartBeater
}

func (n *net) parseFilePublicAddress() (fileAddress []byte) {
	if n.FilePublicAddress != "" {
		var e vnode.EndPoint
		var err error
		e, err = vnode.ParseEndPoint(n.FilePublicAddress)
		if err != nil {
			n.log.Error(fmt.Sprintf("Failed to parse FilePublicAddress: %v", err))
			return nil
		}

		fileAddress, err = e.Serialize()
		if err != nil {
			n.log.Error(fmt.Sprintf("Failed to serialize FilePublicAddress: %v", err))
			return nil
		}
	} else if n.FilePort != 0 {
		fileAddress = make([]byte, 2)
		binary.BigEndian.PutUint16(fileAddress, uint16(n.FilePort))
	}

	return fileAddress
}

func extractFileAddress(sender net2.Addr, fileAddressBytes []byte) (fileAddress string) {
	if len(fileAddressBytes) != 0 {
		var tcp *net2.TCPAddr

		if len(fileAddressBytes) == 2 {
			filePort := binary.BigEndian.Uint16(fileAddressBytes)
			var ok bool
			if tcp, ok = sender.(*net2.TCPAddr); ok {
				return tcp.IP.String() + ":" + strconv.Itoa(int(filePort))
			}
		} else {
			var ep = new(vnode.EndPoint)

			if err := ep.Deserialize(fileAddressBytes); err == nil {
				if ep.Typ.Is(vnode.HostIP) {
					// verify ip
					var ok bool
					if tcp, ok = sender.(*net2.TCPAddr); ok {
						err = netool.CheckRelayIP(tcp.IP, ep.Host)
						if err != nil {
							// invalid ip
							ep.Host = tcp.IP
						}
					}

					fileAddress = ep.String()
				} else {
					tcp, err = net2.ResolveTCPAddr("tcp", ep.String())
					if err == nil {
						fileAddress = ep.String()
					}
				}
			}
		}
	}

	return
}

func (n *net) ProtoData() []byte {
	genesis := n.Chain.GetGenesisSnapshotBlock()
	current := n.Chain.GetLatestSnapshotBlock()

	var key, signature []byte
	if len(n.MinePrivateKey) != 0 {
		key = n.MinePrivateKey.PubByte()
		signature = ed25519.Sign(n.MinePrivateKey, n.nodeID.Bytes())
	}

	pb := &protos.ViteHandshake{
		Genesis:     genesis.Hash.Bytes(),
		Head:        current.Hash.Bytes(),
		Height:      current.Height,
		Key:         key,
		Signature:   signature,
		FileAddress: n.parseFilePublicAddress(),
	}

	buf, err := proto.Marshal(pb)
	if err != nil {
		return nil
	}

	return buf
}

func (n *net) ReceiveHandshake(msg p2p.HandshakeMsg, protoData []byte, sender net2.Addr) (state interface{}, level p2p.Level, err error) {
	pb := &protos.ViteHandshake{}
	err = proto.Unmarshal(protoData, pb)
	if err != nil {
		return
	}

	if pb.Key != nil {
		right := ed25519.Verify(pb.Key, msg.ID.Bytes(), pb.Signature)
		if !right {
			err = errInvalidSignature
			return
		}

		addr := types.PubkeyToAddress(pb.Key)
		if n.Producer != nil && n.Producer.IsProducer(addr) {
			level = p2p.Superior
		}
	}

	// genesis
	genesis := n.Chain.GetGenesisSnapshotBlock()
	if !bytes.Equal(pb.Genesis, genesis.Hash.Bytes()) {
		err = errDiffGenesisBlock
		return
	}

	// head
	var hash types.Hash
	hash, err = types.BytesToHash(pb.Head)
	if err != nil {
		err = errErrorHeadToHash
		return
	}

	state = PeerState{
		Head:        hash,
		Height:      pb.Height,
		FileAddress: extractFileAddress(sender, pb.FileAddress),
	}

	return
}

func (n *net) Name() string {
	return "vite"
}

func (n *net) ID() p2p.ProtocolID {
	return ID
}

func (n *net) Handle(msg p2p.Msg) error {
	if handler := n.handlers.pick(code(msg.Code)); handler != nil {
		p := n.peers.get(msg.Sender.ID())
		if p != nil {
			return handler.handle(msg, p)
		} else {
			return errPeerNotExist
		}
	}

	return p2p.PeerUnknownMessage
}

func (n *net) SetState(state []byte, peer p2p.Peer) {
	var heartbeat = &protos.State{}

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

	forward := chooseForardStrategy(cfg.ForwardStrategy, peers)
	broadcaster := newBroadcaster(peers, cfg.Verifier, feed, newMemBlockStore(1000), forward, nil)

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
	syncer := newSyncer(cfg.Chain, peers, downloader, 10*time.Minute)
	reader := newCacheReader(cfg.Chain, receiver, downloader)

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
		fs:              newFileServer(cfg.FileListenAddress, cfg.Chain, syncConnFac),
		handlers:        newHandlers("vite"),
		log:             netLog,
		hb:              newHeartBeater(peers, cfg.Chain, netLog.New("module", "heartbeat")),
	}

	var err error
	n.query, err = newQueryHandler(cfg.Chain)
	if err != nil {
		panic(errors.New("cannot construct query handler"))
	}

	// GetSubLedgerCode, GetSnapshotBlocksCode, GetAccountBlocksCode, GetChunkCode
	if err = n.handlers.register(n.query); err != nil {
		panic(errors.New("cannot register handler: query"))
	}

	// NewSnapshotBlockCode, NewAccountBlockCode
	if err = n.handlers.register(broadcaster); err != nil {
		panic(errors.New("cannot register handler: broadcaster"))
	}

	// SnapshotBlocksCode, AccountBlocksCode
	if err = n.handlers.register(fetcher); err != nil {
		panic(errors.New("cannot register handler: fetcher"))
	}

	return n
}

type heartBeater struct {
	chain     chainReader
	last      time.Time
	lastPeers map[vnode.NodeID]struct{}
	ps        *peerSet
	log       log15.Logger
}

func newHeartBeater(ps *peerSet, chain chainReader, log log15.Logger) *heartBeater {
	return &heartBeater{
		ps:    ps,
		chain: chain,
		log:   log,
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
			continue
		}
		heartBeat.Peers = append(heartBeat.Peers, &protos.State_Peer{
			ID:     id[:],
			Status: protos.State_Disconnected,
		})
	}

	for id = range idMap {
		if _, ok = h.lastPeers[id]; ok {
			continue
		}
		heartBeat.Peers = append(heartBeat.Peers, &protos.State_Peer{
			ID:     id[:],
			Status: protos.State_Disconnected,
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

func (n *net) Start(svr p2p.P2P) (err error) {
	if atomic.CompareAndSwapInt32(&n.running, 0, 1) {
		n.nodeID = svr.Config().Node().ID

		if n.Producer != nil && n.MinePrivateKey != nil {
			addr := types.PubkeyToAddress(n.MinePrivateKey.PubByte())
			if n.Producer.IsProducer(addr) {
				// todo set finder
			}
		}

		if err = n.fs.start(); err != nil {
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

		_ = n.fs.stop()

		n.query.stop()

		n.fetcher.stop()

		return nil
	}

	return errNetIsNotRunning
}

func (n *net) Info() NodeInfo {
	return NodeInfo{
		PeerCount: n.peers.count(),
		Latency:   n.broadcaster.Statistic(),
	}
}

type NodeInfo struct {
	PeerCount int     `json:"peerCount"`
	Latency   []int64 `json:"latency"` // [0,1,12,24]
}
