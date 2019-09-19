package filters

import (
	"context"
	"errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/rpc"
	"github.com/vitelabs/go-vite/rpcapi/api"
	"github.com/vitelabs/go-vite/vite"
	"sync"
	"time"
)

var (
	deadline = 5 * time.Minute // consider a filter inactive if it has not been polled for within deadline
)

type filter struct {
	typ              FilterType
	deadline         *time.Timer
	param            api.FilterParam
	s                *RpcSubscription
	blocks           []*AccountBlock
	blocksWithHeight []*AccountBlockWithHeight
	logs             []*Logs
	snapshotBlocks   []*SnapshotBlock
	onroadMsgs       []*OnroadMsg
}

type SubscribeApi struct {
	vite        *vite.Vite
	log         log15.Logger
	filterMap   map[rpc.ID]*filter
	filterMapMu sync.Mutex
	eventSystem *EventSystem
}

func NewSubscribeApi(vite *vite.Vite) *SubscribeApi {
	if Es == nil {
		panic("Set \"SubscribeEnabled\" to \"true\" in node_config.json")
	}
	s := &SubscribeApi{
		vite:        vite,
		log:         log15.New("module", "rpc_api/subscribe_api"),
		filterMap:   make(map[rpc.ID]*filter),
		eventSystem: Es,
	}
	go s.timeoutLoop()
	return s
}

func (s *SubscribeApi) timeoutLoop() {
	s.log.Info("start timeout loop")
	// delete timeout filters every 5 minutes
	ticker := time.NewTicker(5 * time.Minute)
	for {
		<-ticker.C
		s.filterMapMu.Lock()
		for id, f := range s.filterMap {
			select {
			case <-f.deadline.C:
				f.s.Unsubscribe()
				delete(s.filterMap, id)
			default:
				continue
			}
		}
		s.filterMapMu.Unlock()
	}
}

type RpcFilterParam struct {
	AddrRange map[string]*api.Range `json:"addrRange"`
	Topics    [][]types.Hash        `json:"topics"`
}

type AccountBlock struct {
	Hash    types.Hash `json:"hash"`
	Removed bool       `json:"removed"`
}

type OnroadMsg struct {
	Hash    types.Hash `json:"hash"`
	Closed  bool       `json:"closed"`
	Removed bool       `json:"removed"`
}

type OnroadMsgV2 struct {
	Hash     types.Hash `json:"hash"`
	Received bool       `json:"received"`
	Removed  bool       `json:"removed"`
}

type AccountBlockWithHeight struct {
	Hash      types.Hash `json:"hash"`
	Height    uint64     `json:"height"` // Deprecated
	HeightStr string     `json:"heightStr"`
	Removed   bool       `json:"removed"`
}
type AccountBlockWithHeightV2 struct {
	Hash    types.Hash `json:"hash"`
	Height  string     `json:"height"`
	Removed bool       `json:"removed"`
}

type SnapshotBlock struct {
	Hash      types.Hash `json:"hash"`
	Height    uint64     `json:"height"` // Deprecated
	HeightStr string     `json:"heightStr"`
	Removed   bool       `json:"removed"`
}
type SnapshotBlockV2 struct {
	Hash    types.Hash `json:"hash"`
	Height  string     `json:"height"`
	Removed bool       `json:"removed"`
}

type Logs struct {
	Log              *ledger.VmLog  `json:"log"`
	AccountBlockHash types.Hash     `json:"accountBlockHash"`
	AccountHeight    string         `json:"accountHeight"`
	Addr             *types.Address `json:"addr"`
	Removed          bool           `json:"removed"`
}
type LogsV2 struct {
	Log              *ledger.VmLog  `json:"vmlog"`
	AccountBlockHash types.Hash     `json:"accountBlockHash"`
	AccountHeight    string         `json:"accountBlockHeight"`
	Addr             *types.Address `json:"address"`
	Removed          bool           `json:"removed"`
}

