package trie_gc

import (
	"errors"
	"fmt"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/chain_db/database"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/generator"
	"github.com/vitelabs/go-vite/trie"
)

func (gc *collector) recoverGenesis() error {
	// recover genesis trie
	batch := new(leveldb.Batch)

	genesisSnapshotBlock := gc.chain.NewGenesisSnapshotBlock()
	trieSaveCallback1, err := genesisSnapshotBlock.StateTrie.Save(batch)
	if err != nil {
		return err
	}

	_, genesisConsensusGroupBlockVC := gc.chain.NewGenesisConsensusGroupBlock()
	trieSaveCallback2, err := genesisConsensusGroupBlockVC.UnsavedCache().Trie().Save(batch)
	if err != nil {
		return err
	}

	_, genesisMintageBlockVC := gc.chain.NewGenesisMintageBlock()
	trieSaveCallback3, err := genesisMintageBlockVC.UnsavedCache().Trie().Save(batch)
	if err != nil {
		return err
	}

	_, genesisMintageSendBlockVC := gc.chain.NewGenesisMintageSendBlock()
	trieSaveCallback4, err := genesisMintageSendBlockVC.UnsavedCache().Trie().Save(batch)
	if err != nil {
		return err
	}

	_, genesisRegisterBlockVC := gc.chain.NewGenesisRegisterBlock()
	trieSaveCallback5, err := genesisRegisterBlockVC.UnsavedCache().Trie().Save(batch)
	if err != nil {
		return err
	}

	secondSnapshotBlock := gc.chain.NewSecondSnapshotBlock()
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

	return nil
}

func (gc *collector) saveTrie(t *trie.Trie) error {
	batch := new(leveldb.Batch)

	trieSaveCallback, saveTrieErr := t.Save(batch)
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
	return nil
}

func isNodeExist(chainInstance Chain, node *trie.TrieNode) (bool, error) {
	db := chainInstance.ChainDb().Db()

	nodeHash := node.Hash()
	dbKey, _ := database.EncodeKey(database.DBKP_TRIE_NODE, nodeHash.Bytes())
	ok, err := db.Has(dbKey, nil)
	if !ok || err != nil {
		return ok, err
	}

	if node.NodeType() == trie.TRIE_HASH_NODE {
		dbKey2, _ := database.EncodeKey(database.DBKP_TRIE_REF_VALUE, node.Value())
		ok2, err := db.Has(dbKey2, nil)
		if !ok || err != nil {
			return ok2, err
		}
	}
	return true, nil

}

// Recover data when delete too much data
func (gc *collector) Recover() (returnErr error) {
	// clear trie node pool
	gc.chain.CleanTrieNodePool()

	defer func() {
		// finally, start gc
		if returnErr != nil {
			fmt.Println("Recover failed, error is " + returnErr.Error())
		}
	}()

	const PRINT_PER_COUNT = 10000

	if err := gc.recoverGenesis(); err != nil {
		return errors.New("recoverGenesis failed, error is " + err.Error())
	}

	latestBlockEventId, err := gc.chain.GetLatestBlockEventId()
	if err != nil {
		return errors.New("GetLatestBlockEventId failed, error is " + err.Error())
	}

	fmt.Println("Recovering data, dont't shut down...")

	accountTypeCache := make(map[types.Address]byte)

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

				var accountType byte

				if cacheAccountType, ok := accountTypeCache[block.AccountAddress]; ok {
					accountType = cacheAccountType
				} else {

					dbAccountType, err := gc.chain.AccountType(&block.AccountAddress)
					if err != nil {
						return errors.New("gc.chain.AccountType failed, error is " + err.Error())
					}
					accountType = byte(dbAccountType)
					accountTypeCache[block.AccountAddress] = accountType
				}

				if accountType == 3 && block.IsSendBlock() {
					// the send block of contract address need be ignored.
					continue
				}

				vmCtxtList, _ := generator.RecoverVmContext(gc.chain, block)
				if len(vmCtxtList) <= 0 {
					err := errors.New(fmt.Sprintf("RecoverVmContext failed, error is len(vmCtxtList) <= 0, block.Hash is %s, block.Height is %d", block.Hash, block.Height))
					return err
				}

				for index, vmCtxt := range vmCtxtList {
					if index == 0 {
						firstTrieHash := vmCtxt.UnsavedCache().Trie().Hash()

						if firstTrieHash == nil {
							firstTrieHash = &types.Hash{}
						}
						if block.StateHash != *firstTrieHash {
							err := errors.New(fmt.Sprintf("recover failed, trie hash is not correct, block.Hash is %s, block.StateHash is %s, firstTrieHash is %s",
								block.Hash, block.StateHash, firstTrieHash))
							return err
						}
					}
					if err := gc.saveTrie(vmCtxt.UnsavedCache().Trie()); err != nil {
						return errors.New("saveTrie failed, error is " + err.Error())
					}
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

				if err := gc.saveTrie(newStateTrie); err != nil {
					return errors.New("saveTrie failed, error is " + err.Error())
				}
			}
		}

		if i%PRINT_PER_COUNT == 0 {
			fmt.Printf("Recover %d/%d\n", i, latestBlockEventId)
		}

	}

	fmt.Printf("Recover %d/%d\n", latestBlockEventId, latestBlockEventId)
	fmt.Println("Data recovery complete")
	return nil
}
