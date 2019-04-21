package tree

import (
	"fmt"
	"sync"

	"github.com/vitelabs/go-vite/common/types"

	"github.com/go-errors/errors"
	"github.com/vitelabs/go-vite/monitor"
)

type branch struct {
	*branchBase

	root Branch

	childrenMu sync.RWMutex
	children   map[string]*branch
}

func (self *branch) Size() uint64 {
	return self.size()
}

func (self *branch) GetKnotAndBranch(height uint64) (Knot, Branch) {
	if height > self.headHeight {
		return nil, nil
	}
	block := self.getHeightBlock(height)
	if block != nil {
		return block, self
	}
	refers := make(map[string]Branch)
	refer := self.root
	for {
		if refer == nil {
			return nil, nil
		}
		b := refer.GetKnot(height, false)
		if b != nil {
			return b, refer
		} else {
			if _, ok := refers[refer.Id()]; ok {
				monitor.LogEvent("pool", "getBlockError")
				return nil, nil
			}
			refers[refer.Id()] = refer
			refer = refer.Root()
		}
	}
	return nil, nil
}

func (self *branch) AddHead(ks ...Knot) error {
	for _, v := range ks {
		self.addHead(v)
	}
	return nil
}

func (self *branch) HeadHH() (uint64, types.Hash) {
	return self.headHH()
}

func (self *branch) TailHH() (uint64, types.Hash) {
	return self.tailHH()
}

func (self *branch) ContainsKnot(height uint64, hash types.Hash, flag bool) bool {
	return self.contains(height, hash, flag)
}

func (self *branch) GetKnot(height uint64, flag bool) Knot {
	w := self.getKnot(height, flag)
	if w == nil {
		return nil
	}
	return w
}

func (self *branch) Id() string {
	return self.branchId()
}

func (self *branch) Type() BranchType {
	return Normal
}

func (self *branch) prune() {
	tail := self.root.GetKnot(self.tailHeight, false)
	if tail == nil {
		panic("tail is nil")
	}
	if self.root.Type() == Normal {
		self.root.(*branch).prune()
	}
	for i := self.tailHeight + 1; i <= self.headHeight; i++ {
		selfB := self.getKnot(i, false)
		block := self.root.GetKnot(i, true)
		if block != nil && block.Hash() == selfB.Hash() {
			fmt.Printf("remove tail[%s][%d-%s]\n", self.branchId(), block.Height(), block.Hash())
			self.RemoveTail(block)
		} else {
			break
		}
	}
}

func (self *branch) exchangeAllRoot() {
	for {
		root := self.root
		if root.Type() == Disk {
			break
		}
		self.exchangeRoot(self.root.(*branch))
	}
}

func (self *branch) exchangeRoot(root *branch) error {
	if root.Id() != self.root.Id() {
		return errors.New("root not match")
	}

	if tailEquals(root, self) {
		root.removeChild(self)
		self.updateRootSimple(root, root.root)
		root.updateRoot(root.root, self)
		return nil
	}

	// from.tailHeight <= to.tailHeight  && from.headHeight > to.tail.Height
	selfTailHeight := self.tailHeight
	rootTailHeight := root.tailHeight
	rootHeadHeight := root.headHeight
	if rootTailHeight <= selfTailHeight && rootHeadHeight >= selfTailHeight {
		for i := selfTailHeight; i > rootTailHeight; i-- {
			w := root.GetKnot(i, false)
			if w != nil {
				self.AddTail(w)
			}
		}
		for i := rootTailHeight + 1; i <= rootTailHeight; i++ {
			w := root.GetKnot(i, false)
			if w != nil {
				root.RemoveTail(w)
			}
		}
		root.removeChild(self)
		self.updateRootSimple(root, root.root)
		root.updateRoot(root.root, self)
		return nil
	} else {
		return errors.Errorf("err for exchangeRoot.root:%s, self:%s, rootTail:%s, rootHead:%s, selfTail:%s, selfHead:%s",
			root.Id(), self.Id(), root.SprintTail(), root.SprintHead(), self.SprintTail(), self.SprintHead())

	}
}

