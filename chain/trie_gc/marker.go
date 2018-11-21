package trie_gc

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/trie"
	"time"
)

type marker struct {
	chain              Chain
	db                 *leveldb.DB
	markHeightPerRound uint64
	cleanCountPerRound uint64
	cleanInterval      time.Duration

	clearedHeight uint64
	markedHeight  uint64

	triePool *trie.TrieNodePool
}

func newMarker(chain Chain, db *leveldb.DB) (*marker, error) {
	m := &marker{
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

func (m *marker) ClearedHeight() uint64 {
	return m.clearedHeight
}

func (m *marker) fixTargetHeight(targetHeight uint64) uint64 {
	correctTargetHeight := targetHeight
	if m.markedHeight > 0 && m.clearedHeight < m.markedHeight {
		correctTargetHeight = m.markedHeight
	}

	return correctTargetHeight
}

func (m *marker) Mark(targetHeight uint64, terminal <-chan struct{}) error {
	targetHeight = m.fixTargetHeight(targetHeight)

	for m.markedHeight < targetHeight {
		select {
		case <-terminal:
			return nil
		default:
			currentTargetHeight := m.markedHeight + m.markHeightPerRound
			if currentTargetHeight > targetHeight {
				currentTargetHeight = targetHeight
			}

			blocks, err := m.chain.GetSnapshotBlocksByHeight(m.markedHeight, currentTargetHeight-m.markedHeight, true, false)
			if err != nil {
				return err
			}

			if err := m.saveMarkedHashSet(m.getNodeHashSet(blocks)); err != nil {
				return err
			}

			if err := m.saveMarkedHeight(currentTargetHeight); err != nil {
				return err
			}
			m.markedHeight = currentTargetHeight
		}
	}
	return nil
}

func (m *marker) FilterMarked() error {
	// markedHeight := m.markedHeight
	block, err := m.chain.GetSnapshotBlockByHeight(m.markedHeight)
	if err != nil {
		return err
	}

	if err := m.deleteMarkedHashSet(m.getNodeHashSet([]*ledger.SnapshotBlock{block})); err != nil {
		return err
	}

	return nil
}

func (m *marker) Clean(terminal <-chan struct{}) error {
	// do clean
	for !m.isAllCleared() {
		select {
		case <-terminal:
			return nil
		default:
			needCleanHashList, err := m.getMarkedHashList(m.cleanCountPerRound)
			if err != nil {
				return err
			}

			// delete chain
			if err := trie.DeleteNodes(m.chain.ChainDb().Db(), needCleanHashList); err != nil {
				return err
			}

			if err := m.deleteMarkedHashList(needCleanHashList); err != nil {
				return err
			}
			time.Sleep(m.cleanInterval)
		}

	}

	clearedHeight := m.markedHeight
	if err := m.saveClearedHeight(clearedHeight); err != nil {
		return err
	}
	m.clearedHeight = clearedHeight

	return nil
}

func (m *marker) getNodeHashSet(blocks []*ledger.SnapshotBlock) map[types.Hash]struct{} {
	hashSet := make(map[types.Hash]struct{})

	for _, block := range blocks {
		stateTrie := trie.NewTrie(m.chain.ChainDb().Db(), &block.StateHash, m.triePool)
		hashList := stateTrie.NodeHashList()
		for _, nodeHash := range hashList {
			if _, ok := hashSet[*nodeHash]; !ok {
				hashSet[*nodeHash] = struct{}{}
			}
		}
	}

	return hashSet
}

func (m *marker) isAllCleared() bool {
	return false
}
func (m *marker) getMarkedHashList(count uint64) ([]types.Hash, error) {
	return nil, nil
}

func (m *marker) saveMarkedHashSet(hashSet map[types.Hash]struct{}) error {
	return nil
}

func (m *marker) deleteMarkedHashSet(hashSet map[types.Hash]struct{}) error {
	return nil
}

func (m *marker) deleteMarkedHashList(hashList []types.Hash) error {
	return nil
}

func (m *marker) getClearedHeight() (uint64, error) {
	return 0, nil
}

func (m *marker) saveClearedHeight(clearedHeight uint64) error {
	return nil
}

func (m *marker) getMarkedHeight() (uint64, error) {
	return 0, nil
}

func (m *marker) saveMarkedHeight(markedHeight uint64) error {
	return nil
}
