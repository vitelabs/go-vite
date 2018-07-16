package handler

import (
	"github.com/vitelabs/go-vite/protocols"
	"github.com/vitelabs/go-vite/ledger/access"
	"github.com/vitelabs/go-vite/ledger"
	"math/big"
	"github.com/vitelabs/go-vite/common/types"
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
		// pm := sc.vite.Pm()
	}()
	return nil
}

// HandleBlockHash
func (sc *SnapshotChain) HandleSendBlocks (msg protocols.SnapshotBlocksMsg, peer *protocols.Peer) error {
	go func() {
		sc.scAccess.WriteBlockList(msg)
	}()

	return nil
}

func (sc *SnapshotChain) InsertMiningBlock () error {
	return nil
}

func (sc *SnapshotChain) StopAllWrite () error {
	return nil
}

func (sc *SnapshotChain) StartAllWrite () error {
	return nil
}

func (sc *SnapshotChain) GetLatestBlock () (*ledger.SnapshotBlock, error) {
	return sc.scAccess.GetLatestBlock()
}

func (sc *SnapshotChain) GetBlockByHash (hash *types.Hash) (*ledger.SnapshotBlock, error) {
	return sc.scAccess.GetBlockByHash(hash)
}

func (sc *SnapshotChain) GetBlockByHeight (height *big.Int) (*ledger.SnapshotBlock, error) {
	return sc.scAccess.GetLatestBlock()
}