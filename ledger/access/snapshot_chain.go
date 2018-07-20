package access

import (
	"github.com/vitelabs/go-vite/vitedb"
	"github.com/vitelabs/go-vite/ledger"
	"sync"
	"github.com/syndtr/goleveldb/leveldb"
	"errors"
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"bytes"
	"math/big"
	"log"
)

type SnapshotChainAccess struct {
	store *vitedb.SnapshotChain
	accountStore  *vitedb.Account
	bwMutex sync.RWMutex
}

var snapshotChainAccess = &SnapshotChainAccess {
	store: vitedb.GetSnapshotChain(),
	accountStore:  vitedb.GetAccount(),
}

func GetSnapshotChainAccess () *SnapshotChainAccess {
	return snapshotChainAccess
}

func (sca *SnapshotChainAccess) CheckAndCreateGenesisBlock () {
	genesisBlock, err := sca.store.GetBLockByHeight(big.NewInt(1))
	if err != nil && err != leveldb.ErrNotFound{
		log.Fatal(errors.New("Create genesis block failed. Error is " + err.Error()))
	}

	if genesisBlock == nil {
		sca.createGenesisBlock()
	} else {
		if ok := sca.checkGenesisBlock(genesisBlock); !ok {
			log.Fatal("Genesis block is invalid.")
		}
	}
}

func (sca *SnapshotChainAccess) createGenesisBlock () {
	genesisBlock := ledger.GetGenesisSnapshot()
	err := sca.store.BatchWrite(nil, func(batch *leveldb.Batch) error {
		 if err := sca.writeBlock(batch, genesisBlock , nil); err != nil {
			return err
		 }
		 return nil
	})

	if err != nil {
		log.Fatal("Create genesis block failed. Error is " + err.Error())
	}
}

func (sca *SnapshotChainAccess) checkGenesisBlock (genesisBlock *ledger.SnapshotBlock) bool {
	return genesisBlock.PrevHash == nil &&
		bytes.Equal(genesisBlock.Producer.Bytes(), ledger.GenesisSnapshotBlock.Producer.Bytes()) &&
		bytes.Equal(genesisBlock.Signature, ledger.GenesisSnapshotBlock.Signature) &&
		bytes.Equal(genesisBlock.Hash.Bytes(), ledger.GenesisSnapshotBlock.Hash.Bytes()) &&
		bytes.Equal(genesisBlock.PublicKey, ledger.GenesisSnapshotBlock.PublicKey) &&
		genesisBlock.Timestamp == ledger.GenesisSnapshotBlock.Timestamp &&
		genesisBlock.Height.Cmp( big.NewInt(1)) == 0
}


func (sca *SnapshotChainAccess) GetBlockByHeight (height *big.Int) (*ledger.SnapshotBlock, error) {
	block, err:= sca.store.GetBLockByHeight(height)
	if err != nil {
		return nil, err
	}
	return block, nil
}

func (sca *SnapshotChainAccess) GetBlockByHash (blockHash *types.Hash) (*ledger.SnapshotBlock, error) {
	block, err:= sca.store.GetBlockByHash(blockHash)
	if err != nil {
		return nil, err
	}
	return block, nil
}

func (sca *SnapshotChainAccess) GetBlocksFromOrigin (originBlockHash *types.Hash, count uint64, forward bool) (ledger.SnapshotBlockList, error) {
	return sca.store.GetBlocksFromOrigin(originBlockHash, count, forward)
}

func (sca *SnapshotChainAccess) GetBlockList (index int, num int, count int) ([]*ledger.SnapshotBlock, error) {
	blockList, err:= sca.store.GetBlockList(index, num, count)
	if err != nil {
		return nil, err
	}
	return blockList, nil
}

func (sca *SnapshotChainAccess) GetLatestBlock() (*ledger.SnapshotBlock, error){
	return sca.store.GetLatestBlock()
}

