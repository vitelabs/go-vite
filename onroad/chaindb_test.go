package onroad

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

type mockChain interface {
	LatestSnapshotBlock() (*ledger.SnapshotBlock, error)
	GetLatestAccountBlock(addr types.Address) (*ledger.AccountBlock, error)
	GetContractMeta(contractAddress types.Address) (meta *ledger.ContractMeta, err error)
	IsSeedConfirmedNTimes(blockHash types.Hash, n uint64) (bool, error)
	GetConfirmedTimes(blockHash types.Hash) (uint64, error)
	LoadOnRoad(gid types.Gid) (map[types.Address]map[types.Address][]ledger.HashHeight, error)
	GetAccountBlockByHash(blockHash types.Hash) (*ledger.AccountBlock, error)
	GetCompleteBlockByHash(blockHash types.Hash) (*ledger.AccountBlock, error)
	GetStakeQuota(addr types.Address) (*types.Quota, error)
	GetStakeQuotas(addrList []types.Address) (map[types.Address]*types.Quota, error)
	IsGenesisAccountBlock(hash types.Hash) bool
	InsertAccountBlock(b *ledger.AccountBlock) error
	PrepareDeleteAccountBlocks(b *ledger.AccountBlock) error
	PrepareDeleteSnapshotBlocks(chunks []*ledger.SnapshotChunk) error
	InsertSnapshotBlocks(chunks []*ledger.SnapshotChunk) error
	DeleteOnRoad(toAddress types.Address, sendBlockHash types.Hash) error
}

type Account struct {
	addr         types.Address
	contractMeta *ledger.ContractMeta
	hasRsBlock   bool
	quota        uint64

	blocksMap        map[types.Hash]*ledger.AccountBlock
	sendBlocksMap    map[types.Hash]*ledger.AccountBlock
	receiveBlocksMap map[types.Hash]*ledger.AccountBlock
	onRoadBlocks     map[types.Hash]*ledger.AccountBlock
}

func MockAccount(meta *ledger.ContractMeta, sendCreate *ledger.AccountBlock, hasRsBlock bool) *Account {
	account := &Account{
		addr:             types.Address{},
		quota:            0,
		blocksMap:        make(map[types.Hash]*ledger.AccountBlock),
		sendBlocksMap:    make(map[types.Hash]*ledger.AccountBlock),
		receiveBlocksMap: make(map[types.Hash]*ledger.AccountBlock),
		onRoadBlocks:     make(map[types.Hash]*ledger.AccountBlock),
	}
	if meta != nil {
		account.contractMeta = meta
		account.hasRsBlock = hasRsBlock
		account.blocksMap[sendCreate.Hash] = sendCreate
		account.sendBlocksMap[sendCreate.Hash] = sendCreate
		account.onRoadBlocks[sendCreate.Hash] = sendCreate
	}
	return account
}

type ChainDb struct {
	AccChain      map[types.Address]*Account
	SnapshotChain []*ledger.SnapshotBlock
}

func MockChainDb(newContractCount uint8) mockChain {
	db := &ChainDb{
		AccChain:      make(map[types.Address]*Account),
		SnapshotChain: make([]*ledger.SnapshotBlock, 0),
	}
	for i := uint8(0); i < newContractCount; i++ {
		meta, b := mockSendCreate(types.DELEGATE_GID, 1, 1)
		rsAc := MockAccount(meta, b, true)
		db.AccChain[rsAc.addr] = rsAc
	}
	return db
}

/* method to query mockChain db */

func (db *ChainDb) LatestSnapshotBlock() (*ledger.SnapshotBlock, error) {
	return db.SnapshotChain[len(db.SnapshotChain)-1], nil
}

func (db *ChainDb) GetLatestAccountBlock(addr types.Address) (*ledger.AccountBlock, error) {
	return nil, nil
}

func (db *ChainDb) GetContractMeta(contractAddress types.Address) (meta *ledger.ContractMeta, err error) {
	value, ok := db.AccChain[contractAddress]
	if ok && value != nil {
		return value.contractMeta, nil
	}
	return nil, nil
}

func (db *ChainDb) IsSeedConfirmedNTimes(blockHash types.Hash, n uint64) (bool, error) {
	return true, nil
}

func (db *ChainDb) GetConfirmedTimes(blockHash types.Hash) (uint64, error) { return 0, nil }

func (db *ChainDb) LoadOnRoad(gid types.Gid) (map[types.Address]map[types.Address][]ledger.HashHeight, error) {
	return nil, nil
}

func (db *ChainDb) GetAccountBlockByHash(blockHash types.Hash) (*ledger.AccountBlock, error) {
	return nil, nil
}

func (db *ChainDb) GetCompleteBlockByHash(blockHash types.Hash) (*ledger.AccountBlock, error) {
	return nil, nil
}

func (db *ChainDb) GetStakeQuota(addr types.Address) (*types.Quota, error) { return nil, nil }

func (db *ChainDb) GetStakeQuotas(addrList []types.Address) (map[types.Address]*types.Quota, error) {
	return nil, nil
}

func (db *ChainDb) IsGenesisAccountBlock(hash types.Hash) bool { return false }

/* methods which can manipulate the mockChain */

func (db *ChainDb) InsertAccountBlock(b *ledger.AccountBlock) error { return nil }

func (db *ChainDb) PrepareDeleteAccountBlocks(b *ledger.AccountBlock) error { return nil }

func (db *ChainDb) PrepareDeleteSnapshotBlocks(chunks []*ledger.SnapshotChunk) error { return nil }

func (db *ChainDb) InsertSnapshotBlocks(chunks []*ledger.SnapshotChunk) error { return nil }

func (db *ChainDb) DeleteOnRoad(toAddress types.Address, sendBlockHash types.Hash) error { return nil }

/* for mock new db data */

func mockSendCreate(gid types.Gid, sendCTimes, seedCTimes uint8) (*ledger.ContractMeta, *ledger.AccountBlock) {
	b := &ledger.AccountBlock{
		BlockType: ledger.BlockTypeSendCreate,
	}
	meta := &ledger.ContractMeta{
		Gid:                gid,
		SendConfirmedTimes: sendCTimes,
		CreateBlockHash:    b.Hash,
		QuotaRatio:         0,
		SeedConfirmedTimes: seedCTimes,
	}
	return meta, b
}

func mockNewSnapshotBlock() *ledger.SnapshotBlock { return nil }

func mockNewSendBlock() *ledger.AccountBlock { return nil }

func mockNewRSBlock() *ledger.AccountBlock { return nil }
