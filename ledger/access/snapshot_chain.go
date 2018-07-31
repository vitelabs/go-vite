package access

import (
	"bytes"
	"errors"
	errors2 "github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vitedb"
	"log"
	"math/big"
	"sync"
)

type SnapshotChainAccess struct {
	store        *vitedb.SnapshotChain
	accountStore *vitedb.Account
	bwMutex      sync.RWMutex
}

var snapshotChainAccess *SnapshotChainAccess

func GetSnapshotChainAccess() *SnapshotChainAccess {
	if snapshotChainAccess == nil {
		snapshotChainAccess = &SnapshotChainAccess{
			store:        vitedb.GetSnapshotChain(),
			accountStore: vitedb.GetAccount(),
		}
	}
	return snapshotChainAccess
}

func (sca *SnapshotChainAccess) CheckAndCreateGenesisBlocks() {

	// Check snapshotGenesisBlock
	snapshotGenesisBlock, err := sca.store.GetBLockByHeight(big.NewInt(1))
	if err != nil && err != leveldb.ErrNotFound {
		log.Fatal(errors2.Wrap(err, "CheckAndCreateGenesisBlocks"))
	}

	if snapshotGenesisBlock == nil {
		sca.WriteGenesisBlock()
	} else {
		if ok := snapshotGenesisBlock.IsGenesisBlock(); !ok {
			// Fixme
			err := vitedb.ClearAndReNewDb(vitedb.DB_LEDGER)
			if err != nil {
				log.Fatal(errors2.Wrap(errors.New("SnapshotGenesisBlock is not valid. ClearAndReNewDb failed."), "CheckAndCreateGenesisBlocks"))
			}

			log.Println("CheckAndCreateGenesisBlocks: ClearAndReNewDb")
			sca.CheckAndCreateGenesisBlocks()
		}
	}

	accountChainAccess = GetAccountChainAccess()

	// Check accountGenesisBlockFirst
	accountGenesisBlockFirst, err := accountChainAccess.GetBlockByHash(ledger.AccountGenesisBlockFirst.Hash)
	if err != nil && err != leveldb.ErrNotFound {
		log.Fatal(errors2.Wrap(err, "CheckAccountGenesisBlockFirst"))
	}

	if accountGenesisBlockFirst == nil {
		accountChainAccess.WriteGenesisBlock()
	} else {
		if ok := accountGenesisBlockFirst.IsGenesisBlock(); !ok {
			log.Fatal(errors2.Wrap(errors.New("AccountGenesisBlockFirst is not valid."), "CheckAndCreateAccountGenesisBlockFirst"))
		}
	}

	// Check accountGenesisBlockSecond
	accountGenesisBlockSecond, err := accountChainAccess.GetBlockByHash(ledger.AccountGenesisBlockSecond.Hash)
	if err != nil && err != leveldb.ErrNotFound {
		log.Fatal(errors2.Wrap(err, "CheckAccountGenesisBlockSecond"))
	}

	if accountGenesisBlockSecond == nil {
		accountChainAccess.WriteGenesisSecondBlock()
	} else {
		if ok := accountGenesisBlockSecond.IsGenesisSecondBlock(); !ok {
			log.Fatal(errors2.Wrap(errors.New("AccountGenesisBlockSecond is not valid."), "CheckAndCreateAccountGenesisBlockSecond"))
		}
	}

}

func (sca *SnapshotChainAccess) WriteGenesisBlock() {
	if err := sca.WriteBlock(ledger.SnapshotGenesisBlock, nil); err != nil {
		log.Fatal(errors2.Wrap(err, "snapshotChain.WriteGenesisBlock"))
	}
	log.Println("snapshotChain.WriteGenesisBlock success.")
}

func (sca *SnapshotChainAccess) GetBlockByHeight(height *big.Int) (*ledger.SnapshotBlock, error) {
	block, err := sca.store.GetBLockByHeight(height)
	if err != nil {
		return nil, err
	}
	return block, nil
}

func (sca *SnapshotChainAccess) GetBlockByHash(blockHash *types.Hash) (*ledger.SnapshotBlock, error) {
	block, err := sca.store.GetBlockByHash(blockHash)
	if err != nil {
		return nil, err
	}
	return block, nil
}

func (sca *SnapshotChainAccess) GetBlocksFromOrigin(originBlockHash *types.Hash, count uint64, forward bool) (ledger.SnapshotBlockList, error) {
	return sca.store.GetBlocksFromOrigin(originBlockHash, count, forward)
}

func (sca *SnapshotChainAccess) GetBlockList(index int, num int, count int) ([]*ledger.SnapshotBlock, error) {
	blockList, err := sca.store.GetBlockList(index, num, count)
	if err != nil {
		return nil, err
	}
	return blockList, nil
}

func (sca *SnapshotChainAccess) GetLatestBlock() (*ledger.SnapshotBlock, error) {
	return sca.store.GetLatestBlock()
}

func (sca *SnapshotChainAccess) WriteBlockList(blockList []*ledger.SnapshotBlock, signFunc signSnapshotBlockFuncType) error {
	batch := new(leveldb.Batch)
	var err error
	for _, block := range blockList {
		err = sca.writeBlock(batch, block, signFunc)
		if err != nil {
			return err
		}
	}
	return nil
}

