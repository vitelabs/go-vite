package api

import (
	"context"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/rpc"
	"github.com/vitelabs/go-vite/rpcapi/api/filters"
	"github.com/vitelabs/go-vite/vite"
	"sync"
	"time"
)

type FilterType byte

const (
	UnknownSubscription FilterType = iota
	LogsSubscription
	// TODO other types
)

type filter struct {
	typ      FilterType
	param    FilterParam
	deadline *time.Timer
	hashes   []types.Hash
	logs     []*ledger.VmLog
}

type SubscribeApi struct {
	vite        *vite.Vite
	log         log15.Logger
	filterMap   map[rpc.ID]*filter
	filterMapMu sync.Mutex
	eventSystem *filters.EventSystem
}

func NewSubscribeApi(vite *vite.Vite) *SubscribeApi {
	return &SubscribeApi{
		vite:        vite,
		log:         log15.New("module", "rpc_api/subscribe_api"),
		filterMap:   make(map[rpc.ID]*filter),
		eventSystem: filters.NewEventSystem(vite),
	}
}

type FilterParam struct {
	FromSnapshotBlockHeight uint64          `json:"fromHeight"`
	ToSnapshotBlockHeight   uint64          `json:"toHeight"`
	AccountAddrList         []types.Address `json:"addrList"`
	Topics                  [][]types.Hash  `json:"topics"`
	AccountHash             *types.Hash     `json:"accountHash"`
	SnapshotHash            *types.Hash     `json:"snapshotHash"`
}

func (s *SubscribeApi) NewAccountBlocks(ctx context.Context) (*rpc.Subscription, error) {
	notifier, supported := rpc.NotifierFromContext(ctx)
	if !supported {
		return &rpc.Subscription{}, rpc.ErrNotificationsUnsupported
	}
	// TODO new go thread to listen to event channel
	rpcSub := notifier.CreateSubscription()
	return rpcSub, nil
}

func (s *SubscribeApi) NewSnapshotBlocks(ctx context.Context) (*rpc.Subscription, error) {
	notifier, supported := rpc.NotifierFromContext(ctx)
	if !supported {
		return &rpc.Subscription{}, rpc.ErrNotificationsUnsupported
	}
	// TODO new go thread to listen to event channel
	rpcSub := notifier.CreateSubscription()
	return rpcSub, nil
}

func (s *SubscribeApi) NewLogs(ctx context.Context, param FilterParam) (*rpc.Subscription, error) {
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

func (s *SubscribeApi) NewSnapshotBlocksFilter() (rpc.ID, error) {
	// TODO
	return "", nil
}

func (s *SubscribeApi) NewLogsFilter() (rpc.ID, error) {
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
