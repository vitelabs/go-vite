package filters

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/rpc"
	"github.com/vitelabs/go-vite/rpcapi/api"
	"github.com/vitelabs/go-vite/vite"
	"sync"
	"time"
)

type FilterType byte

const (
	// UnknownSubscription indicates an unknown subscription type
	UnknownSubscription FilterType = iota
	ConfirmedLogsSubscription
	LogsSubscription
	NewAndConfirmedLogsSubscription
	ConfirmedAccountBlocksSubscription
	AccountBlocksSubscription
	// LastIndexSubscription keeps track of the last index
	LastIndexSubscription
)

type subscription struct {
	id                      rpc.ID
	typ                     FilterType
	createTime              time.Time
	installed               chan struct{}
	err                     chan error
	param                   FilterParam
	accountBlockCh          chan []*AccountBlockMsg
	confirmedAccountBlockCh chan []*ConfirmedAccountBlockMsg
	logsCh                  chan []*LogsMsg
}

type EventSystem struct {
	vite      *vite.Vite
	chain     *ChainSubscribe
	install   chan *subscription         // install filter
	uninstall chan *subscription         // remove filter
	acCh      chan AccountChainEvent     // Channel to receive new account chain event
	spCh      chan SnapshotChainEvent    // Channel to receive new snapshot chain event
	acDelCh   chan AccountChainDelEvent  // Channel to receive new account chain delete event when account chain fork
	spDelCh   chan SnapshotChainDelEvent // Channel to receive new snapshot chain delete event when snapshot chain fork
	stop      chan struct{}
}

const (
	acChanSize    = 10
	spChanSize    = 10
	acDelChanSize = 10
	spDelChanSize = 10
)

func NewEventSystem(v *vite.Vite) *EventSystem {
	es := &EventSystem{
		vite:    v,
		acCh:    make(chan AccountChainEvent, acChanSize),
		spCh:    make(chan SnapshotChainEvent, spChanSize),
		acDelCh: make(chan AccountChainDelEvent, acDelChanSize),
		spDelCh: make(chan SnapshotChainDelEvent, spDelChanSize),
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
	index := make(map[FilterType]map[rpc.ID]*subscription)
	for i := UnknownSubscription; i < LastIndexSubscription; i++ {
		index[i] = make(map[rpc.ID]*subscription)
	}

	for {
		select {
		case acEvent := <-es.acCh:
			es.handleAcEvent(&acEvent)
		case spEvent := <-es.spCh:
			es.handleSpEvent(&spEvent)
		case acDelEvent := <-es.acDelCh:
			es.handleAcDelEvent(&acDelEvent)
		case spDelEvent := <-es.spDelCh:
			es.handleSpDelEvent(&spDelEvent)
		case i := <-es.install:
			index[i.typ][i.id] = i
			close(i.installed)
		case u := <-es.uninstall:
			delete(index[u.typ], u.id)
			close(u.err)

		// system stopped
		case <-es.stop:
			return
		}
	}
}

func (es *EventSystem) handleAcEvent(acEvent *AccountChainEvent) {
	// TODO
}
func (es *EventSystem) handleSpEvent(acEvent *SnapshotChainEvent) {
	// TODO
	fmt.Printf("handle snapshot block event: %v\n", acEvent.SnapshotBlock.Hash)
}
func (es *EventSystem) handleAcDelEvent(acEvent *AccountChainDelEvent) {
	// TODO
}
func (es *EventSystem) handleSpDelEvent(acEvent *SnapshotChainDelEvent) {
	// TODO
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
			}
		}
		<-s.Err()
	})
}

type AccountBlockBase struct {
	From      types.Address
	To        types.Address
	BlockType byte
}

type AccountBlockMsg struct {
	AccountBlockBase
	Delete bool
}

type ConfirmedAccountBlockMsg struct {
	AccountBlockList []AccountBlockBase
	SnapshotHash     types.Hash
	Delete           bool
}

type VmLogBase struct {
	Logs             []ledger.VmLog
	AccountBlockHash types.Hash
}
type LogsMsg struct {
	Logs         []VmLogBase
	SnapshotHash *types.Hash
	Delete       bool
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
	}
	return es.subscribe(sub)
}

func (es *EventSystem) SubscribeLogs(p FilterParam, ch chan []*LogsMsg) (*RpcSubscription, error) {
	// TODO following code has bug
	from := uint64(0)
	to := uint64(0)
	if fromBlock, err := api.StringToUint64(p.FromBlock); err != nil {
		from = fromBlock
	}
	if toBlock, err := api.StringToUint64(p.ToBlock); err != nil {
		to = toBlock
	}

	// only interested in new logs
	if p.FromBlock == rpc.NewBlockNumber && p.ToBlock == rpc.NewBlockNumber {
		return es.subscribeLogs(p, ch), nil
	}
	// only interested in new confirmed logs
	if p.FromBlock == rpc.LatestBlockNumber && p.ToBlock == rpc.LatestBlockNumber {
		return es.subscribeConfirmedLogs(p, ch), nil
	}
	// only interested in confirmed logs within a specific block range
	if from >= 0 && to >= 0 && to >= from {
		return es.subscribeConfirmedLogs(p, ch), nil
	}
	// interested in confirmed logs from a specific block number, new confirmed logs and new logs
	if p.FromBlock >= rpc.LatestBlockNumber && p.ToBlock == rpc.NewBlockNumber {
		return es.subscribeNewAndConfirmedLogs(p, ch), nil
	}
	// interested in logs from a specific block number to new confirmed blocks
	if from >= 0 && p.ToBlock == rpc.LatestBlockNumber {
		return es.subscribeConfirmedLogs(p, ch), nil
	}
	return nil, errors.New("invalid from and to block combination")
}

func (es *EventSystem) subscribeLogs(p FilterParam, ch chan []*LogsMsg) *RpcSubscription {
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
	}
	return es.subscribe(sub)
}

func (es *EventSystem) subscribeNewAndConfirmedLogs(p FilterParam, ch chan []*LogsMsg) *RpcSubscription {
	sub := &subscription{
		id:                      rpc.NewID(),
		typ:                     NewAndConfirmedLogsSubscription,
		param:                   p,
		createTime:              time.Now(),
		installed:               make(chan struct{}),
		err:                     make(chan error),
		accountBlockCh:          make(chan []*AccountBlockMsg),
		confirmedAccountBlockCh: make(chan []*ConfirmedAccountBlockMsg),
		logsCh:                  ch,
	}
	return es.subscribe(sub)
}

func (es *EventSystem) subscribeConfirmedLogs(p FilterParam, ch chan []*LogsMsg) *RpcSubscription {
	sub := &subscription{
		id:                      rpc.NewID(),
		typ:                     ConfirmedLogsSubscription,
		param:                   p,
		createTime:              time.Now(),
		installed:               make(chan struct{}),
		err:                     make(chan error),
		accountBlockCh:          make(chan []*AccountBlockMsg),
		confirmedAccountBlockCh: make(chan []*ConfirmedAccountBlockMsg),
		logsCh:                  ch,
	}
	return es.subscribe(sub)
}

func (es *EventSystem) subscribe(sub *subscription) *RpcSubscription {
	es.install <- sub
	<-sub.installed
	return &RpcSubscription{ID: sub.id}
}
