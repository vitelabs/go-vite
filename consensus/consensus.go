package consensus

import (
	"math/big"
	"sync"
	"time"

	"github.com/vitelabs/go-vite/consensus/db"

	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
)

type Verifier interface {
	VerifyAccountProducer(block *ledger.AccountBlock) (bool, error)
	VerifyABsProducer(abs map[types.Gid][]*ledger.AccountBlock) ([]*ledger.AccountBlock, error)
	VerifySnapshotProducer(block *ledger.SnapshotBlock) (bool, error)
}

type Event struct {
	Gid     types.Gid
	Address types.Address
	Stime   time.Time
	Etime   time.Time

	Timestamp         time.Time // add to block
	SnapshotTimeStamp time.Time // add to block

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
	VoteTimeToIndex(gid types.Gid, t2 time.Time) (uint64, error)
	VoteIndexToTime(gid types.Gid, i uint64) (*time.Time, *time.Time, error)
}

type APIReader interface {
	ReadVoteMap(t time.Time) ([]*VoteDetails, *ledger.HashHeight, error)
	ReadSuccessRateForAPI(start, end uint64) ([]map[types.Address]*consensus_db.Content, error)
}

type Life interface {
	Start()
	Init() error
	Stop()
}

type SbpStats struct {
	Index            uint64
	BlockNum         uint64
	ExceptedBlockNum uint64
	VoteCnt          *big.Int
	Name             string
}

type DayStats struct {
	Index uint64
	stats map[string]*SbpStats
	// todo
	VoteSum *big.Int
}
type SBPStatReader interface {
	DayStats(startIndex uint64, endIndex uint64) ([]*DayStats, error)
}

type Consensus interface {
	Verifier
	Subscriber
	Reader
	Life
	API() APIReader
}

// update committee result
type consensus struct {
	common.LifecycleStatus

	mLog log15.Logger

	genesis time.Time

	rw *chainRw

	snapshot  DposReader
	contracts *contractsCs

	dposWrapper *dposReader

	// subscribes map[types.Gid]map[string]*subscribeEvent
	subscribes sync.Map

	api APIReader

	wg     sync.WaitGroup
	closed chan struct{}
}

func (self *consensus) API() APIReader {
	return self.api
}

func NewConsensus(ch Chain) *consensus {
	log := log15.New("module", "consensus")
	rw := newChainRw(ch, log)
	self := &consensus{rw: rw}
	self.mLog = log

	return self
}
