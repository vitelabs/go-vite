package consensus

import (
	"math/big"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
)

// Ensure that all nodes get same result
type teller struct {
	info *membersInfo
	//voteCache map[int32]*electionResult
	voteCache *lru.Cache
	rw        *chainRw
	algo      *algo
	gid       types.Gid

	mLog log15.Logger
}

func newTeller(info *membersInfo, gid types.Gid, rw *chainRw, log log15.Logger) *teller {

	t := &teller{rw: rw}
	//t.info = &membersInfo{genesisTime: genesisTime, memberCnt: memberCnt, interval: interval, perCnt: perCnt, randCnt: 2, LowestLimit: big.NewInt(1000)}
	t.info = info
	t.gid = gid
	t.algo = &algo{info: t.info}
	t.mLog = log.New("gid", gid.String())
	t.mLog.Info("new teller.", "membersInfo", info.String())
	cache, err := lru.New(1024 * 10)
	if err != nil {
		panic(err)
	}
	t.voteCache = cache
	return t
}

func (self *teller) voteResults(b *ledger.SnapshotBlock) ([]types.Address, *ledger.HashHeight, error) {
	head := self.rw.GetLatestSnapshotBlock()

	if b.Height > head.Height {
		return nil, nil, errors.New("rollback happened, block height[" + strconv.FormatUint(b.Height, 10) + "], head height[" + strconv.FormatUint(head.Height, 10) + "]")
	}

	headH := ledger.HashHeight{Height: b.Height, Hash: b.Hash}
	addressList, e := self.calVotes(headH)
	if e != nil {
		return nil, nil, e
	}
	return addressList, &headH, nil
}

func (self *teller) electionIndex(index int32) (*electionResult, error) {
	sTime := self.info.genSTime(index - 1)

	block, e := self.rw.GetSnapshotBeforeTime(sTime)
	if e != nil {
		self.mLog.Error("geSnapshotBeferTime fail.", "err", e)
		return nil, e
	}

	voteResults, hashH, err := self.voteResults(block)
	if err != nil {
		return nil, err
	}

	plans := self.info.genPlan(index, voteResults)
	plans.Hash = hashH.Hash
	plans.Height = hashH.Height
	return plans, nil
}

func (self *teller) electionTime(t time.Time) (*electionResult, error) {
	index := self.info.time2Index(t)
	return self.electionIndex(index)
}
func (self *teller) time2Index(t time.Time) int32 {
	index := self.info.time2Index(t)
	return index
}

func (self *teller) findSeed(votes []*Vote) int64 {
	result := big.NewInt(0)
	for _, v := range votes {
		result.Add(result, v.balance)
	}
	return result.Int64()
}
func (self *teller) convertToAddress(votes []*Vote) []types.Address {
	var result []types.Address
	for _, v := range votes {
		result = append(result, v.addr)
	}
	return result
}
func (self *teller) calVotes(hashH ledger.HashHeight) ([]types.Address, error) {
	// load from cache
	r, ok := self.voteCache.Get(hashH.Hash)
	if ok {
		return r.([]types.Address), nil
	}
	// record vote
	votes, err := self.rw.CalVotes(self.gid, self.info, hashH)
	if err != nil {
		return nil, err
	}
	// filter size of members
	finalVotes := self.algo.filterVotes(votes, &hashH)
	// shuffle the members
	finalVotes = self.algo.shuffleVotes(finalVotes, &hashH)

	address := self.convertToAddress(finalVotes)

	// update cache
	self.voteCache.Add(hashH.Hash, address)
	return address, nil
}

type byBalance []*Vote

func (a byBalance) Len() int      { return len(a) }
func (a byBalance) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a byBalance) Less(i, j int) bool {
	r := a[i].balance.Cmp(a[j].balance)
	if r == 0 {
		return a[i].name < a[j].name
	} else {
		return r < 0
	}
}
