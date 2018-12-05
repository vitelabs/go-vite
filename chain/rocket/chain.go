package rocket

import (
	"github.com/vitelabs/go-vite/chain_db"
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/log15"
	"path/filepath"
	"sync"
)

type chain struct {
	chainDb           *chain_db.ChainDb
	cfg               *config.Chain
	dataDir           string
	log               log15.Logger
	createAccountLock sync.Mutex

	ledgerDirName string
}

func NewChain(cfg *config.Config) Chain {
	chain := &chain{
		dataDir:       cfg.DataDir,
		cfg:           cfg.Chain,
		ledgerDirName: "ledger",
	}

	if chain.cfg == nil {
		chain.cfg = &config.Chain{}
	}

	// chainDb
	chainDb := chain_db.NewChainDb(filepath.Join(chain.dataDir, chain.ledgerDirName))
	if chainDb == nil {
		chain.log.Crit("NewChain failed, db init failed", "method", "Init")
	}
	chain.chainDb = chainDb

	return chain
}
