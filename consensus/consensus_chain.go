package consensus

import (
	"math/big"
	"time"

	"github.com/vitelabs/go-vite/common/fork"

	"github.com/vitelabs/go-vite/ledger"

	"github.com/hashicorp/golang-lru"
	"github.com/vitelabs/go-vite/common/types"
)

type Point interface {
	PrevHash() *types.Hash
	NextHash() *types.Hash
	Height() uint64
}

type PointLinkedArray interface {
	GetByHeight(height uint64) (Point, error)
	Append(block Point) error
	NextHeight(height uint64) uint64
}

type hashPoint struct {
	prevHash *types.Hash
	nextHash *types.Hash
	height   uint64
}

func (self *hashPoint) PrevHash() *types.Hash {
	return self.prevHash
}

func (self *hashPoint) NextHash() *types.Hash {
	return self.nextHash
}

func (self *hashPoint) Height() uint64 {
	return self.height
}

type hourLinkedArray struct {
	hours map[uint64]*hourPoint
}

func (self *hourLinkedArray) GetByHeight(height uint64) (Point, error) {
	panic("implement me")
}

func (self *hourLinkedArray) Append(block Point) error {
	panic("implement me")
}

func (self *hourLinkedArray) NextHeight(height uint64) uint64 {
	panic("implement me")
}

// hour = 48 * period
type hourPoint struct {
	hashPoint
}

type SBPInfos map[types.Address]*SBPInfo

func (self SBPInfos) Get(address types.Address) *SBPInfo {
	info, ok := self[address]
	if ok {
		return info
	}

	tmp := &SBPInfo{}
	self[address] = tmp
	return tmp
}

func NewSBPInfos() SBPInfos {
	return make(map[types.Address]*SBPInfo)
}

type SBPInfo struct {
	ExpectedNum int32
	FactualNum  int32
}

func (self *SBPInfo) AddNum(expectedNum int32, factualNum int32) *SBPInfo {
	self.ExpectedNum = self.ExpectedNum + expectedNum
	self.FactualNum = self.FactualNum + factualNum
	return self
}
func (self *SBPInfo) Rate() int32 {
	if self.ExpectedNum == 0 {
		return -1
	}
	if self.FactualNum == 0 {
		return 0
	}
	result := big.NewInt(0).Div(big.NewInt(int64(self.FactualNum*1000000)), big.NewInt(int64(self.ExpectedNum)))
	return int32(result.Int64())
}

func (self *hourPoint) GetSBPInfos() SBPInfos {
	panic("implement me")
}

// day = 24 * hour = 24 * 48 * period
type dayPoint struct {
	hashPoint
}

type dayLinkedArray struct {
	hours map[uint64]*hourPoint
}

func (self *dayLinkedArray) GetByHeight(height uint64) (Point, error) {
	panic("implement me")
}

func (self *dayLinkedArray) Append(block Point) error {
	panic("implement me")
}

func (self *dayLinkedArray) NextHeight(height uint64) uint64 {
	panic("implement me")
}

// period = 75s
type periodPoint struct {
	hashPoint
	empty bool
	// hash exist
	proof *ledger.HashHeight
	// beforeTime + hash
	proof2 *ledger.HashHeight
	stime  *time.Time
	etime  *time.Time
	sbps   map[types.Address]*SBPInfo
}

var empty = make(map[types.Address]*SBPInfo)

func (self *periodPoint) GetSBPInfos() SBPInfos {
	if self.empty {
		return empty
	}
	return self.sbps
}

type periodLinkedArray struct {
	//periods map[uint64]*periodPoint
	periods  *lru.Cache
	rw       ch
	snapshot *teller
}

func newPeriodPointArray(rw ch) *periodLinkedArray {
	cache, err := lru.New(4 * 24 * 60)
	if err != nil {
		panic(err)
	}
	return &periodLinkedArray{rw: rw, periods: cache}
}

func (self *periodLinkedArray) GetByHeight(height uint64) (Point, error) {
	value, ok := self.periods.Get(height)
	if !ok || value == nil {
		result, err := self.getByHeight(height)
		if err != nil {
			return nil, err
		}
		if result != nil {
			self.Append(result)
			return result, nil
		} else {
			return nil, nil
		}
	}
	point := value.(*periodPoint)
	valid := self.checkValid(point)
	if !valid {
		result, err := self.getByHeight(height)
		if err != nil {
			return nil, err
		}
		if result != nil {
			self.Append(result)
			return result, nil
		} else {
			return nil, nil
		}
	}
	return point, nil
}

