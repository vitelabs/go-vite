package consensus

import (
	"time"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/consensus/core"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
)

type simpleCs struct {
	info *core.GroupInfo
	//voteCache map[int32]*electionResult
	//voteCache *lru.Cache
	algo core.Algo

	log log15.Logger
}

func newSimpleCs(log log15.Logger) *simpleCs {
	cs := &simpleCs{}
	cs.log = log.New("gid", "snapshot")

	cs.info = newSimpleAccountInfo()
	cs.algo = core.NewAlgo(cs.info)
	return cs
}

func newSimpleInfo() *core.GroupInfo {
	t, err := time.Parse("2016-01-02 15:04:05", "2019-03-26 12:00:03")
	if err != nil {
		panic(err)
	}

	group := types.ConsensusGroupInfo{
		Gid:                    types.Gid{},
		NodeCount:              5,
		Interval:               1,
		PerCount:               3,
		RandCount:              1,
		RandRank:               100,
		CountingTokenId:        types.CreateTokenTypeId(),
		RegisterConditionId:    0,
		RegisterConditionParam: nil,
		VoteConditionId:        0,
		VoteConditionParam:     nil,
		Owner:                  types.Address{},
		PledgeAmount:           nil,
		WithdrawHeight:         0,
	}

	info := core.NewGroupInfo(t, group)
	return info
}

func newSimpleAccountInfo() *core.GroupInfo {
	t, err := time.Parse("2016-01-02 15:04:05", "2019-03-26 12:00:03")
	if err != nil {
		panic(err)
	}

	group := types.ConsensusGroupInfo{}

	info := core.NewGroupInfo(t, group)
	return info
}

func (self *simpleCs) time2Index(t time.Time) uint64 {
	return self.info.Time2Index(t)
}

func (self *simpleCs) index2Time(i uint64) (time.Time, time.Time) {
	sTime := self.info.GenSTime(i)
	eTime := self.info.GenETime(i)
	return sTime, eTime
}

func (self *simpleCs) electionTime(t time.Time) (*electionResult, error) {
	index := self.info.Time2Index(t)
	return self.electionIndex(index)
}

func (self *simpleCs) electionIndex(index uint64) (*electionResult, error) {
	var voteResults []types.Address

	addrs := []string{"vite_360232b0378111b122685a15e612143dc9a89cfa7e803f4b5a",
		"vite_ce18b99b46c70c8e6bf34177d0c5db956a8c3ea7040a1c1e25",
		"vite_40996a2ba285ad38930e09a43ee1bd0d84f756f65318e8073a",
		"vite_c8c70c248536117c54d5ffd9724428c58c7fc57f3183508b3d",
		"vite_50f3ede3d3098ae7236f65bb320578ca13dd52516aafc1a10c"}

	for _, v := range addrs {
		addr, err := types.HexToAddress(v)
		if err != nil {
			panic(err)
		}
		voteResults = append(voteResults, addr)
	}

	plans := genElectionResult(self.info, index, voteResults)
	return plans, nil
}

func (self *simpleCs) VerifySnapshotProducer(header *ledger.SnapshotBlock) (bool, error) {
	electionResult, err := self.electionTime(*header.Timestamp)
	if err != nil {
		return false, err
	}

	return self.verifyProducer(*header.Timestamp, header.Producer(), electionResult), nil
}

func (self *simpleCs) verifyProducer(t time.Time, address types.Address, result *electionResult) bool {
	if result == nil {
		return false
	}
	for _, plan := range result.Plans {
		if plan.Member == address {
			if plan.STime == t {
				return true
			}
		}
	}
	return false
}
