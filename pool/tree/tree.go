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
	RemoveTail(k Knot)
	AddTail(k Knot)
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
	AddHead(k ...Knot) error

	//AddTail(k Knot) error
	//RemoveTail(k Knot) error
	//RemoveHead(k Knot) error
	//Insert(k ...Knot)
	Size() uint64
	Root() Branch
	Id() string
	Type() BranchType
}

type Tree interface {
	Root() Branch
	Main() Branch
	Branches() map[string]Branch
	PruneTree() []Branch
	FindBranch(height uint64, hash types.Hash) Branch
	ForkBranch(b Branch, height uint64, hash types.Hash) Branch
	SwitchMainTo(b Branch) error
	SwitchMainToEmpty() error
	FindForkPointFromMain(target Branch) (Knot, Knot, error)
	Init(name string, root Branch) error
}
