package consensus

import (
	"time"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/consensus/core"
	"github.com/vitelabs/go-vite/ledger"
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

	Timestamp      time.Time  // add to block
	SnapshotHash   types.Hash // add to block
	SnapshotHeight uint64     // add to block

	VoteTime time.Time // voteTime
}

type electionResult struct {
	Plans  []*core.MemberPlan
	STime  time.Time
	ETime  time.Time
	Index  uint64
	Hash   types.Hash
	Height uint64
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
