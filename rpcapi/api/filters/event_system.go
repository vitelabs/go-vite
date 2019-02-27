package filters

import (
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/rpc"
	"github.com/vitelabs/go-vite/vite"
	"sync"
	"time"
)

type FilterType byte

var Es *EventSystem

const (
	// UnknownSubscription indicates an unknown subscription type
	UnknownSubscription FilterType = iota
	ConfirmedLogsSubscription
	LogsSubscription
	ConfirmedAccountBlocksSubscription
	AccountBlocksSubscription
	// LastIndexSubscription keeps track of the last index
	LastIndexSubscription
)

type heightRange struct {
	fromHeight uint64
	toHeight   uint64
}

type filterParam struct {
	snapshotRange *heightRange
	addrRange     map[types.Address]heightRange
	topics        [][]types.Hash
	accountHash   *types.Hash
	snapshotHash  *types.Hash
}

type subscription struct {
	id                      rpc.ID
	typ                     FilterType
	createTime              time.Time
	installed               chan struct{}
	err                     chan error
	param                   *filterParam
	accountBlockCh          chan []*AccountBlockMsg
	confirmedAccountBlockCh chan []*ConfirmedAccountBlockMsg
	logsCh                  chan []*LogsMsg
	confirmedLogsCh         chan []*LogsMsg
}

type EventSystem struct {
	vite      *vite.Vite
	chain     *ChainSubscribe
	install   chan *subscription         // install filter
	uninstall chan *subscription         // remove filter
	acCh      chan []*AccountChainEvent  // Channel to receive new account chain event
	spCh      chan []*SnapshotChainEvent // Channel to receive new snapshot chain event
	acDelCh   chan AccountChainDelEvent  // Channel to receive new account chain delete event when account chain fork
	spDelCh   chan SnapshotChainDelEvent // Channel to receive new snapshot chain delete event when snapshot chain fork
	stop      chan struct{}
	log       log15.Logger
}

const (
	acChanSize    = 100
	spChanSize    = 10
	acDelChanSize = 10
	spDelChanSize = 10
	installSize   = 10
	uninstallSize = 10
)

func NewEventSystem(v *vite.Vite) *EventSystem {
	es := &EventSystem{
		vite:      v,
		acCh:      make(chan []*AccountChainEvent, acChanSize),
		spCh:      make(chan []*SnapshotChainEvent, spChanSize),
		acDelCh:   make(chan AccountChainDelEvent, acDelChanSize),
		spDelCh:   make(chan SnapshotChainDelEvent, spDelChanSize),
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
	fmt.Println("start event loop")
	index := make(map[FilterType]map[rpc.ID]*subscription)
	for i := UnknownSubscription; i < LastIndexSubscription; i++ {
		index[i] = make(map[rpc.ID]*subscription)
	}

	for {
		select {
		case acEvent := <-es.acCh:
			es.handleAcEvent(index, acEvent)
		case spEvent := <-es.spCh:
			es.handleSpEvent(index, spEvent)
		case acDelEvent := <-es.acDelCh:
			es.handleAcDelEvent(&acDelEvent)
		case spDelEvent := <-es.spDelCh:
			es.handleSpDelEvent(&spDelEvent)
		case i := <-es.install:
			fmt.Println("install " + i.id)
			index[i.typ][i.id] = i
			close(i.installed)
		case u := <-es.uninstall:
			fmt.Println("uninstall " + u.id)
			delete(index[u.typ], u.id)
			close(u.err)

		// system stopped
		case <-es.stop:
			for _, subscriptions := range index {
				for _, s := range subscriptions {
					fmt.Println("close " + s.id)
					close(s.err)
				}
			}
			index = nil
			return
		}
	}
}

func (es *EventSystem) handleAcEvent(filters map[FilterType]map[rpc.ID]*subscription, acEvent []*AccountChainEvent) {
	if len(acEvent) == 0 {
		return
	}
	// handle account blocks
	msgs := make([]*AccountBlockMsg, len(acEvent))
	for i, e := range acEvent {
		msgs[i] = &AccountBlockMsg{Hash: e.Hash, Removed: false}
	}
	for _, f := range filters[AccountBlocksSubscription] {
		f.accountBlockCh <- msgs
	}
	// handle logs
	for _, f := range filters[LogsSubscription] {
		var logs []*LogsMsg
		for _, e := range acEvent {
			if matchedLogs := filterLogs(e, f.param, false); len(matchedLogs) > 0 {
				logs = append(logs, matchedLogs...)
			}
		}
		if len(logs) > 0 {
			f.logsCh <- logs
		}
	}
}

func (es *EventSystem) handleSpEvent(filters map[FilterType]map[rpc.ID]*subscription, scEvents []*SnapshotChainEvent) {
	if len(scEvents) == 0 {
		return
	}
	msgs := make([]*ConfirmedAccountBlockMsg, len(scEvents))
	for scIndex, scEvent := range scEvents {
		hashList := make([]types.Hash, len(scEvent.Content))
		// handle account blocks
		for i, e := range scEvent.Content {
			hashList[i] = e.Hash
		}
		msgs[scIndex] = &ConfirmedAccountBlockMsg{hashList, scEvent.SnapshotHash, false}
		// handle logs
		for _, f := range filters[LogsSubscription] {
			var logs []*LogsMsg
			for _, e := range scEvent.Content {
				if matchedLogs := filterConfirmedLogs(e, scEvent.SnapshotHash, scEvent.SnapshotHeight, f.param, false); len(matchedLogs) > 0 {
					logs = append(logs, matchedLogs...)
				}
			}
			if len(logs) > 0 {
				f.confirmedLogsCh <- logs
			}
		}
	}
	for _, f := range filters[AccountBlocksSubscription] {
		f.confirmedAccountBlockCh <- msgs
	}

}
func (es *EventSystem) handleAcDelEvent(acEvent *AccountChainDelEvent) {
	// TODO
}
func (es *EventSystem) handleSpDelEvent(acEvent *SnapshotChainDelEvent) {
	// TODO
}

func filterLogs(e *AccountChainEvent, filter *filterParam, removed bool) []*LogsMsg {
	if len(e.Logs) == 0 {
		return nil
	}
	var logs []*LogsMsg
	if filter.accountHash != nil && *filter.accountHash != e.Hash {
		return nil
	}
	if filter.addrRange != nil {
		if hr, ok := filter.addrRange[e.Addr]; !ok {
			return nil
		} else if (hr.fromHeight > 0 && hr.fromHeight > e.Height) || (hr.toHeight > 0 && hr.toHeight < e.Height) {
			return nil
		}
	}
	for _, l := range e.Logs {
		if len(l.Topics) < len(filter.topics) {
			return nil
		}
		for i, topicRange := range filter.topics {
			flag := false
			if len(topicRange) == 0 {
				flag = true
				continue
			}
			for _, topic := range topicRange {
				if topic == l.Topics[i] {
					flag = true
					continue
				}
			}
			if !flag {
				return nil
			}
		}
		logs = append(logs, &LogsMsg{l, e.Hash, nil, &e.Addr, removed})
	}
	return logs
}

func filterConfirmedLogs(e *AccountChainEvent, snapshotHash types.Hash, snapshotHeight uint64, filter *filterParam, removed bool) []*LogsMsg {
	// TODO
	return nil
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
			case <-s.sub.confirmedAccountBlockCh:
			case <-s.sub.logsCh:
			case <-s.sub.confirmedLogsCh:
			}
		}
		<-s.Err()
	})
}

