package store

import (
	"sync"

	"strconv"

	"github.com/vitelabs/go-vite/interval/common"
)

func newMemoryStore() BlockStore {
	self := &blockMemoryStore{}
	for _, genesis := range genesisBlocks {
		self.PutAccount(genesis.Signer(), genesis)
		self.SetAccountHead(genesis.Signer(), &common.HashHeight{Hash: genesis.Hash(), Height: genesis.Height()})
	}

	return self
}

// thread safe block memory store
type blockMemoryStore struct {
	snapshotHeight sync.Map
	snapshotHash   sync.Map
	accountHeight  sync.Map
	accountHash    sync.Map
	// key: source hash val: received
	sourceHash sync.Map

	head sync.Map

	sMu sync.Mutex
	aMu sync.Mutex
}

var snapshotHeadKey = "s_head_key"

func (store *blockMemoryStore) GetSnapshotHead() *common.HashHeight {
	value, ok := store.head.Load(snapshotHeadKey)
	if !ok {
		return nil
	}
	return value.(*common.HashHeight)
}

func (store *blockMemoryStore) GetAccountHead(address string) *common.HashHeight {
	value, ok := store.head.Load(address)
	if !ok {
		return nil
	}
	return value.(*common.HashHeight)
}

func (store *blockMemoryStore) SetSnapshotHead(hashH *common.HashHeight) {
	if hashH == nil {
		store.head.Delete(snapshotHeadKey)
	} else {
		store.head.Store(snapshotHeadKey, hashH)
	}
}

func (store *blockMemoryStore) SetAccountHead(address string, hashH *common.HashHeight) {
	if hashH == nil {
		store.head.Delete(address)
	} else {
		store.head.Store(address, hashH)
	}

}

func (store *blockMemoryStore) DeleteSnapshot(hashH common.HashHeight) {
	store.snapshotHeight.Delete(hashH.Height)
	store.snapshotHash.Delete(hashH.Hash)
}

func (store *blockMemoryStore) DeleteAccount(address string, hashH common.HashHeight) {
	store.accountHash.Delete(hashH.Hash)
	store.accountHeight.Delete(store.genKey(address, hashH.Height))
}

func (store *blockMemoryStore) PutSnapshot(block *common.SnapshotBlock) {
	store.snapshotHash.Store(block.Hash(), block)
	store.snapshotHeight.Store(block.Height(), block)
}

func (store *blockMemoryStore) PutAccount(address string, block *common.AccountStateBlock) {
	store.accountHash.Store(block.Hash(), block)
	store.accountHeight.Store(store.genKey(address, block.Height()), block)
}

func (store *blockMemoryStore) GetSnapshotByHash(hash string) *common.SnapshotBlock {
	value, ok := store.snapshotHash.Load(hash)
	if !ok {
		return nil
	}
	return value.(*common.SnapshotBlock)
}

func (store *blockMemoryStore) GetSnapshotByHeight(height uint64) *common.SnapshotBlock {
	value, ok := store.snapshotHeight.Load(height)
	if !ok {
		return nil
	}
	return value.(*common.SnapshotBlock)
}

func (store *blockMemoryStore) GetAccountByHash(address, hash string) *common.AccountStateBlock {
	value, ok := store.accountHash.Load(hash)
	if !ok {
		return nil
	}
	return value.(*common.AccountStateBlock)
}

func (store *blockMemoryStore) GetAccountBySourceHash(hash string) *common.AccountStateBlock {
	h, ok := store.sourceHash.Load(hash)
	if !ok {
		return nil
	}
	hashH := h.(*common.AccountHashH)
	return store.GetAccountByHeight(hashH.Addr, hashH.Height)
}
func (store *blockMemoryStore) PutSourceHash(hash string, h *common.AccountHashH) {
	store.sourceHash.Store(hash, h)
}
func (store *blockMemoryStore) DeleteSourceHash(hash string) {
	store.sourceHash.Delete(hash)
}

func (store *blockMemoryStore) GetAccountByHeight(address string, height uint64) *common.AccountStateBlock {
	value, ok := store.accountHeight.Load(store.genKey(address, height))
	if !ok {
		return nil
	}
	return value.(*common.AccountStateBlock)

}

func (store *blockMemoryStore) genKey(address string, height uint64) string {
	return address + "_" + strconv.FormatUint(height, 10)
}
