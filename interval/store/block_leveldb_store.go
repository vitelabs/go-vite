package store

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/interval/common"
	"github.com/vitelabs/go-vite/interval/store/serializer/gencode"
)

func newBlockLeveldbStore(path string) BlockStore {
	db, err := leveldb.OpenFile(path, nil)
	if err != nil {
		panic(err)
	}
	self := &blockLeveldbStore{db: db}
	self.initAccountGenesis()
	return self
}

// thread safe block levelDB store
type blockLeveldbStore struct {
	db *leveldb.DB
}

func (store *blockLeveldbStore) initAccountGenesis() {
	for _, genesis := range genesisBlocks {
		block := store.GetAccountByHeight(genesis.Signer(), common.FirstHeight)
		if block == nil {
			store.PutAccount(genesis.Signer(), genesis)
		}
		head := store.GetAccountHead(genesis.Signer())
		if head == nil {
			store.SetAccountHead(genesis.Signer(), &common.HashHeight{Hash: genesis.Hash(), Height: genesis.Height()})
		}
	}
}

func (store *blockLeveldbStore) genSnapshotHashKey(hash common.Hash) []byte {
	return store.genHashKey("sh", hash)
}
func (store *blockLeveldbStore) genAccountHashKey(address common.Address, hash common.Hash) []byte {
	return store.genHashKey("ah_"+address.String(), hash)
}
func (store *blockLeveldbStore) genAccountSourceHashKey(hash common.Hash) []byte {
	return store.genHashKey("as_", hash)
}
func (store *blockLeveldbStore) genSnapshotHeightKey(height common.Height) []byte {
	return store.genHeightKey("se", height)
}
func (store *blockLeveldbStore) genAccountHeightKey(address common.Address, height common.Height) []byte {
	return store.genHeightKey("ae_"+address.String(), height)
}

func (store *blockLeveldbStore) genSnapshotHeadKey() []byte {
	return []byte("she")
}

func (store *blockLeveldbStore) genAccountHeadKey(address common.Address) []byte {
	return []byte("ahe_" + address)
}

func (store *blockLeveldbStore) genHashKey(prefix string, hash common.Hash) []byte {
	return []byte(prefix + "_" + hash.String())
}
func (store *blockLeveldbStore) genHeightKey(prefix string, height common.Height) []byte {
	return []byte(prefix + "_" + height.String())
}

func (store *blockLeveldbStore) PutSnapshot(block *common.SnapshotBlock) {
	heightKey := store.genSnapshotHeightKey(block.Height())
	hashKey := store.genSnapshotHashKey(block.Hash())
	blockByt, e := gencode.SerializeSnapshotBlock(block)
	if e != nil {
		panic(e)
	}

	store.db.Put(hashKey, block.Height().Bytes(), nil)
	store.db.Put(heightKey, blockByt, nil)
}

func (store *blockLeveldbStore) PutAccount(address common.Address, block *common.AccountStateBlock) {
	heightKey := store.genAccountHeightKey(address, block.Height())
	hashKey := store.genAccountHashKey(address, block.Hash())

	blockByt, e := gencode.SerializeAccountBlock(block)
	if e != nil {
		panic(e)
	}

	store.db.Put(hashKey, block.Height().Bytes(), nil)
	store.db.Put(heightKey, blockByt, nil)
}

func (store *blockLeveldbStore) DeleteSnapshot(hashH common.HashHeight) {
	heightKey := store.genSnapshotHeightKey(hashH.Height)
	hashKey := store.genSnapshotHashKey(hashH.Hash)
	store.db.Delete(heightKey, nil)
	store.db.Delete(hashKey, nil)
}

func (store *blockLeveldbStore) DeleteAccount(address common.Address, hashH common.HashHeight) {
	heightKey := store.genAccountHeightKey(address, hashH.Height)
	hashKey := store.genAccountHashKey(address, hashH.Hash)
	store.db.Delete(heightKey, nil)
	store.db.Delete(hashKey, nil)
}

func (store *blockLeveldbStore) SetSnapshotHead(hashH *common.HashHeight) {
	key := store.genSnapshotHeadKey()
	bytes, e := gencode.SerializeHashHeight(hashH)
	if e != nil {
		panic(e)
	}
	store.db.Put(key, bytes, nil)
}

func (store *blockLeveldbStore) SetAccountHead(address common.Address, hashH *common.HashHeight) {
	key := store.genAccountHeadKey(address)

	bytes, e := gencode.SerializeHashHeight(hashH)
	if e != nil {
		panic(e)
	}
	store.db.Put(key, bytes, nil)
}

func (store *blockLeveldbStore) GetSnapshotHead() *common.HashHeight {
	key := store.genSnapshotHeadKey()
	byt, err := store.db.Get(key, nil)
	if err != nil {
		return nil
	}
	hashH, e := gencode.DeserializeHashHeight(byt)
	if e != nil {
		return nil
	}
	return hashH
}

func (store *blockLeveldbStore) GetAccountHead(address common.Address) *common.HashHeight {
	key := store.genAccountHeadKey(address)
	byt, err := store.db.Get(key, nil)
	if err != nil {
		return nil
	}
	hashH, e := gencode.DeserializeHashHeight(byt)
	if e != nil {
		return nil
	}
	return hashH
}

func (store *blockLeveldbStore) GetSnapshotByHash(hash common.Hash) *common.SnapshotBlock {
	key := store.genSnapshotHashKey(hash)
	value, err := store.db.Get(key, nil)
	if err != nil {
		return nil
	}
	return store.GetSnapshotByHeight(common.BytesToHeight(value))

}

func (store *blockLeveldbStore) GetSnapshotByHeight(height common.Height) *common.SnapshotBlock {
	key := store.genSnapshotHeightKey(height)
	value, err := store.db.Get(key, nil)
	if err != nil {
		return nil
	}
	block, e := gencode.DeserializeSnapshotBlock(value)
	if e != nil {
		return nil
	}
	return block
}

func (store *blockLeveldbStore) GetAccountByHash(address common.Address, hash common.Hash) *common.AccountStateBlock {
	key := store.genAccountHashKey(address, hash)
	value, err := store.db.Get(key, nil)
	if err != nil {
		return nil
	}
	return store.GetAccountByHeight(address, common.BytesToHeight(value))
}

func (store *blockLeveldbStore) GetAccountByHeight(address common.Address, height common.Height) *common.AccountStateBlock {
	key := store.genAccountHeightKey(address, height)
	value, err := store.db.Get(key, nil)
	if err != nil {
		return nil
	}
	block, e := gencode.DeserializeAccountBlock(value)
	if e != nil {
		return nil
	}
	return block
}

func (store *blockLeveldbStore) GetAccountBySourceHash(hash common.Hash) *common.AccountStateBlock {
	key := store.genAccountSourceHashKey(hash)
	value, err := store.db.Get(key, nil)
	if err != nil {
		return nil
	}
	h, e := gencode.DeserializeAccountHashH(value)
	if e != nil {
		return nil
	}
	return store.GetAccountByHeight(h.Addr, h.Height)
}

func (store *blockLeveldbStore) PutSourceHash(hash common.Hash, aH *common.AccountHashH) {
	key := store.genAccountSourceHashKey(hash)
	byt, e := gencode.SerializeAccountHashH(aH)
	if e != nil {
		panic(e)
	}
	store.db.Put(key, byt, nil)
}

func (store *blockLeveldbStore) DeleteSourceHash(hash common.Hash) {
	key := store.genAccountSourceHashKey(hash)
	store.db.Delete(key, nil)
}
