package pool

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/pool/tree"
)

type branchChain struct {
	rw      chainRw
	chainId string
	v       *ForkVersion
}

func (self *branchChain) AddTail(k tree.Knot) {
	panic("not support")
}

func (self *branchChain) SprintTail() string {
	return "DISK TAIL"
}

func (self *branchChain) SprintHead() string {
	panic("implement me")
}

func (self *branchChain) RemoveTail(k tree.Knot) {
	panic("implement me")
}

func (self *branchChain) GetKnotAndBranch(height uint64) (tree.Knot, tree.Branch) {
	panic("implement me")
}

func (self *branchChain) TailHH() (uint64, types.Hash) {
	panic("implement me")
}

func (self *branchChain) Size() uint64 {
	panic("implement me")
}

func (self *branchChain) AddHead(k ...tree.Knot) error {
	panic("implement me")
}

func (self *branchChain) GetKnot(height uint64, flag bool) tree.Knot {
	panic("implement me")
}

func (self *branchChain) ContainsKnot(height uint64, hash types.Hash, flag bool) bool {
	panic("implement me")
}

func (self *branchChain) HeadHH() (uint64, types.Hash) {
	panic("implement me")
}

func (self *branchChain) Root() tree.Branch {
	panic("implement me")
}

func (self *branchChain) Id() string {
	panic("implement me")
}

func (self *branchChain) Type() tree.BranchType {
	return tree.Disk
}
