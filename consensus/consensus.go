package consensus

import (
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/consensus/consensus_db"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
)

type Verifier interface {
	VerifyAccountProducer(block *ledger.AccountBlock) (bool, error)
	VerifySnapshotProducer(block *ledger.SnapshotBlock) (bool, error)
}

type Event struct {
	Gid     types.Gid
	Address types.Address
	Stime   time.Time
	Etime   time.Time

	Timestamp         time.Time  // add to block
	SnapshotHash      types.Hash // add to block
	SnapshotHeight    uint64     // add to block
	SnapshotTimeStamp time.Time  // add to block

	VoteTime    time.Time // voteTime
	PeriodStime time.Time // start time for period
	PeriodEtime time.Time // end time for period
}

type ProducersEvent struct {
	Addrs []types.Address
	Index uint64
	Gid   types.Gid
}

type Subscriber interface {
	Subscribe(gid types.Gid, id string, addr *types.Address, fn func(Event))
	UnSubscribe(gid types.Gid, id string)
	SubscribeProducers(gid types.Gid, id string, fn func(event ProducersEvent))
}

type Reader interface {
	ReadByIndex(gid types.Gid, index uint64) ([]*Event, uint64, error)
	ReadByTime(gid types.Gid, t time.Time) ([]*Event, uint64, error)
	ReadVoteMapByTime(gid types.Gid, index uint64) ([]*VoteDetails, *ledger.HashHeight, error)
	ReadVoteMapForAPI(gid types.Gid, t time.Time) ([]*VoteDetails, *ledger.HashHeight, error)
	ReadSuccessRateForAPI(start, end uint64) ([]SBPInfos, error)
	ReadSuccessRate2ForAPI(start, end uint64) ([]SBPInfos, error)
	VoteTimeToIndex(gid types.Gid, t2 time.Time) (uint64, error)
	VoteIndexToTime(gid types.Gid, i uint64) (*time.Time, *time.Time, error)
}

type Life interface {
	Start()
	Init() error
	Stop()
}

type Consensus interface {
	Verifier
	Subscriber
	Reader
	Life
}

// update committee result
type consensus struct {
	common.LifecycleStatus

	mLog log15.Logger

	genesis time.Time

	rw *chainRw

	snapshot  *snapshotCs
	contracts map[types.Gid]*contractsCs

	// subscribes map[types.Gid]map[string]*subscribeEvent
	subscribes sync.Map

	wg     sync.WaitGroup
	closed chan struct{}
}

func NewConsensus(genesisTime time.Time, ch ch) *committee {
	points, err := newPeriodPointArray(ch)
	if err != nil {
		panic(errors.Wrap(err, "create period point fail."))
	}

	rw := &chainRw{rw: ch, periodPoints: points}
	committee := &committee{rw: rw, periods: points, genesis: genesisTime, whiteProducers: make(map[string]bool)}
	committee.mLog = log15.New("module", "consensus/committee")
	db, err := ch.NewDb("consensus")
	if err != nil {
		panic(err)
	}
	committee.dbCache = &DbCache{db: consensus_db.NewConsensusDB(db)}
	cache, err := lru.New(1024 * 10)
	if err != nil {
		panic(err)
	}
	committee.lruCache = cache
	return committee
}
