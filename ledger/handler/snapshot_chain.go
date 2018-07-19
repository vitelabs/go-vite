package handler

import (
	"github.com/vitelabs/go-vite/protocols"
	"github.com/vitelabs/go-vite/ledger/access"
	"github.com/vitelabs/go-vite/ledger"
	"math/big"
	"github.com/vitelabs/go-vite/common/types"
	"log"
)

type SyncInfo struct {
	BeginHeight *big.Int
	TargetHeight *big.Int
	CurrentHeight *big.Int
}



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
		blocks, err := sc.scAccess.GetBlocksFromOrigin(&msg.Origin, msg.Count, msg.Forward)
		if err != nil {
			log.Println(err)
			return
		}

		sc.vite.Pm().SendMsg(peer, &protocols.Msg{
			Code: protocols.SnapshotBlocksMsgCode,
			Payload: blocks,
		})
	}()
	return nil
}

//func sortSnapshotBlocks (msg protocols.SnapshotBlocksMsg) (protocols.SnapshotBlocksMsg) {
//	newMsg :=  protocols.SnapshotBlocksMsg{}
//	for e := range msg {
//
//	}
//}

// HandleBlockHash
func (sc *SnapshotChain) HandleSendBlocks (msg protocols.SnapshotBlocksMsg, peer *protocols.Peer) error {
	go func() {
		globalRWMutex.RLock()
		defer globalRWMutex.RUnlock()

		for _, block := range msg {
			err := sc.scAccess.WriteBlock(block)
			if err != nil {
				log.Println(err)
			}


		}
	}()

	return nil
}

var firstSyncDone = false
var syncInfo = &SyncInfo{}

func (sc *SnapshotChain) syncPeer (peer *protocols.Peer) error {
	latestBlock, err := sc.scAccess.GetLatestBlock()
	if err != nil {
		return err
	}


	if !firstSyncDone {
		if syncInfo.BeginHeight == nil {
			syncInfo.BeginHeight = latestBlock.Height
		}
		syncInfo.TargetHeight = peer.Height
	}


	count := &big.Int{}
	count.Sub(peer.Height, latestBlock.Height)

	sc.vite.Pm().SendMsg(peer, &protocols.Msg{
		Code: protocols.GetSnapshotBlocksMsgCode,
		Payload: &protocols.GetSnapshotBlocksMsg{
			Origin: *latestBlock.Hash,
			Count: count.Uint64(),
			Forward: true,
		},
	})


	return nil
}

func (sc *SnapshotChain) SyncPeer (peer *protocols.Peer) {
	// Do syncing
	err := sc.syncPeer(peer)


	// Syncing done
	sc.vite.Pm().SyncDone()

	if err != nil {
		log.Println(err)
		// If the first syncing goes wrong, try to sync again.
		if !firstSyncDone {
			sc.vite.Pm().Sync()
		}
	} else {
		firstSyncDone = true
	}


}



func (sc *SnapshotChain) WriteMiningBlock () error {
	globalRWMutex.RLock()
	defer globalRWMutex.RUnlock()


	return nil
}

func (sc *SnapshotChain) StopAllWrite () error {
	globalRWMutex.Lock()
	return nil
}

func (sc *SnapshotChain) StartAllWrite () error {
	globalRWMutex.Unlock()
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

func (sc *SnapshotChain) GetFirstSyncInfo () (*SyncInfo) {
	return syncInfo
}