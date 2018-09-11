package handler

import (
	"bytes"
	"errors"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/ledger/access"
	"github.com/vitelabs/go-vite/ledger/cache/pending"
	"github.com/vitelabs/go-vite/log15"
	protoTypes "github.com/vitelabs/go-vite/protocols/types"
	"math/big"
	"strconv"
	"sync"
	"time"
)

var acLog = log15.New("module", "ledger/handler/account_chain")

type downloadTask struct {
	downloadId uint64
	tryTime    int
	finished   chan int
}

type AccountChain struct {
	vite Vite
	// Handle block
	acAccess *access.AccountChainAccess
	aAccess  *access.AccountAccess
	scAccess *access.SnapshotChainAccess
	uAccess  *access.UnconfirmedAccess
	tAccess  *access.TokenAccess

	downloadLock      sync.Mutex
	downloadTasks     map[string]*downloadTask
	downloadIdAddress map[uint64]*types.Address

	pool *pending.AccountchainPool
}

func NewAccountChain(vite Vite) *AccountChain {
	ac := &AccountChain{
		vite:     vite,
		acAccess: access.GetAccountChainAccess(),
		aAccess:  access.GetAccountAccess(),
		scAccess: access.GetSnapshotChainAccess(),
		uAccess:  access.GetUnconfirmedAccess(),
		tAccess:  access.GetTokenAccess(),

		downloadTasks:     make(map[string]*downloadTask),
		downloadIdAddress: make(map[uint64]*types.Address),
	}
	ac.pool = pending.NewAccountchainPool(ac)
	ac.startDownloaderTimer()

	return ac
}

// HandleBlockHash
func (ac *AccountChain) HandleGetBlocks(msg *protoTypes.GetAccountBlocksMsg, peer *protoTypes.Peer, id uint64) error {
	go func() {
		blocks, err := ac.acAccess.GetBlocksFromOrigin(&msg.Origin, msg.Count, msg.Forward)
		if err != nil {
			acLog.Error(err.Error())
		}

		// send out
		acLog.Info("AccountChain.HandleGetBlocks: send " + strconv.Itoa(len(blocks)) + " blocks.")

		var neterr error
		if blocks != nil {
			neterr = ac.vite.Pm().SendMsg(peer, &protoTypes.Msg{
				Code:    protoTypes.AccountBlocksMsgCode,
				Payload: &blocks,
				Id:      id,
			})
		} else {
			neterr = ac.vite.Pm().SendMsg(peer, &protoTypes.Msg{
				Code:    protoTypes.SnapshotBlocksMsgCode,
				Payload: nil,
				Id:      id,
			})
		}

		if neterr != nil {
			scLog.Info("AccountChain HandleGetBlocks: Send account blocks to network failed, error is " + neterr.Error())
		}
	}()
	return nil
}

func (ac *AccountChain) ProcessBlock(block *ledger.AccountBlock) {
	globalRWMutex.RLock()
	defer globalRWMutex.RUnlock()

	if block.PublicKey == nil || block.Hash == nil || block.Signature == nil {
		// Discard the block.
		acLog.Info("AccountChain HandleSendBlocks: discard block, because block.PublicKey or block.Hash or block.Signature is nil.")
		return
	}
	// Verify hash
	computedHash, err := block.ComputeHash()

	if err != nil {
		// Discard the block.
		acLog.Error(err.Error())
		return
	}

	if !bytes.Equal(computedHash.Bytes(), block.Hash.Bytes()) {
		// Discard the block.
		acLog.Info("AccountChain HandleSendBlocks: discard block " + block.Hash.String() + ", because the computed hash is " + computedHash.String() + " and the block hash is " + block.Hash.String())
		return
	}
	// Verify signature
	isVerified, verifyErr := crypto.VerifySig(block.PublicKey, block.Hash.Bytes(), block.Signature)

	if verifyErr != nil || !isVerified {
		// Discard the block.
		acLog.Info("AccountChain HandleSendBlocks: discard block " + block.Hash.String() + ", because verify signature failed.")
		return
	}

	// Write block
	acLog.Info("AccountChain HandleSendBlocks: try write a block from network")
	writeErr := ac.acAccess.WriteBlock(block, nil)

	if writeErr != nil {
		acLog.Error("AccountChain HandleSendBlocks: Write error.", "Error", writeErr)
	}

	return
}

// HandleBlockHash
func (ac *AccountChain) HandleSendBlocks(msg *protoTypes.AccountBlocksMsg, peer *protoTypes.Peer, id uint64) error {
	ac.finishDownload(id)
	ac.pool.Add(*msg)

	return nil
}