// Deprecated: use subscribe_createSnapshotBlockFilter instead
func (s *SubscribeApi) NewSnapshotBlocksFilter() (rpc.ID, error) {
	return s.createSnapshotBlockFilter(SnapshotBlocksSubscription)
}
func (s *SubscribeApi) CreateSnapshotBlockFilter() (rpc.ID, error) {
	return s.createSnapshotBlockFilter(SnapshotBlocksSubscriptionV2)
}
func (s *SubscribeApi) createSnapshotBlockFilter(ft FilterType) (rpc.ID, error) {
	s.log.Info("createSnapshotBlockFilter")
	var (
		sbCh  = make(chan []*SnapshotBlock)
		sbSub = s.eventSystem.SubscribeSnapshotBlocks(sbCh, ft)
	)

	s.filterMapMu.Lock()
	s.filterMap[sbSub.ID] = &filter{typ: sbSub.sub.typ, deadline: time.NewTimer(deadline), s: sbSub}
	s.filterMapMu.Unlock()

	go func() {
		for {
			select {
			case sb := <-sbCh:
				s.filterMapMu.Lock()
				if f, found := s.filterMap[sbSub.ID]; found {
					f.snapshotBlocks = append(f.snapshotBlocks, sb...)
				}
				s.filterMapMu.Unlock()
			case <-sbSub.Err():
				s.filterMapMu.Lock()
				delete(s.filterMap, sbSub.ID)
				s.filterMapMu.Unlock()
				return
			}
		}
	}()

	return sbSub.ID, nil
}

// Deprecated: use subscribe_createAccountBlockFilter instead
func (s *SubscribeApi) NewAccountBlocksFilter() (rpc.ID, error) {
	return s.createAccountBlockFilter()
}
func (s *SubscribeApi) CreateAccountBlockFilter() (rpc.ID, error) {
	return s.createAccountBlockFilter()
}
func (s *SubscribeApi) createAccountBlockFilter() (rpc.ID, error) {
	s.log.Info("createAccountBlockFilter")
	var (
		acCh  = make(chan []*AccountBlock)
		acSub = s.eventSystem.SubscribeAccountBlocks(acCh)
	)

	s.filterMapMu.Lock()
	s.filterMap[acSub.ID] = &filter{typ: acSub.sub.typ, deadline: time.NewTimer(deadline), s: acSub}
	s.filterMapMu.Unlock()

	go func() {
		for {
			select {
			case ac := <-acCh:
				s.filterMapMu.Lock()
				if f, found := s.filterMap[acSub.ID]; found {
					f.blocks = append(f.blocks, ac...)
				}
				s.filterMapMu.Unlock()
			case <-acSub.Err():
				s.filterMapMu.Lock()
				delete(s.filterMap, acSub.ID)
				s.filterMapMu.Unlock()
				return
			}
		}
	}()

	return acSub.ID, nil
}

// Deprecated: use subscribe_createAccountBlockFilterByAddress instead
func (s *SubscribeApi) NewAccountBlocksByAddrFilter(addr types.Address) (rpc.ID, error) {
	return s.createAccountBlockFilterByAddress(addr, AccountBlocksWithHeightSubscription)
}
func (s *SubscribeApi) CreateAccountBlockFilterByAddress(addr types.Address) (rpc.ID, error) {
	return s.createAccountBlockFilterByAddress(addr, AccountBlocksWithHeightSubscriptionV2)
}
func (s *SubscribeApi) createAccountBlockFilterByAddress(addr types.Address, ft FilterType) (rpc.ID, error) {
	s.log.Info("createAccountBlockFilterByAddress")
	var (
		acCh  = make(chan []*AccountBlockWithHeight)
		acSub = s.eventSystem.SubscribeAccountBlocksByAddr(addr, acCh, ft)
	)

	s.filterMapMu.Lock()
	s.filterMap[acSub.ID] = &filter{typ: acSub.sub.typ, deadline: time.NewTimer(deadline), s: acSub}
	s.filterMapMu.Unlock()

	go func() {
		for {
			select {
			case ac := <-acCh:
				s.filterMapMu.Lock()
				if f, found := s.filterMap[acSub.ID]; found {
					f.blocksWithHeight = append(f.blocksWithHeight, ac...)
				}
				s.filterMapMu.Unlock()
			case <-acSub.Err():
				s.filterMapMu.Lock()
				delete(s.filterMap, acSub.ID)
				s.filterMapMu.Unlock()
				return
			}
		}
	}()

	return acSub.ID, nil
}

