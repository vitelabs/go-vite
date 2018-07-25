package handler

import (
	"bytes"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/ledger/access"
	"github.com/vitelabs/go-vite/ledger/cache/pending"
	"github.com/vitelabs/go-vite/ledger/handler_interface"
	"github.com/vitelabs/go-vite/log"
	protoTypes "github.com/vitelabs/go-vite/protocols/types"
	"math/big"
	"strconv"
	"sync"
	"time"
)

type SnapshotChain struct {
	// Handle block
	vite     Vite
	scAccess *access.SnapshotChainAccess
	acAccess *access.AccountChainAccess
	aAccess  *access.AccountAccess

	syncDownChannelList []chan<- int
}

func NewSnapshotChain(vite Vite) *SnapshotChain {
	return &SnapshotChain{
		vite:     vite,
		scAccess: access.GetSnapshotChainAccess(),
		acAccess: access.GetAccountChainAccess(),
		aAccess:  access.GetAccountAccess(),
	}
}

var registerChannelLock sync.Mutex

func (sc *SnapshotChain) registerFirstSyncDown(firstSyncDownChan chan<- int) {
	registerChannelLock.Lock()
	sc.syncDownChannelList = append(sc.syncDownChannelList, firstSyncDownChan)
	registerChannelLock.Unlock()

	if syncInfo.IsFirstSyncDone {
		sc.onFirstSyncDown()
	}
}

func (sc *SnapshotChain) onFirstSyncDown() {
	registerChannelLock.Lock()
	syncInfo.IsFirstSyncDone = true
	go func() {
		defer registerChannelLock.Unlock()
		for _, syncDownChannel := range sc.syncDownChannelList {
			syncDownChannel <- 0
		}
		sc.syncDownChannelList = []chan<- int{}
	}()
}

// HandleGetBlock
func (sc *SnapshotChain) HandleGetBlocks(msg *protoTypes.GetSnapshotBlocksMsg, peer *protoTypes.Peer) error {
	go func() {
		blocks, err := sc.scAccess.GetBlocksFromOrigin(&msg.Origin, msg.Count, msg.Forward)
		if err != nil {
			log.Info(err.Error())
			return
		}

		log.Info("SnapshotChain HandleGetBlocks: Send " + strconv.Itoa(len(blocks)) + " snapshot blocks to network.")
		neterr := sc.vite.Pm().SendMsg(peer, &protoTypes.Msg{
			Code:    protoTypes.SnapshotBlocksMsgCode,
			Payload: &blocks,
		})

		if neterr != nil {
			log.Info("SnapshotChain HandleGetBlocks: Send snapshot blocks to network failed, error is " + neterr.Error())
		}

	}()
	return nil
}

var pendingPool *pending.SnapshotchainPool