func (ac *AccountChain) finishDownload(id uint64) {
	ac.downloadLock.Lock()
	defer ac.downloadLock.Unlock()

	address := ac.downloadIdAddress[id]
	if address == nil {
		return
	}

	if downloadTask, ok := ac.downloadTasks[address.String()]; ok {
		if downloadTask.downloadId == id {
			downloadTask.finished <- 1
		}
	}
}

func (ac *AccountChain) startDownloaderTimer() {
	go func() {
		for {
			ac.downloadLock.Lock()
			for _, downloadTask := range ac.downloadTasks {
				downloadTask.tryTime++
				if downloadTask.tryTime == 5 {
					downloadTask.finished <- 1
				}
			}
			ac.downloadLock.Unlock()
			turnInterval := time.Duration(2000)
			time.Sleep(turnInterval * time.Millisecond)
		}

	}()
}

func (ac *AccountChain) GetLatestBlock(addr *types.Address) (*ledger.AccountBlock, *types.GetError) {
	block, err := ac.acAccess.GetLatestBlockByAccountAddress(addr)
	if err != nil {
		return nil, &types.GetError{
			Code: 1,
			Err:  err,
		}
	}
	return block, nil
}

func (ac *AccountChain) Download(peer *protoTypes.Peer, needSyncData []*access.WscNeedSyncErrData) {
	for _, item := range needSyncData {
		ac.downloadLock.Lock()
		if _, ok := ac.downloadTasks[(*item).AccountAddress.String()]; ok {
			ac.downloadLock.Unlock()
			continue
		}
		ac.downloadLock.Unlock()

		latestBlock, err := ac.acAccess.GetLatestBlockByAccountAddress(item.AccountAddress)
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

		downloadId := getNewNetTaskId()
		ac.vite.Pm().SendMsg(peer, &protoTypes.Msg{
			Code: protoTypes.GetAccountBlocksMsgCode,
			Payload: &protoTypes.GetAccountBlocksMsg{
				Origin:  *item.TargetBlockHash,
				Count:   gap.Uint64(),
				Forward: false,
			},
			Id: downloadId,
		})

		ac.downloadLock.Lock()
		ac.downloadIdAddress[downloadId] = item.AccountAddress
		ac.downloadTasks[(*item).AccountAddress.String()] = &downloadTask{
			downloadId: downloadId,
			tryTime:    0,
			finished:   make(chan int, 2),
		}
		ac.downloadLock.Unlock()

	}

	for _, item := range needSyncData {
		key := (*item).AccountAddress.String()
		downloadTask := ac.downloadTasks[key]
		<-downloadTask.finished

		ac.downloadLock.Lock()
		delete(ac.downloadIdAddress, downloadTask.downloadId)
		delete(ac.downloadTasks, key)
		ac.downloadLock.Unlock()
	}
}

// AccAddr = account address
func (ac *AccountChain) GetAccountByAccAddr(addr *types.Address) (*ledger.AccountMeta, error) {
	return ac.aAccess.GetAccountMeta(addr)
}

// AccAddr = account address
func (ac *AccountChain) GetBlocksByAccAddr(addr *types.Address, index, num, count int) (ledger.AccountBlockList, *types.GetError) {
	return ac.acAccess.GetBlockListByAccountAddress(index, num, count, addr)
}

func (ac *AccountChain) CreateTx(block *ledger.AccountBlock) error {
	return ac.CreateTxWithPassphrase(block, "")
}

