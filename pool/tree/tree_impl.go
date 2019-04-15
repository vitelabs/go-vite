package tree

import (
	"sync"

	"github.com/go-errors/errors"
	"github.com/vitelabs/go-vite/common/types"
)

type tree struct {
	branchMu   sync.RWMutex
	branchList map[string]*branch

	main *branch
	root Branch
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

func (self *tree) Init(root Branch) error {
	self.root = root
	height, hash := root.HeadHH()
	self.main = newBranch(newBranchBase(height, hash, height, hash), root)
	self.addBranch(self.main)
	return nil
}

func (self *tree) Root() Branch {
	return self.root
}

func (self *tree) Main() Branch {
	return self.main
}

func (self *tree) ForkBranch(b Branch, height uint64, hash types.Hash) Branch {
	knot := b.GetKnot(height, true)
	new := newBranch(newBranchBase(knot.Height(), knot.Hash(), knot.Height(), knot.Hash()), b)
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

	if self.root.HeadHH() == target.tailHH() {
		self.main = target
	} else {
		return errors.New("new chain tail fail.")
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

func (self *tree) addBranch(b *branch) {
	self.branchMu.Lock()
	defer self.branchMu.Unlock()
	self.branchList[b.Id()] = b
}
