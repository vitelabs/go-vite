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
	"github.com/vitelabs/go-vite/log15"
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
	STATUS_DOWNLOADING
)

const (
	DOWNLOADTIMES_LIMIT = 30
)

type SnapshotChain struct {
	// Handle block
	vite     Vite
	scAccess *access.SnapshotChainAccess
	acAccess *access.AccountChainAccess
	aAccess  *access.AccountAccess

	status              int // 0 init, 1 first syncing, 2 normal , 3 forking,
	syncDownChannelList []chan<- int

	downloadId       uint64
	downloadTryTimes int

	handleSendLock sync.Mutex
	pool           *pending.SnapshotchainPool
}

func NewSnapshotChain(vite Vite) *SnapshotChain {
	sc := &SnapshotChain{
		vite:     vite,
		scAccess: access.GetSnapshotChainAccess(),
		acAccess: access.GetAccountChainAccess(),
		aAccess:  access.GetAccountAccess(),

		status: 0,
	}

	sc.pool = pending.NewSnapshotchainPool(sc)
	return sc
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
func (sc *SnapshotChain) HandleGetBlocks(msg *protoTypes.GetSnapshotBlocksMsg, peer *protoTypes.Peer, id uint64) error {
	go func() {
		scLog.Info("SnapshotChain HandleGetBlocks: GetBlocksFromOrigin, msg.Origin is " + msg.Origin.String() +
			", msg.Count is " + strconv.Itoa(int(msg.Count)) + ", msg.Forward is " + strconv.FormatBool(msg.Forward))

		blocks, err := sc.scAccess.GetBlocksFromOrigin(&msg.Origin, msg.Count, msg.Forward)
		if err != nil {
			scLog.Error("SnapshotChain HandleGetBlocks: Error is " + err.Error())
		}

		scLog.Info("SnapshotChain HandleGetBlocks: Send " + strconv.Itoa(len(blocks)) + " snapshot blocks to network.")
		var neterr error

		if blocks != nil {
			neterr = sc.vite.Pm().SendMsg(peer, &protoTypes.Msg{
				Code:    protoTypes.SnapshotBlocksMsgCode,
				Payload: &blocks,
				Id:      id,
			})
		} else {
			neterr = sc.vite.Pm().SendMsg(peer, &protoTypes.Msg{
				Code:    protoTypes.SnapshotBlocksMsgCode,
				Payload: nil,
				Id:      id,
			})
		}

		if neterr != nil {
			scLog.Info("SnapshotChain HandleGetBlocks: Send snapshot blocks to network failed, error is " + neterr.Error())
		}

	}()
	return nil
}

func (sc *SnapshotChain) ProcessBlock(block *ledger.SnapshotBlock, peer *protoTypes.Peer, id uint64) int {
	globalRWMutex.RLock()
	defer globalRWMutex.RUnlock()

	// Timeout
	if sc.status == STATUS_DOWNLOADING ||
		sc.status == STATUS_FORKING {
		sc.downloadTryTimes++
		if sc.downloadTryTimes < DOWNLOADTIMES_LIMIT {
			return pending.NO_DISCARD
		}
		sc.downloadTryTimes = 0
		sc.status = STATUS_RUNNING
	}

	scLog.Info("SnapshotChain HandleSendBlocks: Start process block " + block.Hash.String() + ", block height is " + block.Height.String())
	if block.PublicKey == nil || block.Hash == nil || block.Signature == nil {
		scLog.Info("SnapshotChain HandleSendBlocks: discard block  , because block.PublicKey or block.Hash or block.Signature is nil.")
		// Discard block
		return pending.DISCARD
	}

	r, err := sc.vite.Verifier().Verify(sc, block)

	if !r {
		if err != nil {
			scLog.Error("SnapshotChain HandleSendBlocks: Verify failed.", " err", err)
		}
		scLog.Error("SnapshotChain HandleSendBlocks: Verify failed.")
		// Discard block
		return pending.DISCARD
	}

	// Verify hash
	computedHash, err := block.ComputeHash()
	if err != nil {
		scLog.Error(err.Error())
		// Discard block
		return pending.DISCARD
	}

	if !bytes.Equal(computedHash.Bytes(), block.Hash.Bytes()) {
		// Discard the block.
		scLog.Info("SnapshotChain HandleSendBlocks: discard block " + block.Hash.String() + ", because the computed hash is " + computedHash.String() + " and the block hash is " + block.Hash.String())
		return pending.DISCARD
	}

	// Verify signature
	isVerified, verifyErr := crypto.VerifySig(block.PublicKey, block.Hash.Bytes(), block.Signature)
	if !isVerified || verifyErr != nil {
		// Discard the block.
		scLog.Info("SnapshotChain HandleSendBlocks: discard block " + block.Hash.String() + ", because verify signature failed.")
		return pending.DISCARD
	}

	wbErr := sc.scAccess.WriteBlock(block, nil)
	if wbErr != nil {
		switch wbErr.(type) {
		case *access.ScWriteError:
			scWriteError := wbErr.(*access.ScWriteError)
			if scWriteError.Code == access.WscNeedSyncErr {
				needSyncData := scWriteError.Data.([]*access.WscNeedSyncErrData)

				scLog.Info("Need sync data. ", "need_sync_length", len(needSyncData))

				sc.vite.Ledger().Ac().Download(peer, needSyncData)

				scLog.Info("Sync data finished.")

				return pending.TRY_AGAIN
			} else if scWriteError.Code == access.WscPrevHashErr {
				preBlock := scWriteError.Data.(*ledger.SnapshotBlock)

				gap := &big.Int{}
				gap.Sub(block.Height, preBlock.Height)

				isFork := false
				needDeleteCount := big.NewInt(0)

				if gap.Cmp(big.NewInt(0)) <= 0 {
					if sc.scAccess.CheckExists(block.Hash) {
						// Need resolve fork
						isFork = true
						needDeleteCount.Abs(gap)
						needDeleteCount.Add(needDeleteCount, big.NewInt(1))
					} else {
						// Not fork, then discard
						return pending.DISCARD
					}
				} else if gap.Cmp(big.NewInt(1)) == 0 {
					// Need resolve fork
					isFork = true
				}

				if isFork {
					sc.status = STATUS_FORKING
					deleteCount := big.NewInt(3)

					deleteCount.Add(deleteCount, needDeleteCount)

					err := sc.scAccess.DeleteBlocks(preBlock.Hash, deleteCount.Uint64())
					if err != nil {
						scLog.Error("SnapshotChain.HandleSendBlocks: Delete failed, error is " + err.Error())
						return pending.DISCARD
					}
					gap.Add(gap, deleteCount)
				} else {
					sc.status = STATUS_DOWNLOADING
				}

				// Download fragment
				scLog.Info("SnapshotChain.HandleSendBlocks: start download, origin is " + block.Hash.String() +
					", count is " + gap.String() + ", forward is false")

				// Start download, Limit is 1000
				if gap.Cmp(big.NewInt(5000)) <= 0 {
					sc.download(peer, &protoTypes.Msg{
						Code: protoTypes.GetSnapshotBlocksMsgCode,
						Payload: &protoTypes.GetSnapshotBlocksMsg{
							Origin:  *block.Hash,
							Count:   gap.Uint64(),
							Forward: false,
						},
					})
				} else {
					latestBlock, err := sc.scAccess.GetLatestBlock()
					if err != nil {
						scLog.Info("SnapshotChain.HandleSendBlocks: GetLatestBlock failed, err is " + err.Error())
					} else {
						sc.download(peer, &protoTypes.Msg{
							Code: protoTypes.GetSnapshotBlocksMsgCode,
							Payload: &protoTypes.GetSnapshotBlocksMsg{
								Origin:  *latestBlock.Hash,
								Count:   5000,
								Forward: true,
							},
						})
					}
				}

				return pending.DISCARD
			}
		}

		// Let the pool discard the block.
		scLog.Info("SnapshotChain.HandleSendBlocks: write failed, error is " + wbErr.Error())
		return pending.DISCARD
	}

	if !sc.isFirstSyncDone() {
		syncInfo.CurrentHeight = block.Height

		if syncInfo.CurrentHeight.Cmp(syncInfo.TargetHeight) >= 0 {
			sc.onFirstSyncDown()
		}
	}

	return pending.DISCARD
}

// HandleBlockHash
func (sc *SnapshotChain) HandleSendBlocks(netMsg *protoTypes.SnapshotBlocksMsg, peer *protoTypes.Peer, id uint64) error {
	sc.handleSendLock.Lock()
	defer sc.handleSendLock.Unlock()

	if (sc.status == STATUS_DOWNLOADING ||
		sc.status == STATUS_FORKING) && id == sc.downloadId {
		sc.status = STATUS_RUNNING
	}

	msg := []*ledger.SnapshotBlock(*netMsg)
	scLog.Info("SnapshotChain.HandleSendBlocks: receive " + strconv.Itoa(len(msg)) + " blocks")
	sc.pool.Add(msg, peer, id)

	return nil
}

var syncInfo = &handler_interface.SyncInfo{}

func (sc *SnapshotChain) download(peer *protoTypes.Peer, msg *protoTypes.Msg) {
	sc.downloadId = getNewNetTaskId()
	// Set msg
	msg.Id = sc.downloadId

	sc.vite.Pm().SendMsg(peer, msg)
}

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

	// Download 1, trigger download process
	sc.download(peer, &protoTypes.Msg{
		Code: protoTypes.GetSnapshotBlocksMsgCode,
		Payload: &protoTypes.GetSnapshotBlocksMsg{
			Origin:  peer.Head,
			Count:   1,
			Forward: false,
		},
	})

	return nil
}

