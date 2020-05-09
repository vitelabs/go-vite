package tree

import (
	"time"

	"github.com/vitelabs/go-vite/interval/common"
)

type BranchType uint8

const (
	Normal BranchType = 0
	Disk              = 1
)

type Knot interface {
	Height() uint64
	Hash() common.Hash
	PrevHash() common.Hash
}

type BranchBase interface {
	MatchHead(hash common.Hash) bool
	SprintTail() string
	SprintHead() string
	UTime() time.Time
}

type Branch interface {
	BranchBase
	GetKnot(height uint64, flag bool) Knot
	GetHash(height uint64, flag bool) *common.Hash
	GetKnotAndBranch(height uint64) (Knot, Branch)
	GetHashAndBranch(height uint64) (*common.Hash, Branch)
	ContainsKnot(height uint64, hash common.Hash, flag bool) bool
	HeadHH() (uint64, common.Hash)
	TailHH() (uint64, common.Hash)
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
	Brothers(b Branch) []Branch
	PruneTree() []Branch
	FindBranch(height uint64, hash common.Hash) Branch
	ForkBranch(b Branch, height uint64, hash common.Hash) Branch

	RootHeadAdd(k Knot) error

	AddHead(b Branch, k Knot) error
	RemoveTail(b Branch, k Knot) error
	AddTail(b Branch, k Knot) error

	SwitchMainTo(b Branch) error
	SwitchMainToEmpty() error
	FindForkPointFromMain(target Branch) (Knot, Knot, error)
	Init(name string, root Branch) error

	Exists(hash common.Hash) bool
	Size() uint64
}
