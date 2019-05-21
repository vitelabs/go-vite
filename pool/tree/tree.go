package tree

import (
	"github.com/vitelabs/go-vite/common/types"
)

type BranchType uint8

const (
	Normal BranchType = 0
	Disk              = 1
)

type Knot interface {
	Height() uint64
	Hash() types.Hash
	PrevHash() types.Hash
}

type BranchBase interface {
	MatchHead(hash types.Hash) bool
	SprintTail() string
	SprintHead() string
}

type Branch interface {
	BranchBase
	GetKnot(height uint64, flag bool) Knot
	GetKnotAndBranch(height uint64) (Knot, Branch)
	ContainsKnot(height uint64, hash types.Hash, flag bool) bool
	HeadHH() (uint64, types.Hash)
	TailHH() (uint64, types.Hash)
	Linked(root Branch) bool

	Size() uint64
	Root() Branch
	ID() string
	Type() BranchType
}

type Tree interface {
	Root() Branch
	Main() Branch
	Branches() map[string]Branch
	PruneTree() []Branch
	FindBranch(height uint64, hash types.Hash) Branch
	ForkBranch(b Branch, height uint64, hash types.Hash) Branch

	RootHeadAdd(k Knot) error

	AddHead(b Branch, k Knot) error
	RemoveTail(b Branch, k Knot) error
	AddTail(b Branch, k Knot) error

	SwitchMainTo(b Branch) error
	SwitchMainToEmpty() error
	FindForkPointFromMain(target Branch) (Knot, Knot, error)
	Init(name string, root Branch) error

	Exists(hash types.Hash) bool
	Size() uint64
}
