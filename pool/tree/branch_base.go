package tree

import (
	"fmt"
	"sync"

	"github.com/vitelabs/go-vite/common/types"
)

type branchBase struct {
	heightBlocks map[uint64]Knot
	headHeight   uint64 //  branch size is zero when headHeight==tailHeight
	headHash     types.Hash
	tailHeight   uint64
	tailHash     types.Hash
	id           string
	heightMu     sync.RWMutex
}

func (self *branchBase) size() uint64 {
	return self.headHeight - self.tailHeight
}

func (self *branchBase) SprintHead() string {
	return fmt.Sprintf("%d-%s", self.headHeight, self.headHash)
}

func (self *branchBase) SprintTail() string {
	return fmt.Sprintf("%d-%s", self.tailHeight, self.tailHash)
}

func (self *branchBase) headHH() (uint64, types.Hash) {
	return self.headHeight, self.headHash
}
func (self *branchBase) tailHH() (uint64, types.Hash) {
	return self.tailHeight, self.tailHash
}

func (self *branchBase) branchId() string {
	return self.id
}

func (self *branchBase) getHeightBlock(height uint64) Knot {
	self.heightMu.RLock()
	defer self.heightMu.RUnlock()
	block, ok := self.heightBlocks[height]
	if ok {
		return block
	} else {
		return nil
	}
}

func (self *branchBase) updateHeightBlock(height uint64, b Knot) {
	if b != nil {
		self.heightBlocks[height] = b
	} else {
		// nil means delete
		delete(self.heightBlocks, height)
	}
}

func (self *branchBase) addHead(w Knot) {
	self.heightMu.Lock()
	defer self.heightMu.Unlock()
	if self.headHash != w.PrevHash() {
		panic("add head")
	}
	self.headHash = w.Hash()
	self.headHeight = w.Height()
	self.updateHeightBlock(w.Height(), w)
}

func (self *branchBase) MatchHead(hash types.Hash) bool {
	if self.headHash == hash {
		return true
	}
	return false
}

func (self *branchBase) removeTail(w Knot) {
	self.heightMu.Lock()
	defer self.heightMu.Unlock()
	if self.tailHash != w.PrevHash() {
		panic("remove fail")
	}
	self.tailHash = w.Hash()
	self.tailHeight = w.Height()
	self.updateHeightBlock(w.Height(), nil)
}

func (self *branchBase) removeHead(w Knot) {
	self.heightMu.Lock()
	defer self.heightMu.Unlock()
	if self.headHash != w.Hash() {
		panic("remove head")
	}
	self.headHash = w.PrevHash()
	self.headHeight = w.Height() - 1
	self.updateHeightBlock(w.Height(), nil)
}

func (self *branchBase) AddTail(w Knot) {
	self.heightMu.Lock()
	defer self.heightMu.Unlock()
	if self.tailHash != w.Hash() {
		panic("add tail")
	}
	self.tailHash = w.PrevHash()
	self.tailHeight = w.Height() - 1
	self.updateHeightBlock(w.Height(), w)
}

func newBranchBase(tailHeight uint64, tailHash types.Hash, headHeight uint64, headHash types.Hash, id string) *branchBase {
	b := &branchBase{}
	b.tailHeight = tailHeight
	b.tailHash = tailHash
	b.headHeight = headHeight
	b.headHash = headHash
	b.id = id
	b.heightBlocks = make(map[uint64]Knot)
	return b
}
