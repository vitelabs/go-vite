package trie_gc

import (
	"encoding/binary"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/vitelabs/go-vite/chain_db/database"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/trie"
	"time"
)

type Marker struct {
	chain              Chain
	db                 *leveldb.DB
	markHeightPerRound uint64
	cleanCountPerRound uint64
	cleanInterval      time.Duration

	clearedHeight uint64
	markedHeight  uint64

	triePool *trie.TrieNodePool
}

func NewMarker(chain Chain, db *leveldb.DB) (*Marker, error) {
	m := &Marker{
		chain:              chain,
		db:                 db,
		markHeightPerRound: 100,
		cleanCountPerRound: 100 * 100,
		cleanInterval:      time.Millisecond * 2,
		triePool:           trie.NewTrieNodePool(),
	}

	if clearedHeight, err := m.getClearedHeight(); err != nil {
		return nil, err
	} else {
		m.clearedHeight = clearedHeight
	}

	if markedHeight, err := m.getMarkedHeight(); err != nil {
		return nil, err
	} else {
		m.markedHeight = markedHeight
	}

	return m, nil
}
func (m *Marker) Db() *leveldb.DB {
	return m.db
}
func (m *Marker) SetMarkedHeight(markedHeight uint64) {
	m.markedHeight = markedHeight
}

func (m *Marker) MarkedHeight() uint64 {
	return m.markedHeight
}

func (m *Marker) ClearedHeight() uint64 {
	return m.clearedHeight
}

func (m *Marker) fixTargetHeight(targetHeight uint64) uint64 {
	correctTargetHeight := targetHeight
	if m.markedHeight > 0 && m.clearedHeight < m.markedHeight {
		correctTargetHeight = m.markedHeight
	}

	return correctTargetHeight
}

func (m *Marker) Mark(targetHeight uint64, terminal <-chan struct{}) (bool, error) {
	targetHeight = m.fixTargetHeight(targetHeight)

	for m.markedHeight < targetHeight {
		select {
		case <-terminal:
			return true, nil
		default:
			currentTargetHeight := m.markedHeight + m.markHeightPerRound
			if currentTargetHeight > targetHeight {
				currentTargetHeight = targetHeight
			}

			blocks, err := m.chain.GetSnapshotBlocksByHeight(m.markedHeight, currentTargetHeight-m.markedHeight, true, false)
			if err != nil {
				return false, err
			}
			if hashSet, err := m.getNodeHashSet(blocks); err == nil {
				if err := m.saveMarkedHashSet(hashSet); err != nil {
					return false, err
				}
			} else {
				return false, err
			}

			if err := m.saveMarkedHeight(currentTargetHeight); err != nil {
				return false, err
			}
			m.markedHeight = currentTargetHeight
		}
	}
	return false, nil
}

func (m *Marker) FilterMarked() error {
	// markedHeight := m.markedHeight
	block, err := m.chain.GetSnapshotBlockByHeight(m.markedHeight)
	if err != nil {
		return err
	}

	if hashSet, err := m.getNodeHashSet([]*ledger.SnapshotBlock{block}); err == nil {
		if err := m.deleteMarkedHashSet(hashSet); err != nil {
			return err
		}
	} else {
		return err
	}
	return nil
}

func (m *Marker) Clean(terminal <-chan struct{}) (bool, error) {
	// do clean
	for {
		if cleared, err := m.isAllCleared(); cleared {
			break
		} else if err != nil {
			return false, err
		}
		select {
		case <-terminal:
			return true, nil
		default:
			needCleanHashList, err := m.getMarkedHashList(m.cleanCountPerRound)
			if err != nil {
				return false, err
			}

			// delete chain
			if err := trie.DeleteNodes(m.chain.ChainDb().Db(), needCleanHashList); err != nil {
				return false, err
			}

			if err := m.deleteMarkedHashList(needCleanHashList); err != nil {
				return false, err
			}
			time.Sleep(m.cleanInterval)
		}

	}

	clearedHeight := m.markedHeight
	if err := m.saveClearedHeight(clearedHeight); err != nil {
		return false, err
	}
	m.clearedHeight = clearedHeight

	return false, nil
}