func (sc *SnapshotChain) SyncPeer(peer *protoTypes.Peer) {
	// Syncing done, modify in future
	defer sc.vite.Pm().SyncDone()
	if sc.status == STATUS_INIT {
		sc.status = STATUS_FIRST_SYNCING
	}

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
		scLog.Error(err.Error())

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
	globalRWMutex.Lock()
	defer globalRWMutex.Unlock()
	if sc.status != STATUS_RUNNING {
		err := errors.New("Node status is " + strconv.Itoa(sc.status) + ", can't mining")
		scLog.Error("SnapshotChain WriteMiningBlock: " + err.Error())
		return err
	}

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
		scLog.Error("WriteMiningBlock broadcast failed", "error", sendErr)
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
			scLog.Error(err.Error())
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

	return latestBlock, nil
}

func (sc *SnapshotChain) GetBlockByHash(hash *types.Hash) (*ledger.SnapshotBlock, error) {
	return sc.scAccess.GetBlockByHash(hash)
}

func (sc *SnapshotChain) GetBlockByHeight(height *big.Int) (*ledger.SnapshotBlock, error) {
	return sc.scAccess.GetBlockByHeight(height)
}
func (sc *SnapshotChain) isFirstSyncDone() bool {
	if sc.status >= STATUS_FIRST_SYNCING && syncInfo.CurrentHeight != nil && syncInfo.TargetHeight != nil {
		return  syncInfo.CurrentHeight.Cmp(syncInfo.TargetHeight) >= 0
	}
	return false
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
