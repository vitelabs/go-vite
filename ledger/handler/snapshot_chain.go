package handler

import (
	"bytes"
	"github.com/inconshreveable/log15"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/ledger/access"
	"github.com/vitelabs/go-vite/ledger/cache/pending"
	"github.com/vitelabs/go-vite/ledger/handler_interface"
	protoTypes "github.com/vitelabs/go-vite/protocols/types"
	"math/big"
	"strconv"
	"sync"
	"time"
)

var scLog = log15.New("module", "ledger/handler/snapshot_chain")

const (
	STATUS_INIT = iota
	STATUS_FIRST_SYNCING
	STATUS_RUNNING
	STATUS_FORKING
)

type SnapshotChain struct {
	// Handle block
	vite     Vite
	scAccess *access.SnapshotChainAccess
	acAccess *access.AccountChainAccess
	aAccess  *access.AccountAccess

	status              int // 0 init, 1 first syncing, 2 normal , 3 forking,
	syncDownChannelList []chan<- int
}

func NewSnapshotChain(vite Vite) *SnapshotChain {
	return &SnapshotChain{
		vite:     vite,
		scAccess: access.GetSnapshotChainAccess(),
		acAccess: access.GetAccountChainAccess(),
		aAccess:  access.GetAccountAccess(),

		status: 0,
	}
}

var registerChannelLock sync.Mutex

func (sc *SnapshotChain) registerFirstSyncDown(firstSyncDownChan chan<- int) {
	registerChannelLock.Lock()
	sc.syncDownChannelList = append(sc.syncDownChannelList, firstSyncDownChan)
	registerChannelLock.Unlock()

	if sc.isFirstSyncDone() {
		sc.onFirstSyncDown()
	}
}

func (sc *SnapshotChain) onFirstSyncDown() {
	registerChannelLock.Lock()

	if sc.status == STATUS_FIRST_SYNCING {
		sc.status = STATUS_RUNNING
	}

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
		scLog.Info("SnapshotChain HandleGetBlocks: GetBlocksFromOrigin, msg.Origin is " + msg.Origin.String() +
			", msg.Count is " + strconv.Itoa(int(msg.Count)) + ", msg.Forward is " + strconv.FormatBool(msg.Forward))

		blocks, err := sc.scAccess.GetBlocksFromOrigin(&msg.Origin, msg.Count, msg.Forward)
		if err != nil {
			scLog.Error("SnapshotChain HandleGetBlocks: Error is " + err.Error())
			return
		}

		scLog.Info("SnapshotChain HandleGetBlocks: Send " + strconv.Itoa(len(blocks)) + " snapshot blocks to network.")
		neterr := sc.vite.Pm().SendMsg(peer, &protoTypes.Msg{
			Code:    protoTypes.SnapshotBlocksMsgCode,
			Payload: &blocks,
		})

		if neterr != nil {
			scLog.Info("SnapshotChain HandleGetBlocks: Send snapshot blocks to network failed, error is " + neterr.Error())
		}

	}()
	return nil
}

var pendingPool *pending.SnapshotchainPool

// Fixme
var currentMaxHeight = big.NewInt(0)

