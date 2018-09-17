package net

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/seiflotfy/cuckoofilter"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"io/ioutil"
	"log"
	"net"
	"sync"
	"time"
)

type Config struct {
	NetID uint64
}

type Net struct {
	*Config
	start         time.Time
	peers         *peerSet
	snapshotFeed  *snapshotBlockFeed
	accountFeed   *accountBlockFeed
	term          chan struct{}
	pool          *reqPool
	FromHeight    uint64
	TargetHeight  uint64
	syncState     SyncState
	slock         sync.RWMutex // use for syncState change
	stateFeed     *SyncStateFeed
	SnapshotChain BlockChain
	blockRecord   *cuckoofilter.CuckooFilter // record blocks has retrieved from network
	log log15.Logger
}

func New(cfg *Config) *Net {
	peerSet := NewPeerSet()

	n := &Net{
		Config:       cfg,
		peers:        peerSet,
		snapshotFeed: new(snapshotBlockFeed),
		accountFeed:  new(accountBlockFeed),
		stateFeed:    new(SyncStateFeed),
		term:         make(chan struct{}),
		pool:         NewReqPool(peerSet),
		blockRecord:  cuckoofilter.NewCuckooFilter(10000),
		log: log15.New("module", "vite/net"),
	}

	return n
}

func (n *Net) Start() {
	n.start = time.Now()
	go n.preSync()
}

func (n *Net) Stop() {
	select {
	case <-n.term:
	default:
		close(n.term)
	}
}

func (n *Net) Syncing() bool {
	n.slock.RLock()
	defer n.slock.RUnlock()
	return n.syncState == Syncing
}

func (n *Net) startSync() {
	n.SetSyncState(Syncing)
	go n.checkChainHeight()
}

func (n *Net) checkChainHeight()  {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	begin := time.Now()

	for {
		select {
		case now := <- ticker.C:
			current, err := n.SnapshotChain.GetLatestSnapshotBlock()
			if err != nil {
				return
			}
			if current.Height == n.TargetHeight {
				n.SetSyncState(Syncdone)
			}
			if now.Sub(begin) > waitForChainGrow {
				n.SetSyncState(Syncerr)
			}
		case <- n.term:
			return
		}
	}
}

func (n *Net) SetSyncState(st SyncState) {
	n.slock.Lock()
	defer n.slock.Unlock()
	n.syncState = st
	n.stateFeed.Notify(st)
}

func (n *Net) SyncState() SyncState {
	n.slock.Lock()
	defer n.slock.Unlock()

	return n.syncState
}

func (n *Net) ReceiveConn(conn net.Conn) {
	select {
	case <-n.term:
	default:
	}

}

func (n *Net) HandlePeer(p *Peer) {
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
		n.removePeer(p)
		return
	}

	n.startPeer(p)
}

