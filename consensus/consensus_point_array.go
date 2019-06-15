package consensus

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-errors/errors"
	"github.com/hashicorp/golang-lru"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/consensus/cdb"
	"github.com/vitelabs/go-vite/consensus/core"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
)

func newDayLinkedArray(hour LinkedArray,
	db *cdb.ConsensusDB,
	proof RollbackProof,
	fn func(b byte, u uint64, hashes types.Hash) (*cdb.VoteContent, error),
	genesisTime time.Time,
	log log15.Logger) *linkedArray {
	dayArr := &linkedArray{}
	dayArr.rate = 24 // 24 hour in day
	dayArr.prefix = cdb.IndexPointDay
	dayArr.lowerArr = hour
	dayArr.db = db
	dayArr.proof = proof
	dayArr.extraDataFn = fn

	dayArr.TimeIndex = core.NewTimeIndex(genesisTime, time.Hour*24)
	dayArr.log = log
	return dayArr
}

func newHourLinkedArray(period LinkedArray, db *cdb.ConsensusDB, proof RollbackProof, intervalSec time.Duration, genesisTime time.Time, log log15.Logger) *linkedArray {
	hourArr := &linkedArray{}
	hourArr.rate = uint64(time.Hour / intervalSec)
	hourArr.prefix = cdb.IndexPointHour
	hourArr.lowerArr = period
	hourArr.db = db
	hourArr.proof = proof
	hourArr.TimeIndex = core.NewTimeIndex(genesisTime, time.Hour)
	hourArr.log = log
	return hourArr
}

// LinkedArray can read SBP statistics of day, hour, period
type LinkedArray interface {
	core.TimeIndex
	GetByIndex(index uint64) (*cdb.Point, error)
	GetByIndexWithProof(index uint64, proofHash types.Hash) (*cdb.Point, error)
}

type linkedArray struct {
	core.TimeIndex
	prefix   byte
	rate     uint64
	db       *cdb.ConsensusDB
	lowerArr LinkedArray

	proof RollbackProof

	extraDataFn func(b byte, u uint64, hashes types.Hash) (*cdb.VoteContent, error)

	log log15.Logger
}

func (arr *linkedArray) GetByIndex(index uint64) (*cdb.Point, error) {
	_, etime := arr.Index2Time(index)
	hash, err := arr.proof.ProofHash(etime)
	if err != nil {
		return nil, err
	}
	return arr.GetByIndexWithProof(index, hash)
}

func (arr *linkedArray) GetByIndexWithProof(index uint64, proofHash types.Hash) (*cdb.Point, error) {
	point, exists, err := arr.getByIndexWithProofFromDb(index, proofHash)
	if err != nil {
		return nil, err
	}
	if point != nil {
		return point, nil
	}

	point, err = arr.getByIndexWithProofFromKernel(index, proofHash)
	if err != nil {
		return nil, err
	}
	if point == nil {
		return nil, errors.Errorf("index[%d][%d]proof[%s] Get fail", arr.prefix, index, proofHash)
	}
	if exists && point.IsEmpty() {
		arr.db.DeletePointByHeight(arr.prefix, index)
	} else {
		err = arr.db.StorePointByHeight(arr.prefix, index, point)
		if err != nil {
			bytes, _ := json.Marshal(point)
			arr.log.Error("store point by height Fail.", "index", index, "point", string(bytes))
		}
	}
	return point, nil
}

