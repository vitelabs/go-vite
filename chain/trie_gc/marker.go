package trie_gc

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/vitelabs/go-vite/chain_db/database"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/trie"
	"sync"
)

type Marker struct {
	chain              Chain
	markHeightPerRound uint64

	triePool *trie.TrieNodePool

	lock           sync.Mutex
	stopSaveMinGap uint64
}

func NewMarker(chain Chain) *Marker {
	m := &Marker{
		chain:              chain,
		markHeightPerRound: 100,
		triePool:           trie.NewTrieNodePool(),

		stopSaveMinGap: 8,
	}

	return m
}

func (m *Marker) getTargetHeight() uint64 {
	latestSnapshotBlock := m.chain.GetLatestSnapshotBlock()
	if latestSnapshotBlock == nil {
		return 0
	}

	return m.chain.GetLatestSnapshotBlock().Height
}

func (m *Marker) Mark(fromHeight uint64, terminal <-chan struct{}) (hashSet map[types.Hash]struct{}, isTerminal bool, err error) {
	markedHeight := fromHeight
	markedHashSet := make(map[types.Hash]struct{})

	targetHeight := m.getTargetHeight()

	for {
		for markedHeight < targetHeight {
			select {
			case <-terminal:
				return nil, true, nil
			default:
				currentTargetHeight := markedHeight + m.markHeightPerRound
				if currentTargetHeight > targetHeight {
					currentTargetHeight = targetHeight
				}

				blocks, err := m.chain.GetSnapshotBlocksByHeight(markedHeight, currentTargetHeight-markedHeight, true, false)
				if err != nil {
					return nil, false, err
				}
				if err := m.setNodeHashSet(blocks, markedHashSet); err != nil {
					return nil, false, err
				}

				// if err := m.saveMarkedHeight(currentTargetHeight); err != nil {
				//		return false, err
				// }
				markedHeight = currentTargetHeight
			}
		}

		// stop save
		m.chain.StopSaveTrie()

		nextTargetHeight := m.getTargetHeight()
		if nextTargetHeight <= targetHeight {
			break
		}

		gap := nextTargetHeight - targetHeight
		if gap > m.stopSaveMinGap {
			m.chain.StartSaveTrie()
			// unlock
		}

		targetHeight = nextTargetHeight
	}

	// Clean
	return markedHashSet, false, nil

}

//func (m *Marker) _Mark(targetHeight uint64, terminal <-chan struct{}) (bool, error) {
//	targetHeight = m.fixTargetHeight(targetHeight)
//
//	m.markedHashSet = make(map[types.Hash]struct{})
//	for m.markedHeight < targetHeight {
//		select {
//		case <-terminal:
//			return true, nil
//		default:
//			currentTargetHeight := m.markedHeight + m.markHeightPerRound
//			if currentTargetHeight > targetHeight {
//				currentTargetHeight = targetHeight
//			}
//
//			blocks, err := m.chain.GetSnapshotBlocksByHeight(m.markedHeight, currentTargetHeight-m.markedHeight, true, false)
//			if err != nil {
//				return false, err
//			}
//			if err := m.setNodeHashSet(blocks, m.markedHashSet); err != nil {
//				return false, err
//			}
//
//			// if err := m.saveMarkedHeight(currentTargetHeight); err != nil {
//			//		return false, err
//			// }
//			m.markedHeight = currentTargetHeight
//		}
//	}
//	return false, nil
//}
//
//func (m *Marker) FilterMarked() error {
//	// markedHeight := m.markedHeight
//	block, err := m.chain.GetSnapshotBlockByHeight(m.markedHeight)
//	if err != nil {
//		return err
//	}
//
//	if hashSet, err := m.getNodeHashSet([]*ledger.SnapshotBlock{block}); err == nil {
//		if err := m.deleteMarkedHashSet(hashSet); err != nil {
//			return err
//		}
//	} else {
//		return err
//	}
//	return nil
//}

func (m *Marker) Clean(hashSet map[types.Hash]struct{}) (bool, error) {
	// do clean
	// clear cache
	// clear db
	dbkey, _ := database.EncodeKey(database.DBKP_TRIE_NODE)
	iter := m.chain.TrieDb().NewIterator(util.BytesPrefix(dbkey), nil)
	defer iter.Release()

	batch := new(leveldb.Batch)
	for iter.Next() {
		key := iter.Key()
		hash, _ := types.BytesToHash(key[1:])

		if _, ok := hashSet[hash]; !ok {
			batch.Delete(iter.Key())
		}
	}
	if err := m.chain.ChainDb().Commit(batch); err != nil {
		return false, err
	}

	// clear cache
	m.chain.CleanTrieNodePool()

	if err := iter.Error(); err != nil && err != leveldb.ErrNotFound {
		return false, err
	}
	return false, nil

	//for {
	//	if cleared, err := m.isAllCleared(); cleared {
	//		break
	//	} else if err != nil {
	//		return false, err
	//	}
	//	select {
	//	case <-terminal:
	//		return true, nil
	//	default:
	//		needCleanHashList, err := m.getMarkedHashList(m.cleanCountPerRound)
	//		if err != nil {
	//			return false, err
	//		}
	//
	//		// delete chain
	//		if err := trie.DeleteNodes(m.chain.ChainDb().Db(), needCleanHashList); err != nil {
	//			return false, err
	//		}
	//
	//		if err := m.deleteMarkedHashList(needCleanHashList); err != nil {
	//			return false, err
	//		}
	//	}
	//
	//}
	//
	//clearedHeight := m.markedHeight
	//if err := m.saveClearedHeight(clearedHeight); err != nil {
	//	return false, err
	//}
	//m.clearedHeight = clearedHeight
	//
	//return false, nil
}

