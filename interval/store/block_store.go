package store

import (
	"time"

	"path"

	"github.com/vitelabs/go-vite/interval/common"
	"github.com/vitelabs/go-vite/interval/common/config"
	"github.com/vitelabs/go-vite/interval/common/log"
	"github.com/vitelabs/go-vite/interval/tools"
)

type BlockStore interface {
	PutSnapshot(block *common.SnapshotBlock)
	PutAccount(address string, block *common.AccountStateBlock)
	DeleteSnapshot(hashH common.HashHeight)
	DeleteAccount(address string, hashH common.HashHeight)

	SetSnapshotHead(hashH *common.HashHeight)
	SetAccountHead(address string, hashH *common.HashHeight)

	GetSnapshotHead() *common.HashHeight
	GetAccountHead(address string) *common.HashHeight

	GetSnapshotByHash(hash string) *common.SnapshotBlock
	GetSnapshotByHeight(height uint64) *common.SnapshotBlock

	GetAccountByHash(address string, hash string) *common.AccountStateBlock
	GetAccountByHeight(address string, height uint64) *common.AccountStateBlock

	GetAccountBySourceHash(hash string) *common.AccountStateBlock
	PutSourceHash(hash string, block *common.AccountHashH)
	DeleteSourceHash(hash string)
}

var genesisBlocks []*common.AccountStateBlock

func init() {
	var genesisAccounts = []string{"viteshan", "jie"}
	for _, a := range genesisAccounts {
		genesis := common.NewAccountBlock(common.FirstHeight, "", "", a, time.Unix(0, 0),
			200, 0, 0, "", common.GENESIS, a, a, nil)
		genesis.SetHash(tools.CalculateAccountHash(genesis))
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
