package consensus

import (
	"math/big"
	"time"

	"github.com/vitelabs/go-vite/common/fork"

	"sort"

	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/common/fork"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/consensus/core"
	"github.com/vitelabs/go-vite/ledger"
)

type ch interface {
	GetLatestSnapshotBlock() *ledger.SnapshotBlock

	GetConsensusGroupList(snapshotHash types.Hash) ([]*types.ConsensusGroupInfo, error)                                                     // Get all consensus group
	GetRegisterList(snapshotHash types.Hash, gid types.Gid) ([]*types.Registration, error)                                                  // Get register for consensus group
	GetVoteMap(snapshotHash types.Hash, gid types.Gid) ([]*types.VoteInfo, error)                                                           // Get the candidate's vote
	GetBalanceList(snapshotHash types.Hash, tokenTypeId types.TokenTypeId, addressList []types.Address) (map[types.Address]*big.Int, error) // Get balance for addressList
	GetSnapshotBlockBeforeTime(timestamp *time.Time) (*ledger.SnapshotBlock, error)

	GetContractGidByAccountBlock(block *ledger.AccountBlock) (*types.Gid, error)
	GetSnapshotBlockByHeight(height uint64) (*ledger.SnapshotBlock, error)
	GetSnapshotBlockByHash(hash *types.Hash) (*ledger.SnapshotBlock, error)
	GetSnapshotBlocksAfterAndEqualTime(endHeight uint64, startTime *time.Time, producer *types.Address) ([]*ledger.SnapshotBlock, error)
	IsGenesisSnapshotBlock(block *ledger.SnapshotBlock) bool
	NewDb(dbDir string) (*leveldb.DB, error)
}

type chainRw struct {
	rw ch

	hourPoints   PointLinkedArray
	dayPoints    PointLinkedArray
	periodPoints PointLinkedArray
}

type VoteDetails struct {
	core.Vote
	CurrentAddr  types.Address
	RegisterList []types.Address
	Addr         map[types.Address]*big.Int
}
type ByBalance []*VoteDetails

func (a ByBalance) Len() int      { return len(a) }
func (a ByBalance) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByBalance) Less(i, j int) bool {

	r := a[j].Balance.Cmp(a[i].Balance)
	if r == 0 {
		return a[i].Name < a[j].Name
	} else {
		return r < 0
	}
}

func (self *chainRw) GetSnapshotBeforeTime(t time.Time) (*ledger.SnapshotBlock, error) {
	block, e := self.rw.GetSnapshotBlockBeforeTime(&t)

	if e != nil {
		return nil, e
	}

	if block == nil {
		return nil, errors.New("before time[" + t.String() + "] block not exist")
	}
	return block, nil
}

func (self *chainRw) GetSeedsBeforeHashH(lastBlock *ledger.SnapshotBlock, dur time.Duration) (map[types.Address]uint64, error) {
	startT := lastBlock.Timestamp.Add(-dur)
	//blocks, err := self.rw.GetSnapshotBlocksAfterAndEqualTime(&ledger.HashHeight{Hash: lastBlock.Hash, Height: lastBlock.Height}, &startT)
	blocks, err := self.rw.GetSnapshotBlocksAfterAndEqualTime(lastBlock.Height, &startT, nil)
	if err != nil {
		return nil, err
	}
	snapshots, _ := self.groupSnapshotBySeedExist(blocks)

	m := make(map[types.Address][]*ledger.SnapshotBlock)
	for _, block := range snapshots {
		if block.SeedHash == nil {
			continue
		}
		producer := block.Producer()
		bs, ok := m[producer]
		if len(bs) >= 2 {
			continue
		}
		if ok {
			m[producer] = append(m[producer], block)
		} else {
			var arr []*ledger.SnapshotBlock
			arr = append(arr, block)
			m[producer] = arr
		}
	}

	seedM := make(map[types.Address]uint64)
	for k, v := range m {
		if len(v) != 2 {
			continue
		}

		seed := self.getSeed(v[0], v[1])
		if seed != 0 {
			seedM[k] = seed
		}
	}

	return seedM, nil
}