func (m *Marker) getNodeHashSet(blocks []*ledger.SnapshotBlock) (map[types.Hash]struct{}, error) {
	hashSet := make(map[types.Hash]struct{})
	var accountStateHashList []types.Hash
	for _, block := range blocks {
		stateTrie := trie.NewTrie(m.chain.ChainDb().Db(), &block.StateHash, m.triePool)
		nodeList := stateTrie.NodeList()

		for _, node := range nodeList {
			nodeHash := node.Hash()
			if _, ok := hashSet[*nodeHash]; !ok {
				hashSet[*nodeHash] = struct{}{}
				if node.IsLeafNode() {
					value := stateTrie.LeafNodeValue(node)
					if len(value) == types.HashSize {
						accountStateHash, err := types.BytesToHash(value)
						if err != nil {
							return nil, err
						}
						accountStateHashList = append(accountStateHashList, accountStateHash)
					}

				}
			}
		}
	}

	for _, stateHash := range accountStateHashList {
		stateTrie := trie.NewTrie(m.chain.ChainDb().Db(), &stateHash, m.triePool)
		nodeList := stateTrie.NodeList()
		for _, node := range nodeList {
			nodeHash := node.Hash()
			if _, ok := hashSet[*nodeHash]; !ok {
				hashSet[*nodeHash] = struct{}{}
			}
		}

	}

	return hashSet, nil
}

func (m *Marker) isAllCleared() (bool, error) {
	dbKey, _ := database.EncodeKey(DBKP_MARKED_HASHLIST)

	iter := m.db.NewIterator(util.BytesPrefix(dbKey), nil)
	defer iter.Release()

	if !iter.Next() {
		if err := iter.Error(); err != nil && err != leveldb.ErrNotFound {
			return false, err
		}
		return true, nil
	}

	return false, nil
}
func (m *Marker) getMarkedHashList(count uint64) ([]types.Hash, error) {
	dbKey, _ := database.EncodeKey(DBKP_MARKED_HASHLIST)

	iter := m.db.NewIterator(util.BytesPrefix(dbKey), nil)
	defer iter.Release()
	var hashList []types.Hash
	for i := uint64(0); i < count && iter.Next(); i++ {
		key := iter.Key()
		hash, _ := types.BytesToHash(key[1:])
		hashList = append(hashList, hash)
	}

	if err := iter.Error(); err != nil && err != leveldb.ErrNotFound {
		return nil, err
	}
	return hashList, nil
}

func (m *Marker) saveMarkedHashSet(hashSet map[types.Hash]struct{}) error {
	batch := new(leveldb.Batch)
	for hash := range hashSet {
		dbKey, _ := database.EncodeKey(DBKP_MARKED_HASHLIST, hash.Bytes())
		batch.Put(dbKey, []byte{})
	}

	return m.db.Write(batch, nil)
}

func (m *Marker) deleteMarkedHashSet(hashSet map[types.Hash]struct{}) error {
	batch := new(leveldb.Batch)
	for hash := range hashSet {
		dbKey, _ := database.EncodeKey(DBKP_MARKED_HASHLIST, hash.Bytes())
		batch.Delete(dbKey)
	}

	return m.db.Write(batch, nil)
}

func (m *Marker) deleteMarkedHashList(hashList []types.Hash) error {
	batch := new(leveldb.Batch)
	for _, hash := range hashList {
		dbKey, _ := database.EncodeKey(DBKP_MARKED_HASHLIST, hash.Bytes())
		batch.Delete(dbKey)
	}

	return m.db.Write(batch, nil)
}

func (m *Marker) getClearedHeight() (uint64, error) {
	dbKey, _ := database.EncodeKey(DBKP_CLEARED_HEIGHT)
	value, err := m.db.Get(dbKey, nil)
	if err != nil {
		if err != leveldb.ErrNotFound {
			return 0, err
		}
		return 0, nil
	}
	return binary.BigEndian.Uint64(value), nil
}

func (m *Marker) saveClearedHeight(clearedHeight uint64) error {
	dbKey, _ := database.EncodeKey(DBKP_CLEARED_HEIGHT)

	heightBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(heightBytes, clearedHeight)

	return m.db.Put(dbKey, heightBytes, nil)
}

func (m *Marker) getMarkedHeight() (uint64, error) {
	dbKey, _ := database.EncodeKey(DBKP_MARKED_HEIGHT)
	value, err := m.db.Get(dbKey, nil)
	if err != nil {
		if err != leveldb.ErrNotFound {
			return 0, err
		}
		return 0, nil
	}
	return binary.BigEndian.Uint64(value), nil
}

func (m *Marker) saveMarkedHeight(markedHeight uint64) error {
	dbKey, _ := database.EncodeKey(DBKP_MARKED_HEIGHT)

	heightBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(heightBytes, markedHeight)

	return m.db.Put(dbKey, heightBytes, nil)
}
