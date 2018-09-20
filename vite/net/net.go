package net

import (
	"github.com/pkg/errors"
	"github.com/seiflotfy/cuckoofilter"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/p2p"
	"io/ioutil"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

type Config struct {
	NetID      uint64
	Port       uint16
	BlockChain BlockChain
}

type Net struct {
	*Config
	start         time.Time
	peers         *peerSet
	snapshotFeed  *snapshotBlockFeed
	accountFeed   *accountBlockFeed
	term          chan struct{}
	pool          *requestPool
	FromHeight    uint64
	TargetHeight  uint64
	syncState     int32 // atomic
	downloaded    int32 // atomic, indicate whether the first sync data has download from bestPeer
	stateFeed     *SyncStateFeed
	SnapshotChain BlockChain
	blockRecord   *cuckoofilter.CuckooFilter // record blocks has retrieved from network
	log           log15.Logger
	Protocols     []*p2p.Protocol
	wg            sync.WaitGroup
	fileServer    *FileServer
}

func New(cfg *Config) (*Net, error) {
	peerSet := NewPeerSet()

	fileServer, err := newFileServer(cfg.Port)
	if err != nil {
		return nil, err
	}

	n := &Net{
		Config:        cfg,
		peers:         peerSet,
		snapshotFeed:  new(snapshotBlockFeed),
		accountFeed:   new(accountBlockFeed),
		stateFeed:     new(SyncStateFeed),
		term:          make(chan struct{}),
		pool:          newRequestPool(peerSet),
		SnapshotChain: cfg.BlockChain,
		blockRecord:   cuckoofilter.NewCuckooFilter(10000),
		log:           log15.New("module", "vite/net"),
		fileServer:    fileServer,
	}

	n.Protocols = make([]*p2p.Protocol, len(cmdSets))
	for i, cmdset := range cmdSets {
		n.Protocols[i] = &p2p.Protocol{
			Name: CmdSetName,
			ID:   cmdset,
			Handle: func(p *p2p.Peer, rw p2p.MsgReadWriter) error {
				// will be called by p2p.Peer.runProtocols use goroutine
				peer := newPeer(p, rw, cmdset)
				return n.HandlePeer(peer)
			},
		}
	}

	return n, nil
}

func (n *Net) Start() {
	n.start = time.Now()
}

func (n *Net) Stop() {
	select {
	case <-n.term:
	default:
		close(n.term)
	}

	n.fileServer.stop()
}

func (n *Net) Syncing() bool {
	return atomic.LoadInt32(&n.syncState) == int32(Syncing)
}

//func (n *Net) startSync() {
//	n.SetSyncState(Syncing)
//	go n.checkChainHeight()
//}
//
//func (n *Net) checkChainHeight() {
//	ticker := time.NewTicker(1 * time.Minute)
//	defer ticker.Stop()
//
//	begin := time.Now()
//
//	for {
//		select {
//		case now := <-ticker.C:
//			current, err := n.SnapshotChain.GetLatestSnapshotBlock()
//			if err != nil {
//				return
//			}
//			if current.Height == n.TargetHeight {
//				n.SetSyncState(Syncdone)
//			}
//			if now.Sub(begin) > waitForChainGrow {
//				n.SetSyncState(Syncerr)
//			}
//		case <-n.term:
//			return
//		}
//	}
//}

func (n *Net) SetSyncState(st SyncState) {
	atomic.StoreInt32(&n.syncState, int32(st))
	n.stateFeed.Notify(st)
}

func (n *Net) SyncState() SyncState {
	return SyncState(atomic.LoadInt32(&n.syncState))
}

// the main method, handle peer
func (n *Net) HandlePeer(p *Peer) error {
	head, err := n.SnapshotChain.GetLatestSnapshotBlock()
	if err != nil {
		log.Fatal("cannot get current block", err)
	}

	genesis, err := n.SnapshotChain.GetGenesesBlock()
	if err != nil {
		log.Fatal("cannot get genesis block", err)
	}

	err = p.Handshake(n.NetID, head.Height, head.Hash, genesis.Hash)
	if err != nil {
		return err
	}

	return n.startPeer(p)
}

func (n *Net) startPeer(p *Peer) error {
	n.peers.Add(p)
	defer n.peers.Del(p)

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	go n.sync()

	for {
		select {
		case <-n.term:
			return p2p.DiscQuitting
		case err := <-p.reqErr:
			return err
		case <-ticker.C:
			current, err := n.SnapshotChain.GetLatestSnapshotBlock()
			if err == nil {
				p.Send(StatusCode, &BlockID{
					Hash:   current.Hash,
					Height: current.Height,
				})
			}

		default:
			if err := n.handleMsg(p); err != nil {
				return err
			}
		}
	}
}

func (n *Net) handleMsg(p *Peer) (err error) {
	msg, err := p.ts.ReadMsg()
	if err != nil {
		return
	}
	defer msg.Discard()

	code := cmd(msg.Cmd)
	if code == HandshakeCode {
		return errHandshakeTwice
	}

	payload, err := ioutil.ReadAll(msg.Payload)
	if err != nil {
		return
	}

	switch code {
	case StatusCode:
		status := new(BlockID)
		err = status.Deserialize(payload)
		if err != nil {
			return
		}
		p.SetHead(status.Hash, status.Height)
	case GetSubLedgerCode:
		seg := new(Segment)
		err = seg.Deserialize(payload)
		if err != nil {
			return
		}

		snapshotblocks, accountblocks, err := n.SnapshotChain.GetSubLedger(seg.From.Height, seg.To.Height)
		if err != nil {
			p.Send(ExceptionCode, Missing)
		} else {
			p.Send(SubLedgerCode, &SubLedger{
				SBlocks: snapshotblocks,
				ABlocks: accountblocks,
			})
		}
	case GetSnapshotBlocksCode:

	case GetAccountBlocksCode:
		as := new(AccountSegment)
		err = as.Deserialize(payload)
		if err != nil {
			return
		}

		accountMap, err := n.SnapshotChain.GetAccountBlockMap(*as)
		if err != nil {
			p.Send(ExceptionCode, Missing)
		} else {
			p.Send(AccountBlocksCode, accountMap)
		}
	case FileListCode:
		fs := new(FileList)
		err = fs.Deserialize(payload)
		if err != nil {
			return
		}
		// todo get file

	case SubLedgerCode:
		subledger := new(SubLedger)
		err = subledger.Deserialize(payload)
		if err != nil {
			return
		}

		p.receive(SubLedgerCode, subledger)

		for _, block := range subledger.SBlocks {
			p.SeeBlock(block.Hash)
		}
		for _, block := range subledger.ABlocks {
			p.SeeBlock(block.Hash)
		}
		//n.receiveSnapshotBlock(subledger.snapshotblocks)
		//n.receiveAccountBlocks(subledger.accountblocks)
	case SnapshotBlocksCode:
		blocks := new(SnapshotBlocks)

		err = blocks.Deserialize(payload)
		if err != nil {
			return
		}

		p.receive(SnapshotBlocksCode, blocks)

		for _, block := range blocks.Blocks {
			p.SeeBlock(block.Hash)
		}
		n.receiveSnapshotBlock(blocks)
	case AccountBlocksCode:
		blocks := new(AccountBlocks)

		err = blocks.Deserialize(payload)
		if err != nil {
			return
		}

		p.receive(AccountBlocksCode, blocks)

		for _, block := range blocks.Blocks {
			p.SeeBlock(block.Hash)
		}
		n.receiveAccountBlocks(blocks)
		n.BroadcastAccountBlocks(blocks)
	case NewSnapshotBlockCode:
		block := new(ledger.SnapshotBlock)
		err = block.Deserialize(payload)
		if err != nil {
			return
		}

		p.SeeBlock(block.Hash)
		n.BroadcastSnapshotBlock(block)
	case ExceptionCode:
		exception, err := deserializeException(payload)
		if err != nil {
			return
		}
		p.receive(ExceptionCode, exception)
	default:
		return errors.New("unknown message")
	}

	return nil
}

func (n *Net) sync() {
	// set syncing
	if !atomic.CompareAndSwapInt32(&n.syncState, 0, 1) {
		return
	}

	// set syncdone
	defer atomic.CompareAndSwapInt32(&n.syncState, 1, 2)

	for {
		select {
		case <-n.term:
			return
		default:
			if n.peers.Count() >= enoughPeers {
				goto SYNC
			} else {
				time.Sleep(3 * time.Second)
			}
		}
	}

SYNC:
	bestPeer := n.peers.BestPeer()
	if bestPeer == nil {
		n.log.Info("syncdone: bestPeer is nil")
		return
	}

	current, err := n.SnapshotChain.GetLatestSnapshotBlock()
	if err != nil {
		n.log.Error("can`t getLatestSnapshotBlock", "error", err)
		return
	}

	peerHead := bestPeer.Head()

	peerHeight := peerHead.Height
	if peerHeight <= current.Height {
		n.log.Info("syncdone: bestPeer is lower than me")
		return
	}

	n.FromHeight = current.Height
	n.TargetHeight = peerHeight

	seg := &Segment{
		Origin: &BlockID{current.Hash, current.Height},
		To:     &BlockID{peerHead.Hash, peerHeight},
		Count:  peerHeight - current.Height,
	}

	req := newRequest(GetSubLedgerCode, seg, subledgerTimeout)
	n.pool.add(req)
}

func (n *Net) BroadcastSnapshotBlock(block *ledger.SnapshotBlock) {
	peers := n.peers.UnknownBlock(block.Hash)

	for _, peer := range peers {
		go peer.SendNewSnapshotBlock(block)
	}
}

func (n *Net) BroadcastSnapshotBlocks(blocks []*ledger.SnapshotBlock) {
	for _, block := range blocks {
		go func(b *ledger.SnapshotBlock) {
			peers := n.peers.UnknownBlock(block.Hash)

			for _, peer := range peers {
				go peer.SendNewSnapshotBlock(block)
			}
		}(block)
	}
}

func (n *Net) BroadcastAccountBlocks(blocks []*ledger.AccountBlock) {
	for _, block := range blocks {
		peers := n.peers.UnknownBlock(block.Hash)
		for _, peer := range peers {
			go func(peers []*Peer, b *ledger.AccountBlock) {
				peer.SendAccountBlocks([]*ledger.AccountBlock{block})
			}(peers, block)
		}
	}
}

func (n *Net) FetchSnapshotBlocks(start types.Hash, count uint64) {
	// if the sync data has not downloaded, then ignore fetch request
	if atomic.LoadInt32(&n.downloaded) == 0 {
		return
	}

	req := newRequest(GetSnapshotBlocksCode, &Segment{
		From: &BlockID{
			Hash: start,
		},
		Count: count,
	}, snapshotBlocksTimeout)

	n.pool.add(req)
}

func (n *Net) FetchAccountBlocks(start types.Hash, count uint64, address types.Address) {
	// if the sync data has not downloaded, then ignore fetch request
	if atomic.LoadInt32(&n.downloaded) == 0 {
		return
	}

	req := newRequest(GetAccountBlocksCode, &AccountSegment{
		Address: address,
		Segment: &Segment{
			From: &BlockID{
				Hash: start,
			},
			Count: count,
		},
	}, accountBlocksTimeout)

	n.pool.add(req)
}

func (n *Net) SubscribeAccountBlock(fn func(block *ledger.AccountBlock)) (subId int) {
	return n.accountFeed.Sub(fn)
}

func (n *Net) UnsubscribeAccountBlock(subId int) {
	n.accountFeed.Unsub(subId)
}

func (n *Net) receiveAccountBlocks(blocks []*ledger.AccountBlock) {
	for _, block := range blocks {
		// todo verify
		if !n.blockRecord.Lookup(block.Hash[:]) {
			n.blockRecord.InsertUnique(block.Hash[:])
			n.accountFeed.Notify(block)
		}
	}
}

func (n *Net) SubscribeSnapshotBlock(fn func(block *ledger.SnapshotBlock)) (subId int) {
	return n.snapshotFeed.Sub(fn)
}

func (n *Net) UnsubscribeSnapshotBlock(subId int) {
	n.snapshotFeed.Unsub(subId)
}

func (n *Net) receiveSnapshotBlock(blocks []*ledger.SnapshotBlock) {
	for _, block := range blocks {
		// todo verify
		if !n.blockRecord.Lookup(block.Hash[:]) {
			n.blockRecord.InsertUnique(block.Hash[:])
			n.snapshotFeed.Notify(block)
		}
	}
}

func (n *Net) SubscribeSyncStatus(fn func(SyncState)) (subId int) {
	return n.stateFeed.Sub(fn)
}

func (n *Net) UnsubscribeSyncStatus(subId int) {
	n.stateFeed.Unsub(subId)
}

// get current netInfo (peers, syncStatus, ...)
func (n *Net) Status() *NetStatus {
	running := true
	select {
	case <-n.term:
		running = false
	default:
	}

	return &NetStatus{
		Peers:     n.peers.Info(),
		Running:   running,
		Uptime:    time.Now().Sub(n.start),
		SyncState: SyncState(atomic.LoadInt32(&n.syncState)),
	}
}

type NetStatus struct {
	Peers     []*PeerInfo
	SyncState SyncState
	Uptime    time.Duration
	Running   bool
}
