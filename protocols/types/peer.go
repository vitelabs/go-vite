package types

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/p2p"
	"math/big"
	"sync"
)

// @section Peer for protocol handle, not p2p Peer.
type Peer struct {
	*p2p.Peer
	ID 		string
	Head 	types.Hash
	Height	*big.Int
	Version int
	RW 		MsgReadWriter
	Lock 	sync.RWMutex
}

func (p *Peer) Update(status *StatusMsg) {
	p.Lock.Lock()
	defer p.Lock.Unlock()

	p.Height = status.Height
	p.Head = status.CurrentBlock
}
