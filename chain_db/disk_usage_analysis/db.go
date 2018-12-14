package disk_usage_analysis

import (
	"github.com/vitelabs/go-vite/chain_db"
	"github.com/vitelabs/go-vite/node"
	"path/filepath"
)

func newChainDb(dataDir string) *chain_db.ChainDb {
	return chain_db.NewChainDb(filepath.Join(node.DefaultDataDir(), dataDir, "ledger"))
}
