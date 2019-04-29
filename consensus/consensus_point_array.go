package consensus

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-errors/errors"
	"github.com/hashicorp/golang-lru"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/consensus/core"
	"github.com/vitelabs/go-vite/consensus/db"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
)

func newDayLinkedArray(hour LinkedArray,
	db *consensus_db.ConsensusDB,
	proof RollbackProof,
	fn func(b byte, u uint64, hashes types.Hash) (*consensus_db.VoteContent, error),
	genesisTime time.Time,
	log log15.Logger) *linkedArray {
	dayArr := &linkedArray{}
	dayArr.rate = 24 // 24 hour in day
	dayArr.prefix = consensus_db.INDEX_Point_DAY
	dayArr.lowerArr = hour
	dayArr.db = db
	dayArr.proof = proof
	dayArr.extraDataFn = fn

	dayArr.TimeIndex = core.NewTimeIndex(genesisTime, time.Hour*24)
	dayArr.log = log
	return dayArr
}

func newHourLinkedArray(period LinkedArray, db *consensus_db.ConsensusDB, proof RollbackProof, intervalSec time.Duration, genesisTime time.Time, log log15.Logger) *linkedArray {
	hourArr := &linkedArray{}
	hourArr.rate = uint64(time.Hour / intervalSec)
	hourArr.prefix = consensus_db.INDEX_Point_HOUR
	hourArr.lowerArr = period
	hourArr.db = db
	hourArr.proof = proof
	hourArr.TimeIndex = core.NewTimeIndex(genesisTime, time.Hour)
	hourArr.log = log
	return hourArr
}

type LinkedArray interface {
	core.TimeIndex
	GetByIndex(index uint64) (*consensus_db.Point, error)
	GetByIndexWithProof(index uint64, proofHash types.Hash) (*consensus_db.Point, error)
}

type linkedArray struct {
	core.TimeIndex
	prefix   byte
	rate     uint64
	db       *consensus_db.ConsensusDB
	lowerArr LinkedArray

	proof RollbackProof

	extraDataFn func(b byte, u uint64, hashes types.Hash) (*consensus_db.VoteContent, error)

	log log15.Logger
}

func (self *linkedArray) GetByIndex(index uint64) (*consensus_db.Point, error) {
	//proofTime := self.genesisTime.Add(self.indexInterval * time.Duration(index))
	_, etime := self.Index2Time(index)
	hash, err := self.proof.ProofHash(etime)
	if err != nil {
		return nil, err
	}
	return self.GetByIndexWithProof(index, hash)
}

func (self *linkedArray) GetByIndexWithProof(index uint64, proofHash types.Hash) (*consensus_db.Point, error) {
	point, exists, err := self.getByIndexWithProofFromDb(index, proofHash)
	if err != nil {
		return nil, err
	}
	if point != nil {
		return point, nil
	}

	point, err = self.getByIndexWithProofFromKernel(index, proofHash)
	if err != nil {
		return nil, err
	}
	if point == nil {
		return nil, errors.Errorf("index[%d][%d]proof[%s] Get fail", self.prefix, index, proofHash)
	}
	if exists && point.IsEmpty() {
		self.db.DeletePointByHeight(self.prefix, index)
	} else {
		err = self.db.StorePointByHeight(self.prefix, index, point)
		if err != nil {
			bytes, _ := json.Marshal(point)
			self.log.Error("store point by height Fail.", "index", index, "point", string(bytes))
		}
	}
	return point, nil
}

// result maybe nil
func (self *linkedArray) getByIndexWithProofFromDb(index uint64, proofHash types.Hash) (*consensus_db.Point, bool, error) {
	var exists = false
	// get point info from db
	point, err := self.db.GetPointByHeight(self.prefix, index)
	if err != nil {
		return nil, exists, err
	}
	if point == nil {
		// proof for empty
		emptyProofResult, err := self.proof.ProofEmpty(self.Index2Time(index))
		if err != nil {
			return nil, exists, err
		}
		if emptyProofResult {
			return consensus_db.NewEmptyPoint(), exists, nil
		}
	} else {
		exists = true
		if proofHash == point.Hash {
			// from db and proof
			return point, exists, nil
		}
	}
	return nil, exists, nil
}

// result must not nil
func (self *linkedArray) getByIndexWithProofFromKernel(index uint64, proofHash types.Hash) (*consensus_db.Point, error) {
	result := consensus_db.NewEmptyPoint()
	start := index * self.rate
	end := start + self.rate
	tmpProofHash := proofHash
	for i := end; i > start; i-- {
		p, err := self.lowerArr.GetByIndexWithProof(i-1, tmpProofHash)
		if err != nil {
			return nil, err
		}
		if p.IsEmpty() {
			continue
		}
		tmpProofHash = p.PrevHash
		if err := result.LeftAppend(p); err != nil {
			return nil, err
		}
	}
	if !result.IsEmpty() && self.extraDataFn != nil {
		voteContent, err := self.extraDataFn(self.prefix, index, proofHash)
		if err != nil {
			return nil, err
		}
		if voteContent != nil {
			result.Votes = voteContent
		}
	}
	return result, nil
}

