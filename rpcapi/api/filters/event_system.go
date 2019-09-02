package filters

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/rpc"
	"github.com/vitelabs/go-vite/rpcapi/api"
	"github.com/vitelabs/go-vite/vite"
	"sync"
	"time"
)

type FilterType byte

var Es *EventSystem

const (
	LogsSubscription FilterType = iota
	LogsSubscriptionV2
	AccountBlocksSubscription
	AccountBlocksWithHeightSubscription
	AccountBlocksWithHeightSubscriptionV2
	OnroadBlocksSubscription
	OnroadBlocksSubscriptionV2
	SnapshotBlocksSubscription
	SnapshotBlocksSubscriptionV2
)

type subscription struct {
	id                       rpc.ID
	typ                      FilterType
	createTime               time.Time
	installed                chan struct{}
	err                      chan error
	param                    *api.FilterParam
	addr                     types.Address
	snapshotBlockCh          chan []*SnapshotBlock
	accountBlockCh           chan []*AccountBlock
	accountBlockWithHeightCh chan []*AccountBlockWithHeight
	logsCh                   chan []*Logs
	onroadMsgCh              chan []*OnroadMsg
}

type EventSystem struct {
	vite      *vite.Vite
	chain     *ChainSubscribe
	install   chan *subscription        // install filter
	uninstall chan *subscription        // remove filter
	acCh      chan []*AccountChainEvent // Channel to receive new account chain event
	acDelCh   chan []*AccountChainEvent // Channel to receive new account chain delete event when account chain fork
	sbCh      chan []*SnapshotChainEvent
	sbDelCh   chan []*SnapshotChainEvent
	stop      chan struct{}
	log       log15.Logger
}

const (
	acChanSize    = 100
	acDelChanSize = 10
	sbChanSize    = 10
	sbDelChanSize = 10
	installSize   = 10
	uninstallSize = 10
)

func NewEventSystem(v *vite.Vite) *EventSystem {
	es := &EventSystem{
		vite:      v,
		acCh:      make(chan []*AccountChainEvent, acChanSize),
		acDelCh:   make(chan []*AccountChainEvent, acDelChanSize),
		sbCh:      make(chan []*SnapshotChainEvent, sbChanSize),
		sbDelCh:   make(chan []*SnapshotChainEvent, sbDelChanSize),
		install:   make(chan *subscription, installSize),
		uninstall: make(chan *subscription, uninstallSize),
		stop:      make(chan struct{}),
		log:       log15.New("module", "rpc_api/event_system"),
	}
	return es
}

func (es *EventSystem) Start() {
	es.chain = NewChainSubscribe(es.vite, es)
	go es.eventLoop()
}

func (es *EventSystem) Stop() {
	close(es.stop)
	es.chain.Stop()
}

func (es *EventSystem) eventLoop() {
	es.log.Info("start event loop")
	index := make(map[FilterType]map[rpc.ID]*subscription)
	for i := LogsSubscription; i <= SnapshotBlocksSubscriptionV2; i++ {
		index[i] = make(map[rpc.ID]*subscription)
	}

	for {
		select {
		case acEvent := <-es.acCh:
			es.handleAcEvent(index, acEvent, false)
		case acDelEvent := <-es.acDelCh:
			es.handleAcEvent(index, acDelEvent, true)
		case sbEvent := <-es.sbCh:
			es.handleSbEvent(index, sbEvent, false)
		case sbDelEvent := <-es.sbDelCh:
			es.handleSbEvent(index, sbDelEvent, true)
		case i := <-es.install:
			es.log.Info("install ", "id", i.id)
			index[i.typ][i.id] = i
			close(i.installed)
		case u := <-es.uninstall:
			es.log.Info("uninstall ", "id", u.id)
			delete(index[u.typ], u.id)
			close(u.err)

		// system stopped
		case <-es.stop:
			for _, subscriptions := range index {
				for _, s := range subscriptions {
					close(s.err)
				}
			}
			index = nil
			es.log.Info("stop event loop")
			return
		}
	}
}

func (es *EventSystem) handleSbEvent(filters map[FilterType]map[rpc.ID]*subscription, sbEvent []*SnapshotChainEvent, removed bool) {
	if len(sbEvent) == 0 {
		return
	}
	blocks := make([]*SnapshotBlock, len(sbEvent))
	for i, e := range sbEvent {
		blocks[i] = &SnapshotBlock{Hash: e.Hash, Height: e.Height, HeightStr: api.Uint64ToString(e.Height), Removed: removed}
	}
	for _, f := range filters[SnapshotBlocksSubscription] {
		f.snapshotBlockCh <- blocks
	}
	for _, f := range filters[SnapshotBlocksSubscriptionV2] {
		f.snapshotBlockCh <- blocks
	}
}

