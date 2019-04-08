package consensus

import (
	"encoding/json"
	"time"

	"github.com/go-errors/errors"
	"github.com/hashicorp/golang-lru"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/consensus/db"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
)

type TimeIndex struct {
	GenesisTime time.Time
	// second
	Interval time.Duration
}

func (self *TimeIndex) Index2Time(index uint64) (time.Time, time.Time) {
	sTime := self.GenesisTime.Add(self.Interval * time.Duration(index))
	eTime := self.GenesisTime.Add(self.Interval * time.Duration(index+1))
	return sTime, eTime
}

func newDayLinkedArray(hour LinkedArray, db *consensus_db.ConsensusDB, proof RollbackProof, genesisTime time.Time, log log15.Logger) *linkedArray {
	dayArr := &linkedArray{}
	dayArr.rate = DAY_TO_HOUR
	dayArr.prefix = consensus_db.INDEX_Point_DAY
	dayArr.lowerArr = hour
	dayArr.db = db
	dayArr.proof = proof
	dayArr.timeIndex = &TimeIndex{GenesisTime: genesisTime, Interval: time.Duration(DAY_TO_PERIOD*PERIOD_TO_SECS) * time.Second}
	dayArr.log = log
	return dayArr
}

func newHourLinkedArray(period LinkedArray, db *consensus_db.ConsensusDB, proof RollbackProof, genesisTime time.Time, log log15.Logger) *linkedArray {
	hourArr := &linkedArray{}
	hourArr.rate = HOUR_TO_PERIOD
	hourArr.prefix = consensus_db.INDEX_Point_HOUR
	hourArr.lowerArr = period
	hourArr.db = db
	hourArr.proof = proof
	hourArr.timeIndex = &TimeIndex{GenesisTime: genesisTime, Interval: time.Duration(HOUR_TO_PERIOD*PERIOD_TO_SECS) * time.Second}
	hourArr.log = log
	return hourArr
}

type LinkedArray interface {
	GetByIndex(index uint64) (*consensus_db.Point, error)
	GetByIndexWithProof(index uint64, proofHash types.Hash) (*consensus_db.Point, error)
}

type linkedArray struct {
	prefix   byte
	rate     uint64
	db       *consensus_db.ConsensusDB
	lowerArr LinkedArray

	proof RollbackProof

	timeIndex *TimeIndex
	//genesisTime   time.Time
	//indexInterval time.Duration

	log log15.Logger
}

func (self *linkedArray) GetByIndex(index uint64) (*consensus_db.Point, error) {
	//proofTime := self.genesisTime.Add(self.indexInterval * time.Duration(index))
	_, etime := self.timeIndex.Index2Time(index)
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
		emptyProofResult, err := self.proof.ProofEmpty(self.timeIndex.Index2Time(index))
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
	return result, nil
}

var PERIOD_TO_SECS = uint64(75)
var HOUR_TO_PERIOD = uint64(48)
var DAY_TO_HOUR = uint64(24)
var DAY_TO_PERIOD = uint64(24 * 48)

//// hour = 48 * period
//type hourPoint struct {
//	hashPoint
//}

type SBPInfo struct {
	ExpectedNum int32
	FactualNum  int32
}

type periodLinkedArray struct {
	//periods map[uint64]*periodPoint
	periods   *lru.Cache
	rw        Chain
	snapshot  DposReader
	timeIndex *TimeIndex
	proof     RollbackProof
	log       log15.Logger
}

func newPeriodPointArray(rw Chain, cs DposReader, proof RollbackProof, log log15.Logger) *periodLinkedArray {
	cache, err := lru.New(4 * 24 * 60)
	if err != nil {
		panic(err)
	}
	index := &TimeIndex{}
	index.GenesisTime = cs.GetInfo().GenesisTime
	index.Interval = time.Second * time.Duration(cs.GetInfo().PlanInterval)
	return &periodLinkedArray{rw: rw, periods: cache, snapshot: cs, log: log, timeIndex: index, proof: proof}
}

func (self *periodLinkedArray) GetByIndexWithProof(index uint64, proofHash types.Hash) (*consensus_db.Point, error) {
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
		proofEmptyResult, err := self.proof.ProofEmpty(self.timeIndex.Index2Time(index))
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
	_, etime := self.timeIndex.Index2Time(index)

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

	if self.rw.IsGenesisSnapshotBlock(proofHash) {
		return consensus_db.NewEmptyPoint(), nil
	}

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

	// actually no block
	if len(blocks) == 0 {
		return consensus_db.NewEmptyPoint(), nil
	}

	result, err := self.snapshot.ElectionIndex(index)
	if err != nil {
		return nil, err
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
