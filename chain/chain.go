package chain

import (
	"github.com/vitelabs/go-vite/chain_db"
	"github.com/vitelabs/go-vite/compress"
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/log15"
	"path/filepath"
)

var chainLog = log15.New("module", "chain")

type Chain struct {
	chainDb    *chain_db.ChainDb
	compressor *compress.Compressor
}

func NewChain(cfg *config.Config) *Chain {
	chainDb := chain_db.NewChainDb(filepath.Join(cfg.DataDir, "chain"))
	if chainDb == nil {
		chainLog.Error("NewChain failed")
		return nil
	}

	compressor := compress.NewCompressor()
	return &Chain{
		chainDb:    chainDb,
		compressor: compressor,
	}
}

func (c *Chain) Start() {
	// Start compress in the background
	c.compressor.Start()
}

func (c *Chain) Stop() {
	// Stop compress
	c.compressor.Stop()
}