func (es *EventSystem) handleAcEvent(filters map[FilterType]map[rpc.ID]*subscription, acEvent []*AccountChainEvent, removed bool) {
	if len(acEvent) == 0 {
		return
	}
	msgs := make([]*AccountBlock, len(acEvent))
	heightMsgs := make(map[types.Address][]*AccountBlockWithHeight)
	onroadMsgs := make(map[types.Address][]*OnroadMsg)
	deletedSendBlockHash := make(map[types.Hash]types.Address)
	for i, e := range acEvent {
		msgs[i] = &AccountBlock{Hash: e.Hash, Removed: removed}
		if _, ok := heightMsgs[e.Addr]; !ok {
			heightMsgs[e.Addr] = make([]*AccountBlockWithHeight, 0)
		}
		heightMsgs[e.Addr] = append(heightMsgs[e.Addr], &AccountBlockWithHeight{Height: e.Height, HeightStr: api.Uint64ToString(e.Height), Hash: e.Hash, Removed: removed})

		if removed {
			if ledger.IsSendBlock(e.BlockType) {
				deletedSendBlockHash[e.Hash] = e.ToAddr
			} else {
				if len(e.SendBlockList) > 0 {
					for _, sendBlock := range e.SendBlockList {
						deletedSendBlockHash[sendBlock.Hash] = sendBlock.ToAddr
					}
				}
			}
		} else {
			if ledger.IsSendBlock(e.BlockType) {
				onroadMsgs = appendOnroadMsg(onroadMsgs, e.ToAddr, e.Hash, false, removed)
			} else {
				onroadMsgs = appendOnroadMsg(onroadMsgs, e.Addr, e.FromBlockHash, true, removed)
				if len(e.SendBlockList) > 0 {
					for _, sendBlock := range e.SendBlockList {
						onroadMsgs = appendOnroadMsg(onroadMsgs, sendBlock.ToAddr, sendBlock.Hash, false, removed)
					}
				}
			}
		}
	}
	if removed {
		for _, e := range acEvent {
			if ledger.IsReceiveBlock(e.BlockType) {
				if _, ok := deletedSendBlockHash[e.FromBlockHash]; !ok {
					onroadMsgs = appendOnroadMsg(onroadMsgs, e.Addr, e.FromBlockHash, false, false)
				} else {
					delete(deletedSendBlockHash, e.FromBlockHash)
				}
			}
		}
	}
	for deletedSendHash, toAddr := range deletedSendBlockHash {
		onroadMsgs = appendOnroadMsg(onroadMsgs, toAddr, deletedSendHash, false, true)
	}
	// handle account blocks
	for _, f := range filters[AccountBlocksSubscription] {
		f.accountBlockCh <- msgs
	}
	// handle accountBlocksWithHeight
	for _, f := range filters[AccountBlocksWithHeightSubscription] {
		if hashHeightMsgs, ok := heightMsgs[f.addr]; ok {
			f.accountBlockWithHeightCh <- hashHeightMsgs
		}
	}
	for _, f := range filters[AccountBlocksWithHeightSubscriptionV2] {
		if hashHeightMsgs, ok := heightMsgs[f.addr]; ok {
			f.accountBlockWithHeightCh <- hashHeightMsgs
		}
	}
	// handle onroad blocks
	for _, f := range filters[OnroadBlocksSubscription] {
		if onroadMsgs, ok := onroadMsgs[f.addr]; ok {
			f.onroadMsgCh <- onroadMsgs
		}
	}
	for _, f := range filters[OnroadBlocksSubscriptionV2] {
		if onroadMsgs, ok := onroadMsgs[f.addr]; ok {
			f.onroadMsgCh <- onroadMsgs
		}
	}
	// handle logs
	for _, f := range filters[LogsSubscription] {
		var logs []*Logs
		for _, e := range acEvent {
			if matchedLogs := filterLogs(e, f.param, removed); len(matchedLogs) > 0 {
				logs = append(logs, matchedLogs...)
			}
		}
		if len(logs) > 0 {
			f.logsCh <- logs
		}
	}
	for _, f := range filters[LogsSubscriptionV2] {
		var logs []*Logs
		for _, e := range acEvent {
			if matchedLogs := filterLogs(e, f.param, removed); len(matchedLogs) > 0 {
				logs = append(logs, matchedLogs...)
			}
		}
		if len(logs) > 0 {
			f.logsCh <- logs
		}
	}
}

func appendOnroadMsg(onroadMsgs map[types.Address][]*OnroadMsg, toAddr types.Address, hash types.Hash, closed, removed bool) map[types.Address][]*OnroadMsg {
	if _, ok := onroadMsgs[toAddr]; !ok {
		onroadMsgs[toAddr] = make([]*OnroadMsg, 0)
	}
	onroadMsgs[toAddr] = append(onroadMsgs[toAddr], &OnroadMsg{Hash: hash, Closed: closed, Removed: removed})
	return onroadMsgs
}

