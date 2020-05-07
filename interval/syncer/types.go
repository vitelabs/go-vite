package syncer

import "github.com/vitelabs/go-vite/interval/common"

type stateMsg struct {
	Height uint64
	Hash   string
}

type accountBlocksMsg struct {
	Address string
	Blocks  []*common.AccountStateBlock
}
type snapshotBlocksMsg struct {
	Blocks []*common.SnapshotBlock
}

type accountHashesMsg struct {
	Address string
	Hashes  []common.HashHeight
}
type snapshotHashesMsg struct {
	Hashes []common.HashHeight
}

type requestAccountHashMsg struct {
	Address string
	Height  uint64
	Hash    string
	PrevCnt uint64
}

type requestSnapshotHashMsg struct {
	Height  uint64
	Hash    string
	PrevCnt uint64
}

type requestAccountBlockMsg struct {
	Address string
	Hashes  []common.HashHeight
}

type requestSnapshotBlockMsg struct {
	Hashes []common.HashHeight
}

type peerState struct {
	Height uint64
	Hash   string
}