type SBPInfo struct {
	ExpectedNum int32
	FactualNum  int32
}

type periodLinkedArray struct {
	core.TimeIndex
	//periods map[uint64]*periodPoint
	periods  *lru.Cache
	rw       Chain
	snapshot DposReader
	proof    RollbackProof
	log      log15.Logger
}

func newPeriodPointArray(rw Chain, cs DposReader, proof RollbackProof, log log15.Logger) *periodLinkedArray {
	cache, err := lru.New(4 * 24 * 60)
	if err != nil {
		panic(err)
	}
	index := core.NewTimeIndex(cs.GetInfo().GenesisTime, time.Second*time.Duration(cs.GetInfo().PlanInterval))
	return &periodLinkedArray{TimeIndex: index, rw: rw, periods: cache, snapshot: cs, log: log, proof: proof}
}

func (self *periodLinkedArray) GetByIndexWithProof(index uint64, proofHash types.Hash) (*consensus_db.Point, error) {
	if self.rw.IsGenesisSnapshotBlock(proofHash) {
		return consensus_db.NewEmptyPoint(), nil
	}
	point, err := self.getByIndexWithProofFromDb(index, proofHash)
	if err != nil {
		return nil, err
	}
	if point != nil {
		return point, nil
	}

	point, err = self.getByIndexWithProofFromChain(index, proofHash)
	if err != nil {
		return nil, err
	}
	if point == nil {
		return nil, errors.Errorf("period index[%d]proof[%s] Get fail", index, proofHash)
	}
	err = self.set(index, point)
	if err != nil {
		bytes, _ := json.Marshal(point)
		self.log.Error("store period point by height Fail.", "index", index, "point", string(bytes))
	}

	return point, nil
}

func (self *periodLinkedArray) getByIndexWithProofFromDb(index uint64, proofHash types.Hash) (*consensus_db.Point, error) {
	// get point info from db
	value, ok := self.periods.Get(index)
	if !ok || value == nil {
		proofEmptyResult, err := self.proof.ProofEmpty(self.Index2Time(index))
		if err != nil {
			return nil, err
		}
		if proofEmptyResult {
			return consensus_db.NewEmptyPoint(), nil
		}
	} else {
		point := value.(*consensus_db.Point)
		if proofHash == point.Hash {
			// from db and proof
			return point, nil
		}
	}
	return nil, nil
}

func (self *periodLinkedArray) GetByIndex(index uint64) (*consensus_db.Point, error) {
	_, etime := self.Index2Time(index)

	proofHash, err := self.proof.ProofHash(etime)
	if err != nil {
		return nil, err
	}
	return self.GetByIndexWithProof(index, proofHash)
}

func (self *periodLinkedArray) set(index uint64, block *consensus_db.Point) error {
	self.periods.Add(index, block)
	return nil
}

func (self *periodLinkedArray) getByIndexWithProofFromChain(index uint64, proofHash types.Hash) (*consensus_db.Point, error) {
	stime, etime := self.snapshot.Index2Time(index)

	block, err := self.rw.GetSnapshotBlockByHash(proofHash)
	if err != nil {
		return nil, err
	}
	if block == nil {
		return nil, errors.Errorf("period index[%d] proof[%s] not exist.", index, proofHash)
	}

	blocks, err := self.rw.GetSnapshotHeadersAfterOrEqualTime(&ledger.HashHeight{Hash: block.Hash, Height: block.Height}, &stime, nil)
	if err != nil {
		return nil, err
	}

	for _, v := range blocks {
		self.log.Debug(fmt.Sprintf("[A]index:%d, height:%d, producer:%s, hash:%s", index, v.Height, v.Producer(), v.Hash))
	}

	// actually no block
	if len(blocks) == 0 {
		return consensus_db.NewEmptyPoint(), nil
	}

	result, err := self.snapshot.ElectionIndex(index)
	if err != nil {
		return nil, err
	}
	for _, v := range result.Plans {
		self.log.Debug(fmt.Sprintf("[E]index:%d, producer:%s, stime:%s ", index, v.Member, v.STime))
	}

	return self.genPeriodPoint(index, &stime, &etime, proofHash, blocks, result)
}

func (self *periodLinkedArray) genPeriodPoint(index uint64, stime *time.Time, etime *time.Time, proofHash types.Hash, blocks []*ledger.SnapshotBlock, result *electionResult) (*consensus_db.Point, error) {
	point := consensus_db.NewEmptyPoint()

	point.PrevHash = blocks[len(blocks)-1].PrevHash
	point.Hash = blocks[0].Hash
	for _, v := range blocks {
		sbp, ok := point.Sbps[v.Producer()]
		if !ok {
			point.Sbps[v.Producer()] = &consensus_db.Content{FactualNum: 1, ExpectedNum: 0}
		} else {
			sbp.AddNum(0, 1)
		}
	}

	for _, v := range result.Plans {
		sbp, ok := point.Sbps[v.Member]
		if !ok {
			point.Sbps[v.Member] = &consensus_db.Content{FactualNum: 0, ExpectedNum: 1}
		} else {
			sbp.AddNum(1, 0)
		}
	}
	return point, nil
}