func filterLogs(e *AccountChainEvent, filter *api.FilterParam, removed bool) []*Logs {
	if len(e.Logs) == 0 {
		return nil
	}
	var logs []*Logs
	if filter.AddrRange != nil {
		if hr, ok := filter.AddrRange[e.Addr]; !ok {
			return nil
		} else if (hr.FromHeight > 0 && hr.FromHeight > e.Height) || (hr.ToHeight > 0 && hr.ToHeight < e.Height) {
			return nil
		}
	}
	for _, l := range e.Logs {
		if api.FilterLog(filter, l) {
			logs = append(logs, &Logs{l, e.Hash, api.Uint64ToString(e.Height), &e.Addr, removed})
		}
	}
	return logs
}

type RpcSubscription struct {
	ID        rpc.ID
	sub       *subscription
	unSubOnce sync.Once
	es        *EventSystem
}

func (s *RpcSubscription) Err() <-chan error {
	return s.sub.err
}

func (s *RpcSubscription) Unsubscribe() {
	s.unSubOnce.Do(func() {
	uninstallLoop:
		for {
			select {
			case s.es.uninstall <- s.sub:
				break uninstallLoop
			case <-s.sub.accountBlockCh:
			case <-s.sub.accountBlockWithHeightCh:
			case <-s.sub.logsCh:
			case <-s.sub.snapshotBlockCh:
			case <-s.sub.onroadMsgCh:
			}
		}
		<-s.Err()
	})
}

func (es *EventSystem) SubscribeAccountBlocks(ch chan []*AccountBlock) *RpcSubscription {
	sub := &subscription{
		id:                       rpc.NewID(),
		typ:                      AccountBlocksSubscription,
		createTime:               time.Now(),
		installed:                make(chan struct{}),
		err:                      make(chan error),
		snapshotBlockCh:          make(chan []*SnapshotBlock),
		accountBlockCh:           ch,
		accountBlockWithHeightCh: make(chan []*AccountBlockWithHeight),
		logsCh:                   make(chan []*Logs),
		onroadMsgCh:              make(chan []*OnroadMsg),
	}
	return es.subscribe(sub)
}

func (es *EventSystem) SubscribeAccountBlocksByAddr(addr types.Address, ch chan []*AccountBlockWithHeight, ft FilterType) *RpcSubscription {
	sub := &subscription{
		id:                       rpc.NewID(),
		typ:                      ft,
		addr:                     addr,
		createTime:               time.Now(),
		installed:                make(chan struct{}),
		err:                      make(chan error),
		snapshotBlockCh:          make(chan []*SnapshotBlock),
		accountBlockCh:           make(chan []*AccountBlock),
		accountBlockWithHeightCh: ch,
		logsCh:                   make(chan []*Logs),
		onroadMsgCh:              make(chan []*OnroadMsg),
	}
	return es.subscribe(sub)
}

func (es *EventSystem) SubscribeOnroadBlocksByAddr(addr types.Address, ch chan []*OnroadMsg, ft FilterType) *RpcSubscription {
	sub := &subscription{
		id:                       rpc.NewID(),
		typ:                      ft,
		addr:                     addr,
		createTime:               time.Now(),
		installed:                make(chan struct{}),
		err:                      make(chan error),
		snapshotBlockCh:          make(chan []*SnapshotBlock),
		accountBlockCh:           make(chan []*AccountBlock),
		accountBlockWithHeightCh: make(chan []*AccountBlockWithHeight),
		logsCh:                   make(chan []*Logs),
		onroadMsgCh:              ch,
	}
	return es.subscribe(sub)
}

func (es *EventSystem) SubscribeSnapshotBlocks(ch chan []*SnapshotBlock, ft FilterType) *RpcSubscription {
	sub := &subscription{
		id:                       rpc.NewID(),
		typ:                      ft,
		createTime:               time.Now(),
		installed:                make(chan struct{}),
		err:                      make(chan error),
		snapshotBlockCh:          ch,
		accountBlockCh:           make(chan []*AccountBlock),
		accountBlockWithHeightCh: make(chan []*AccountBlockWithHeight),
		logsCh:                   make(chan []*Logs),
		onroadMsgCh:              make(chan []*OnroadMsg),
	}
	return es.subscribe(sub)
}

func (es *EventSystem) SubscribeLogs(p *api.FilterParam, ch chan []*Logs, ft FilterType) *RpcSubscription {
	sub := &subscription{
		id:                       rpc.NewID(),
		typ:                      ft,
		param:                    p,
		createTime:               time.Now(),
		installed:                make(chan struct{}),
		err:                      make(chan error),
		snapshotBlockCh:          make(chan []*SnapshotBlock),
		accountBlockCh:           make(chan []*AccountBlock),
		accountBlockWithHeightCh: make(chan []*AccountBlockWithHeight),
		logsCh:                   ch,
		onroadMsgCh:              make(chan []*OnroadMsg),
	}
	return es.subscribe(sub)
}

func (es *EventSystem) subscribe(s *subscription) *RpcSubscription {
	es.install <- s
	<-s.installed
	return &RpcSubscription{ID: s.id, sub: s, es: es}
}