func (m *Marker) setNodeHashSet(blocks []*ledger.SnapshotBlock, hashSet map[types.Hash]struct{}) error {
	var accountStateHashList []types.Hash

	inHashSet := func(node *trie.TrieNode) bool {
		if _, ok := hashSet[*node.Hash()]; ok {
			return false
		}
		return true
	}
	for _, block := range blocks {
		stateTrie := trie.NewTrie(m.chain.ChainDb().Db(), &block.StateHash, m.triePool)
		ni := stateTrie.NewNodeIterator()
		for ni.Next(inHashSet) {
			node := ni.Node()

			nodeHash := node.Hash()
			if _, ok := hashSet[*nodeHash]; !ok {
				hashSet[*nodeHash] = struct{}{}
				if node.IsLeafNode() {
					value := stateTrie.LeafNodeValue(node)
					if len(value) == types.HashSize {
						accountStateHash, err := types.BytesToHash(value)
						if err != nil {
							return err
						}

						if _, ok := hashSet[*nodeHash]; !ok {
							accountStateHashList = append(accountStateHashList, accountStateHash)
						}

					}

				}
			}
		}
	}

	for _, stateHash := range accountStateHashList {
		stateTrie := trie.NewTrie(m.chain.ChainDb().Db(), &stateHash, m.triePool)
		ni := stateTrie.NewNodeIterator()
		for ni.Next(inHashSet) {
			node := ni.Node()
			nodeHash := node.Hash()

			hashSet[*nodeHash] = struct{}{}
		}
	}

	return nil
}

//
//func (m *Marker) isAllCleared() (bool, error) {
//	dbKey, _ := database.EncodeKey(DBKP_MARKED_HASHLIST)
//
//	iter := m.db.NewIterator(util.BytesPrefix(dbKey), nil)
//	defer iter.Release()
//
//	if !iter.Next() {
//		if err := iter.Error(); err != nil && err != leveldb.ErrNotFound {
//			return false, err
//		}
//		return true, nil
//	}
//
//	return false, nil
//}

//func (m *Marker) getMarkedHashList(count uint64) ([]types.Hash, error) {
//	dbKey, _ := database.EncodeKey(DBKP_MARKED_HASHLIST)
//
//	iter := m.db.NewIterator(util.BytesPrefix(dbKey), nil)
//	defer iter.Release()
//	var hashList []types.Hash
//	for i := uint64(0); i < count && iter.Next(); i++ {
//		key := iter.Key()
//		hash, _ := types.BytesToHash(key[1:])
//		hashList = append(hashList, hash)
//	}
//
//	if err := iter.Error(); err != nil && err != leveldb.ErrNotFound {
//		return nil, err
//	}
//	return hashList, nil
//}
//
//func (m *Marker) saveMarkedHashSet(hashSet map[types.Hash]struct{}) error {
//	batch := new(leveldb.Batch)
//	for hash := range hashSet {
//		dbKey, _ := database.EncodeKey(DBKP_MARKED_HASHLIST, hash.Bytes())
//		batch.Put(dbKey, []byte{})
//	}
//
//	return m.db.Write(batch, nil)
//}
//
//func (m *Marker) deleteMarkedHashSet(hashSet map[types.Hash]struct{}) error {
//	batch := new(leveldb.Batch)
//	for hash := range hashSet {
//		dbKey, _ := database.EncodeKey(DBKP_MARKED_HASHLIST, hash.Bytes())
//		batch.Delete(dbKey)
//	}
//
//	return m.db.Write(batch, nil)
//}
//
//func (m *Marker) deleteMarkedHashList(hashList []types.Hash) error {
//	batch := new(leveldb.Batch)
//	for _, hash := range hashList {
//		dbKey, _ := database.EncodeKey(DBKP_MARKED_HASHLIST, hash.Bytes())
//		batch.Delete(dbKey)
//	}
//
//	return m.db.Write(batch, nil)
//}
//
//func (m *Marker) getClearedHeight() (uint64, error) {
//	dbKey, _ := database.EncodeKey(DBKP_CLEARED_HEIGHT)
//	value, err := m.db.Get(dbKey, nil)
//	if err != nil {
//		if err != leveldb.ErrNotFound {
//			return 0, err
//		}
//		return 0, nil
//	}
//	return binary.BigEndian.Uint64(value), nil
//}
//
//func (m *Marker) saveClearedHeight(clearedHeight uint64) error {
//	dbKey, _ := database.EncodeKey(DBKP_CLEARED_HEIGHT)
//
//	heightBytes := make([]byte, 8)
//	binary.BigEndian.PutUint64(heightBytes, clearedHeight)
//
//	return m.db.Put(dbKey, heightBytes, nil)
//}
//
//func (m *Marker) getMarkedHeight() (uint64, error) {
//	dbKey, _ := database.EncodeKey(DBKP_MARKED_HEIGHT)
//	value, err := m.db.Get(dbKey, nil)
//	if err != nil {
//		if err != leveldb.ErrNotFound {
//			return 0, err
//		}
//		return 0, nil
//	}
//	return binary.BigEndian.Uint64(value), nil
//}
//
//func (m *Marker) saveMarkedHeight(markedHeight uint64) error {
//	dbKey, _ := database.EncodeKey(DBKP_MARKED_HEIGHT)
//
//	heightBytes := make([]byte, 8)
//	binary.BigEndian.PutUint64(heightBytes, markedHeight)
//
//	return m.db.Put(dbKey, heightBytes, nil)
//}
