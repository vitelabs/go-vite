package filters

import (
	"context"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/rpc"
	"github.com/vitelabs/go-vite/vite"
	"sync"
	"time"
)

type filter struct {
	typ      FilterType
	deadline *time.Timer
	hashes   []types.Hash
	logs     []*ledger.VmLog
	param    FilterParam
	s        *RpcSubscription
}

type SubscribeApi struct {
	vite        *vite.Vite
	log         log15.Logger
	filterMap   map[rpc.ID]*filter
	filterMapMu sync.Mutex
	eventSystem *EventSystem
}

func NewSubscribeApi(vite *vite.Vite) *SubscribeApi {
	s := &SubscribeApi{
		vite:        vite,
		log:         log15.New("module", "rpc_api/subscribe_api"),
		filterMap:   make(map[rpc.ID]*filter),
		eventSystem: NewEventSystem(vite),
	}
	go s.timeoutLoop()
	return s
}

func (s *SubscribeApi) timeoutLoop() {
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

type FilterParam struct {
	FromBlock       string          `json:"fromBlock"`
	ToBlock         string          `json:"toBlock"`
	AccountAddrList []types.Address `json:"addrList"`
	Topics          [][]types.Hash  `json:"topics"`
	AccountHash     *types.Hash     `json:"accountHash"`
	SnapshotHash    *types.Hash     `json:"snapshotHash"`
}

func (s *SubscribeApi) AccountBlocks(ctx context.Context) (*rpc.Subscription, error) {
	notifier, supported := rpc.NotifierFromContext(ctx)
	if !supported {
		return &rpc.Subscription{}, rpc.ErrNotificationsUnsupported
	}
	rpcSub := notifier.CreateSubscription()

	go func() {
		accountBlockHashCh := make(chan []*AccountBlockMsg, 128)
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

func (s *SubscribeApi) ConfirmedAccountBlocks(ctx context.Context) (*rpc.Subscription, error) {
	notifier, supported := rpc.NotifierFromContext(ctx)
	if !supported {
		return &rpc.Subscription{}, rpc.ErrNotificationsUnsupported
	}
	rpcSub := notifier.CreateSubscription()

	go func() {
		acMsg := make(chan []*AccountBlockMsg, 128)
		sub := s.eventSystem.SubscribeAccountBlocks(acMsg)

		for {
			select {
			case msg := <-acMsg:
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

func (s *SubscribeApi) Logs(ctx context.Context, param FilterParam) (*rpc.Subscription, error) {
	notifier, supported := rpc.NotifierFromContext(ctx)
	if !supported {
		return &rpc.Subscription{}, rpc.ErrNotificationsUnsupported
	}
	// TODO create new event channel and subscribe, new go thread to listen to event channel
	rpcSub := notifier.CreateSubscription()
	return rpcSub, nil
}

func (s *SubscribeApi) GetLogs(ctx context.Context, param FilterParam) ([]*ledger.VmLog, error) {
	// TODO
	return nil, nil
}

func (s *SubscribeApi) GetFilterLogs(ctx context.Context, ID rpc.ID) ([]*ledger.VmLog, error) {
	// TODO
	return nil, nil
}

func (s *SubscribeApi) NewFilter(param FilterParam) (rpc.ID, error) {
	// TODO
	return "", nil
}

func (s *SubscribeApi) NewAccountBlocksFilter() (rpc.ID, error) {
	// TODO
	return "", nil
}

func (s *SubscribeApi) NewConfirmedAccountBlocksFilter() (rpc.ID, error) {
	// TODO
	return "", nil
}

func (s *SubscribeApi) NewLogsFilter() (rpc.ID, error) {
	// TODO
	return "", nil
}

func (s *SubscribeApi) NewConfirmedLogsFilter() (rpc.ID, error) {
	// TODO
	return "", nil
}

func (s *SubscribeApi) UninstallFilter(id rpc.ID) bool {
	// TODO
	return false
}

func (s *SubscribeApi) GetFilterChanges(id rpc.ID) (interface{}, error) {
	// TODO
	return "", nil
}
