package consensus

import (
	"fmt"
	"math/big"
	"sort"
	"sync"
	"time"

	"github.com/vitelabs/go-vite/pool/lock"

	"github.com/vitelabs/go-vite/chain"

	"github.com/hashicorp/golang-lru"
	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/consensus/core"
	"github.com/vitelabs/go-vite/consensus/db"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
)

// Chain refer to chain.Chain
type Chain interface {
	GetGenesisSnapshotBlock() *ledger.SnapshotBlock
	GetLatestSnapshotBlock() *ledger.SnapshotBlock

	/*
	*	Event Manager
	 */
	Register(listener chain.EventListener)
	UnRegister(listener chain.EventListener)

	// todo
	GetConsensusGroupList(snapshotHash types.Hash) ([]*types.ConsensusGroupInfo, error)                                                 // Get all consensus group
	GetRegisterList(snapshotHash types.Hash, gid types.Gid) ([]*types.Registration, error)                                              // Get register for consensus group
	GetVoteList(snapshotHash types.Hash, gid types.Gid) ([]*types.VoteInfo, error)                                                      // Get the candidate's vote
	GetConfirmedBalanceList(addrList []types.Address, tokenID types.TokenTypeId, sbHash types.Hash) (map[types.Address]*big.Int, error) // Get balance for addressList
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
	genesisTime  time.Time
	rw           Chain
	rollbackLock lock.ChainRollback

	hourPoints   LinkedArray
	dayPoints    LinkedArray
	periodPoints *periodLinkedArray

	dbCache  *consensus_db.ConsensusDB
	lruCache *lru.Cache

	started chan struct{}
	wg      sync.WaitGroup

	log log15.Logger
}

func newChainRw(rw Chain, log log15.Logger, rollbackLock lock.ChainRollback) *chainRw {
	cRw := &chainRw{rw: rw}

	cRw.genesisTime = *rw.GetGenesisSnapshotBlock().Timestamp

	db, err := rw.NewDb("consensus")
	if err != nil {
		panic(err)
	}
	cRw.dbCache = consensus_db.NewConsensusDB(db)
	cache, err := lru.New(1024 * 10)
	if err != nil {
		panic(err)
	}
	cRw.lruCache = cache
	cRw.log = log
	cRw.rollbackLock = rollbackLock
	return cRw

}

func (cRw *chainRw) initArray(cs *snapshotCs) {
	if cs == nil {
		panic("snapshot cs is nil.")
	}
	proof := newRollbackProof(cRw.rw)
	cRw.periodPoints = newPeriodPointArray(cRw.rw, cs, proof, cRw.log)
	cRw.hourPoints = newHourLinkedArray(cRw.periodPoints, cRw.dbCache, proof, time.Duration(cs.GetInfo().PlanInterval)*time.Second, cRw.genesisTime, cRw.log)
	cRw.dayPoints = newDayLinkedArray(cRw.hourPoints, cRw.dbCache, proof, cs.dayVoteStat, cRw.genesisTime, cRw.log)
}

func (cRw *chainRw) Start() error {
	cRw.started = make(chan struct{})

	// todo register chain
	go func() {
		cRw.wg.Add(1)
		defer cRw.wg.Done()
		startedCh := cRw.started
		ticker := time.NewTicker(time.Second * 30)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				block := cRw.GetLatestSnapshotBlock()
				t := block.Timestamp

				index := cRw.dayPoints.Time2Index(*t)
				point, err := cRw.dayPoints.GetByIndex(index)
				if err != nil {
					cRw.log.Error("can't get day info by index", "index", index, "time", t)
				} else {
					cRw.log.Info("get day by info index", "index", index, "time", t, "point", point.Json())
				}
			case <-startedCh:
				return
			}
		}
	}()
	return nil

}

func (cRw *chainRw) Stop() error {
	// todo register chain
	close(cRw.started)
	cRw.wg.Wait()
	return nil
}

// VoteDetails is an extension for core.Vote
type VoteDetails struct {
	core.Vote
	CurrentAddr  types.Address
	RegisterList []types.Address
	Addr         map[types.Address]*big.Int
}

// ByBalance sorts a slice of VoteDetails in decreasing order.
// Sorts by name for equal balance
type ByBalance []*VoteDetails

func (a ByBalance) Len() int      { return len(a) }
func (a ByBalance) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByBalance) Less(i, j int) bool {

	r := a[j].Balance.Cmp(a[i].Balance)
	if r == 0 {
		return a[i].Name < a[j].Name
	}
	return r < 0
}

func (cRw *chainRw) GetSnapshotBeforeTime(t time.Time) (*ledger.SnapshotBlock, error) {
	// todo if t < genesisTime, return genesis block
	block, e := cRw.rw.GetSnapshotHeaderBeforeTime(&t)

	if e != nil {
		return nil, e
	}

	if block == nil {
		return nil, errors.New("before time[" + t.String() + "] block not exist")
	}
	return block, nil
}

func (cRw *chainRw) GetSeedsBeforeHashH(hash types.Hash) uint64 {
	return cRw.rw.GetRandomSeed(hash, 25)
}

func (cRw *chainRw) CalVotes(info *core.GroupInfo, hashH ledger.HashHeight) ([]*core.Vote, error) {
	cRw.rollbackLock.RLockRollback()
	defer cRw.rollbackLock.RUnLockRollback()
	return core.CalVotes(info.ConsensusGroupInfo, hashH.Hash, cRw.rw)
}

