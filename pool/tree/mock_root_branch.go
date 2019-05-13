package tree

import (
	"fmt"

	"github.com/vitelabs/go-vite/common/types"
)

type mockBranchRoot struct {
	chainId    string
	base       *branchBase
	headHeight uint64
}

func newMockBranchRoot() *mockBranchRoot {
	root := &mockBranchRoot{chainId: "disk", headHeight: 0}
	root.base = newBranchBase(emptyKnot.Height(), emptyKnot.Hash(), emptyKnot.Height(), emptyKnot.Hash(), "disk-base")
	return root
}

func (self *mockBranchRoot) RemoveTail(k Knot) error {
	panic("implement me")
}

func (self *mockBranchRoot) MatchHead(hash types.Hash) bool {
	_, h := self.HeadHH()
	return hash == h
}

func (self *mockBranchRoot) Linked(root Branch) bool {
	panic("not support")
}

func (self *mockBranchRoot) AddTail(k Knot) {
	panic("not support")
}

func (self *mockBranchRoot) SprintTail() string {
	return "DISK TAIL"
}

func (self *mockBranchRoot) SprintHead() string {
	h1, h2 := self.HeadHH()
	return fmt.Sprintf("%d-%s", h1, h2)
}

func (self *mockBranchRoot) GetKnotAndBranch(height uint64) (Knot, Branch) {
	return self.GetKnot(height, true), self
}

func (self *mockBranchRoot) TailHH() (uint64, types.Hash) {
	panic("not support")
}

func (self *mockBranchRoot) Size() uint64 {
	u, _ := self.HeadHH()
	return u
}

func (self *mockBranchRoot) AddHead(k Knot) error {
	panic("not support")
}

func (self *mockBranchRoot) addHead(ks ...Knot) error {
	for _, v := range ks {
		self.base.addHead(v)
		self.headHeight = v.Height()
	}
	return nil
}

func (self *mockBranchRoot) GetKnot(height uint64, flag bool) Knot {
	knot := self.base.getHeightBlock(height)
	if knot == nil {
		return nil
	}
	return knot
}

func (self *mockBranchRoot) ContainsKnot(height uint64, hash types.Hash, flag bool) bool {
	panic("implement me")
}

func (self *mockBranchRoot) Head() Knot {
	head := self.base.getHeightBlock(self.headHeight)
	if head == nil {
		return emptyKnot
	}

	return head
}

func (self *mockBranchRoot) HeadHH() (uint64, types.Hash) {
	head := self.Head()
	return head.Height(), head.Hash()
}

func (self *mockBranchRoot) Root() Branch {
	panic("not support")
}

func (self *mockBranchRoot) ID() string {
	return self.chainId
}

func (self *mockBranchRoot) Type() BranchType {
	return Disk
}