// HandleBlockHash
func (sc *SnapshotChain) HandleSendBlocks(msg *protoTypes.SnapshotBlocksMsg, peer *protoTypes.Peer) error {
	if pendingPool == nil {
		log.Info("SnapshotChain HandleSendBlocks: Init pending.SnapshotchainPool.")
		pendingPool = pending.NewSnapshotchainPool(func(block *ledger.SnapshotBlock) bool {
			globalRWMutex.RLock()
			defer globalRWMutex.RUnlock()

			log.Info("SnapshotChain HandleSendBlocks: Start process block " + block.Hash.String())
			if block.PublicKey == nil || block.Hash == nil || block.Signature == nil {
				// Let the pool discard the block.
				log.Info("SnapshotChain HandleSendBlocks: discard block " + block.Hash.String() + ", because block.PublicKey or block.Hash or block.Signature is nil.")
				return true
			}

			r, err := sc.vite.Verifier().Verify(sc, block)

			if !r {
				if err != nil {
					log.Info("SnapshotChain HandleSendBlocks: Verify failed. Error is " + err.Error())
				}
				log.Info("SnapshotChain HandleSendBlocks: Verify failed.")
				// Let the pool discard the block.
				return true
			}

			// Verify hash
			computedHash, err := block.ComputeHash()
			if err != nil {
				// Discard the block.
				log.Info(err.Error())
				return true
			}

			if !bytes.Equal(computedHash.Bytes(), block.Hash.Bytes()) {
				// Discard the block.
				log.Info("SnapshotChain HandleSendBlocks: discard block " + block.Hash.String() + ", because the computed hash is " + computedHash.String() + " and the block hash is " + block.Hash.String())
				return true
			}

			// Verify signature
			isVerified, verifyErr := crypto.VerifySig(block.PublicKey, block.Hash.Bytes(), block.Signature)
			if !isVerified || verifyErr != nil {
				// Let the pool discard the block.
				log.Info("SnapshotChain HandleSendBlocks: discard block " + block.Hash.String() + ", because verify signature failed.")
				return true
			}

			wbErr := sc.scAccess.WriteBlock(block, nil)
			if wbErr != nil {
				switch wbErr.(type) {
				case *access.ScWriteError:
					scWriteError := wbErr.(*access.ScWriteError)
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
							if item.TargetBlockHeight.Cmp(currentBlockHeight) <= 0 {
								// Don't sync when the height of target block is lower
								continue
							}

							gap := &big.Int{}
							gap = gap.Sub(item.TargetBlockHeight, currentBlockHeight)

							sc.vite.Pm().SendMsg(peer, &protoTypes.Msg{
								Code: protoTypes.GetAccountBlocksMsgCode,
								Payload: &protoTypes.GetAccountBlocksMsg{
									Origin:  *item.TargetBlockHash,
									Count:   gap.Uint64(),
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

			if !syncInfo.IsFirstSyncDone {
				syncInfo.CurrentHeight = block.Height

				if syncInfo.CurrentHeight.Cmp(syncInfo.TargetHeight) >= 0 {
					sc.onFirstSyncDown()
				}
			}

			return true
		})
	}

	pendingPool.Add(*msg)
	log.Info("SnapshotChain.HandleSendBlocks: receive " + strconv.Itoa(len(*msg)) + " blocks")

	return nil
}

var syncInfo = &handler_interface.SyncInfo{
	IsFirstSyncDone:  false,
	IsFirstSyncStart: false,
}

func (sc *SnapshotChain) syncPeer(peer *protoTypes.Peer) error {
	latestBlock, err := sc.scAccess.GetLatestBlock()
	if err != nil {
		return err
	}

	if !syncInfo.IsFirstSyncDone {
		syncInfo.IsFirstSyncStart = true
		if syncInfo.BeginHeight == nil {
			syncInfo.BeginHeight = latestBlock.Height
		}

		log.Info("syncPeer: syncInfo.BeginHeight is " + syncInfo.BeginHeight.String())
		syncInfo.TargetHeight = peer.Height
		syncInfo.CurrentHeight = syncInfo.BeginHeight
		log.Info("syncPeer: syncInfo.TargetHeight is " + peer.Height.String())
	}

	count := &big.Int{}
	count.Sub(peer.Height, latestBlock.Height)
	count.Add(count, big.NewInt(1))

	sc.vite.Pm().SendMsg(peer, &protoTypes.Msg{
		Code: protoTypes.GetSnapshotBlocksMsgCode,
		Payload: &protoTypes.GetSnapshotBlocksMsg{
			Origin:  *latestBlock.Hash,
			Count:   count.Uint64(),
			Forward: true,
		},
	})

	return nil
}

func (sc *SnapshotChain) SyncPeer(peer *protoTypes.Peer) {
	// Syncing done, modify in future
	defer sc.vite.Pm().SyncDone()
	if peer == nil {
		if !syncInfo.IsFirstSyncDone {
			sc.onFirstSyncDown()
			log.Info("SnapshotChain.SyncPeer: sync finished.")
		}
		return
	}
	// Do syncing
	log.Info("SyncPeer: start sync peer.")
	err := sc.syncPeer(peer)

	if err != nil {
		log.Info(err.Error())

		// If the first syncing goes wrong, try to sync again.
		go func() {
			time.Sleep(time.Duration(1000))
			sc.vite.Pm().Sync()
		}()
	}
}

func (sc *SnapshotChain) WriteMiningBlock(block *ledger.SnapshotBlock) error {
	globalRWMutex.Lock()
	defer globalRWMutex.Unlock()

	latestBlock, glbErr := sc.GetLatestBlock()
	if glbErr != nil {
		return errors.Wrap(glbErr, "WriteMiningBlock")
	}
	var gnsErr error
	block.Snapshot, gnsErr = sc.getNeedSnapshot()

	if gnsErr != nil {
		return errors.Wrap(glbErr, "WriteMiningBlock")
	}

	block.PrevHash = latestBlock.Hash
	block.Amount = big.NewInt(0)

	log.Info("SnapshotChain WriteMiningBlock: create a new snapshot block.")
	err := sc.scAccess.WriteBlock(block, func(block *ledger.SnapshotBlock) (*ledger.SnapshotBlock, error) {
		var signErr error

		block.Signature, block.PublicKey, signErr =
			sc.vite.WalletManager().KeystoreManager.SignData(*block.Producer, block.Hash.Bytes())

		return block, signErr
	})

	if err != nil {
		log.Info("SnapshotChain WriteMiningBlock: Write a new snapshot block failed. Error is " + err.Error())
		return err
	}

	// Broadcast
	log.Info("SnapshotChain WriteMiningBlock: Broadcast a new snapshot block.")
	sendErr := sc.vite.Pm().SendMsg(nil, &protoTypes.Msg{
		Code:    protoTypes.SnapshotBlocksMsgCode,
		Payload: &protoTypes.SnapshotBlocksMsg{block},
	})

	if sendErr != nil {
		log.Info("WriteMiningBlock broadcast failed, error is " + sendErr.Error())
		return sendErr
	}

	return nil
}

func (sc *SnapshotChain) getNeedSnapshot() (map[string]*ledger.SnapshotItem, error) {
	accountAddressList, err := sc.aAccess.GetAccountList()
	if err != nil {
		return nil, err
	}

	// Scan all accounts. Optimize in the future.
	needSnapshot := make(map[string]*ledger.SnapshotItem)

	for _, accountAddress := range accountAddressList {
		latestBlock, err := sc.acAccess.GetLatestBlockByAccountAddress(accountAddress)

		if latestBlock == nil || latestBlock.Meta.IsSnapshotted {
			continue
		}

		latestBlock.AccountAddress = accountAddress
		if err != nil {
			log.Info(err.Error())
			continue
		}

		needSnapshot[latestBlock.AccountAddress.Hex()] = &ledger.SnapshotItem{
			AccountBlockHeight: latestBlock.Meta.Height,
			AccountBlockHash:   latestBlock.Hash,
		}
	}

	return needSnapshot, nil
}

func (sc *SnapshotChain) GetLatestBlock() (*ledger.SnapshotBlock, error) {
	return sc.scAccess.GetLatestBlock()
}

func (sc *SnapshotChain) GetBlockByHash(hash *types.Hash) (*ledger.SnapshotBlock, error) {
	return sc.scAccess.GetBlockByHash(hash)
}

func (sc *SnapshotChain) GetBlockByHeight(height *big.Int) (*ledger.SnapshotBlock, error) {
	return sc.scAccess.GetBlockByHeight(height)
}

func (sc *SnapshotChain) GetFirstSyncInfo() *handler_interface.SyncInfo {
	return syncInfo
}
