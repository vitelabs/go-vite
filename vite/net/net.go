package net

import (
	"github.com/pkg/errors"
	"github.com/seiflotfy/cuckoofilter"
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/p2p"
	"github.com/vitelabs/go-vite/vite/net/message"
	"sync"
	"sync/atomic"
	"time"
)

type Chain interface {
	GetAccountBlockMap(queryParams map[types.Address]*chain.BlockMapQueryParam) map[types.Address][]*ledger.AccountBlock
	GetAbHashList(originBlockHash *types.Hash, count, step int, forward bool) ([]*types.Hash, error)
	GetSubLedgerByHeight(startHeight uint64, count int, forward bool) ([]string, [][2]uint64)
	GetSubLedgerByHash(startBlockHash *types.Hash, count int, forward bool) ([]string, [][2]uint64, error)
	GetConfirmSubLedger(fromHeight uint64, toHeight uint64) ([]*ledger.SnapshotBlock, map[types.Address][]*ledger.AccountBlock, error)
	GetSnapshotBlocksByHash(originBlockHash *types.Hash, count uint64, forward, containSnapshotContent bool) ([]*ledger.SnapshotBlock, error)
	GetSnapshotBlocksByHeight(height uint64, count uint64, forward, containSnapshotContent bool) ([]*ledger.SnapshotBlock, error)
	GetLatestSnapshotBlock() *ledger.SnapshotBlock
	GetGenesisSnapshotBlock() *ledger.SnapshotBlock
}

type Config struct {
	NetID      uint64
	Port       uint16
	Chain Chain
}

type Net struct {
	*Config
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
	current := n.Chain.GetLatestSnapshotBlock()
	genesis := n.Chain.GetGenesisSnapshotBlock()

	err := p.Handshake(&message.HandShake{
		CmdSet:       p.CmdSet,
		NetID:        n.NetID,
		Height:       current.Height,
		Port:         n.Port,
		Current: current.Hash,
		Genesis: genesis.Hash,
	})

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
			current := n.Chain.GetLatestSnapshotBlock()
			p.Send(StatusCode, 0, &message.BlockID{
				Hash:   current.Hash,
				Height: current.Height,
			})
		default:
			if err := n.handleMsg(p); err != nil {
				return err
			}
		}
	}

	close(p.term)
	return nil
}