func (self *periodLinkedArray) Append(block Point) error {
	if block.(*periodPoint).etime == nil {
		panic("nilnilnil")
	}
	self.periods.Add(block.Height(), block)
	return nil
}

func (self *periodLinkedArray) NextHeight(height uint64) uint64 {
	return height + 1
}

func (self *periodLinkedArray) getByHeight(height uint64) (*periodPoint, error) {
	stime, etime := self.snapshot.index2Time(height)
	// todo opt
	endSnapshotBlock, err := self.rw.GetSnapshotBlockBeforeTime(&etime)
	if err != nil {
		return nil, err
	}
	if endSnapshotBlock.Timestamp.Before(stime) {
		return self.emptyPoint(height, &stime, &etime, endSnapshotBlock)
	}

	if self.rw.IsGenesisSnapshotBlock(endSnapshotBlock) {
		return self.emptyPoint(height, &stime, &etime, endSnapshotBlock)
	}

	if !fork.IsMintFork(endSnapshotBlock.Height) {
		return self.emptyPoint(height, &stime, &etime, endSnapshotBlock)
	}

	blocks, err := self.rw.GetSnapshotBlocksAfterAndEqualTime(endSnapshotBlock.Height, &stime, nil)
	if err != nil {
		return nil, err
	}

	// actually no block
	if len(blocks) == 0 {
		return self.emptyPoint(height, &stime, &etime, endSnapshotBlock)
	}

	result, err := self.snapshot.electionIndex(height)
	if err != nil {
		return nil, err
	}

	return self.genPeriodPoint(height, &stime, &etime, endSnapshotBlock, blocks, result)
}

func (self *periodLinkedArray) checkValid(point *periodPoint) bool {
	proof := point.proof
	if proof != nil {
		block, _ := self.rw.GetSnapshotBlockByHash(&proof.Hash)
		if block == nil {
			return false
		} else {
			return true
		}
	}

	proof2 := point.proof2
	if proof2 != nil {
		if point.etime == nil {
			panic("etime is nil")
		}
		block, _ := self.rw.GetSnapshotBlockBeforeTime(point.etime)
		if block != nil && block.Hash == proof2.Hash {
			return true
		} else {
			return false
		}
	}
	return false
}

func (self *periodLinkedArray) emptyPoint(height uint64, stime, etime *time.Time, endSnapshotBlock *ledger.SnapshotBlock) (*periodPoint, error) {
	point := &periodPoint{}
	point.height = height
	point.stime = stime
	point.etime = etime
	point.empty = true

	block, err := self.rw.GetSnapshotBlockByHeight(endSnapshotBlock.Height + 1)
	if err != nil {
		return nil, err
	}
	if block != nil && block.Timestamp.After(*etime) {
		point.proof = &ledger.HashHeight{Hash: block.Hash, Height: block.Height}
	} else {
		point.proof2 = &ledger.HashHeight{Hash: endSnapshotBlock.Hash, Height: endSnapshotBlock.Height}
	}
	return point, nil
}
func (self *periodLinkedArray) genPeriodPoint(height uint64, stime *time.Time, etime *time.Time, endSnapshot *ledger.SnapshotBlock, blocks []*ledger.SnapshotBlock, result *electionResult) (*periodPoint, error) {
	point := &periodPoint{}
	point.height = height
	point.stime = stime
	point.etime = etime
	point.empty = false

	block, err := self.rw.GetSnapshotBlockByHeight(endSnapshot.Height + 1)
	if err != nil {
		return nil, err
	}
	if block != nil && (block.Timestamp.Nanosecond() >= etime.Nanosecond()) {
		point.proof = &ledger.HashHeight{Hash: block.Hash, Height: block.Height}
		point.nextHash = &block.Hash
	} else {
		point.proof2 = &ledger.HashHeight{Hash: endSnapshot.Hash, Height: endSnapshot.Height}
	}
	point.prevHash = &blocks[len(blocks)-1].Hash

	sbps := make(map[types.Address]*SBPInfo)
	for _, v := range blocks {
		sbp, ok := sbps[v.Producer()]
		if !ok {
			sbps[v.Producer()] = &SBPInfo{FactualNum: 1, ExpectedNum: 0}
		} else {
			sbp.AddNum(0, 1)
		}
	}

	for _, v := range result.Plans {
		sbp, ok := sbps[v.Member]
		if !ok {
			sbps[v.Member] = &SBPInfo{FactualNum: 0, ExpectedNum: 1}
		} else {
			sbp.AddNum(1, 0)
		}
	}
	point.sbps = sbps
	return point, nil
}