func (n *Net) HandleMsg(p *Peer) (err error) {
	msg, err := p.ts.ReadMsg()
	if err != nil {
		return
	}
	defer msg.Discard()

	Code := Cmd(msg.Cmd)
	if Code == HandshakeCode {
		return errHandshakeTwice
	}

	payload, err := ioutil.ReadAll(msg.Payload)
	if err != nil {
		return
	}

	switch Cmd(msg.Cmd) {
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
			p.Send(SubLedgerCode, &subLedgerMsg{
				snapshotblocks: snapshotblocks,
				accountblocks:  accountblocks,
			})
		}
	case GetSnapshotBlockHeadersCode:
	case GetSnapshotBlockBodiesCode:
	case GetSnapshotBlocksCode:
	case GetSnapshotBlocksByHashCode:
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
	case GetAccountBlocksByHashCode:
	case SubLedgerCode:
		subledger := new(subLedgerMsg)
		err = subledger.Deserialize(payload)
		if err != nil {
			return
		}

		p.receive(SubLedgerCode, subledger)

		for _, block := range subledger.snapshotblocks {
			p.SeeBlock(block.Hash)
		}
		for _, block := range subledger.accountblocks {
			p.SeeBlock(block.Hash)
		}
		n.receiveSnapshotBlock(subledger.snapshotblocks)
		n.receiveAccountBlocks(subledger.accountblocks)
	case SnapshotBlockHeadersCode:
	case SnapshotBlockBodiesCode:
	case SnapshotBlocksCode:
		blocks := []*ledger.SnapshotBlock

		err = blocks.Deserialize(payload)
		if err != nil {
			return
		}

		p.receive(SnapshotBlocksCode, blocks)

		for _, block := range blocks {
			p.SeeBlock(block.Hash)
		}
		n.receiveSnapshotBlock(blocks)
	case AccountBlocksCode:
		blocks := []*ledger.AccountBlock

		err = blocks.Deserialize(payload)
		if err != nil {
			return
		}

		p.receive(AccountBlocksCode, blocks)

		for _, block := range blocks {
			p.SeeBlock(block.Hash)
		}
		n.receiveAccountBlocks(blocks)
	case NewSnapshotBlockCode:
		block := new(ledger.SnapshotBlock)
		err = block.Deserialize(payload)
		if err != nil {
			return
		}

		p.receive(NewSnapshotBlockCode, block)
		p.SeeBlock(block.Hash)
		n.BroadcastSnapshotBlock(block, false)
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

func (n *Net) BroadcastSnapshotBlock(block *ledger.SnapshotBlock, propagate bool) {
	peers := n.peers.UnknownBlock(block.Hash)

	for _, peer := range peers {
		go peer.SendNewSnapshotBlock(block)
	}
}

func (n *Net) BroadcastSnapshotBlocks(blocks []*ledger.SnapshotBlock, propagate bool) {
	for _, block := range blocks {
		go func(b *ledger.SnapshotBlock) {
			peers := n.peers.UnknownBlock(block.Hash)

			for _, peer := range peers {
				go peer.SendNewSnapshotBlock(block)
			}
		}(block)
	}
}

func (n *Net) BroadcastAccountBlocks(blocks []*ledger.AccountBlock, propagate bool) {
	for _, block := range blocks {
		peers := n.peers.UnknownBlock(block.Hash)
		for _, peer := range peers {
			go func(peers []*Peer, b *ledger.AccountBlock) {
				peer.SendAccountBlocks([]*ledger.AccountBlock{block})
			}(peers, block)
		}
	}
}

func (n *Net) FetchSnapshotBlocks(s *Segment) {
	req := newReq(GetSnapshotBlocksCode, s, func(cmd Cmd, i interface{}) (done bool, err error) {
		if cmd != SnapshotBlocksCode {
			return false, nil
		}
		blocks, ok := i.(SnapshotBlocksMsg)
		if !ok {
			return false, nil
		}
		n.receiveSnapshotBlock(blocks)

		if uint64(len(blocks)) != (s.To.Height - s.From.Height) {
			return false, fmt.Errorf("incomplete snapshotblocks")
		}

		return true, nil
	}, snapshotBlocksTimeout)

	n.pool.Add(req)
}

func (n *Net) FetchSnapshotBlocksByHash(hashes []types.Hash) {
	strangeHashes := make([]types.Hash, 0, len(hashes))
	for _, hash := range hashes {
		if !n.blockRecord.Lookup(hash[:]) {
			strangeHashes = append(strangeHashes, hash)
		}
	}

	if len(strangeHashes) != 0 {
		req := newReq(GetAccountBlocksByHashCode, strangeHashes, func(cmd Cmd, i interface{}) (done bool, err error) {

		}, snapshotBlocksTimeout)

		n.pool.Add(req)
	}
}

func (n *Net) FetchAccountBlocks(as AccountSegment) {
	req := newReq(GetAccountBlocksCode, as, func(cmd Cmd, i interface{}) (done bool, err error) {
		if cmd != AccountBlocksCode {
			return false, nil
		}
		mapBlocks, ok := i.(AccountBlocksMsg)
		if !ok {
			return false, nil
		}
		for account, blocks := range mapBlocks {
			if as[account] == nil {
				return false, fmt.Errorf("missing accountblocks of account %s", account)
			}

			n.receiveAccountBlocks(blocks)

			if uint64(len(blocks)) != (as[account].To.Height - as[account].From.Height) {
				return false, fmt.Errorf("incomplete accountblocks of account %s", account)
			}
		}

		return true, nil
	}, accountBlocksTimeout)

	n.pool.Add(req)
}

func (n *Net) FetchAccountBlocksByHash(hashes []types.Hash) {
	strangeHashes := make([]types.Hash, 0, len(hashes))
	for _, hash := range hashes {
		if !n.blockRecord.Lookup(hash[:]) {
			strangeHashes = append(strangeHashes, hash)
		}
	}

	if len(strangeHashes) != 0 {
		req := newReq(GetSnapshotBlocksByHashCode, strangeHashes, func(cmd Cmd, i interface{}) (done bool, err error) {

		}, snapshotBlocksTimeout)

		n.pool.Add(req)
	}
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

func (n *Net) preSync() {
	timer := time.NewTimer(waitEnoughPeers)
	defer timer.Stop()

	loop:
	for {
		select {
		case <- timer.C:
			break loop
		case <- n.term:
			return
		default:
			if n.peers.Count() >= enoughPeers {
				break loop
			}
		}
	}

	n.syncWithPeer(n.peers.BestPeer())
}

func (n *Net) syncWithPeer(p *Peer) error {
	if n.Syncing() {
		return errSynced
	}

	current, err := n.SnapshotChain.GetLatestSnapshotBlock()
	if err != nil {
		n.log.Error("getLatestSnapshotBlock error", "error", err)
		return nil
	}

	peerHeight := p.Head().Height
	if peerHeight <= current.Height {
		return nil
	}

	n.FromHeight = current.Height
	n.TargetHeight = peerHeight
	n.startSync()

	seg := &Segment{
		From:    &BlockID{current.Hash, current.Height},
		To:      &BlockID{p.head, p.height},
		Step:    0,
		Forward: true,
	}

	req := newReq(GetSubLedgerCode, seg, func(cmd Cmd, v interface{}) (done bool, err error) {
		if cmd == ExceptionCode && v == Missing {
			return false, Missing
		}

		if cmd != SubLedgerCode {
			return false, nil
		}
		subledger, ok := v.(subLedgerMsg)
		if !ok {
			return false, fmt.Errorf("not SubLedgerMsg")
		}
		if uint64(len(subledger.snapshotblocks)) != (seg.To.Height - seg.From.Height) {
			return false, fmt.Errorf("incomplete snapshot blocks")
		}
		return true, nil
	}, subledgerTimeout)

	timer := time.NewTimer(subledgerTimeout)
	defer timer.Stop()

	n.pool.Add(req)

	for {
		var done bool
		var err error

		select {
		case done = <- req.done:
			case err = <- req.errch:
		case <- timer.C:
			return errMsgTimeout
		}

		if done && err == nil {
			return nil
		}
	}

	return nil
}

func (n *Net) startPeer(p *Peer) {
	n.peers.Add(p)

	go n.heartbeat(p)

	go n.syncWithPeer(p)

	for {
		select {
		case <- n.term:
			return
		default:
		}

		if err := n.HandleMsg(p); err != nil {
			n.removePeer(p)
		}
	}
}

func (n *Net) heartbeat(p *Peer)  {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <- n.term:
			return
		case <- ticker.C:
			current, err := n.SnapshotChain.GetLatestSnapshotBlock()
			if err != nil {
				return
			}

			p.Send(StatusCode, &BlockID{
				Hash:   current.Hash,
				Height: current.Height,
			})
		}
	}
}

func (n *Net) removePeer(p *Peer) {
	p.Destroy()
	n.peers.Del(p)
}

// get current netInfo (peers, syncStatus, ...)
func (n *Net) Status() *NetStatus {
	running := true
	select {
	case <-n.term:
		running = false
	default:
	}

	n.slock.RLock()
	st := n.syncState
	n.slock.RUnlock()

	return &NetStatus{
		Peers:     n.peers.Info(),
		Running:   running,
		Uptime:    time.Now().Sub(n.start),
		SyncState: st,
	}
}

type NetStatus struct {
	Peers     []*PeerInfo
	SyncState SyncState
	Uptime    time.Duration
	Running   bool
}