// Deprecated: use subscribe_createUnreceivedBlockFilterByAddress instead
func (s *SubscribeApi) NewOnroadBlocksByAddrFilter(addr types.Address) (rpc.ID, error) {
	return s.createUnreceivedBlockFilterByAddress(addr, OnroadBlocksSubscription)
}
func (s *SubscribeApi) CreateUnreceivedBlockFilterByAddress(addr types.Address) (rpc.ID, error) {
	return s.createUnreceivedBlockFilterByAddress(addr, OnroadBlocksSubscriptionV2)
}
func (s *SubscribeApi) createUnreceivedBlockFilterByAddress(addr types.Address, ft FilterType) (rpc.ID, error) {
	s.log.Info("createUnreceivedBlockFilterByAddress")
	var (
		acCh  = make(chan []*OnroadMsg)
		acSub = s.eventSystem.SubscribeOnroadBlocksByAddr(addr, acCh, ft)
	)

	s.filterMapMu.Lock()
	s.filterMap[acSub.ID] = &filter{typ: acSub.sub.typ, deadline: time.NewTimer(deadline), s: acSub}
	s.filterMapMu.Unlock()

	go func() {
		for {
			select {
			case ac := <-acCh:
				s.filterMapMu.Lock()
				if f, found := s.filterMap[acSub.ID]; found {
					f.onroadMsgs = append(f.onroadMsgs, ac...)
				}
				s.filterMapMu.Unlock()
			case <-acSub.Err():
				s.filterMapMu.Lock()
				delete(s.filterMap, acSub.ID)
				s.filterMapMu.Unlock()
				return
			}
		}
	}()

	return acSub.ID, nil
}

// Deprecated: use subscribe_createVmLogFilter instead
func (s *SubscribeApi) NewLogsFilter(param RpcFilterParam) (rpc.ID, error) {
	return s.createVmLogFilter(param.AddrRange, param.Topics, LogsSubscription)
}
func (s *SubscribeApi) CreateVmLogFilter(param api.VmLogFilterParam) (rpc.ID, error) {
	return s.createVmLogFilter(param.AddrRange, param.Topics, LogsSubscriptionV2)
}
func (s *SubscribeApi) createVmLogFilter(rangeMap map[string]*api.Range, topics [][]types.Hash, ft FilterType) (rpc.ID, error) {
	s.log.Info("createVmLogFilter")
	p, err := api.ToFilterParam(rangeMap, topics)
	if err != nil {
		return "", err
	}
	var (
		logsCh  = make(chan []*Logs)
		logsSub = s.eventSystem.SubscribeLogs(p, logsCh, ft)
	)

	s.filterMapMu.Lock()
	s.filterMap[logsSub.ID] = &filter{typ: logsSub.sub.typ, deadline: time.NewTimer(deadline), s: logsSub}
	s.filterMapMu.Unlock()

	go func() {
		for {
			select {
			case l := <-logsCh:
				s.filterMapMu.Lock()
				if f, found := s.filterMap[logsSub.ID]; found {
					f.logs = append(f.logs, l...)
				}
				s.filterMapMu.Unlock()
			case <-logsSub.Err():
				s.filterMapMu.Lock()
				delete(s.filterMap, logsSub.ID)
				s.filterMapMu.Unlock()
				return
			}
		}
	}()

	return logsSub.ID, nil
}

func (s *SubscribeApi) UninstallFilter(id rpc.ID) bool {
	s.log.Info("UninstallFilter")
	s.filterMapMu.Lock()
	f, found := s.filterMap[id]
	if found {
		delete(s.filterMap, id)
	}
	s.filterMapMu.Unlock()
	if found {
		f.s.Unsubscribe()
	}
	return found
}

type AccountBlocksMsg struct {
	Blocks []*AccountBlock `json:"result"`
	Id     rpc.ID          `json:"subscription"`
}

type AccountBlocksWithHeightMsg struct {
	Blocks []*AccountBlockWithHeight `json:"result"`
	Id     rpc.ID                    `json:"subscription"`
}
type AccountBlocksWithHeightMsgV2 struct {
	Blocks []*AccountBlockWithHeightV2 `json:"result"`
	Id     rpc.ID                      `json:"subscription"`
}

type LogsMsg struct {
	Logs []*Logs `json:"result"`
	Id   rpc.ID  `json:"subscription"`
}
type LogsMsgV2 struct {
	Logs []*LogsV2 `json:"result"`
	Id   rpc.ID    `json:"subscription"`
}

type OnroadBlocksMsg struct {
	Blocks []*OnroadMsg `json:"result"`
	Id     rpc.ID       `json:"subscription"`
}

type OnroadBlocksMsgV2 struct {
	Blocks []*OnroadMsgV2 `json:"result"`
	Id     rpc.ID         `json:"subscription"`
}