func (sca *SnapshotChainAccess) WriteBlockList (blockList []*ledger.SnapshotBlock, signFunc signSnapshotBlockFuncType) error {
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

func (sca *SnapshotChainAccess) WriteBlock (block *ledger.SnapshotBlock, signFunc signSnapshotBlockFuncType) error {
	err := sca.store.BatchWrite(nil, func(batch *leveldb.Batch) error {
		return sca.writeBlock(batch, block, signFunc)
	})
	if err != nil {
		fmt.Println("Write block failed, block data is ")
		fmt.Printf("%+v\n", block)
	} else {
		fmt.Println("Write Snapshot block " + block.Hash.String() + " succeed")
	}

	return err
}

type signSnapshotBlockFuncType func(*ledger.SnapshotBlock)(*ledger.SnapshotBlock, error)


func (sca *SnapshotChainAccess) writeBlock (batch *leveldb.Batch, block *ledger.SnapshotBlock, signFunc signSnapshotBlockFuncType) *ScWriteError {
	if block == nil {
		return &ScWriteError {
			Code: WscDefaultErr,
			Err: errors.New("The written block is not available."),
		}
	}
	// Mutex.lock
	sca.bwMutex.Lock()
	defer sca.bwMutex.Unlock()

	if block.Hash == nil {
		if err := block.SetHash(); err != nil {
			return &ScWriteError{
				Code: WscSetHashErr,
				Err: err,
			}
		}
	}

	isGenesisBlock := bytes.Equal(block.Hash.Bytes(), ledger.GenesisSnapshotBlock.Hash.Bytes())
	// Judge whether the prehash is valid
	if !isGenesisBlock {
		preSnapshotBlock, err := sca.store.GetLatestBlock()
		if err != nil {
			return &ScWriteError {
				Code: WscDefaultErr,
				Err: err,
			}
		}
		if !bytes.Equal(block.PrevHash.Bytes(), preSnapshotBlock.Hash.Bytes()){
			return &ScWriteError {
				Code: WscPrevHashErr,
				Err: errors.New("PreHash of the written block doesn't direct to the latest block hash."),
				Data: preSnapshotBlock,
			}
		}
		newSnapshotHeight := &big.Int{}
		block.Height = newSnapshotHeight.Add(preSnapshotBlock.Height, big.NewInt(1))
	}

	// Check account block availability
	if !isGenesisBlock {
		snapshot := block.Snapshot

		var needSyncAccountBlocks []*WscNeedSyncErrData

		for addr, snapshotItem := range snapshot {
			blockMeta, err := accountChainAccess.GetBlockMetaByHash(snapshotItem.AccountBlockHash)
			accountAddress, _ := types.HexToAddress(addr)
			if err != nil || blockMeta == nil {
				needSyncAccountBlocks = append(needSyncAccountBlocks, &WscNeedSyncErrData{
					AccountAddress: &accountAddress,
					TargetBlockHash: snapshotItem.AccountBlockHash,
					TargetBlockHeight: snapshotItem.AccountBlockHeight,
				})
			}  else {
				// Modify block meta status.
				accountChainAccess.store.WriteBlockMeta(batch, snapshotItem.AccountBlockHash, blockMeta)
			}
		}


		if needSyncAccountBlocks != nil{
			return  &ScWriteError {
				Code: WscNeedSyncErr,
				Data: needSyncAccountBlocks,
			}
		}
	}


	if signFunc != nil && block.Signature == nil {
		var signErr error

		block, signErr = signFunc(block)

		if signErr != nil {
			return &ScWriteError{
				Code: WscSignErr,
				Err: signErr,
			}
		}
	}


	// Get producer account
	producerAccountMeta, gAccMetaErr := sca.accountStore.GetAccountMetaByAddress(block.Producer)
	if gAccMetaErr != nil && gAccMetaErr != leveldb.ErrNotFound{
		if gAccMetaErr != nil {
			return &ScWriteError {
				Code: WscDefaultErr,
				Err: gAccMetaErr,
			}
		}
	}

	// If producer account doen't exist, create it
	if producerAccountMeta == nil {
		writeNewAccountMutex.Lock()
		defer writeNewAccountMutex.Unlock()

		var err error
		producerAccountMeta, err = accountAccess.CreateNewAccountMeta(batch, block.Producer, block.PublicKey)
		if err != nil {
			return &ScWriteError {
				Code: WscDefaultErr,
				Err: err,
			}
		}

		if err = sca.accountStore.WriteMeta(batch, block.Producer, producerAccountMeta); err != nil {
			return &ScWriteError{
				Code: WscDefaultErr,
				Err: err,
			}
		}
	}

	//snapshotBlockHeight:d.[snapshotBlockHash]:[snapshotBlockHeight]
	if wbhErr := sca.store.WriteBlockHeight(batch, block); wbhErr != nil {
		return &ScWriteError {
			Code: WscDefaultErr,
			Err: wbhErr,
		}
	}
	//snapshotBlock:e.[snapshotBlockHeight]:[snapshotBlock]
	if wbErr := sca.store.WriteBlock(batch, block); wbErr != nil {
		return &ScWriteError {
			Code: WscDefaultErr,
			Err: wbErr,
		}
	}
	return nil
}
