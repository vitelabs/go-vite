package store

import (
	"encoding/binary"
	"strconv"

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

func (self *blockLeveldbStore) initAccountGenesis() {
	for _, genesis := range genesisBlocks {
		block := self.GetAccountByHeight(genesis.Signer(), common.FirstHeight)
		if block == nil {
			self.PutAccount(genesis.Signer(), genesis)
		}
		head := self.GetAccountHead(genesis.Signer())
		if head == nil {
			self.SetAccountHead(genesis.Signer(), &common.HashHeight{Hash: genesis.Hash(), Height: genesis.Height()})
		}
	}
}

func (self *blockLeveldbStore) genSnapshotHashKey(hash string) []byte {
	return self.genHashKey("sh", hash)
}
func (self *blockLeveldbStore) genAccountHashKey(address, hash string) []byte {
	return self.genHashKey("ah_"+address, hash)
}
func (self *blockLeveldbStore) genAccountSourceHashKey(hash string) []byte {
	return self.genHashKey("as_", hash)
}
func (self *blockLeveldbStore) genSnapshotHeightKey(height uint64) []byte {
	return self.genHeightKey("se", height)
}
func (self *blockLeveldbStore) genAccountHeightKey(address string, height uint64) []byte {
	return self.genHeightKey("ae_"+address, height)
}

func (self *blockLeveldbStore) genSnapshotHeadKey() []byte {
	return []byte("she")
}

func (self *blockLeveldbStore) genAccountHeadKey(address string) []byte {
	return []byte("ahe_" + address)
}

func (self *blockLeveldbStore) genHashKey(prefix string, hash string) []byte {
	return []byte(prefix + "_" + hash)
}
func (self *blockLeveldbStore) genHeightKey(prefix string, height uint64) []byte {
	return []byte(prefix + "_" + strconv.FormatUint(height, 10))
}

func (self *blockLeveldbStore) PutSnapshot(block *common.SnapshotBlock) {
	var heightbuf [8]byte
	binary.BigEndian.PutUint64(heightbuf[:], block.Height())
	heightKey := self.genSnapshotHeightKey(block.Height())
	hashKey := self.genSnapshotHashKey(block.Hash())
	blockByt, e := gencode.SerializeSnapshotBlock(block)
	if e != nil {
		panic(e)
	}

	self.db.Put(hashKey, heightbuf[:], nil)
	self.db.Put(heightKey, blockByt, nil)
}

func (self *blockLeveldbStore) PutAccount(address string, block *common.AccountStateBlock) {
	var heightbuf [8]byte
	binary.BigEndian.PutUint64(heightbuf[:], block.Height())
	heightKey := self.genAccountHeightKey(address, block.Height())
	hashKey := self.genAccountHashKey(address, block.Hash())

	blockByt, e := gencode.SerializeAccountBlock(block)
	if e != nil {
		panic(e)
	}

	self.db.Put(hashKey, heightbuf[:], nil)
	self.db.Put(heightKey, blockByt, nil)
}

func (self *blockLeveldbStore) DeleteSnapshot(hashH common.HashHeight) {
	heightKey := self.genSnapshotHeightKey(hashH.Height)
	hashKey := self.genSnapshotHashKey(hashH.Hash)
	self.db.Delete(heightKey, nil)
	self.db.Delete(hashKey, nil)
}

func (self *blockLeveldbStore) DeleteAccount(address string, hashH common.HashHeight) {
	heightKey := self.genAccountHeightKey(address, hashH.Height)
	hashKey := self.genAccountHashKey(address, hashH.Hash)
	self.db.Delete(heightKey, nil)
	self.db.Delete(hashKey, nil)
}

func (self *blockLeveldbStore) SetSnapshotHead(hashH *common.HashHeight) {
	key := self.genSnapshotHeadKey()
	bytes, e := gencode.SerializeHashHeight(hashH)
	if e != nil {
		panic(e)
	}
	self.db.Put(key, bytes, nil)
}

func (self *blockLeveldbStore) SetAccountHead(address string, hashH *common.HashHeight) {
	key := self.genAccountHeadKey(address)

	bytes, e := gencode.SerializeHashHeight(hashH)
	if e != nil {
		panic(e)
	}
	self.db.Put(key, bytes, nil)
}

func (self *blockLeveldbStore) GetSnapshotHead() *common.HashHeight {
	key := self.genSnapshotHeadKey()
	byt, err := self.db.Get(key, nil)
	if err != nil {
		return nil
	}
	hashH, e := gencode.DeserializeHashHeight(byt)
	if e != nil {
		return nil
	}
	return hashH
}

func (self *blockLeveldbStore) GetAccountHead(address string) *common.HashHeight {
	key := self.genAccountHeadKey(address)
	byt, err := self.db.Get(key, nil)
	if err != nil {
		return nil
	}
	hashH, e := gencode.DeserializeHashHeight(byt)
	if e != nil {
		return nil
	}
	return hashH
}

func (self *blockLeveldbStore) GetSnapshotByHash(hash string) *common.SnapshotBlock {
	key := self.genSnapshotHashKey(hash)
	value, err := self.db.Get(key, nil)
	if err != nil {
		return nil
	}
	height := binary.BigEndian.Uint64(value)
	return self.GetSnapshotByHeight(height)

}

func (self *blockLeveldbStore) GetSnapshotByHeight(height uint64) *common.SnapshotBlock {
	key := self.genSnapshotHeightKey(height)
	value, err := self.db.Get(key, nil)
	if err != nil {
		return nil
	}
	block, e := gencode.DeserializeSnapshotBlock(value)
	if e != nil {
		return nil
	}
	return block
}

func (self *blockLeveldbStore) GetAccountByHash(address string, hash string) *common.AccountStateBlock {
	key := self.genAccountHashKey(address, hash)
	value, err := self.db.Get(key, nil)
	if err != nil {
		return nil
	}
	height := binary.BigEndian.Uint64(value)
	return self.GetAccountByHeight(address, height)
}

func (self *blockLeveldbStore) GetAccountByHeight(address string, height uint64) *common.AccountStateBlock {
	key := self.genAccountHeightKey(address, height)
	value, err := self.db.Get(key, nil)
	if err != nil {
		return nil
	}
	block, e := gencode.DeserializeAccountBlock(value)
	if e != nil {
		return nil
	}
	return block
}

func (self *blockLeveldbStore) GetAccountBySourceHash(hash string) *common.AccountStateBlock {
	key := self.genAccountSourceHashKey(hash)
	value, err := self.db.Get(key, nil)
	if err != nil {
		return nil
	}
	h, e := gencode.DeserializeAccountHashH(value)
	if e != nil {
		return nil
	}
	return self.GetAccountByHeight(h.Addr, h.Height)
}

func (self *blockLeveldbStore) PutSourceHash(hash string, aH *common.AccountHashH) {
	key := self.genAccountSourceHashKey(hash)
	byt, e := gencode.SerializeAccountHashH(aH)
	if e != nil {
		panic(e)
	}
	self.db.Put(key, byt, nil)
}

func (self *blockLeveldbStore) DeleteSourceHash(hash string) {
	key := self.genAccountSourceHashKey(hash)
	self.db.Delete(key, nil)
}
