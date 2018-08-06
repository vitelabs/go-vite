package types

import (
	"github.com/inconshreveable/log15"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/p2p"
	"math/big"
	"sync"
)

// @section Peer for protocol handle, not p2p Peer.
type Peer struct {
	*p2p.Peer
	ID      string
	Head    types.Hash
	Height  *big.Int
	Version int
	RW      MsgReadWriter
	Lock    sync.RWMutex
	// use this channel to ensure that only one goroutine send msg simultaneously.
	Sending chan struct{}
	Log     log15.Logger
}

func (p *Peer) Update(status *StatusMsg) {
	p.Lock.Lock()
	defer p.Lock.Unlock()

	p.Height = status.Height
	p.Head = status.CurrentBlock
	p.Log.Info("update status", "ID", p.ID, "height", p.Height, "head", p.Head)
}