type SnapshotBlocksMsg struct {
	Blocks []*SnapshotBlock `json:"result"`
	Id     rpc.ID           `json:"subscription"`
}
type SnapshotBlocksMsgV2 struct {
	Blocks []*SnapshotBlockV2 `json:"result"`
	Id     rpc.ID             `json:"subscription"`
}

// Deprecated: use subscribe_getChangesByFilterId instead
func (s *SubscribeApi) GetFilterChanges(id rpc.ID) (interface{}, error) {
	return s.getChangesByFilterId(id)
}
func (s *SubscribeApi) GetChangesByFilterId(id rpc.ID) (interface{}, error) {
	return s.getChangesByFilterId(id)
}
func (s *SubscribeApi) getChangesByFilterId(id rpc.ID) (interface{}, error) {
	s.log.Info("getChangesByFilterId", "id", id)
	s.filterMapMu.Lock()
	defer s.filterMapMu.Unlock()

	if f, found := s.filterMap[id]; found {
		if !f.deadline.Stop() {
			<-f.deadline.C
		}
		f.deadline.Reset(deadline)

		switch f.typ {
		case AccountBlocksSubscription:
			blocks := f.blocks
			f.blocks = nil
			return AccountBlocksMsg{blocks, id}, nil
		case AccountBlocksWithHeightSubscription:
			blocks := f.blocksWithHeight
			f.blocksWithHeight = nil
			return AccountBlocksWithHeightMsg{blocks, id}, nil
		case AccountBlocksWithHeightSubscriptionV2:
			blocks := f.blocksWithHeight
			f.blocksWithHeight = nil
			result := make([]*AccountBlockWithHeightV2, len(blocks))
			for i, b := range blocks {
				result[i] = &AccountBlockWithHeightV2{b.Hash, b.HeightStr, b.Removed}
			}
			return AccountBlocksWithHeightMsgV2{result, id}, nil
		case OnroadBlocksSubscription:
			onroadMsgs := f.onroadMsgs
			f.onroadMsgs = nil
			return OnroadBlocksMsg{onroadMsgs, id}, nil
		case OnroadBlocksSubscriptionV2:
			onroadMsgs := f.onroadMsgs
			f.onroadMsgs = nil
			result := make([]*OnroadMsgV2, len(onroadMsgs))
			for i, o := range onroadMsgs {
				result[i] = &OnroadMsgV2{o.Hash, o.Closed, o.Removed}
			}
			return OnroadBlocksMsgV2{result, id}, nil
		case LogsSubscription:
			logs := f.logs
			f.logs = nil
			return LogsMsg{logs, id}, nil
		case LogsSubscriptionV2:
			logs := f.logs
			f.logs = nil
			result := make([]*LogsV2, len(logs))
			for i, l := range logs {
				result[i] = &LogsV2{l.Log, l.AccountBlockHash, l.AccountHeight, l.Addr, l.Removed}
			}
			return LogsMsgV2{result, id}, nil
		case SnapshotBlocksSubscription:
			snapshotBlocks := f.snapshotBlocks
			f.snapshotBlocks = nil
			return SnapshotBlocksMsg{snapshotBlocks, id}, nil
		case SnapshotBlocksSubscriptionV2:
			snapshotBlocks := f.snapshotBlocks
			f.snapshotBlocks = nil
			result := make([]*SnapshotBlockV2, len(snapshotBlocks))
			for i, b := range snapshotBlocks {
				result[i] = &SnapshotBlockV2{b.Hash, b.HeightStr, b.Removed}
			}
			return SnapshotBlocksMsgV2{result, id}, nil
		}
	}

	return nil, errors.New("filter not found")
}

