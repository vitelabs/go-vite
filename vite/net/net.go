package net

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/seiflotfy/cuckoofilter"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/compress"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/p2p"
	"github.com/vitelabs/go-vite/vite/net/message"
	"sync"
	"sync/atomic"
	"time"
)

// all query include start block
type Chain interface {
	// query common Hash
	GetAbHashList(start, count, step uint64, forward bool) ([]*ledger.HashHeight, error)

	// the second return value mean chunk befor/after file
	GetSubLedgerByHeight(start, count uint64, forward bool) ([]string, [][2]uint64)
	GetSubLedgerByHash(origin *types.Hash, count uint64, forward bool) ([]string, [][2]uint64, error)

	// query chunk
	GetConfirmSubLedger(start, end uint64) ([]*ledger.SnapshotBlock, map[types.Address][]*ledger.AccountBlock, error)

	GetSnapshotBlocksByHash(origin *types.Hash, count uint64, forward, content bool) ([]*ledger.SnapshotBlock, error)
	GetSnapshotBlocksByHeight(height, count uint64, forward, content bool) ([]*ledger.SnapshotBlock, error)

	GetAccountBlocksByHash(addr types.Address, origin *types.Hash, count uint64, forward bool) ([]*ledger.AccountBlock, error)
	GetAccountBlocksByHeight(addr types.Address, start, count uint64, forward bool) ([]*ledger.AccountBlock, error)

	GetLatestSnapshotBlock() *ledger.SnapshotBlock
	GetGenesisSnapshotBlock() *ledger.SnapshotBlock

	Compressor() *compress.Compressor
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
	Protocols     []*p2p.Protocol	// mount to p2p.Server
	wg            sync.WaitGroup
	fileServer    *FileServer
	newBlocks []*ledger.SnapshotBlock	// before syncDone, cache newBlocks
}

func New(cfg *Config) (*Net, error) {
	peerSet := NewPeerSet()
	//pool := newRequestPool(peerSet)

	fileServer, err := newFileServer(cfg.Port, cfg.Chain, peerSet, nil)
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
		case err := <-p.errch:
			return err
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
		status := new(ledger.HashHeight)
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

		var files []*message.File
		var batch [][2]uint64
		if req.From.Height != 0 {
			//files, batch = n.Chain.GetSubLedgerByHeight(req.From.Height, req.Count, req.Forward)
		} else {
			//files, batch, err = n.Chain.GetSubLedgerByHash(&req.From.Hash, req.Count, req.Forward)
		}

		if err != nil {
			return p.Send(ExceptionCode, msg.Id, message.Missing)
		} else {
			return p.Send(FileListCode, msg.Id, &message.FileList{
				Files: files,
				Chunk: batch,
				Nonce: 0,
			})
		}
	case GetChunkCode:
		req := new(message.Chunk)
		err = req.Deserialize(msg.Payload)
		if err != nil {
			return
		}

		sblocks, mblocks, err := n.Chain.GetConfirmSubLedger(req.Start, req.End)
		if err == nil {
			return p.SendSubLedger(&message.SubLedger{
				SBlocks: sblocks,
				ABlocks: mblocks,
			}, msg.Id)
		}
	case GetSnapshotBlocksCode:
		req := new(message.GetSnapshotBlocks)
		err = req.Deserialize(msg.Payload)
		if err != nil {
			return
		}

		var blocks []*ledger.SnapshotBlock
		if req.From.Height != 0 {
			blocks, err = n.Chain.GetSnapshotBlocksByHeight(req.From.Height, req.Count, req.Forward, false)
		} else {
			blocks, err = n.Chain.GetSnapshotBlocksByHash(&req.From.Hash, req.Count, req.Forward, false)
		}

		var ids []*ledger.HashHeight
		if err != nil {
			n.log.Error("GetSnapshotBlocks<%v/%d, %d, %v, false> error: %v", req.From.Hash, req.From.Height, req.Count, req.Forward, err)

			count := req.From.Height / hashStep
			if count > maxStepHashCount {
				count = maxStepHashCount
			}
			// from high to low
			ids, err = n.Chain.GetAbHashList(req.From.Height, count, hashStep, false)
			if err != nil {
				n.log.Error(fmt.Sprintf("GetAbHashList<%d, %d, %d, %v> error: %v", req.From.Height, 100, 20, err))
				return err
			}

			return p.SendFork(&message.Fork{
				List: ids,
			}, msg.Id)
		} else {
			return p.SendSnapshotBlocks(blocks, msg.Id)
		}
	case GetFullSnapshotBlocksCode:
		req := new(message.GetSnapshotBlocks)
		err = req.Deserialize(msg.Payload)
		if err != nil {
			return
		}

		var blocks []*ledger.SnapshotBlock
		if req.From.Height != 0 {
			blocks, err = n.Chain.GetSnapshotBlocksByHeight(req.From.Height, req.Count, req.Forward, true)
		} else {
			blocks, err = n.Chain.GetSnapshotBlocksByHash(&req.From.Hash, req.Count, req.Forward, true)
		}

		if err != nil {
			return p.Send(FullSnapshotBlocksCode, msg.Id, &message.SnapshotBlocks{
				Blocks: blocks,
			})
		} else {
			return p.Send(ExceptionCode, msg.Id, message.Missing)
		}
	case GetAccountBlocksCode:
		as := new(message.GetAccountBlocks)
		err = as.Deserialize(msg.Payload)
		if err != nil {
			return
		}

		var blocks []*ledger.AccountBlock
		if as.From.Height != 0 {
			blocks, err = n.Chain.GetAccountBlocksByHeight(as.Address, as.From.Height, as.Count, as.Forward)
		} else {
			blocks, err = n.Chain.GetAccountBlocksByHash(as.Address, &as.From.Hash, as.Count, as.Forward)
		}

		if err != nil {
			return p.SendAccountBlocks(as.Address, blocks, msg.Id)
		} else {
			return p.Send(ExceptionCode, msg.Id, message.Missing)
		}
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
		for _, ablocks := range subledger.ABlocks {
			for _, ablock := range ablocks {
				p.SeeBlock(ablock.Hash)
			}
		}
		// todo receive blocks
	case SnapshotBlocksCode:
		blocks := new(message.SnapshotBlocks)

		err = blocks.Deserialize(msg.Payload)
		if err != nil {
			return
		}

		for _, block := range blocks.Blocks {
			p.SeeBlock(block.Hash)
		}
		// todo receive blocks
	case AccountBlocksCode:
		blocks := new(message.AccountBlocks)

		err = blocks.Deserialize(msg.Payload)
		if err != nil {
			return
		}

		for _, block := range blocks.Blocks {
			p.SeeBlock(block.Hash)
		}
		// todo receive blocks
	case NewSnapshotBlockCode:
		block := new(ledger.SnapshotBlock)
		err = block.Deserialize(msg.Payload)
		if err != nil {
			return
		}

		p.SeeBlock(block.Hash)
		n.BroadcastSnapshotBlock(block)
		n.receiveNewBlocks(block)
	case ForkCode:
		fork := new(message.Fork)
		err := fork.Deserialize(msg.Payload)
		if err != nil {
			return
		}

		var blocks []*ledger.SnapshotBlock
		var commonID *ledger.HashHeight
		for _, id := range fork.List {
			blocks, err = n.Chain.GetSnapshotBlocksByHeight(id.Height, 1, true, false)
			if err == nil && blocks[0].Hash == id.Hash {
				commonID = id
				break
			}
		}

		if commonID != nil {
			// get commonID, resend the request message
			return p.GetSnapshotBlocks(&message.GetSnapshotBlocks{
				From:    commonID,
				Count:   p.height - commonID.Height,
				Forward: true,
			})
		} else {
			// can`t get commonID, then getSnapshotBlocks from the lowest block
			// last item is lowest
			lowest := fork.List[len(fork.List)-1].Height
			return p.GetSnapshotBlocks(&message.GetSnapshotBlocks{
				From:    &ledger.HashHeight{Height: lowest,},
				Count:   p.height - lowest,
				Forward: true,
			})
		}

	case ExceptionCode:
		//exception, err := message.DeserializeException(msg.Payload)
		//if err != nil {
		//	return
		//}
		// todo handle exception
	default:
		return errors.New("unknown message")
	}

	return nil
}

