package handler

import (
	"github.com/vitelabs/go-vite/protocols"
	"github.com/vitelabs/go-vite/ledger/access"
)

type SnapshotChain struct {
	// Handle block
	vite Vite

	scAccess *access.SnapshotChainAccess
}

func NewSnapshotChain (vite Vite) (*SnapshotChain) {
	return &SnapshotChain{
		vite: vite,
		scAccess: access.GetSnapshotChainAccess(),
	}
}

// HandleGetBlock
func (sc *SnapshotChain) HandleGetBlocks (msg *protocols.GetSnapshotBlocksMsg, peer *protocols.Peer) error {
	go func() {
		sc.scAccess.GetBlocksFromOrigin(&msg.Origin, msg.Count, msg.Forward)
		// send out
	}()
	return nil
}

// HandleBlockHash
func (sc *SnapshotChain) HandleSendBlocks (msg protocols.SnapshotBlocksMsg, peer *protocols.Peer) error {
	go func() {
		sc.scAccess.WriteBlockList(msg)
	}()

	return nil}