func (self *chainRw) CalVotes(info *core.GroupInfo, block ledger.HashHeight) ([]*core.Vote, error) {
	return core.CalVotes(info, block, self.rw)
}

func (self *chainRw) CalVoteDetails(gid types.Gid, info *core.GroupInfo, block ledger.HashHeight) ([]*VoteDetails, error) {
	// query register info
	registerList, _ := self.rw.GetRegisterList(block.Hash, gid)
	// query vote info
	votes, _ := self.rw.GetVoteMap(block.Hash, gid)

	var registers []*VoteDetails

	// cal candidate
	for _, v := range registerList {
		registers = append(registers, self.GenVoteDetails(block.Hash, v, votes, info.CountingTokenId))
	}
	sort.Sort(ByBalance(registers))
	return registers, nil
}

func (self *chainRw) GenVoteDetails(snapshotHash types.Hash, registration *types.Registration, infos []*types.VoteInfo, id types.TokenTypeId) *VoteDetails {
	var addrs []types.Address
	for _, v := range infos {
		if v.NodeName == registration.Name {
			addrs = append(addrs, v.VoterAddr)
		}
	}
	balanceMap, _ := self.rw.GetBalanceList(snapshotHash, id, addrs)
	balanceTotal := big.NewInt(0)
	for _, v := range balanceMap {
		balanceTotal.Add(balanceTotal, v)
	}
	return &VoteDetails{
		Vote: core.Vote{
			Name:    registration.Name,
			Addr:    registration.NodeAddr,
			Balance: balanceTotal,
		},
		CurrentAddr:  registration.NodeAddr,
		RegisterList: registration.HisAddrList,
		Addr:         balanceMap,
	}
}

func (self *chainRw) GetMemberInfo(gid types.Gid, genesis time.Time) *core.GroupInfo {
	// todo consensus group maybe change ??
	var result *core.GroupInfo
	head := self.rw.GetLatestSnapshotBlock()
	consensusGroupList, _ := self.rw.GetConsensusGroupList(head.Hash)
	for _, v := range consensusGroupList {
		if v.Gid == gid {
			result = core.NewGroupInfo(genesis, *v)
		}
	}

	return result
}

func (self *chainRw) getGid(block *ledger.AccountBlock) (types.Gid, error) {
	gid, e := self.rw.GetContractGidByAccountBlock(block)
	return *gid, e
}
func (self *chainRw) GetLatestSnapshotBlock() *ledger.SnapshotBlock {
	return self.rw.GetLatestSnapshotBlock()
}
func (self *chainRw) checkSnapshotHashValid(startHeight uint64, startHash types.Hash, actual types.Hash, voteTime time.Time) error {
	if startHash == actual {
		return nil
	}
	startB, e := self.rw.GetSnapshotBlockByHash(&startHash)
	if e != nil {
		return e
	}
	if startB == nil {
		return errors.Errorf("start snapshot block is nil. hashH:%s-%d", startHash, startHeight)
	}
	actualB, e := self.rw.GetSnapshotBlockByHash(&actual)
	if e != nil {
		return e
	}
	if actualB == nil {
		return errors.Errorf("refer snapshot block is nil. hashH:%s", actual)
	}
	if fork.IsMintFork(actualB.Height) {
		header := self.rw.GetLatestSnapshotBlock()
		if header.Timestamp.Before(voteTime) {
			return errors.Errorf("snapshot header time must >= voteTime, headerTime:%s, voteTime:%s, headerHash:%s:%d", header.Timestamp, voteTime, header.Hash, header.Height)
		}
	}

	block, _ := self.rw.GetSnapshotBlockByHeight(actualB.Height + 1)
	if block != nil {
		if block.Timestamp.Before(voteTime) {
			return errors.Errorf("refer snapshot do not match first one before voteTime.")
		}
	}

	if actualB.Height < startB.Height {
		return errors.Errorf("refer snapshot block height must >= start snapshot block height")
	}
	return nil
}

