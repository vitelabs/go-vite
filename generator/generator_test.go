package generator

import (
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/trie"
	"math/big"
	"testing"
	"time"
)

func TestNewGenerator(t *testing.T) {
	//db, addr1, hash12, snapshot2, _ := PrepareDb()
	//gen := NewGenerator(db, db)
	//gen.PrepareVm(&snapshot2.Hash, &hash12, &addr1)
}

func TestGenerator_GenerateWithMessage(t *testing.T) {

}

func TestGenerator_PackBlockWithSendBlock(t *testing.T) {

}

func TestGenerator_GenerateWithOnroad(t *testing.T) {

}

var (
	attovPerVite    = big.NewInt(1e18)
	viteTotalSupply = new(big.Int).Mul(big.NewInt(1e9), attovPerVite)
)

func PrepareDb() (db *testDatabase, addr1 types.Address, hash12 types.Hash, snapshot2 *ledger.SnapshotBlock, timestamp int64) {
	db = NewNoDatabase()
	addr1, _ = types.BytesToAddress(helper.HexToBytes("CA35B7D915458EF540ADE6068DFE2F44E8FA733C"))
	timestamp = 1536214502
	t1 := time.Unix(timestamp-1, 0)
	snapshot1 := &ledger.SnapshotBlock{Height: 1, Timestamp: &t1, Hash: types.DataHash([]byte{10, 1})}
	db.snapshotBlockList = append(db.snapshotBlockList, snapshot1)
	t2 := time.Unix(timestamp, 0)
	snapshot2 = &ledger.SnapshotBlock{Height: 2, Timestamp: &t2, Hash: types.DataHash([]byte{10, 2})}
	db.snapshotBlockList = append(db.snapshotBlockList, snapshot2)

	hash11 := types.DataHash([]byte{1, 1})
	block11 := &ledger.AccountBlock{
		Height:         1,
		ToAddress:      addr1,
		AccountAddress: addr1,
		BlockType:      ledger.BlockTypeSendCall,
		Fee:            big.NewInt(0),
		Amount:         viteTotalSupply,
		TokenId:        ledger.ViteTokenId,
		SnapshotHash:   snapshot1.Hash,
	}
	db.accountBlockMap[addr1] = make(map[types.Hash]*ledger.AccountBlock)
	db.accountBlockMap[addr1][hash11] = block11
	hash12 = types.DataHash([]byte{1, 2})
	block12 := &ledger.AccountBlock{
		Height:         2,
		ToAddress:      addr1,
		AccountAddress: addr1,
		FromBlockHash:  hash11,
		BlockType:      ledger.BlockTypeReceive,
		PrevHash:       hash11,
		Amount:         viteTotalSupply,
		TokenId:        ledger.ViteTokenId,
		SnapshotHash:   snapshot1.Hash,
	}
	db.accountBlockMap[addr1][hash12] = block12
	return
}

type testDatabase struct {
	snapshotBlockList []*ledger.SnapshotBlock
	accountBlockMap   map[types.Address]map[types.Hash]*ledger.AccountBlock
	contractGidMap    map[types.Address]*types.Gid
}

func NewNoDatabase() *testDatabase {
	return &testDatabase{
		contractGidMap:    make(map[types.Address]*types.Gid),
		snapshotBlockList: make([]*ledger.SnapshotBlock, 0),
		accountBlockMap:   make(map[types.Address]map[types.Hash]*ledger.AccountBlock),
	}
}

func (test *testDatabase) GetAccount(address *types.Address) (*ledger.Account, error) {
	return nil, nil
}

func (test *testDatabase) GetLatestSnapshotBlock() *ledger.SnapshotBlock {
	return nil
}

func (test *testDatabase) GetSnapshotBlockByHash(hash *types.Hash) (*ledger.SnapshotBlock, error) {
	return nil, nil
}

func (test *testDatabase) GetSnapshotBlockByHeight(height uint64) (*ledger.SnapshotBlock, error) {
	return nil, nil
}

func (test *testDatabase) GetSnapshotBlocksByHeight(height uint64, count uint64, forward, containSnapshotContent bool) ([]*ledger.SnapshotBlock, error) {
	return nil, nil
}

func (test *testDatabase) GetAccountBlockByHash(blockHash *types.Hash) (*ledger.AccountBlock, error) {
	return nil, nil
}

func (test *testDatabase) GetStateTrie(hash *types.Hash) *trie.Trie {
	return nil
}

func (test *testDatabase) NewStateTrie() *trie.Trie {
	return nil
}

func (test *testDatabase) GetConfirmAccountBlock(snapshotHeight uint64, address *types.Address) (*ledger.AccountBlock, error) {
	return nil, nil

}

func (test *testDatabase) GetContractGid(addr *types.Address) (*types.Gid, error) {
	return nil, nil
}

func (test *testDatabase) GetLatestAccountBlock(addr *types.Address) (*ledger.AccountBlock, error) {
	return nil, nil
}

func (test *testDatabase) SignData(a types.Address, data []byte) (signedData, pubkey []byte, err error) {
	return nil, nil, nil
}

func (test *testDatabase) SignDataWithPassphrase(a types.Address, passphrase string, data []byte) (signedData, pubkey []byte, err error) {
	return nil, nil, nil
}