func (ac *AccountChain) GetBlocks(addr *types.Address, originBlockHash *types.Hash, count uint64) (ledger.AccountBlockList, *types.GetError) {
	if originBlockHash == nil {
		latestblock, err := ac.GetLatestBlock(addr)
		if err != nil {
			return nil, &types.GetError{
				Code: 1,
				Err:  err,
			}
		}

		if latestblock == nil {
			return nil, nil
		}
		originBlockHash = latestblock.Hash
	}
	blocks, err := ac.acAccess.GetBlocksFromOrigin(originBlockHash, count, false)
	if err != nil {
		return nil, &types.GetError{
			Code: 2,
			Err:  err,
		}
	}

	// Reverse
	for i, j := 0, len(blocks)-1; i < j; i, j = i+1, j-1 {
		blocks[i], blocks[j] = blocks[j], blocks[i]
	}

	return blocks, nil
}
func (ac *AccountChain) CreateTxWithPassphrase(block *ledger.AccountBlock, passphrase string) error {
	syncInfo := ac.vite.Ledger().Sc().GetFirstSyncInfo()
	if !syncInfo.IsFirstSyncDone {
		acLog.Error("Sync unfinished, so can't create transaction.")
		return errors.New("sync unfinished, so can't create transaction")
	}

	globalRWMutex.RLock()
	defer globalRWMutex.RUnlock()

	accountMeta, err := ac.aAccess.GetAccountMeta(block.AccountAddress)

	if block.IsSendBlock() {
		if err != nil || accountMeta == nil {
			err := errors.New("CreateTx failed, because account " + block.AccountAddress.String() + " doesn't found.")
			acLog.Error(err.Error())
			return err
		}
	} else {
		if err != nil && err != leveldb.ErrNotFound {
			err := errors.New("AccountChain CreateTx: get account meta failed, error is " + err.Error())
			acLog.Error(err.Error())
			return err
		}
	}

	acLog.Info("AccountChain CreateTx: get account meta success.")

	// Set prevHash
	if block.PrevHash == nil {
		latestBlock, err := ac.acAccess.GetLatestBlockByAccountAddress(block.AccountAddress)
		acLog.Info("AccountChain CreateTx: get latestBlock success.")

		if err != nil {
			return err
		}

		if latestBlock != nil {
			block.PrevHash = latestBlock.Hash
		}
	}

	// Set Snapshot Timestamp
	if block.SnapshotTimestamp == nil {
		currentSnapshotBlock, err := ac.scAccess.GetLatestBlock()
		if err != nil {
			return err
		}

		acLog.Info("AccountChain CreateTx: get currentSnapshotBlock success.")
		// Hack
		if currentSnapshotBlock.Height.Cmp(big.NewInt(20)) <= 0 {
			block.SnapshotTimestamp = currentSnapshotBlock.Hash
		} else {
			snapshotBlockHeight := &big.Int{}
			if currentSnapshotBlock.Height.Cmp(big.NewInt(40)) <= 0 {
				snapshotBlockHeight = big.NewInt(20)
			} else {
				snapshotBlockHeight.Sub(currentSnapshotBlock.Height, big.NewInt(20))
			}
			snapshotBlock, err := ac.scAccess.GetBlockByHeight(snapshotBlockHeight)
			if err != nil {
				return err
			}
			block.SnapshotTimestamp = snapshotBlock.Hash
		}
	}

	// Set Timestamp
	if block.Timestamp == 0 {
		block.Timestamp = uint64(time.Now().Unix())
	}

	// Set Pow params: Nounce、Difficulty
	if block.Nounce == nil {
		block.Nounce = []byte{0, 0, 0, 0, 0}
	}
	if block.Difficulty == nil {
		block.Difficulty = []byte{0, 0, 0, 0, 0}
	}
	if block.FAmount == nil {
		block.FAmount = big.NewInt(0)
	}

	// Set PublicKey
	if accountMeta != nil {
		block.PublicKey = accountMeta.PublicKey
	}

	acLog.Info("AccountChain CreateTx: start write block.")
	writeErr := ac.acAccess.WriteBlock(block, func(accountBlock *ledger.AccountBlock) (*ledger.AccountBlock, error) {
		var signErr error
		if passphrase == "" {
			accountBlock.Signature, accountBlock.PublicKey, signErr =
				ac.vite.WalletManager().KeystoreManager.SignData(*block.AccountAddress, block.Hash.Bytes())

		} else {
			accountBlock.Signature, accountBlock.PublicKey, signErr =
				ac.vite.WalletManager().KeystoreManager.SignDataWithPassphrase(*block.AccountAddress, passphrase, block.Hash.Bytes())
		}
		return accountBlock, signErr
	})

	if writeErr != nil {
		acLog.Error("AccountChain CreateTx: write block failed ", "err", writeErr)
		return writeErr.(*access.AcWriteError).Err
	}

	acLog.Info("AccountChain CreateTx: write block success.")

	// Broadcast
	sendErr := ac.vite.Pm().SendMsg(nil, &protoTypes.Msg{
		Code:    protoTypes.AccountBlocksMsgCode,
		Payload: &protoTypes.AccountBlocksMsg{block},
	})

	acLog.Info("AccountChain CreateTx: broadcast to network.")

	if sendErr != nil {
		acLog.Error("CreateTx broadcast failed", "err", sendErr)
		return sendErr
	}
	return nil
}
