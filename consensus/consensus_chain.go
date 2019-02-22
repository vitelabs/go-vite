package consensus

import (
	"github.com/vitelabs/go-vite/common/types"
)

type Point interface {
	PrevHash() *types.Hash
	NextHash() *types.Hash
	Height() uint64
}

type PointLinkedArray interface {
	GetByHeight(height uint64) Point
	Append(block Point) error
	NextHeight(height uint64) uint64
}

type hashPoint struct {
	prevHash *types.Hash
	nextHash *types.Hash
}

func (self *hashPoint) PrevHash() *types.Hash {
	panic("implement me")
}

func (self *hashPoint) NextHash() *types.Hash {
	panic("implement me")
}

func (self *hashPoint) Height() uint64 {
	panic("implement me")
}

type hourLinkedArray struct {
	hours map[uint64]*hourPoint
}

func (self *hourLinkedArray) GetByHeight(height uint64) Point {
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
	ExpectedNum uint64
	FactualNum  uint64
}

func (self *SBPInfo) AddNum(expectedNum uint64, factualNum uint64) *SBPInfo {
	self.ExpectedNum = self.ExpectedNum + expectedNum
	self.FactualNum = self.FactualNum + factualNum
	return self
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

func (self *dayLinkedArray) GetByHeight(height uint64) Point {
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
}

func (self *periodPoint) GetSBPInfos() SBPInfos {
	panic("implement me")
}

type periodLinkedArray struct {
	hours map[uint64]*periodPoint
}

func (self *periodLinkedArray) GetByHeight(height uint64) Point {
	panic("implement me")
}

func (self *periodLinkedArray) Append(block Point) error {
	panic("implement me")
}

func (self *periodLinkedArray) NextHeight(height uint64) uint64 {
	panic("implement me")
}
