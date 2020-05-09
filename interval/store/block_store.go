package store

import (
	"time"

	"path"

	"github.com/vitelabs/go-vite/interval/common"
	"github.com/vitelabs/go-vite/interval/common/config"
	"github.com/vitelabs/go-vite/interval/common/log"
	"github.com/vitelabs/go-vite/interval/utils"
)

type BlockStore interface {
	PutSnapshot(block *common.SnapshotBlock)
	PutAccount(address common.Address, block *common.AccountStateBlock)
	DeleteSnapshot(hashH common.HashHeight)
	DeleteAccount(address common.Address, hashH common.HashHeight)

	SetSnapshotHead(hashH *common.HashHeight)
	SetAccountHead(address common.Address, hashH *common.HashHeight)

	GetSnapshotHead() *common.HashHeight
	GetAccountHead(address common.Address) *common.HashHeight

	GetSnapshotByHash(hash common.Hash) *common.SnapshotBlock
	GetSnapshotByHeight(height common.Height) *common.SnapshotBlock

	GetAccountByHash(address common.Address, hash common.Hash) *common.AccountStateBlock
	GetAccountByHeight(address common.Address, height common.Height) *common.AccountStateBlock

	GetAccountBySourceHash(hash common.Hash) *common.AccountStateBlock
	PutSourceHash(hash common.Hash, block *common.AccountHashH)
	DeleteSourceHash(hash common.Hash)
}

var genesisBlocks []*common.AccountStateBlock

func init() {
	var genesisAccounts = []common.Address{common.Address("viteshan"), common.Address("jie")}
	for _, a := range genesisAccounts {
		genesis := common.NewAccountBlock(common.FirstHeight, "", "", a, time.Unix(0, 0),
			200, 0, common.GENESIS, a, a, nil)
		genesis.SetHash(utils.CalculateAccountHash(genesis))
		genesisBlocks = append(genesisBlocks, genesis)
	}
}

func NewStore(cfg *config.Chain) BlockStore {
	if cfg.StoreType == common.Memory {
		log.Info("store use memory db.")
		return newMemoryStore()
	} else if cfg.StoreType == common.LevelDB {
		log.Info("store use leveldb.")
		return newBlockLeveldbStore(path.Join(cfg.DataDir, cfg.DBPath))
	} else {
		panic("unknown StoreType")
	}
}
