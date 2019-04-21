package chain_plugins

import (
	"github.com/vitelabs/go-vite/chain/db"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm_db"
	"path"
)

type Plugins struct {
	chain   Chain
	store   *chain_db.Store
	plugins map[string]Plugin
}

func NewPlugins(chainDir string, chain Chain) (*Plugins, error) {
	var err error

	id, err := types.BytesToHash(crypto.Hash256([]byte("plugins")))
	if err != nil {
		return nil, err
	}

	store, err := chain_db.NewStore(path.Join(chainDir, "plugins"), id)
	if err != nil {
		return nil, err
	}

	plugins := map[string]Plugin{
		"filterToken": newFilterToken(store, chain),
		"onRoadInfo":  newOnRoadInfo(store, chain),
	}

	return &Plugins{
		chain:   chain,
		store:   store,
		plugins: plugins,
	}, nil
}

func (p *Plugins) Close() error {
	if err := p.store.Close(); err != nil {
		return err
	}

	return nil
}

func (p *Plugins) GetPlugin(name string) Plugin {
	return p.plugins[name]
}

func (p *Plugins) PrepareInsertAccountBlocks(vmBlocks []*vm_db.VmAccountBlock) error {
	// for recover
	for _, vmBlock := range vmBlocks {
		batch := p.store.NewBatch()

		for _, plugin := range p.plugins {
			if err := plugin.InsertAccountBlock(batch, vmBlock.AccountBlock); err != nil {
				return err
			}
		}
		p.store.WriteAccountBlock(batch, vmBlock.AccountBlock)
	}

	return nil
}

func (p *Plugins) PrepareInsertSnapshotBlocks(chunks []*ledger.SnapshotChunk) error {

	for _, chunk := range chunks {
		batch := p.store.NewBatch()

		for _, plugin := range p.plugins {

			if err := plugin.InsertSnapshotBlock(batch, chunk.SnapshotBlock, chunk.AccountBlocks); err != nil {
				return err
			}
		}
		p.store.WriteSnapshot(batch, chunk.AccountBlocks)

	}

	return nil
}

func (p *Plugins) PrepareDeleteAccountBlocks(blocks []*ledger.AccountBlock) error {
	batch := p.store.NewBatch()

	for _, plugin := range p.plugins {
		if err := plugin.DeleteAccountBlocks(batch, blocks); err != nil {
			return err
		}
	}
	p.store.RollbackAccountBlocks(batch, blocks)

	return nil
}

func (p *Plugins) PrepareDeleteSnapshotBlocks(chunks []*ledger.SnapshotChunk) error {
	batch := p.store.NewBatch()

	for _, plugin := range p.plugins {

		if err := plugin.DeleteSnapshotBlocks(batch, chunks); err != nil {
			return err
		}

	}
	p.store.RollbackSnapshot(batch)

	return nil
}

func (p *Plugins) DeleteSnapshotBlocks(chunks []*ledger.SnapshotChunk) error {
	unconfirmedBlocks := p.chain.GetAllUnconfirmedBlocks()
	if len(unconfirmedBlocks) <= 0 {
		return nil
	}

	for _, block := range unconfirmedBlocks {
		batch := p.store.NewBatch()
		for _, plugin := range p.plugins {

			// recover
			if err := plugin.InsertAccountBlock(batch, block); err != nil {
				return err
			}
		}

		p.store.WriteAccountBlock(batch, block)
	}
	return nil
}

func (p *Plugins) InsertAccountBlocks(blocks []*vm_db.VmAccountBlock) error {
	return nil
}
func (p *Plugins) InsertSnapshotBlocks(chunks []*ledger.SnapshotChunk) error {
	return nil
}
func (p *Plugins) DeleteAccountBlocks(blocks []*ledger.AccountBlock) error {
	return nil
}

func (p *Plugins) checkAndRecover() (*chain_db.Store, error) {
	return nil, nil
}
