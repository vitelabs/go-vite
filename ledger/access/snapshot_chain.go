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
)

type SnapshotChainAccess struct {
	store *vitedb.SnapshotChain
	bwMutex sync.RWMutex
}

var snapshotChainAccess = &SnapshotChainAccess{
	store: vitedb.GetSnapshotChain(),
}

func GetSnapshotChainAccess () *SnapshotChainAccess {
	return snapshotChainAccess
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

func (sca *SnapshotChainAccess) WriteBlockList (blockList []*ledger.SnapshotBlock) error {
	batch := new(leveldb.Batch)
	var err error
	for _, block := range blockList {
		err = sca.writeBlock(batch, block)
		if err != nil {
			return err
		}
	}
	return nil
}

func (sca *SnapshotChainAccess) WriteBlock (block *ledger.SnapshotBlock) error{
	err := sca.store.BatchWrite(nil, func(batch *leveldb.Batch) error {
		return sca.writeBlock(batch, block)
	})
	if err != nil {
		fmt.Println("Write block failed, block data is ")
		fmt.Printf("%+v\n", block)
	} else {
		fmt.Println("Write Snapshot block " + block.Hash.String() + " succeed")
	}
	return err
}

func (sca *SnapshotChainAccess) writeBlock (batch *leveldb.Batch, block *ledger.SnapshotBlock) error {
	if block == nil {
		return errors.New("The written block is not available.")
	}
	//mutex.lock
	sca.bwMutex.Lock()
	defer sca.bwMutex.Unlock()
	//judge whether the prehash is valid
	if !bytes.Equal(block.Hash.Bytes(), ledger.GenesisSnapshotBlockHash.Bytes()) {
		preSnapshotBlock, err := sca.store.GetLatestBlock()
		if err != nil {
			return err
		}
		if !bytes.Equal(block.PrevHash.Bytes(), preSnapshotBlock.Hash.Bytes()){
			return errors.New("PreHash of the written block doesn't direct to the latest block hash.")
		}
		newSnapshotHeight := &big.Int{}
		block.Height = newSnapshotHeight.Add(preSnapshotBlock.Height, big.NewInt(1))
	}
	//snapshotBlockHeight:d.[snapshotBlockHash]:[snapshotBlockHeight]
	if wbhErr := sca.store.WriteBlockHeight(batch, block); wbhErr != nil {
		return wbhErr
	}
	//snapshotBlock:e.[snapshotBlockHeight]:[snapshotBlock]
	if wbErr := sca.store.WriteBlock(batch, block); wbErr != nil {
		return wbErr
	}
	return nil
}

func (sca *SnapshotChainAccess) GetAccountList () ([]*types.Address, error){
	return sca.store.GetAccountList()
}