package handler

import (
	protoTypes "github.com/vitelabs/go-vite/protocols/types"
	"github.com/vitelabs/go-vite/ledger/access"
	"github.com/vitelabs/go-vite/ledger"
	"math/big"
	"github.com/vitelabs/go-vite/common/types"
	"log"
	"github.com/vitelabs/go-vite/ledger/cache/pending"
	"time"
	"github.com/vitelabs/go-vite/ledger/handler_interface"
	"github.com/vitelabs/go-vite/crypto"
	"bytes"
)



type SnapshotChain struct {
	// Handle block
	vite Vite
	scAccess *access.SnapshotChainAccess
	acAccess *access.AccountChainAccess
	aAccess *access.AccountAccess
}

func NewSnapshotChain (vite Vite) (*SnapshotChain) {
	return &SnapshotChain{
		vite: vite,
		scAccess: access.GetSnapshotChainAccess(),
	}
}

// HandleGetBlock
func (sc *SnapshotChain) HandleGetBlocks (msg *protoTypes.GetSnapshotBlocksMsg, peer *protoTypes.Peer) error {
	go func() {
		blocks, err := sc.scAccess.GetBlocksFromOrigin(&msg.Origin, msg.Count, msg.Forward)
		if err != nil {
			log.Println(err)
			return
		}

		sc.vite.Pm().SendMsg(peer, &protoTypes.Msg{
			Code: protoTypes.SnapshotBlocksMsgCode,
			Payload: blocks,
		})
	}()
	return nil
}

var pendingPool *pending.SnapshotchainPool

// HandleBlockHash
func (sc *SnapshotChain) HandleSendBlocks (msg *protoTypes.SnapshotBlocksMsg, peer *protoTypes.Peer) error {
	if pendingPool == nil {
		pendingPool = pending.NewSnapshotchainPool(func (block *ledger.SnapshotBlock) bool {
			globalRWMutex.RLock()
			defer globalRWMutex.RUnlock()

			if block.PublicKey == nil || block.Hash == nil || block.Signature == nil {
				// Let the pool discard the block.
				return true
			}

			// Verify hash
			computedHash, err := block.ComputeHash()
			if err != nil {
				// Discard the block.
				log.Println(err)
				return true
			}

			if !bytes.Equal(computedHash.Bytes(), block.Hash.Bytes()){
				// Discard the block.
				log.Println(err)
				return true
			}

			// Verify signature
			isVerified, verifyErr := crypto.VerifySig(block.PublicKey, block.Hash.Bytes(), block.Signature)
			if !isVerified || verifyErr != nil{
				// Let the pool discard the block.
				return true
			}

			wbErr := sc.scAccess.WriteBlock(block, nil)

			if wbErr != nil {
				log.Println(wbErr)

				switch wbErr.(type) {
				case access.ScWriteError:
					scWriteError := err.(access.ScWriteError)
					if scWriteError.Code == access.WscNeedSyncErr {
						needSyncData := scWriteError.Data.([]*access.WscNeedSyncErrData)
						for _, item := range needSyncData {
							latestBlock, err := sc.acAccess.GetLatestBlockByAccountAddress(item.AccountAddress)
							if err != nil {
								// Sync in the next time
								continue
							}

							currentBlockHeight := big.NewInt(0)
							if latestBlock != nil {
								currentBlockHeight = latestBlock.Meta.Height
							}
							if item.TargetBlockHeight.Cmp(currentBlockHeight) <= 0{
								// Don't sync when the height of target block is lower
								continue
							}

							gap := &big.Int{}
							gap = gap.Sub(item.TargetBlockHeight, currentBlockHeight)

							sc.vite.Pm().SendMsg(peer, &protoTypes.Msg{
								Code: protoTypes.GetAccountBlocksMsgCode,
								Payload: &protoTypes.GetAccountBlocksMsg{
									Origin: *item.TargetBlockHash,
									Count: gap.Uint64(),
									Forward: false,
								},
							})
						}
						return false
					} else if scWriteError.Code == access.WscPrevHashErr {
						preBlock := scWriteError.Data.(*ledger.SnapshotBlock)

						gap := &big.Int{}
						gap.Sub(block.Height, preBlock.Height)

						if gap.Cmp(big.NewInt(1)) <= 0 {
							// Let the pool discard the block.
							return true
						}
						return false
					}
				}

				return false
			}

			if !firstSyncDone {
				syncInfo.CurrentHeight = block.Height
				if syncInfo.CurrentHeight.Cmp(syncInfo.TargetHeight) >= 0 {
					firstSyncDone = true
				}
			}

			return true
		})
	}

	pendingPool.Add(*msg)

	return nil
}