func (n *Net) handleMsg(p *Peer) (err error) {
	msg, err := p.mrw.ReadMsg()
	if err != nil {
		return
	}
	defer msg.Discard()

	code := cmd(msg.Cmd)
	if code == HandshakeCode {
		return errHandshakeTwice
	}

	switch code {
	case StatusCode:
		status := new(message.BlockID)
		err = status.Deserialize(msg.Payload)
		if err != nil {
			return
		}
		p.SetHead(status.Hash, status.Height)
	case GetSubLedgerCode:
		req := new(message.GetSnapshotBlocks)
		err = req.Deserialize(msg.Payload)
		if err != nil {
			return
		}
// todo
//		var files []*message.File
//		var batch [][2]uint64
//		if req.From.Height != 0 {
//			files, batch, err = n.Chain.GetSubLedgerByHeight(&req.From.Hash, req.Count, req.Forward)
//		} else {
//			files, batch, err = n.Chain.GetSubLedgerByHash(&req.From.Hash, req.Count, req.Forward)
//		}

		if err != nil {
			p.Send(ExceptionCode, msg.Id, message.Missing)
		} else {
			p.Send(FileListCode, msg.Id, &message.FileList{
				Files: nil,
				Start: 0,
				End:   0,
				Nonce: 0,
			})
		}
	case GetSnapshotBlocksCode:
		req := new(message.GetSnapshotBlocks)
		err = req.Deserialize(msg.Payload)
		if err != nil {
			return
		}

		var blocks []*ledger.SnapshotBlock
		if req.From.Height != 0 {
			n.Chain.GetSnapshotBlocksByHeight(req.From.Height, req.Count, req.Forward, false)
		} else {
			n.Chain.GetSnapshotBlocksByHash(&req.From.Hash, req.Count, req.Forward, false)
		}

		p.Send(SnapshotBlocksCode, msg.Id, &message.SnapshotBlocks{
			Blocks: blocks,
		})
	case GetFullSnapshotBlocksCode:
		req := new(message.GetSnapshotBlocks)
		err = req.Deserialize(msg.Payload)
		if err != nil {
			return
		}

		var blocks []*ledger.SnapshotBlock
		if req.From.Height != 0 {
			n.Chain.GetSnapshotBlocksByHeight(req.From.Height, req.Count, req.Forward, true)
		} else {
			n.Chain.GetSnapshotBlocksByHash(&req.From.Hash, req.Count, req.Forward, true)
		}

		p.Send(FullSnapshotBlocksCode, msg.Id, &message.SnapshotBlocks{
			Blocks: blocks,
		})
	case GetAccountBlocksCode:
		as := new(message.GetAccountBlocks)
		err = as.Deserialize(msg.Payload)
		if err != nil {
			return
		}

		query := make(map[types.Address]*chain.BlockMapQueryParam, 1)
		query[as.Address] = &chain.BlockMapQueryParam{
			OriginBlockHash: &as.From.Hash,
			Count:           as.Count,
			Forward:         as.Forward,
		}

		mblocks := n.Chain.GetAccountBlockMap(query)
		p.Send(AccountBlocksCode, msg.Id, &message.AccountBlocks{
			Address: as.Address,
			Blocks:  mblocks[as.Address],
		})
	case FileListCode:
		fs := new(message.FileList)
		err = fs.Deserialize(msg.Payload)
		if err != nil {
			return
		}

		// todo request file
	case SubLedgerCode:
		subledger := new(message.SubLedger)
		err = subledger.Deserialize(msg.Payload)
		if err != nil {
			return
		}

		for _, block := range subledger.SBlocks {
			p.SeeBlock(block.Hash)
		}
		for _, block := range subledger.ABlocks {
			p.SeeBlock(block.Hash)
		}
	case SnapshotBlocksCode:
		blocks := new(message.SnapshotBlocks)

		err = blocks.Deserialize(msg.Payload)
		if err != nil {
			return
		}

		for _, block := range blocks.Blocks {
			p.SeeBlock(block.Hash)
		}
	case AccountBlocksCode:
		blocks := new(message.AccountBlocks)

		err = blocks.Deserialize(msg.Payload)
		if err != nil {
			return
		}

		for _, block := range blocks.Blocks {
			p.SeeBlock(block.Hash)
		}
	case NewSnapshotBlockCode:
		block := new(ledger.SnapshotBlock)
		err = block.Deserialize(msg.Payload)
		if err != nil {
			return
		}

		p.SeeBlock(block.Hash)
		n.BroadcastSnapshotBlock(block)
	case ForkCode:
		fork := new(message.Fork)
		err := fork.Deserialize(msg.Payload)
		if err != nil {
			return
		}

		var commonID *message.BlockID
		for _, id := range fork.List {
			blocks, err := n.Chain.GetSnapshotBlocksByHeight(id.Height, 1, true, false)
			if err == nil && blocks[0].Hash == id.Hash {
				commonID = id
				return
			}
		}
		// after get commonID, resend the request message
		p.GetSnapshotBlocks(&message.GetSnapshotBlocks{
			From:    commonID,
			Count:   p.height - commonID.Height,
			Forward: true,
		})
	case ExceptionCode:
		exception, err := message.DeserializeException(msg.Payload)
		if err != nil {
			return
		}
		// todo handle exception
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

	current := n.Chain.GetLatestSnapshotBlock()

	peerHeight := bestPeer.height
	if peerHeight <= current.Height {
		n.log.Info("syncdone: bestPeer is lower than me")
		return
	}

	n.FromHeight = current.Height
	n.TargetHeight = peerHeight

	param := &message.GetSnapshotBlocks{
		From: &message.BlockID{
			Hash:   current.Hash,
			Height: current.Height,
		},
		Count:   peerHeight - n.FromHeight,
		Forward: true,
	}

	bestPeer.GetSubLedger(param)
}

func (n *Net) BroadcastSnapshotBlock(block *ledger.SnapshotBlock) {
	peers := n.peers.UnknownBlock(block.Hash)
	for _, peer := range peers {
		go peer.SendNewSnapshotBlock(block)
	}
}

func (n *Net) BroadcastAccountBlock(addr types.Address, block *ledger.AccountBlock) {
	peers := n.peers.UnknownBlock(block.Hash)
	for _, peer := range peers {
		go func(peer *Peer) {
			peer.SendAccountBlocks(addr, []*ledger.AccountBlock{block}, 0)
		}(peer)
	}
}

func (n *Net) FetchSnapshotBlocks(start types.Hash, count uint64) {
	// if the sync data has not downloaded, then ignore fetch request
	if atomic.LoadInt32(&n.downloaded) == 0 {
		return
	}

	req := newRequest(GetSnapshotBlocksCode, &message.GetSnapshotBlocks{
		From: &message.BlockID{Hash: start,},
		Count: count,
		Forward: true,
	}, snapshotBlocksTimeout)

	n.pool.add(req)
}

func (n *Net) FetchAccountBlocks(start types.Hash, count uint64, address types.Address) {
	// if the sync data has not downloaded, then ignore fetch request
	if atomic.LoadInt32(&n.downloaded) == 0 {
		return
	}

	req := newRequest(GetAccountBlocksCode, &message.GetAccountBlocks{
		Address: address,
		From:    &message.BlockID{Hash:   start,},
		Count:   count,
		Forward: true,
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
		SyncState: SyncState(atomic.LoadInt32(&n.syncState)),
	}
}

type NetStatus struct {
	Peers     []*PeerInfo
	SyncState SyncState
	Running   bool
}
