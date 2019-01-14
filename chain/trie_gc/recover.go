package trie_gc

import (
	"errors"
	"fmt"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/common/types"
)

// Recover data when delete too much data
// TODO account block
func (gc *collector) Recover() (returnErr error) {
	// first, stop gc
	gc.Stop()
	defer func() {
		// finally, start gc
		if returnErr != nil {
			gc.log.Error("Recover failed, error is " + returnErr.Error())
		}
		gc.Start()
	}()

	// recover genesis trie
	batch := new(leveldb.Batch)
	genesisSnapshotBlock := gc.chain.GetGenesisSnapshotBlock()
	trieSaveCallback1, err := genesisSnapshotBlock.StateTrie.Save(batch)
	if err != nil {
		return err
	}

	genesisConsensusGroupBlockVC := gc.chain.GetGenesisConsensusGroupBlockVC()
	trieSaveCallback2, err := genesisConsensusGroupBlockVC.UnsavedCache().Trie().Save(batch)
	if err != nil {
		return err
	}

	genesisMintageBlockVC := gc.chain.GetGenesisMintageBlockVC()
	trieSaveCallback3, err := genesisMintageBlockVC.UnsavedCache().Trie().Save(batch)
	if err != nil {
		return err
	}

	genesisMintageSendBlockVC := gc.chain.GetGenesisMintageSendBlockVC()
	trieSaveCallback4, err := genesisMintageSendBlockVC.UnsavedCache().Trie().Save(batch)
	if err != nil {
		return err
	}

	genesisRegisterBlockVC := gc.chain.GetGenesisRegisterBlockVC()
	trieSaveCallback5, err := genesisRegisterBlockVC.UnsavedCache().Trie().Save(batch)
	if err != nil {
		return err
	}

	secondSnapshotBlock := gc.chain.GetSecondSnapshotBlock()
	trieSaveCallback6, err := secondSnapshotBlock.StateTrie.Save(batch)
	if err != nil {
		return err
	}

	if err := gc.chain.ChainDb().Commit(batch); err != nil {
		return errors.New("Commit failed, error is " + err.Error())
	}

	trieSaveCallback1()
	trieSaveCallback2()
	trieSaveCallback3()
	trieSaveCallback4()
	trieSaveCallback5()
	trieSaveCallback6()

	latestBlockEventId, err := gc.chain.GetLatestBlockEventId()
	if err != nil {
		return errors.New("GetLatestBlockEventId failed, error is " + err.Error())
	}

	fmt.Println("Recovering data, dont't shut down...")
	for i := uint64(1); i <= latestBlockEventId; i++ {
		eventType, hashList, err := gc.chain.GetEvent(i)
		if err != nil {
			return errors.New("GetEvent failed, error is " + err.Error())
		}
		switch eventType {
		// AddAccountBlocksEvent = byte(1)
		case byte(1):
			for _, blockHash := range hashList {
				block, err := gc.chain.GetAccountBlockByHash(&blockHash)
				if err != nil {
					return errors.New("GetAccountBlockByHash failed, error is " + err.Error())
				}

				if block == nil {
					// been rolled back
					continue
				}

				if gc.chain.IsGenesisAccountBlock(block) {
					continue
				}

			}

		// AddSnapshotBlocksEvent = byte(3)
		case byte(3):
			for _, snapshotBlockHash := range hashList {
				snapshotBlock, err := gc.chain.GetSnapshotBlockByHash(&snapshotBlockHash)
				if err != nil {
					return errors.New("GetSnapshotBlockByHash failed, error is " + err.Error())
				}

				if snapshotBlock == nil {
					// been rolled back
					continue
				}

				if gc.chain.IsGenesisSnapshotBlock(snapshotBlock) {
					continue
				}

				prevSnapshotHeight := snapshotBlock.Height - 1

				prevSnapshotBlock, err := gc.chain.GetSnapshotBlockByHeight(prevSnapshotHeight)
				if err != nil {
					return errors.New("GetSnapshotBlockByHeight failed, error is " + err.Error())
				}
				var prevStateHash *types.Hash
				if prevSnapshotBlock != nil {
					prevStateHash = &prevSnapshotBlock.StateHash
				} else {
					prevStateHash = &types.Hash{}
				}

				newStateTrie, err := gc.chain.GenStateTrieFromDb(*prevStateHash, snapshotBlock.SnapshotContent)
				if err != nil {
					return errors.New("GenStateTrieFromDb failed, error is " + err.Error())
				}

				batch := new(leveldb.Batch)

				trieSaveCallback, saveTrieErr := newStateTrie.Save(batch)
				if saveTrieErr != nil {
					return errors.New("newStateTrie.Save failed, error is " + saveTrieErr.Error())
				}

				if err := gc.chain.ChainDb().Commit(batch); err != nil {
					return errors.New("Commit failed, error is " + err.Error())
				}

				// after write db
				if trieSaveCallback != nil {
					trieSaveCallback()
				}

			}
		}

	}

	fmt.Println("Data recovery complete")
	return nil
}