func (self *chainRw) groupSnapshotBySeedExist(blocks []*ledger.SnapshotBlock) ([]*ledger.SnapshotBlock, []*ledger.SnapshotBlock) {
	var exists []*ledger.SnapshotBlock
	var notExists []*ledger.SnapshotBlock

	for _, v := range blocks {
		if v.SeedHash != nil {
			exists = append(exists, v)
		} else {
			notExists = append(notExists, v)
		}
	}

	return exists, notExists
}

func (self *chainRw) getSeed(top *ledger.SnapshotBlock, prev *ledger.SnapshotBlock) uint64 {
	seedHash := prev.SeedHash
	if seedHash == nil {
		return 0
	}
	expectedSeedHash := ledger.ComputeSeedHash(top.Seed, prev.PrevHash, prev.Timestamp)
	if expectedSeedHash == *seedHash {
		return prev.Seed
	}
	return 0
}

// an hour = 48 * period
func (self *chainRw) GetSuccessRateByHour(index uint64) (map[types.Address]int32, error) {
	result := make(map[types.Address]int32)
	hourInfos := NewSBPInfos()
	for i := uint64(0); i < hour; i++ {
		if i > index {
			break
		}
		tmpIndex := index - i
		p, err := self.periodPoints.GetByHeight(tmpIndex)
		if err != nil {
			return nil, err
		}
		point := p.(*periodPoint)
		infos := point.GetSBPInfos()
		for k, v := range infos {
			hourInfos.Get(k).AddNum(v.ExpectedNum, v.FactualNum)
		}
	}

	for k, v := range hourInfos {
		result[k] = v.Rate()
	}
	return result, nil
}

// an hour = 48 * period
func (self *chainRw) GetSuccessRateByHour2(index uint64) (SBPInfos, error) {
	hourInfos := NewSBPInfos()
	for i := uint64(0); i < hour; i++ {
		if i > index {
			break
		}
		tmpIndex := index - i
		p, err := self.periodPoints.GetByHeight(tmpIndex)
		if err != nil {
			return nil, err
		}
		point := p.(*periodPoint)
		infos := point.GetSBPInfos()
		for k, v := range infos {
			hourInfos.Get(k).AddNum(v.ExpectedNum, v.FactualNum)
		}
	}

	return hourInfos, nil
}

// a day = 23 * hour + LatestHour
//func (self *chainRw) GetSuccessRateByDay(index uint64) (map[types.Address]*big.Int, error) {
//	dayInfos := NewSBPInfos()
//	startIndex := uint64(0)
//	endIndex := index
//	if day <= index {
//		startIndex = index - (day - 1)
//	}
//	// [startIndex, endIndex]
//	points := self.genPoints(startIndex, endIndex)
//
//	for _, p := range points {
//		switch p.(type) {
//		case *dayPoint:
//			break
//		case *hourPoint:
//			height, err := self.hourPoints.GetByHeight(p.Height())
//			if err != nil {
//				return nil, err
//			}
//			infos := height.(*hourPoint).GetSBPInfos()
//			for k, v := range infos {
//				dayInfos.Get(k).AddNum(v.ExpectedNum, v.FactualNum)
//			}
//		case *periodPoint:
//			height, err := self.periodPoints.GetByHeight(p.Height())
//			if err != nil {
//				return nil, err
//			}
//			infos := height.(*periodPoint).GetSBPInfos()
//			for k, v := range infos {
//				dayInfos.Get(k).AddNum(v.ExpectedNum, v.FactualNum)
//			}
//		}
//	}
//	// todo
//	return nil, nil
//}
//
//// [startIndex, endIndex]
//func (self *chainRw) genPoints(startIndex uint64, endIndex uint64) []Point {
//	return nil
//}

const (
	period = 1
	hour   = 48 * period
	day    = 24 * hour
)