// Deprecated: use subscribe_createSnapshotBlockSubscription instead
func (s *SubscribeApi) NewSnapshotBlocks(ctx context.Context) (*rpc.Subscription, error) {
	return s.createSnapshotBlockSubscription(ctx, SnapshotBlocksSubscription)
}
func (s *SubscribeApi) CreateSnapshotBlockSubscription(ctx context.Context) (*rpc.Subscription, error) {
	return s.createSnapshotBlockSubscription(ctx, SnapshotBlocksSubscriptionV2)
}
func (s *SubscribeApi) createSnapshotBlockSubscription(ctx context.Context, ft FilterType) (*rpc.Subscription, error) {
	s.log.Info("createSnapshotBlockSubscription")
	notifier, supported := rpc.NotifierFromContext(ctx)
	if !supported {
		return &rpc.Subscription{}, rpc.ErrNotificationsUnsupported
	}
	rpcSub := notifier.CreateSubscription()

	go func() {
		snapshotBlockHashChan := make(chan []*SnapshotBlock, 128)
		sbSub := s.eventSystem.SubscribeSnapshotBlocks(snapshotBlockHashChan, ft)
		for {
			select {
			case h := <-snapshotBlockHashChan:
				if ft == SnapshotBlocksSubscriptionV2 {
					result := make([]*SnapshotBlockV2, len(h))
					for i, b := range h {
						result[i] = &SnapshotBlockV2{b.Hash, b.HeightStr, b.Removed}
					}
					notifier.Notify(rpcSub.ID, result)
				} else {
					notifier.Notify(rpcSub.ID, h)
				}
			case <-rpcSub.Err():
				sbSub.Unsubscribe()
				return
			case <-notifier.Closed():
				sbSub.Unsubscribe()
				return
			}
		}
	}()

	return rpcSub, nil
}

// Deprecated: use subscribe_createAccountBlockSubscription instead
func (s *SubscribeApi) NewAccountBlocks(ctx context.Context) (*rpc.Subscription, error) {
	return s.createAccountBlockSubscription(ctx)
}
func (s *SubscribeApi) CreateAccountBlockSubscription(ctx context.Context) (*rpc.Subscription, error) {
	return s.createAccountBlockSubscription(ctx)
}
func (s *SubscribeApi) createAccountBlockSubscription(ctx context.Context) (*rpc.Subscription, error) {
	s.log.Info("createAccountBlockSubscription")
	notifier, supported := rpc.NotifierFromContext(ctx)
	if !supported {
		return &rpc.Subscription{}, rpc.ErrNotificationsUnsupported
	}
	rpcSub := notifier.CreateSubscription()

	go func() {
		accountBlockHashCh := make(chan []*AccountBlock, 128)
		acSub := s.eventSystem.SubscribeAccountBlocks(accountBlockHashCh)
		for {
			select {
			case h := <-accountBlockHashCh:
				notifier.Notify(rpcSub.ID, h)
			case <-rpcSub.Err():
				acSub.Unsubscribe()
				return
			case <-notifier.Closed():
				acSub.Unsubscribe()
				return
			}
		}
	}()

	return rpcSub, nil
}

// Deprecated: use subscribe_createAccountBlockSubscriptionByAddress instead
func (s *SubscribeApi) NewAccountBlocksByAddr(ctx context.Context, addr types.Address) (*rpc.Subscription, error) {
	return s.createAccountBlockSubscriptionByAddress(ctx, addr, AccountBlocksWithHeightSubscription)
}
func (s *SubscribeApi) CreateAccountBlockSubscriptionByAddress(ctx context.Context, addr types.Address) (*rpc.Subscription, error) {
	return s.createAccountBlockSubscriptionByAddress(ctx, addr, AccountBlocksWithHeightSubscriptionV2)
}
func (s *SubscribeApi) createAccountBlockSubscriptionByAddress(ctx context.Context, addr types.Address, ft FilterType) (*rpc.Subscription, error) {
	s.log.Info("createAccountBlockSubscriptionByAddress")
	notifier, supported := rpc.NotifierFromContext(ctx)
	if !supported {
		return &rpc.Subscription{}, rpc.ErrNotificationsUnsupported
	}
	rpcSub := notifier.CreateSubscription()

	go func() {
		accountBlockCh := make(chan []*AccountBlockWithHeight, 128)
		acSub := s.eventSystem.SubscribeAccountBlocksByAddr(addr, accountBlockCh, ft)
		for {
			select {
			case h := <-accountBlockCh:
				if ft == AccountBlocksWithHeightSubscriptionV2 {
					result := make([]*AccountBlockWithHeightV2, len(h))
					for i, b := range h {
						result[i] = &AccountBlockWithHeightV2{b.Hash, b.HeightStr, b.Removed}
					}
					notifier.Notify(rpcSub.ID, result)
				} else {
					notifier.Notify(rpcSub.ID, h)
				}
			case <-rpcSub.Err():
				acSub.Unsubscribe()
				return
			case <-notifier.Closed():
				acSub.Unsubscribe()
				return
			}
		}
	}()

	return rpcSub, nil
}

