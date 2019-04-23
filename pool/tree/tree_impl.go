package tree

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/go-errors/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/log15"
)

type tree struct {
	branchMu   sync.RWMutex
	branchList map[string]*branch

	main *branch
	root Branch

	idIdx uint32
	name  string
	log   log15.Logger

	hashes sync.Map
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
func (self *tree) knotRemove(k Knot) {
	self.hashes.Delete(k.Hash())
}

func NewTree() *tree {
	return &tree{
		branchList: make(map[string]*branch),
		log:        log15.New("module", "pool/tree"),
	}
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
	if b.Type() == Disk {
		panic("can't fork from disk")
	}
	root := b.(*branch)
	knot := root.GetKnot(height, true)
	if knot == nil {
		panic(fmt.Sprintf("can't get knot by[%d]", height))
	}
	new := newBranch(newBranchBase(knot.Height(), knot.Hash(), knot.Height(), knot.Hash(), self.newBranchId()), b)
	root.addChild(new)
	self.addBranch(new)
	return new
}

func (self *tree) RootHeadAdd(k Knot) error {
	main := self.main
	if main.MatchHead(k.PrevHash()) {
		err := main.AddHead(k)
		if err != nil {
			return err
		}
		err = main.RemoveTail(k)
		if err != nil {
			return err
		}
		return nil
	} else {
		newBranch := self.ForkBranch(main, k.Height()-1, k.PrevHash()).(*branch)
		err := newBranch.AddHead(k)
		if err != nil {
			return err
		}
		err = newBranch.RemoveTail(k)
		if err != nil {
			return err
		}
		newBranch.updateRootSimple(main, self.root)
		main.updateRoot(main.root, newBranch)

		self.main = newBranch
		return nil
	}
}

func (self *tree) RootHeadRemove(k Knot) error {
	main := self.main
	if main.tailHash != k.Hash() {
		return errors.Errorf("root head[%s][%d-%s] remove fail.", main.SprintTail(), k.Height(), k.Hash())
	}
	main.addTail(k)
	return nil
}

func (self *tree) AddHead(b Branch, k Knot) error {
	if b.Type() == Disk {
		return errors.New("can't add head to chain.")
	}
	br := b.(*branch)
	br.addHead(k)
	self.hashes.Store(k.Hash(), true)
	return nil
}

func (self *tree) RemoveTail(b Branch, k Knot) error {
	if b.Type() == Disk {
		return errors.New("can't remove tail from chain.")
	}
	br := b.(*branch)
	br.removeTail(k)
	self.hashes.Delete(k.Hash())
	return nil
}

func (self *tree) AddTail(b Branch, k Knot) error {
	if b.Type() == Disk {
		return errors.New("can't remove tail from chain.")
	}
	br := b.(*branch)
	br.addTail(k)
	self.hashes.Store(k.Hash(), true)
	return nil
}

func (self *tree) Exists(hash types.Hash) bool {
	_, ok := self.hashes.Load(hash)
	return ok
}

func (self *tree) Size() uint64 {
	result := uint64(0)
	self.hashes.Range(func(key, value interface{}) bool {
		result = result + 1
		return true
	})
	return result
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
	target.prune(self)

	// second modify
	err := target.exchangeAllRoot()
	if err != nil {
		return err
	}

	if target.Linked(self.root) {
		self.main = target
	} else {
		panic("new chain tail fail.")
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
		c.prune(self)
	}

	var r []Branch
	for id, c := range self.branchList {
		if id == self.main.Id() {
			continue
		}
		if !c.isGarbage() {
			continue
		}
		err := self.removeBranch(c)
		if err != nil {
			self.log.Error("remove branch fail.", "err", err)
		} else {
			r = append(r, c)
		}
	}
	return r
}

func (self *tree) removeBranch(b *branch) error {
	id := b.Id()
	if id == self.main.Id() {
		return errors.Errorf("not support for main[%s]", id)
	}
	if b.Root().Type() == Disk {
		return errors.Errorf("chain[%s] can't be removed[refer disk].", id)
	}
	root := b.Root().(*branch)
	if b.isLeafBranch() {
		root.removeChild(b)
		delete(self.branchList, id)
		b.destroy(self)
		return nil
	}

	if b.Size() == 0 {
		root.removeChild(b)
		delete(self.branchList, id)
		for _, v := range b.allChildren() {
			root.addChild(v)
		}
	}
	return errors.New("not support")
}

func (self *tree) addBranch(b *branch) {
	self.branchMu.Lock()
	defer self.branchMu.Unlock()
	self.branchList[b.Id()] = b
}