func (self *branch) updateRootSimple(old Branch, new Branch) {
	self.root = new
	if new.Type() == Disk {
		new.(*branch).addChild(self)
	}
}
func (self *branch) updateRoot(old Branch, new Branch) {
	for _, v := range self.allChildren() {
		height, hash := v.tailHH()
		if self.contains(height, hash, false) {
			continue
		}

		if new.ContainsKnot(height, hash, true) {
			continue
		}

		if old.ContainsKnot(height, hash, true) {
			self.removeChild(v)
			v.updateRootSimple(self, old)
			continue
		}

		panic("children fail.")
	}
}

func (self *branch) getKnotAndChain(height uint64) (Knot, Branch) {
	if height > self.headHeight {
		return nil, nil
	}
	block := self.getHeightBlock(height)
	if block != nil {
		return block, self
	}
	refers := make(map[string]Branch)
	refer := self.root
	for {
		if refer == nil {
			return nil, nil
		}
		b := refer.GetKnot(height, false)
		if b != nil {
			return b, refer
		} else {
			if _, ok := refers[refer.Id()]; ok {
				monitor.LogEvent("pool", "GetKnotError")
				return nil, nil
			}
			refers[refer.Id()] = refer
			refer = refer.Root()
		}
	}
	return nil, nil
}

func (self *branch) Root() Branch {
	return self.root
}

func (self *branch) getKnot(height uint64, flag bool) Knot {
	block := self.getHeightBlock(height)
	if block != nil {
		return block
	}
	if flag {
		b, _ := self.getKnotAndChain(height)
		return b
	}
	return nil
}

func (self *branch) contains(height uint64, hash types.Hash, flag bool) bool {
	localResult := self.localContains(height, hash)
	if localResult {
		return true
	}
	if flag == false {
		return false
	}

	knot := self.root.GetKnot(height, flag)
	if knot == nil {
		return false
	}
	return knot.Hash() == hash
}

func (self *branch) localContains(height uint64, hash types.Hash) bool {
	if height > self.tailHeight && self.headHeight <= height {
		k := self.getHeightBlock(height)
		if k != nil {
			return k.Hash() == hash
		} else {
			return false
		}
	} else {
		return false
	}
}

func (self *branch) info() map[string]interface{} {
	result := make(map[string]interface{})
	result["TailHeight"] = self.tailHeight
	result["TailHash"] = self.tailHash
	result["HeadHeight"] = self.headHeight
	result["HeadHash"] = self.headHash
	if self.root != nil {
		result["ReferId"] = self.root.Id()
	}
	result["Id"] = self.Id()
	return result
}

func (self *branch) checkSameTail(target *branch) bool {
	if self.tailHeight != target.tailHeight {
		return false
	}

	if self.tailHash != target.tailHash {
		return false
	}
	return true
}
func (self *branch) removeChild(b *branch) {
	self.childrenMu.Lock()
	defer self.childrenMu.Unlock()

	delete(self.children, b.Id())
}
func (self *branch) addChild(b *branch) {
	self.childrenMu.Lock()
	defer self.childrenMu.Unlock()

	self.children[b.Id()] = b
}

func (self *branch) allChildren() (result []*branch) {
	self.childrenMu.Lock()
	defer self.childrenMu.Unlock()
	for _, v := range self.children {
		result = append(result, v)
	}
	return
}

func (self *branch) Linked(root Branch) bool {
	headHeight, headHash := root.HeadHH()
	if self.tailHeight == headHeight && self.tailHash == headHash {
		return true
	} else {
		return false
	}
}

func newBranch(base *branchBase, root Branch) *branch {
	b := &branch{}
	b.branchBase = base
	b.root = root
	b.children = make(map[string]*branch)
	return b
}

func tailEquals(b1 *branch, b2 *branch) bool {
	if b1.tailHeight == b2.tailHeight && b1.tailHash == b2.tailHash {
		return true
	}
	return false
}