// Deprecated: use subscribe_createUnreceivedBlockSubscriptionByAddress instead
func (s *SubscribeApi) NewOnroadBlocksByAddr(ctx context.Context, addr types.Address) (*rpc.Subscription, error) {
	return s.createUnreceivedBlockSubscriptionByAddress(ctx, addr, OnroadBlocksSubscription)
}
func (s *SubscribeApi) CreateUnreceivedBlockSubscriptionByAddress(ctx context.Context, addr types.Address) (*rpc.Subscription, error) {
	return s.createUnreceivedBlockSubscriptionByAddress(ctx, addr, OnroadBlocksSubscriptionV2)
}
func (s *SubscribeApi) createUnreceivedBlockSubscriptionByAddress(ctx context.Context, addr types.Address, ft FilterType) (*rpc.Subscription, error) {
	s.log.Info("createUnreceivedBlockSubscriptionByAddress")
	notifier, supported := rpc.NotifierFromContext(ctx)
	if !supported {
		return &rpc.Subscription{}, rpc.ErrNotificationsUnsupported
	}
	rpcSub := notifier.CreateSubscription()

	go func() {
		accountBlockHashCh := make(chan []*OnroadMsg, 128)
		acSub := s.eventSystem.SubscribeOnroadBlocksByAddr(addr, accountBlockHashCh, ft)
		for {
			select {
			case h := <-accountBlockHashCh:
				if ft == OnroadBlocksSubscriptionV2 {
					result := make([]*OnroadMsgV2, len(h))
					for i, o := range h {
						result[i] = &OnroadMsgV2{o.Hash, o.Closed, o.Removed}
					}
					notifier.Notify(rpcSub.ID, result)
				} else {
					notifier.Notify(rpcSub.ID, h)
				}
			case <-rpcSub.Err():
				acSub.Unsubscribe()
				return
			case <-notifier.Closed():
				acSub.Unsubscribe()
				return
			}
		}
	}()

	return rpcSub, nil
}

// Deprevated: use subscribe_createVmLogSubscription instead
func (s *SubscribeApi) NewLogs(ctx context.Context, param RpcFilterParam) (*rpc.Subscription, error) {
	return s.createVmLogSubscription(ctx, param.AddrRange, param.Topics, LogsSubscription)
}
func (s *SubscribeApi) CreateVmlogSubscription(ctx context.Context, param api.VmLogFilterParam) (*rpc.Subscription, error) {
	return s.createVmLogSubscription(ctx, param.AddrRange, param.Topics, LogsSubscriptionV2)
}
func (s *SubscribeApi) createVmLogSubscription(ctx context.Context, rangeMap map[string]*api.Range, topics [][]types.Hash, ft FilterType) (*rpc.Subscription, error) {
	s.log.Info("createVmLogSubscription")
	p, err := api.ToFilterParam(rangeMap, topics)
	if err != nil {
		return nil, err
	}

	notifier, supported := rpc.NotifierFromContext(ctx)
	if !supported {
		return &rpc.Subscription{}, rpc.ErrNotificationsUnsupported
	}
	rpcSub := notifier.CreateSubscription()

	go func() {
		logsMsg := make(chan []*Logs, 128)
		sub := s.eventSystem.SubscribeLogs(p, logsMsg, ft)

		for {
			select {
			case msg := <-logsMsg:
				if ft == LogsSubscriptionV2 {
					result := make([]*LogsV2, len(msg))
					for i, l := range msg {
						result[i] = &LogsV2{l.Log, l.AccountBlockHash, l.AccountHeight, l.Addr, l.Removed}
					}
					notifier.Notify(rpcSub.ID, result)
				} else {
					notifier.Notify(rpcSub.ID, msg)
				}

			case <-rpcSub.Err():
				sub.Unsubscribe()
				return
			case <-notifier.Closed():
				sub.Unsubscribe()
				return
			}
		}
	}()
	return rpcSub, nil
}

// Deprecated: use ledger_getVmLogsByFilter instead
func (s *SubscribeApi) GetLogs(param RpcFilterParam) ([]*Logs, error) {
	logs, err := api.GetLogs(s.vite.Chain(), param.AddrRange, param.Topics)
	if err != nil {
		return nil, err
	}
	resultList := make([]*Logs, len(logs))
	for i, l := range logs {
		resultList[i] = &Logs{l.Log, l.AccountBlockHash, l.AccountHeight, l.Addr, false}
	}
	return resultList, nil
}
