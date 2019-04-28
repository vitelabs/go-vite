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
	typ            FilterType
	deadline       *time.Timer
	param          filterParam
	s              *RpcSubscription
	blocks         []*AccountBlock
	logs           []*Logs
	snapshotBlocks []*SnapshotBlock
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

type Range struct {
	FromHeight string `json:"fromHeight"`
	ToHeight   string `json:"toHeight"`
}

func (r *Range) toHeightRange() (*heightRange, error) {
	if r != nil {
		fromHeight, err := api.StringToUint64(r.FromHeight)
		if err != nil {
			return nil, err
		}
		toHeight, err := api.StringToUint64(r.ToHeight)
		if err != nil {
			return nil, err
		}
		if toHeight < fromHeight {
			return nil, errors.New("to height < from height")
		}
		return &heightRange{fromHeight, toHeight}, nil
	}
	return nil, nil
}

type RpcFilterParam struct {
	AddrRange map[string]*Range `json:"addrRange"`
	Topics    [][]types.Hash    `json:"topics"`
}

func (p *RpcFilterParam) toFilterParam() (*filterParam, error) {
	var addrRange map[types.Address]heightRange
	if len(p.AddrRange) == 0 {
		return nil, errors.New("addrRange is nil")
	}
	addrRange = make(map[types.Address]heightRange, len(p.AddrRange))
	for hexAddr, r := range p.AddrRange {
		hr, err := r.toHeightRange()
		if err != nil {
			return nil, err
		}
		if hr == nil {
			hr = &heightRange{0, 0}
		}
		addr, err := types.HexToAddress(hexAddr)
		if err != nil {
			return nil, err
		}
		addrRange[addr] = *hr
	}
	target := &filterParam{
		addrRange: addrRange,
		topics:    p.Topics,
	}
	return target, nil
}

type AccountBlock struct {
	Hash    types.Hash `json:"hash"`
	Removed bool       `json:"removed"`
}

type SnapshotBlock struct {
	Hash    types.Hash `json:"hash"`
	Height  uint64     `json:"height"`
	Removed bool       `json:"removed"`
}

type Logs struct {
	Log              *ledger.VmLog  `json:"log"`
	AccountBlockHash types.Hash     `json:"accountBlockHash"`
	Addr             *types.Address `json:"addr"`
	Removed          bool           `json:"removed"`
}

func (s *SubscribeApi) NewSnapshotBlocksFilter() (rpc.ID, error) {
	s.log.Info("NewSnapshotBlocksFilter")
	var (
		sbCh  = make(chan []*SnapshotBlock)
		sbSub = s.eventSystem.SubscribeSnapshotBlocks(sbCh)
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

func (s *SubscribeApi) NewAccountBlocksFilter() (rpc.ID, error) {
	s.log.Info("NewAccountBlocksFilter")
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

func (s *SubscribeApi) NewLogsFilter(param RpcFilterParam) (rpc.ID, error) {
	s.log.Info("NewLogsFilter")
	p, err := param.toFilterParam()
	if err != nil {
		return "", err
	}
	var (
		logsCh  = make(chan []*Logs)
		logsSub = s.eventSystem.SubscribeLogs(p, logsCh)
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

type LogsMsg struct {
	Logs []*Logs `json:"result"`
	Id   rpc.ID  `json:"subscription"`
}

type SnapshotBlocksMsg struct {
	Blocks []*SnapshotBlock `json:"result"`
	Id     rpc.ID           `json:"subscription"`
}

func (s *SubscribeApi) GetFilterChanges(id rpc.ID) (interface{}, error) {
	s.log.Info("GetFilterChanges", "id", id)
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
		case LogsSubscription:
			logs := f.logs
			f.logs = nil
			return LogsMsg{logs, id}, nil
		case SnapshotBlocksSubscription:
			snapshotBlocks := f.snapshotBlocks
			f.snapshotBlocks = nil
			return SnapshotBlocksMsg{snapshotBlocks, id}, nil
		}
	}

	return nil, errors.New("filter not found")
}

func (s *SubscribeApi) NewSnapshotBlocks(ctx context.Context) (*rpc.Subscription, error) {
	s.log.Info("NewSnapshotBlocks")
	notifier, supported := rpc.NotifierFromContext(ctx)
	if !supported {
		return &rpc.Subscription{}, rpc.ErrNotificationsUnsupported
	}
	rpcSub := notifier.CreateSubscription()

	go func() {
		snapshotBlockHashChan := make(chan []*SnapshotBlock, 128)
		sbSub := s.eventSystem.SubscribeSnapshotBlocks(snapshotBlockHashChan)
		for {
			select {
			case h := <-snapshotBlockHashChan:
				notifier.Notify(rpcSub.ID, h)
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

func (s *SubscribeApi) NewAccountBlocks(ctx context.Context) (*rpc.Subscription, error) {
	s.log.Info("NewAccountBlocks")
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

func (s *SubscribeApi) NewLogs(ctx context.Context, param RpcFilterParam) (*rpc.Subscription, error) {
	s.log.Info("NewLogs")
	p, err := param.toFilterParam()
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
		sub := s.eventSystem.SubscribeLogs(p, logsMsg)

		for {
			select {
			case msg := <-logsMsg:
				notifier.Notify(rpcSub.ID, msg)
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

var getAccountBlocksCount uint64 = 100

func (s *SubscribeApi) GetLogs(param RpcFilterParam) ([]*Logs, error) {
	filterParam, err := param.toFilterParam()
	if err != nil {
		return nil, err
	}
	var logs []*Logs
	for addr, hr := range filterParam.addrRange {
		startHeight := hr.fromHeight
		endHeight := hr.toHeight
		acc, err := s.vite.Chain().GetLatestAccountBlock(addr)
		if err != nil {
			return nil, err
		}
		if acc == nil {
			continue
		}
		if endHeight == 0 || endHeight > acc.Height {
			endHeight = acc.Height
		}
		for {
			end, count, finish := getHeightPage(startHeight, endHeight, getAccountBlocksCount)
			if count == 0 {
				break
			}
			blocks, err := s.vite.Chain().GetAccountBlocksByHeight(addr, end, count)
			if err != nil {
				return nil, err
			}
			for _, b := range blocks {
				if b.LogHash != nil {
					list, err := s.vite.Chain().GetVmLogList(b.LogHash)
					if err != nil {
						return nil, err
					}
					for _, l := range list {
						if filterLog(filterParam, l) {
							logs = append(logs, &Logs{l, b.Hash, &addr, false})
						}
					}
				}
			}
			if finish {
				break
			}
			startHeight = startHeight + count
		}
	}
	return logs, nil
}

func getHeightPage(start uint64, end uint64, count uint64) (uint64, uint64, bool) {
	if end < count || end-count <= start {
		return end, end - start, true
	}
	return end, count, false
}