// result maybe nil
func (arr *linkedArray) getByIndexWithProofFromDb(index uint64, proofHash types.Hash) (*cdb.Point, bool, error) {
	var exists = false
	// get point info from db
	point, err := arr.db.GetPointByHeight(arr.prefix, index)
	if err != nil {
		return nil, exists, err
	}
	if point == nil {
		// proof for empty
		emptyProofResult, err := arr.proof.ProofEmpty(arr.Index2Time(index))
		if err != nil {
			return nil, exists, err
		}
		if emptyProofResult {
			return cdb.NewEmptyPoint(proofHash), exists, nil
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
func (arr *linkedArray) getByIndexWithProofFromKernel(index uint64, proofHash types.Hash) (*cdb.Point, error) {
	result := cdb.NewEmptyPoint(proofHash)
	start := index * arr.rate
	end := start + arr.rate
	tmpProofHash := proofHash
	for i := end; i > start; i-- {
		p, err := arr.lowerArr.GetByIndexWithProof(i-1, tmpProofHash)
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
	if !result.IsEmpty() && arr.extraDataFn != nil {
		voteContent, err := arr.extraDataFn(arr.prefix, index, proofHash)
		if err != nil {
			return nil, err
		}
		if voteContent != nil {
			result.Votes = voteContent
		}
	}
	return result, nil
}

// SBPInfo provide expected and factual block information
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

func (period *periodLinkedArray) GetByIndexWithProof(index uint64, proofHash types.Hash) (*cdb.Point, error) {
	if period.rw.IsGenesisSnapshotBlock(proofHash) {
		return cdb.NewEmptyPoint(proofHash), nil
	}
	point, err := period.getByIndexWithProofFromDb(index, proofHash)
	if err != nil {
		return nil, err
	}
	if point != nil {
		return point, nil
	}

	point, err = period.getByIndexWithProofFromChain(index, proofHash)
	if err != nil {
		return nil, err
	}
	if point == nil {
		return nil, errors.Errorf("period index[%d]proof[%s] Get fail", index, proofHash)
	}
	err = period.set(index, point)
	if err != nil {
		bytes, _ := json.Marshal(point)
		period.log.Error("store period point by height Fail.", "index", index, "point", string(bytes))
	}

	return point, nil
}

func (period *periodLinkedArray) getByIndexWithProofFromDb(index uint64, proofHash types.Hash) (*cdb.Point, error) {
	// get point info from db
	value, ok := period.periods.Get(index)
	if !ok || value == nil {
		proofEmptyResult, err := period.proof.ProofEmpty(period.Index2Time(index))
		if err != nil {
			return nil, err
		}
		if proofEmptyResult {
			return cdb.NewEmptyPoint(proofHash), nil
		}
	} else {
		point := value.(*cdb.Point)
		if proofHash == point.Hash {
			// from db and proof
			return point, nil
		}
	}
	return nil, nil
}

func (period *periodLinkedArray) GetByIndex(index uint64) (*cdb.Point, error) {
	_, etime := period.Index2Time(index)

	proofHash, err := period.proof.ProofHash(etime)
	if err != nil {
		return nil, err
	}
	return period.GetByIndexWithProof(index, proofHash)
}

func (period *periodLinkedArray) set(index uint64, block *cdb.Point) error {
	period.periods.Add(index, block)
	return nil
}

func (period *periodLinkedArray) getByIndexWithProofFromChain(index uint64, proofHash types.Hash) (*cdb.Point, error) {
	stime, etime := period.snapshot.Index2Time(index)

	block, err := period.rw.GetSnapshotBlockByHash(proofHash)
	if err != nil {
		return nil, err
	}
	if block == nil {
		return nil, errors.Errorf("period index[%d] proof[%s] not exist.", index, proofHash)
	}

	if block.Timestamp.Before(stime) {
		return cdb.NewEmptyPoint(proofHash), nil
	}

	blocks, err := period.rw.GetSnapshotHeadersAfterOrEqualTime(&ledger.HashHeight{Hash: block.Hash, Height: block.Height}, &stime, nil)
	if err != nil {
		return nil, err
	}

	for _, v := range blocks {
		period.log.Debug(fmt.Sprintf("[A]index:%d, height:%d, producer:%s, hash:%s", index, v.Height, v.Producer(), v.Hash))
	}

	// actually no block
	if len(blocks) == 0 {
		return cdb.NewEmptyPoint(proofHash), nil
	}

	result, err := period.snapshot.ElectionIndex(index)
	if err != nil {
		return nil, err
	}
	for _, v := range result.Plans {
		period.log.Debug(fmt.Sprintf("[E]index:%d, producer:%s, stime:%s ", index, v.Member, v.STime))
	}

	return period.genPeriodPoint(index, &stime, &etime, proofHash, blocks, result)
}

func (period *periodLinkedArray) genPeriodPoint(index uint64, stime *time.Time, etime *time.Time, proofHash types.Hash, blocks []*ledger.SnapshotBlock, result *electionResult) (*cdb.Point, error) {
	if proofHash != blocks[0].Hash {
		return nil, errors.Errorf("gen period point fail[%s][%s]", proofHash, blocks[0].Hash)
	}
	point := cdb.NewEmptyPoint(proofHash)

	point.PrevHash = blocks[len(blocks)-1].PrevHash
	point.Hash = blocks[0].Hash
	for _, v := range blocks {
		sbp, ok := point.Sbps[v.Producer()]
		if !ok {
			point.Sbps[v.Producer()] = &cdb.Content{FactualNum: 1, ExpectedNum: 0}
		} else {
			sbp.AddNum(0, 1)
		}
	}

	for _, v := range result.Plans {
		sbp, ok := point.Sbps[v.Member]
		if !ok {
			point.Sbps[v.Member] = &cdb.Content{FactualNum: 0, ExpectedNum: 1}
		} else {
			sbp.AddNum(1, 0)
		}
	}
	return point, nil
}
