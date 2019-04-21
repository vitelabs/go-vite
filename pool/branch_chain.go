package pool

import (
	"fmt"

	"github.com/vitelabs/go-vite/ledger"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/pool/tree"
)

type branchChain struct {
	rw      chainRw
	chainId string
	v       *ForkVersion
	head    *ledger.HashHeight
}

func (self *branchChain) MatchHead(hash types.Hash) bool {
	_, h := self.HeadHH()
	return hash == h
}

func (self *branchChain) Linked(root tree.Branch) bool {
	panic("not support")
}

func (self *branchChain) AddTail(k tree.Knot) {
	panic("not support")
}

func (self *branchChain) SprintTail() string {
	return "DISK TAIL"
}

func (self *branchChain) SprintHead() string {
	h1, h2 := self.HeadHH()
	return fmt.Sprintf("%d-%s", h1, h2)
}

func (self *branchChain) RemoveTail(k tree.Knot) {
	panic("not support")
}

func (self *branchChain) GetKnotAndBranch(height uint64) (tree.Knot, tree.Branch) {
	return self.GetKnot(height, true), self
}

func (self *branchChain) TailHH() (uint64, types.Hash) {
	panic("not support")
}

func (self *branchChain) Size() uint64 {
	u, _ := self.HeadHH()
	return u
}

func (self *branchChain) AddHead(k ...tree.Knot) error {
	panic("not support")
}

func (self *branchChain) GetKnot(height uint64, flag bool) tree.Knot {
	return self.rw.getBlock(height)
}

func (self *branchChain) ContainsKnot(height uint64, hash types.Hash, flag bool) bool {
	panic("implement me")
}

func (self *branchChain) Head() commonBlock {
	head := self.rw.head()
	if head == nil {
		return self.rw.getBlock(types.EmptyHeight) // hack implement
	}

	return head
}

func (self *branchChain) HeadHH() (uint64, types.Hash) {
	h := self.head
	if h == nil {
		head := self.Head()
		//self.head = &ledger.HashHeight{Height: head.Height(), Hash: head.Hash()}
		return head.Height(), head.Hash()
	}
	return h.Height, h.Hash
}

func (self *branchChain) Root() tree.Branch {
	panic("not support")
}

func (self *branchChain) Id() string {
	return self.chainId
}

func (self *branchChain) Type() tree.BranchType {
	return tree.Disk
}