var firstSyncDone = false
var syncInfo = &handler_interface.SyncInfo{}

func (sc *SnapshotChain) syncPeer (peer *protoTypes.Peer) error {
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

	sc.vite.Pm().SendMsg(peer, &protoTypes.Msg {
		Code: protoTypes.GetSnapshotBlocksMsgCode,
		Payload: &protoTypes.GetSnapshotBlocksMsg{
			Origin: *latestBlock.Hash,
			Count: count.Uint64(),
			Forward: true,
		},
	})

	return nil
}

func (sc *SnapshotChain) SyncPeer (peer *protoTypes.Peer) {
	// Do syncing
	err := sc.syncPeer(peer)

	// Syncing done, modify in future
	sc.vite.Pm().SyncDone()

	if err != nil {
		log.Println(err)
		// If the first syncing goes wrong, try to sync again.
		go func() {
			time.Sleep(time.Duration(1000))
			sc.vite.Pm().Sync()
		}()
	}

}



func (sc *SnapshotChain) WriteMiningBlock (block *ledger.SnapshotBlock) error {
	globalRWMutex.RLock()
	defer globalRWMutex.RUnlock()

	err := sc.scAccess.WriteBlock(block, func(block *ledger.SnapshotBlock) (*ledger.SnapshotBlock, error) {
		var signErr error

		block.Signature, block.PublicKey, signErr =
				sc.vite.WalletManager().KeystoreManager.SignData(*block.Producer, block.Hash.Bytes())


		return block, signErr
	})

	if err != nil {
		return err
	}

	// Broadcast
	sc.vite.Pm().SendMsg(nil, &protoTypes.Msg {
		Code: protoTypes.SnapshotBlocksMsgCode,
		Payload: &protoTypes.SnapshotBlocksMsg{block},
	})
	return nil
}

func (sc *SnapshotChain) GetNeedSnapshot () ([]*ledger.AccountBlock, error) {
	accountAddressList, err := sc.aAccess.GetAccountList()
	if err != nil {
		return nil, err
	}

	// Scan all accounts. Optimize in the future.
	var needSnapshot []*ledger.AccountBlock
	for _, accountAddress := range accountAddressList {
		latestBlock, err := sc.acAccess.GetLatestBlockByAccountAddress(accountAddress)
		if err != nil {
			log.Println(err)
			continue
		}
		if !latestBlock.Meta.IsSnapshotted {
			needSnapshot = append(needSnapshot, latestBlock)
		}
	}

	return needSnapshot, nil
}

func (sc *SnapshotChain) StopAllWrite () {
	globalRWMutex.Lock()
}

func (sc *SnapshotChain) StartAllWrite () {
	globalRWMutex.Unlock()
}

func (sc *SnapshotChain) GetLatestBlock () (*ledger.SnapshotBlock, error) {
	return sc.scAccess.GetLatestBlock()
}

func (sc *SnapshotChain) GetBlockByHash (hash *types.Hash) (*ledger.SnapshotBlock, error) {
	return sc.scAccess.GetBlockByHash(hash)
}

func (sc *SnapshotChain) GetBlockByHeight (height *big.Int) (*ledger.SnapshotBlock, error) {
	return sc.scAccess.GetBlockByHeight(height)
}

func (sc *SnapshotChain) GetFirstSyncInfo () (*handler_interface.SyncInfo) {
	return syncInfo
}