// HandleBlockHash
func (sc *SnapshotChain) HandleSendBlocks(msg *protoTypes.SnapshotBlocksMsg, peer *protoTypes.Peer) error {
	if pendingPool == nil {
		scLog.Info("SnapshotChain HandleSendBlocks: Init pending.SnapshotchainPool.")
		pendingPool = pending.NewSnapshotchainPool(func(block *ledger.SnapshotBlock) bool {
			globalRWMutex.RLock()
			defer globalRWMutex.RUnlock()

			scLog.Info("SnapshotChain HandleSendBlocks: Start process block " + block.Hash.String() + ", block height is " + block.Height.String())
			if block.PublicKey == nil || block.Hash == nil || block.Signature == nil {
				// Let the pool discard the block.
				scLog.Info("SnapshotChain HandleSendBlocks: discard block  , because block.PublicKey or block.Hash or block.Signature is nil.")
				return true
			}

			r, err := sc.vite.Verifier().Verify(sc, block)

			if !r {
				if err != nil {
					scLog.Info("SnapshotChain HandleSendBlocks: Verify failed. Error is " + err.Error())
				}
				scLog.Info("SnapshotChain HandleSendBlocks: Verify failed.")
				// Let the pool discard the block.
				return true
			}

			// Verify hash
			computedHash, err := block.ComputeHash()
			if err != nil {
				// Discard the block.
				scLog.Info(err.Error())
				return true
			}

			if !bytes.Equal(computedHash.Bytes(), block.Hash.Bytes()) {
				// Discard the block.
				scLog.Info("SnapshotChain HandleSendBlocks: discard block " + block.Hash.String() + ", because the computed hash is " + computedHash.String() + " and the block hash is " + block.Hash.String())
				return true
			}

			// Verify signature
			isVerified, verifyErr := crypto.VerifySig(block.PublicKey, block.Hash.Bytes(), block.Signature)
			if !isVerified || verifyErr != nil {
				// Let the pool discard the block.
				scLog.Info("SnapshotChain HandleSendBlocks: discard block " + block.Hash.String() + ", because verify signature failed.")
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

						if gap.Cmp(big.NewInt(1)) > 0 {
							// Download snapshot block

							scLog.Info("SnapshotChain.HandleSendBlocks: Download snapshot blocks." +
								"Current block height is " + preBlock.Height.String() + ", and target block height is " +
								block.Height.String())

							currentMaxHeight = block.Height

							sc.vite.Pm().SendMsg(peer, &protoTypes.Msg{
								Code: protoTypes.GetSnapshotBlocksMsgCode,
								Payload: &protoTypes.GetSnapshotBlocksMsg{
									Origin:  *block.Hash,
									Count:   gap.Uint64(),
									Forward: false,
								},
							})

							return false
						} else {
							maxBlock := pendingPool.MaxBlock()
							if maxBlock == nil {
								return true
							}

							maxGap := &big.Int{}

							maxGap.Sub(maxBlock.Height, preBlock.Height)

							if maxGap.Cmp(big.NewInt(1)) <= 0 {
								deleteCount := 10
								if sc.status == STATUS_RUNNING {
									sc.status = STATUS_FORKING
								}

								err := sc.scAccess.DeleteBlocks(preBlock.Hash, uint64(deleteCount))
								if err != nil {
									scLog.Error("SnapshotChain.HandleSendBlocks: Delete failed, error is " + err.Error())
									return true
								}

								// Clear pending pool
								pendingPool.Clear()
							}
							// Let the pool discard the block.
							return true
						}

					}
				}

				// Let the pool discard the block.
				scLog.Info("SnapshotChain.HandleSendBlocks: write failed, error is " + wbErr.Error())
				return true
			}

			if block.Height.Cmp(currentMaxHeight) >= 0 {
				currentMaxHeight = block.Height
				if sc.status > STATUS_RUNNING {
					sc.status = STATUS_RUNNING
				}
			}

			if !sc.isFirstSyncDone() {
				syncInfo.CurrentHeight = block.Height

				if syncInfo.CurrentHeight.Cmp(syncInfo.TargetHeight) >= 0 {
					sc.onFirstSyncDown()
				}
			}

			return true
		})
	}

	pendingPool.Add(*msg)
	scLog.Info("SnapshotChain.HandleSendBlocks: receive " + strconv.Itoa(len(*msg)) + " blocks")

	return nil
}

var syncInfo = &handler_interface.SyncInfo{}

func (sc *SnapshotChain) syncPeer(peer *protoTypes.Peer) error {
	latestBlock, err := sc.scAccess.GetLatestBlock()
	if err != nil {
		return err
	}

	if !sc.isFirstSyncDone() {
		if syncInfo.BeginHeight == nil {
			syncInfo.BeginHeight = latestBlock.Height
		}

		scLog.Info("syncPeer: syncInfo.BeginHeight is " + syncInfo.BeginHeight.String())
		syncInfo.TargetHeight = peer.Height
		syncInfo.CurrentHeight = syncInfo.BeginHeight

		scLog.Info("syncPeer: syncInfo.TargetHeight is " + peer.Height.String())
	}

	count := &big.Int{}
	count.Sub(peer.Height, latestBlock.Height)
	count.Add(count, big.NewInt(1))

	sc.vite.Pm().SendMsg(peer, &protoTypes.Msg{
		Code: protoTypes.GetSnapshotBlocksMsgCode,
		Payload: &protoTypes.GetSnapshotBlocksMsg{
			Origin:  peer.Head,
			Count:   count.Uint64(),
			Forward: false,
		},
	})

	return nil
}