func (es *EventSystem) SubscribeAccountBlocks(ch chan []*AccountBlockMsg) *RpcSubscription {
	sub := &subscription{
		id:                      rpc.NewID(),
		typ:                     AccountBlocksSubscription,
		createTime:              time.Now(),
		installed:               make(chan struct{}),
		err:                     make(chan error),
		accountBlockCh:          ch,
		confirmedAccountBlockCh: make(chan []*ConfirmedAccountBlockMsg),
		logsCh:                  make(chan []*LogsMsg),
		confirmedLogsCh:         make(chan []*LogsMsg),
	}
	return es.subscribe(sub)
}

func (es *EventSystem) SubscribeConfirmedAccountBlocks(ch chan []*ConfirmedAccountBlockMsg) *RpcSubscription {
	sub := &subscription{
		id:                      rpc.NewID(),
		typ:                     ConfirmedAccountBlocksSubscription,
		createTime:              time.Now(),
		installed:               make(chan struct{}),
		err:                     make(chan error),
		accountBlockCh:          make(chan []*AccountBlockMsg),
		confirmedAccountBlockCh: ch,
		logsCh:                  make(chan []*LogsMsg),
		confirmedLogsCh:         make(chan []*LogsMsg),
	}
	return es.subscribe(sub)
}

func (es *EventSystem) SubscribeLogs(p *filterParam, ch chan []*LogsMsg) *RpcSubscription {
	sub := &subscription{
		id:                      rpc.NewID(),
		typ:                     LogsSubscription,
		param:                   p,
		createTime:              time.Now(),
		installed:               make(chan struct{}),
		err:                     make(chan error),
		accountBlockCh:          make(chan []*AccountBlockMsg),
		confirmedAccountBlockCh: make(chan []*ConfirmedAccountBlockMsg),
		logsCh:                  ch,
		confirmedLogsCh:         make(chan []*LogsMsg),
	}
	return es.subscribe(sub)
}

func (es *EventSystem) SubscribeConfirmedLogs(p *filterParam, ch chan []*LogsMsg) *RpcSubscription {
	sub := &subscription{
		id:                      rpc.NewID(),
		typ:                     ConfirmedLogsSubscription,
		param:                   p,
		createTime:              time.Now(),
		installed:               make(chan struct{}),
		err:                     make(chan error),
		accountBlockCh:          make(chan []*AccountBlockMsg),
		confirmedAccountBlockCh: make(chan []*ConfirmedAccountBlockMsg),
		logsCh:                  make(chan []*LogsMsg),
		confirmedLogsCh:         ch,
	}
	return es.subscribe(sub)
}

func (es *EventSystem) subscribe(s *subscription) *RpcSubscription {
	es.install <- s
	<-s.installed
	return &RpcSubscription{ID: s.id, sub: s, es: es}
}
