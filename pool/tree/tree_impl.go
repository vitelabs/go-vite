package tree

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/go-errors/errors"
	"github.com/vitelabs/go-vite/common/types"
)

type tree struct {
	branchMu   sync.RWMutex
	branchList map[string]*branch

	main *branch
	root Branch

	idIdx uint32
	name  string
}

func (self *tree) FindBranch(height uint64, hash types.Hash) Branch {
	block := self.main.GetKnot(height, false)
	if block != nil && block.Hash() == hash {
		return self.main
	}
	for _, c := range self.Branches() {
		b := c.GetKnot(height, false)

		if b == nil {
			continue
		} else {
			if b.Hash() == hash {
				return c
			}
		}
	}
	return nil
}
func NewTree() *tree {
	return &tree{branchList: make(map[string]*branch)}
}

func (self *tree) Init(name string, root Branch) error {
	self.name = name
	self.root = root
	height, hash := root.HeadHH()
	self.main = newBranch(newBranchBase(height, hash, height, hash, self.newBranchId()), root)
	self.addBranch(self.main)
	return nil
}

func (self *tree) newBranchId() string {
	return fmt.Sprintf("%s-%d", self.name, atomic.AddUint32(&self.idIdx, 1))
}

func (self *tree) Root() Branch {
	return self.root
}

func (self *tree) Main() Branch {
	return self.main
}

func (self *tree) ForkBranch(b Branch, height uint64, hash types.Hash) Branch {
	knot := b.GetKnot(height, true)
	new := newBranch(newBranchBase(knot.Height(), knot.Hash(), knot.Height(), knot.Hash(), self.newBranchId()), b)
	self.addBranch(new)
	return new
}

func (self *tree) SwitchMainTo(b Branch) error {
	if b.Id() == self.main.Id() {
		return nil
	}
	if b.Type() != Normal {
		return errors.Errorf("branch[%s] type error.", b.Id())
	}
	target := b.(*branch)

	// first prune
	target.prune()
	// second modify
	target.exchangeAllRoot()

	if target.Linked(self.root) {
		self.main = target
	} else {
		return errors.New("new chain tail fail.")
	}
	return nil
}

func (self *tree) SwitchMainToEmpty() error {
	if self.main.Size() == 0 {
		return nil
	}
	height, hash := self.root.HeadHH()
	emptyChain := self.findEmptyForHead(height, hash)
	if emptyChain != nil {
		return self.SwitchMainTo(emptyChain)
	} else {
		return self.SwitchMainTo(self.ForkBranch(self.main, height, hash))
	}
}

// keyPoint, forkPoint, err
func (self *tree) FindForkPointFromMain(target Branch) (Knot, Knot, error) {
	if target.Type() == Disk {
		return nil, nil, errors.New("fail to find fork point from disk.")
	}

	longer, shorter := self.longer(target.(*branch), self.main)
	curHeadHeight := shorter.headHeight

	i := curHeadHeight
	var forkedBlock Knot

	for {
		longerBlock := longer.GetKnot(i, true)
		shorterBlock := shorter.GetKnot(i, true)
		if longerBlock == nil {
			return nil, nil, errors.New("longest chain error.")
		}

		if shorterBlock == nil {
			return nil, nil, errors.New("current chain error.")
		}

		if longerBlock.Hash() == shorterBlock.Hash() {
			forkedBlock = longerBlock
			keyPoint := longer.GetKnot(i+1, true)
			return keyPoint, forkedBlock, nil
		}
		i = i - 1
	}
	return nil, nil, errors.New("can't find fork point")
}

// longer, shorter
func (self *tree) longer(b1 *branch, b2 *branch) (*branch, *branch) {
	if b1.headHeight > b2.headHeight {
		return b1, b2
	} else {
		return b2, b1
	}
}

func (self *tree) findEmptyForHead(headHeight uint64, headHash types.Hash) *branch {
	self.branchMu.RLock()
	defer self.branchMu.RUnlock()
	for _, c := range self.branchList {
		if c.size() == uint64(0) && c.headHash == headHash && c.headHeight == headHeight {
			return c
		}
	}
	return nil
}

func (self *tree) Branches() map[string]Branch {
	self.branchMu.RLock()
	defer self.branchMu.RUnlock()
	branches := self.branchList
	result := make(map[string]Branch, len(branches))
	for k, v := range branches {
		result[k] = v
	}
	return result
}

func (self *tree) PruneTree() []Branch {
	self.branchMu.Lock()
	defer self.branchMu.Unlock()

	// prune every branch
	for id, c := range self.branchList {
		if id == self.main.Id() {
			continue
		}
		c.prune()
	}

	var r []Branch
	for id, c := range self.branchList {
		if id == self.main.Id() {
			continue
		}
		if !c.isGarbage() {
			continue
		}
		self.removeBranch(c)
	}
	return r
}

func (self *tree) removeBranch(b *branch) error {
	if b.isLeafBranch() {
		id := b.Id()
		if b.Root().Type() == Disk {
			return errors.Errorf("chain[%s] can't be removed[refer disk].", id)
		}
		b.Root().(*branch).removeChild(b)
		delete(self.branchList, id)
	}

	return errors.New("not support")
}

//func (self *tree) clearRepeatBranch() []Branch {
//	var r []Branch
//	for id1, c1 := range self.chains {
//		if id1 == self.current.id() {
//			continue
//		}
//		for _, c2 := range self.chains {
//			if c1.tailHeight == c2.tailHeight &&
//				c1.tailHash == c2.tailHash {
//				h := c2.getHeightBlock(c1.headHeight)
//				if h != nil && h.Hash() == c1.headHash {
//					r = append(r, c1)
//				}
//			}
//		}
//	}
//	return r
//}

func (self *tree) addBranch(b *branch) {
	self.branchMu.Lock()
	defer self.branchMu.Unlock()
	self.branchList[b.Id()] = b
}
