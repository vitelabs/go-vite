package consensus

import (
	"fmt"
	"math/big"
	"sort"
	"time"

	"github.com/hashicorp/golang-lru"
	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/consensus/core"
	"github.com/vitelabs/go-vite/consensus/db"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
)

type Chain interface {
	GetGenesisSnapshotBlock() *ledger.SnapshotBlock
	GetLatestSnapshotBlock() *ledger.SnapshotBlock

	// todo
	GetConsensusGroupList(snapshotHash types.Hash) ([]*types.ConsensusGroupInfo, error)                                                 // Get all consensus group
	GetRegisterList(snapshotHash types.Hash, gid types.Gid) ([]*types.Registration, error)                                              // Get register for consensus group
	GetVoteList(snapshotHash types.Hash, gid types.Gid) ([]*types.VoteInfo, error)                                                      // Get the candidate's vote
	GetConfirmedBalanceList(addrList []types.Address, tokenId types.TokenTypeId, sbHash types.Hash) (map[types.Address]*big.Int, error) // Get balance for addressList
	GetSnapshotHeaderBeforeTime(timestamp *time.Time) (*ledger.SnapshotBlock, error)

	GetContractMeta(contractAddress types.Address) (meta *ledger.ContractMeta, err error)

	GetSnapshotBlockByHeight(height uint64) (*ledger.SnapshotBlock, error)
	GetSnapshotBlockByHash(hash types.Hash) (*ledger.SnapshotBlock, error)
	GetSnapshotHeadersAfterOrEqualTime(endHashHeight *ledger.HashHeight, startTime *time.Time, producer *types.Address) ([]*ledger.SnapshotBlock, error)
	IsGenesisSnapshotBlock(hash types.Hash) bool
	GetRandomSeed(snapshotHash types.Hash, n int) uint64
	NewDb(dbDir string) (*leveldb.DB, error)
}

type chainRw struct {
	// todo
	genesisTime time.Time
	rw          Chain

	hourPoints   LinkedArray
	dayPoints    LinkedArray
	periodPoints *periodLinkedArray

	dbCache  *consensus_db.ConsensusDB
	lruCache *lru.Cache

	log log15.Logger
}

func newChainRw(rw Chain, log log15.Logger) *chainRw {
	self := &chainRw{rw: rw}

	self.genesisTime = *rw.GetGenesisSnapshotBlock().Timestamp

	db, err := rw.NewDb("consensus")
	if err != nil {
		panic(err)
	}
	self.dbCache = consensus_db.NewConsensusDB(db)
	cache, err := lru.New(1024 * 10)
	if err != nil {
		panic(err)
	}
	self.lruCache = cache
	self.log = log
	return self

}

func (self *chainRw) initArray(cs DposReader) {
	if cs == nil {
		panic("snapshot cs is nil.")
	}
	proof := newRollbackProof(self.rw)
	self.periodPoints = newPeriodPointArray(self.rw, cs, proof, self.log)
	self.hourPoints = newHourLinkedArray(self.periodPoints, self.dbCache, proof, self.genesisTime, self.log)
	self.dayPoints = newDayLinkedArray(self.hourPoints, self.dbCache, proof, self.genesisTime, self.log)
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
	// todo if t < genesisTime, return genesis block
	block, e := self.rw.GetSnapshotHeaderBeforeTime(&t)

	if e != nil {
		return nil, e
	}

	if block == nil {
		return nil, errors.New("before time[" + t.String() + "] block not exist")
	}
	return block, nil
}

func (self *chainRw) GetSeedsBeforeHashH(lastBlock *ledger.SnapshotBlock) uint64 {
	return self.rw.GetRandomSeed(lastBlock.Hash, 25)
}

func (self *chainRw) CalVotes(info *core.GroupInfo, block ledger.HashHeight) ([]*core.Vote, error) {
	return core.CalVotes(info, block, self.rw)
}

func (self *chainRw) CalVoteDetails(gid types.Gid, info *core.GroupInfo, block ledger.HashHeight) ([]*VoteDetails, error) {
	// query register info
	registerList, _ := self.rw.GetRegisterList(block.Hash, gid)
	// query vote info
	votes, _ := self.rw.GetVoteList(block.Hash, gid)

	var registers []*VoteDetails

	// cal candidate
	for _, v := range registerList {
		registers = append(registers, self.genVoteDetails(block.Hash, v, votes, info.CountingTokenId))
	}
	sort.Sort(ByBalance(registers))
	return registers, nil
}