func (sc *SnapshotChain) SyncPeer(peer *protoTypes.Peer) {
	// Syncing done, modify in future
	defer sc.vite.Pm().SyncDone()

	sc.status = STATUS_FIRST_SYNCING

	if peer == nil {
		if !sc.isFirstSyncDone() {
			sc.onFirstSyncDown()
			scLog.Info("SnapshotChain.SyncPeer: sync finished.")
		}
		return
	}
	// Do syncing
	scLog.Info("SyncPeer: start sync peer.")
	err := sc.syncPeer(peer)

	if err != nil {
		scLog.Info(err.Error())

		// If the first syncing goes wrong, try to sync again.
		go func() {
			time.Sleep(time.Duration(1000))
			sc.vite.Pm().Sync()
		}()
	}
}

func (sc *SnapshotChain) GetConfirmBlock(block *ledger.AccountBlock) (*ledger.SnapshotBlock, error) {
	return sc.acAccess.GetConfirmBlock(block)
}

func (sc *SnapshotChain) GetConfirmTimes(snapshotBlock *ledger.SnapshotBlock) (*big.Int, error) {
	return sc.acAccess.GetConfirmTimes(snapshotBlock)
}

func (sc *SnapshotChain) WriteMiningBlock(block *ledger.SnapshotBlock) error {
	if sc.status != STATUS_RUNNING {
		err := errors.New("Node status is " + strconv.Itoa(sc.status) + ", can't mining")
		scLog.Error("SnapshotChain WriteMiningBlock: " + err.Error())
		return err
	}

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

	scLog.Info("SnapshotChain WriteMiningBlock: create a new snapshot block.")
	err := sc.scAccess.WriteBlock(block, func(block *ledger.SnapshotBlock) (*ledger.SnapshotBlock, error) {
		var signErr error

		block.Signature, block.PublicKey, signErr =
			sc.vite.WalletManager().KeystoreManager.SignData(*block.Producer, block.Hash.Bytes())

		return block, signErr
	})

	if err != nil {
		scLog.Info("SnapshotChain WriteMiningBlock: Write a new snapshot block failed. Error is " + err.Error())
		return err
	}

	// Broadcast
	scLog.Info("SnapshotChain WriteMiningBlock: Broadcast a new snapshot block.")
	sendErr := sc.vite.Pm().SendMsg(nil, &protoTypes.Msg{
		Code:    protoTypes.SnapshotBlocksMsgCode,
		Payload: &protoTypes.SnapshotBlocksMsg{block},
	})

	if sendErr != nil {
		scLog.Info("WriteMiningBlock broadcast failed, error is " + sendErr.Error())
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
			scLog.Info(err.Error())
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
	latestBlock, err := sc.scAccess.GetLatestBlock()
	if err != nil {
		return nil, err
	}

	if latestBlock.Height.Cmp(currentMaxHeight) >= 0 {
		currentMaxHeight = latestBlock.Height
	}
	return latestBlock, nil
}

func (sc *SnapshotChain) GetBlockByHash(hash *types.Hash) (*ledger.SnapshotBlock, error) {
	return sc.scAccess.GetBlockByHash(hash)
}

func (sc *SnapshotChain) GetBlockByHeight(height *big.Int) (*ledger.SnapshotBlock, error) {
	return sc.scAccess.GetBlockByHeight(height)
}
func (sc *SnapshotChain) isFirstSyncDone() bool {
	judge := sc.status > STATUS_FIRST_SYNCING
	return judge
}

func (sc *SnapshotChain) isFirstSyncStart() bool {
	judge := sc.status > STATUS_INIT

	return judge
}

func (sc *SnapshotChain) GetFirstSyncInfo() *handler_interface.SyncInfo {
	return &handler_interface.SyncInfo{
		BeginHeight:      syncInfo.BeginHeight,
		CurrentHeight:    syncInfo.CurrentHeight,
		TargetHeight:     syncInfo.TargetHeight,
		IsFirstSyncDone:  sc.isFirstSyncDone(),
		IsFirstSyncStart: sc.isFirstSyncStart(),
	}
}
