package consensus

import (
	"context"
	"sync"
	"time"

	"github.com/vitelabs/go-vite/v2/common"
	"github.com/vitelabs/go-vite/v2/common/types"
	"github.com/vitelabs/go-vite/v2/interfaces"
	ledger "github.com/vitelabs/go-vite/v2/interfaces/core"
	"github.com/vitelabs/go-vite/v2/ledger/consensus/cdb"
	"github.com/vitelabs/go-vite/v2/ledger/consensus/core"
	"github.com/vitelabs/go-vite/v2/ledger/pool/lock"
	"github.com/vitelabs/go-vite/v2/log15"
)

// Event will trigger when the snapshot block needs production
type Event struct {
	Gid     types.Gid
	Address types.Address
	Stime   time.Time
	Etime   time.Time

	Timestamp time.Time // add to block

	VoteTime    time.Time // voteTime
	PeriodStime time.Time // start time for period
	PeriodEtime time.Time // end time for period
}

// ProducersEvent describes all SBP in one period
type ProducersEvent struct {
	Addrs []types.Address
	Index uint64
	Gid   types.Gid
}

// Subscriber provide an interface to consensus event
type Subscriber interface {
	Subscribe(gid types.Gid, id string, addr *types.Address, fn func(Event))
	UnSubscribe(gid types.Gid, id string)
	SubscribeProducers(gid types.Gid, id string, fn func(event ProducersEvent))
	TriggerMineEvent(addr types.Address) error
}

// Reader can read consensus result
type Reader interface {
	ReadByIndex(gid types.Gid, index uint64) ([]*Event, uint64, error)
	VoteTimeToIndex(gid types.Gid, t2 time.Time) (uint64, error)
	VoteIndexToTime(gid types.Gid, i uint64) (*time.Time, *time.Time, error)
}

// APIReader is just provided for RPC api
type APIReader interface {
	ReadVoteMap(t time.Time) ([]*VoteDetails, *ledger.HashHeight, error)
	ReadSuccessRate(start, end uint64) ([]map[types.Address]*cdb.Content, error)
	ReadByIndex(gid types.Gid, index uint64) ([]*Event, uint64, error)
}

// Life define the life cycle for consensus component
type Life interface {
	Start()
	Init(cfg *ConsensusCfg) error
	Stop()
}

// Consensus include all interface for consensus
type Consensus interface {
	interfaces.ConsensusVerifier
	Subscriber
	Reader
	Life
	API() APIReader
	SBPReader() core.SBPStatReader
}

// update committee result
type consensus struct {
	*ConsensusCfg
	Subscriber
	subscribeTrigger

	common.LifecycleStatus

	mLog log15.Logger

	genesis time.Time

	rw       *chainRw
	rollback lock.ChainRollback

	snapshot  *snapshotCs
	contracts *contractsCs

	dposWrapper *dposReader

	api APIReader

	wg     sync.WaitGroup
	closed chan struct{}

	ctx      context.Context
	cancelFn context.CancelFunc

	tg *trigger
}

func (cs *consensus) SBPReader() core.SBPStatReader {
	return cs.snapshot
}

func (cs *consensus) API() APIReader {
	return cs.api
}

// NewConsensus instantiates a new consensus object
func NewConsensus(ch Chain, rollback lock.ChainRollback) Consensus {
	log := log15.New("module", "consensus")
	rw := newChainRw(ch, log, rollback)

	sub := newConsensusSubscriber()
	cs := &consensus{rw: rw, rollback: rollback}
	cs.mLog = log
	cs.Subscriber = sub
	cs.subscribeTrigger = sub
	cs.snapshot = newSnapshotCs(cs.rw, cs.mLog)
	cs.contracts = newContractCs(cs.rw, cs.mLog)
	cs.dposWrapper = &dposReader{cs.snapshot, cs.contracts, cs.mLog}
	cs.api = &APISnapshot{snapshot: cs.snapshot}

	return cs
}