func (cRw *chainRw) CalVoteDetails(gid types.Gid, info *core.GroupInfo, block ledger.HashHeight) ([]*VoteDetails, error) {
	// query register info
	registerList, _ := cRw.rw.GetRegisterList(block.Hash, gid)
	// query vote info
	votes, _ := cRw.rw.GetVoteList(block.Hash, gid)

	var registers []*VoteDetails

	// cal candidate
	for _, v := range registerList {
		registers = append(registers, cRw.genVoteDetails(block.Hash, v, votes, info.CountingTokenId))
	}
	sort.Sort(ByBalance(registers))
	return registers, nil
}

func (cRw *chainRw) genVoteDetails(snapshotHash types.Hash, registration *types.Registration, infos []*types.VoteInfo, id types.TokenTypeId) *VoteDetails {
	var addrs []types.Address
	for _, v := range infos {
		if v.NodeName == registration.Name {
			addrs = append(addrs, v.VoterAddr)
		}
	}
	balanceMap, _ := cRw.rw.GetConfirmedBalanceList(addrs, id, snapshotHash)
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

func (cRw *chainRw) GetMemberInfo(gid types.Gid) (*core.GroupInfo, error) {
	// todo consensus group maybe change ??
	var result *core.GroupInfo
	head := cRw.rw.GetLatestSnapshotBlock()
	consensusGroupList, err := cRw.rw.GetConsensusGroupList(head.Hash)
	if err != nil {
		return nil, err
	}
	for _, v := range consensusGroupList {
		if v.Gid == gid {
			result = core.NewGroupInfo(cRw.genesisTime, *v)
		}
	}
	if result == nil {
		return nil, errors.Errorf("can't get consensus group[%s] info by [%s-%d].", gid, head.Hash, head.Height)
	}

	return result, nil
}

func (cRw *chainRw) getGid(block *ledger.AccountBlock) (*types.Gid, error) {
	meta, e := cRw.rw.GetContractMeta(block.AccountAddress)
	if e != nil {
		return nil, e
	}
	return &meta.Gid, nil
}
func (cRw *chainRw) GetLatestSnapshotBlock() *ledger.SnapshotBlock {
	return cRw.rw.GetLatestSnapshotBlock()
}
func (cRw *chainRw) checkSnapshotHashValid(startHeight uint64, startHash types.Hash, actual types.Hash, voteTime time.Time) error {
	if startHash == actual {
		return nil
	}
	startB, e := cRw.rw.GetSnapshotBlockByHash(startHash)
	if e != nil {
		return e
	}
	if startB == nil {
		return errors.Errorf("start snapshot block is nil. hashH:%s-%d", startHash, startHeight)
	}
	actualB, e := cRw.rw.GetSnapshotBlockByHash(actual)
	if e != nil {
		return e
	}
	if actualB == nil {
		return errors.Errorf("refer snapshot block is nil. hashH:%s", actual)
	}

	header := cRw.rw.GetLatestSnapshotBlock()
	if header.Timestamp.Before(voteTime) {
		return errors.Errorf("snapshot header time must >= voteTime, headerTime:%s, voteTime:%s, headerHash:%s:%d", header.Timestamp, voteTime, header.Hash, header.Height)
	}

	if actualB.Height < startB.Height {
		return errors.Errorf("refer snapshot block height must >= start snapshot block height")
	}
	return nil
}

// an hour = 48 * period
func (cRw *chainRw) GetSuccessRateByHour(index uint64) (map[types.Address]int32, error) {
	result := make(map[types.Address]int32)
	hourInfos := make(map[types.Address]*consensus_db.Content)
	for i := uint64(0); i < hour; i++ {
		if i > index {
			break
		}
		tmpIndex := index - i
		p, err := cRw.periodPoints.GetByIndex(tmpIndex)
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
func (cRw *chainRw) GetSuccessRateByHour2(index uint64) (map[types.Address]*consensus_db.Content, error) {
	hourInfos := make(map[types.Address]*consensus_db.Content)
	for i := uint64(0); i < hour; i++ {
		if i > index {
			break
		}
		tmpIndex := index - i
		p, err := cRw.periodPoints.GetByIndex(tmpIndex)
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
func (cRw *chainRw) getSnapshotVoteCache(hashes types.Hash) ([]types.Address, bool) {
	result, b := cRw.dbCache.GetElectionResultByHash(hashes)
	if b != nil {
		return nil, false
	}
	if result == nil {
		return nil, false
	}
	return result, true
}
func (cRw *chainRw) updateSnapshotVoteCache(hashes types.Hash, addresses []types.Address) {
	cRw.dbCache.StoreElectionResultByHash(hashes, addresses)
}

func (cRw *chainRw) getContractVoteCache(hashes types.Hash) ([]types.Address, bool) {
	if cRw.lruCache == nil {
		return nil, false
	}
	value, ok := cRw.lruCache.Get(hashes)
	if ok {
		return value.([]types.Address), ok
	}
	return nil, ok
}

func (cRw *chainRw) updateContractVoteCache(hashes types.Hash, addrArr []types.Address) {
	if cRw.lruCache != nil {
		cRw.log.Info(fmt.Sprintf("store election result %s, %+v\n", hashes, addrArr))
		cRw.lruCache.Add(hashes, addrArr)
	}
}

const (
	period = 1
	hour   = 48 * period
	day    = 24 * hour
)