func (self *chainRw) genVoteDetails(snapshotHash types.Hash, registration *types.Registration, infos []*types.VoteInfo, id types.TokenTypeId) *VoteDetails {
	var addrs []types.Address
	for _, v := range infos {
		if v.NodeName == registration.Name {
			addrs = append(addrs, v.VoterAddr)
		}
	}
	balanceMap, _ := self.rw.GetConfirmedBalanceList(addrs, id, snapshotHash)
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

func (self *chainRw) GetMemberInfo(gid types.Gid) (*core.GroupInfo, error) {
	// todo consensus group maybe change ??
	var result *core.GroupInfo
	head := self.rw.GetLatestSnapshotBlock()
	consensusGroupList, err := self.rw.GetConsensusGroupList(head.Hash)
	if err != nil {
		return nil, err
	}
	for _, v := range consensusGroupList {
		if v.Gid == gid {
			result = core.NewGroupInfo(self.genesisTime, *v)
		}
	}
	if result == nil {
		return nil, errors.Errorf("can't get consensus group[%s] info by [%s-%d].", gid, head.Hash, head.Height)
	}

	return result, nil
}

func (self *chainRw) getGid(block *ledger.AccountBlock) (*types.Gid, error) {
	meta, e := self.rw.GetContractMeta(block.AccountAddress)
	if e != nil {
		return nil, e
	}
	return &meta.Gid, nil
}
func (self *chainRw) GetLatestSnapshotBlock() *ledger.SnapshotBlock {
	return self.rw.GetLatestSnapshotBlock()
}
func (self *chainRw) checkSnapshotHashValid(startHeight uint64, startHash types.Hash, actual types.Hash, voteTime time.Time) error {
	if startHash == actual {
		return nil
	}
	startB, e := self.rw.GetSnapshotBlockByHash(startHash)
	if e != nil {
		return e
	}
	if startB == nil {
		return errors.Errorf("start snapshot block is nil. hashH:%s-%d", startHash, startHeight)
	}
	actualB, e := self.rw.GetSnapshotBlockByHash(actual)
	if e != nil {
		return e
	}
	if actualB == nil {
		return errors.Errorf("refer snapshot block is nil. hashH:%s", actual)
	}

	header := self.rw.GetLatestSnapshotBlock()
	if header.Timestamp.Before(voteTime) {
		return errors.Errorf("snapshot header time must >= voteTime, headerTime:%s, voteTime:%s, headerHash:%s:%d", header.Timestamp, voteTime, header.Hash, header.Height)
	}

	if actualB.Height < startB.Height {
		return errors.Errorf("refer snapshot block height must >= start snapshot block height")
	}
	return nil
}

// an hour = 48 * period
func (self *chainRw) GetSuccessRateByHour(index uint64) (map[types.Address]int32, error) {
	result := make(map[types.Address]int32)
	hourInfos := make(map[types.Address]*consensus_db.Content)
	for i := uint64(0); i < hour; i++ {
		if i > index {
			break
		}
		tmpIndex := index - i
		p, err := self.periodPoints.GetByIndex(tmpIndex)
		if err != nil {
			return nil, err
		}
		if p == nil {
			continue
		}
		infos := p.Sbps
		for k, v := range infos {
			c, ok := hourInfos[k]
			if !ok {
				hourInfos[k] = v.Copy()
			} else {
				c.Merge(v)
			}
		}
	}

	for k, v := range hourInfos {
		result[k] = v.Rate()
	}
	return result, nil
}

// an hour = 48 * period
func (self *chainRw) GetSuccessRateByHour2(index uint64) (map[types.Address]*consensus_db.Content, error) {
	hourInfos := make(map[types.Address]*consensus_db.Content)
	for i := uint64(0); i < hour; i++ {
		if i > index {
			break
		}
		tmpIndex := index - i
		p, err := self.periodPoints.GetByIndex(tmpIndex)
		if err != nil {
			return nil, err
		}
		infos := p.Sbps
		for k, v := range infos {
			c, ok := hourInfos[k]
			if !ok {
				hourInfos[k] = v.Copy()
			} else {
				c.Merge(v)
			}
		}
	}

	return hourInfos, nil
}
func (self *chainRw) getSnapshotVoteCache(hashes types.Hash) ([]types.Address, bool) {
	result, b := self.dbCache.GetElectionResultByHash(hashes)
	if b != nil {
		return nil, false
	}
	if result == nil {
		return nil, false
	}
	return result, true
}
func (self *chainRw) updateSnapshotVoteCache(hashes types.Hash, addresses []types.Address) {
	self.dbCache.StoreElectionResultByHash(hashes, addresses)
}

func (self *chainRw) getContractVoteCache(hashes types.Hash) ([]types.Address, bool) {
	if self.lruCache == nil {
		return nil, false
	}
	value, ok := self.lruCache.Get(hashes)
	if ok {
		return value.([]types.Address), ok
	}
	return nil, ok
}

func (self *chainRw) updateContractVoteCache(hashes types.Hash, addrArr []types.Address) {
	if self.lruCache != nil {
		self.log.Info(fmt.Sprintf("store election result %s, %+v\n", hashes, addrArr))
		self.lruCache.Add(hashes, addrArr)
	}
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
