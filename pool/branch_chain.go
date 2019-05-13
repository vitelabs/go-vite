package pool

import (
	"fmt"

	"github.com/vitelabs/go-vite/common"

	"github.com/vitelabs/go-vite/ledger"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/pool/tree"
)

type branchChain struct {
	rw      chainRw
	chainID string
	v       *common.Version
	head    *ledger.HashHeight
	t       tree.Tree
}

func (disk *branchChain) RemoveTail(k tree.Knot) error {
	panic("implement me")
}

func (disk *branchChain) MatchHead(hash types.Hash) bool {
	_, h := disk.HeadHH()
	return hash == h
}

func (disk *branchChain) Linked(root tree.Branch) bool {
	panic("not support")
}

func (disk *branchChain) AddTail(k tree.Knot) {
	panic("not support")
}

func (disk *branchChain) SprintTail() string {
	return "DISK TAIL"
}

func (disk *branchChain) SprintHead() string {
	h1, h2 := disk.HeadHH()
	return fmt.Sprintf("%d-%s", h1, h2)
}

func (disk *branchChain) GetKnotAndBranch(height uint64) (tree.Knot, tree.Branch) {
	return disk.GetKnot(height, true), disk
}

func (disk *branchChain) TailHH() (uint64, types.Hash) {
	panic("not support")
}

func (disk *branchChain) Size() uint64 {
	u, _ := disk.HeadHH()
	return u
}

func (disk *branchChain) AddHead(k tree.Knot) error {
	panic("not support")
}

func (disk *branchChain) GetKnot(height uint64, flag bool) tree.Knot {
	return disk.rw.getBlock(height)
}

func (disk *branchChain) ContainsKnot(height uint64, hash types.Hash, flag bool) bool {
	fmt.Printf("%d, %s, %t, tree:%s\n", height, hash, flag, tree.PrintTree(disk.t))
	panic("implement me")
}

func (disk *branchChain) Head() commonBlock {
	head := disk.rw.head()
	if head == nil {
		return disk.rw.getBlock(types.EmptyHeight) // hack implement
	}

	return head
}

func (disk *branchChain) HeadHH() (uint64, types.Hash) {
	h := disk.head
	if h == nil {
		head := disk.Head()
		return head.Height(), head.Hash()
	}
	return h.Height, h.Hash
}

func (disk *branchChain) Root() tree.Branch {
	panic("not support")
}

func (disk *branchChain) ID() string {
	return disk.chainID
}

func (disk *branchChain) Type() tree.BranchType {
	return tree.Disk
}