func (sca *SnapshotChainAccess) WriteBlock(block *ledger.SnapshotBlock, signFunc signSnapshotBlockFuncType) error {
	err := sca.store.BatchWrite(nil, func(batch *leveldb.Batch) error {
		// When *ScWriteError data type convert to error interface, nil become non-nil. So need return nil manually
		if err := sca.writeBlock(batch, block, signFunc); err != nil {
			return err
		}
		return nil
	})

	return err
}

type signSnapshotBlockFuncType func(*ledger.SnapshotBlock) (*ledger.SnapshotBlock, error)

func (sca *SnapshotChainAccess) writeBlock(batch *leveldb.Batch, block *ledger.SnapshotBlock, signFunc signSnapshotBlockFuncType) *ScWriteError {
	if block == nil {
		return &ScWriteError{
			Code: WscDefaultErr,
			Err:  errors.New("The written block is not available."),
		}
	}

	// Mutex.lock
	sca.bwMutex.Lock()
	defer sca.bwMutex.Unlock()

	isGenesisBlock := block.IsGenesisBlock()

	// Judge whether the prehash is valid
	if !isGenesisBlock {
		preSnapshotBlock, err := sca.store.GetLatestBlock()
		if err != nil {
			return &ScWriteError{
				Code: WscDefaultErr,
				Err:  err,
			}
		}
		if !bytes.Equal(block.PrevHash.Bytes(), preSnapshotBlock.Hash.Bytes()) {
			return &ScWriteError{
				Code: WscPrevHashErr,
				Err:  errors.New("PreHash of the written block doesn't direct to the latest block hash."),
				Data: preSnapshotBlock,
			}
		}

		if block.Height == nil {
			newSnapshotHeight := &big.Int{}
			block.Height = newSnapshotHeight.Add(preSnapshotBlock.Height, big.NewInt(1))
		}
	}

	// Check account block availability
	if !isGenesisBlock && block.Snapshot != nil {
		snapshot := block.Snapshot

		var needSyncAccountBlocks []*WscNeedSyncErrData

		accountChainAccess = GetAccountChainAccess()

		for addr, snapshotItem := range snapshot {
			blockMeta, err := accountChainAccess.GetBlockMetaByHash(snapshotItem.AccountBlockHash)
			accountAddress, _ := types.HexToAddress(addr)
			if err != nil || blockMeta == nil {
				needSyncAccountBlocks = append(needSyncAccountBlocks, &WscNeedSyncErrData{
					AccountAddress:    &accountAddress,
					TargetBlockHash:   snapshotItem.AccountBlockHash,
					TargetBlockHeight: snapshotItem.AccountBlockHeight,
				})
			} else {
				// Modify block meta status.
				blockMeta.IsSnapshotted = true
				accountChainAccess.store.WriteBlockMeta(batch, snapshotItem.AccountBlockHash, blockMeta)
			}
		}

		if needSyncAccountBlocks != nil {
			return &ScWriteError{
				Code: WscNeedSyncErr,
				Data: needSyncAccountBlocks,
			}
		}
	}

	if block.Hash == nil {
		hash, err := block.ComputeHash()
		if err != nil {
			return &ScWriteError{
				Code: WscSetHashErr,
				Err:  err,
			}
		}

		block.Hash = hash
	}

	if signFunc != nil && block.Signature == nil {
		var signErr error

		block, signErr = signFunc(block)

		if signErr != nil {
			return &ScWriteError{
				Code: WscSignErr,
				Err:  signErr,
			}
		}
	}

	// Get producer account
	producerAccountMeta, gAccMetaErr := sca.accountStore.GetAccountMetaByAddress(block.Producer)
	if gAccMetaErr != nil && gAccMetaErr != leveldb.ErrNotFound {
		if gAccMetaErr != nil {
			return &ScWriteError{
				Code: WscDefaultErr,
				Err:  gAccMetaErr,
			}
		}
	}

	// If producer account doesn't exist, create it
	if producerAccountMeta == nil {
		writeNewAccountMutex.Lock()
		defer writeNewAccountMutex.Unlock()

		var err error

		producerAccountMeta, err = GetAccountAccess().CreateNewAccountMeta(batch, block.Producer, block.PublicKey)
		if err != nil {
			return &ScWriteError{
				Code: WscDefaultErr,
				Err:  err,
			}
		}

		// Write account meta
		if err = sca.accountStore.WriteMeta(batch, block.Producer, producerAccountMeta); err != nil {
			return &ScWriteError{
				Code: WscDefaultErr,
				Err:  err,
			}
		}

		// Write account id index
		if err := sca.accountStore.WriteAccountIdIndex(batch, producerAccountMeta.AccountId, block.Producer); err != nil {
			return &ScWriteError{
				Code: WscDefaultErr,
				Err:  err,
			}
		}

	}

	//snapshotBlockHeight:d.[snapshotBlockHash]:[snapshotBlockHeight]
	if wbhErr := sca.store.WriteBlockHeight(batch, block); wbhErr != nil {
		return &ScWriteError{
			Code: WscDefaultErr,
			Err:  errors2.Wrap(wbhErr, "WriteBlockHeight"),
		}
	}

	//snapshotBlock:e.[snapshotBlockHeight]:[snapshotBlock]
	if wbErr := sca.store.WriteBlock(batch, block); wbErr != nil {
		return &ScWriteError{
			Code: WscDefaultErr,
			Err:  errors2.Wrap(wbErr, "WriteBlock"),
		}
	}
	return nil
}