func (n *Net) receiveNewBlocks(block *ledger.SnapshotBlock) {
	if n.Syncing() {
		n.newBlocks = append(n.newBlocks, block)
	} else {
		// todo
	}
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
		From: &ledger.HashHeight{
			Hash:   current.Hash,
			Height: current.Height,
		},
		Count:   peerHeight - n.FromHeight,
		Forward: true,
	}

	bestPeer.GetSubLedger(param)
}

type Broadcaster interface {
	BroadcastSnapshotBlock(block *ledger.SnapshotBlock)
	BroadcastAccountBlock(addr types.Address, block *ledger.AccountBlock)
}

type Fetcher interface {
	FetchSnapshotBlocks(start types.Hash, count uint64)
	FetchAccountBlocks(start types.Hash, count uint64, address types.Address)
}

type SnapshotBlockCallback func(block *ledger.SnapshotBlock)
type AccountblockCallback func(addr types.Address, block *ledger.AccountBlock)
type SyncStateCallback func(SyncState)

type Subscriber interface {
	// return the subId, use to unsubscibe
	// subId is always larger than 0
	SubscribeAccountBlock(fn AccountblockCallback) (subId int)
	// if subId is 0, then ignore
	UnsubscribeAccountBlock(subId int)

	// return the subId, use to unsubscibe
	// subId is always larger than 0
	SubscribeSnapshotBlock(fn SnapshotBlockCallback) (subId int)
	// if subId is 0, then ignore
	UnsubscribeSnapshotBlock(subId int)

	// return the subId, use to unsubscibe
	// subId is always larger than 0
	SubscribeSyncStatus(fn SyncStateCallback) (subId int)
	// if subId is 0, then ignore
	UnsubscribeSyncStatus(subId int)
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

	//req := newRequest(GetSnapshotBlocksCode, &message.GetSnapshotBlocks{
	//	From: &ledger.HashHeight{Hash: start,},
	//	Count: count,
	//	Forward: true,
	//}, snapshotBlocksTimeout)
	//
	//n.pool.add(req)
}

func (n *Net) FetchAccountBlocks(start types.Hash, count uint64, address types.Address) {
	// if the sync data has not downloaded, then ignore fetch request
	if atomic.LoadInt32(&n.downloaded) == 0 {
		return
	}

	//req := newRequest(GetAccountBlocksCode, &message.GetAccountBlocks{
	//	Address: address,
	//	From:    &ledger.HashHeight{Hash:   start,},
	//	Count:   count,
	//	Forward: true,
	//}, accountBlocksTimeout)
	//
	//n.pool.add(req)
}

func (n *Net) SubscribeAccountBlock(fn func(addr types.Address, block *ledger.AccountBlock)) (subId int) {
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
