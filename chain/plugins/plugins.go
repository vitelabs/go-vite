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
	store   *chain_db.Store
	plugins map[string]Plugin
}

func NewPlugins(chainDir string, chain Chain) (*Plugins, error) {
	var err error

	id, err := types.BytesToHash(crypto.Hash256([]byte("plugins")))
	if err != nil {
		return nil, err
	}

	store, err := chain_db.NewStore(path.Join(chainDir, "plugins"), 0, id)
	if err != nil {
		return nil, err
	}

	plugins := map[string]Plugin{
		"filterToken": newFilterToken(store, chain),
	}

	return &Plugins{
		store:   store,
		plugins: plugins,
	}, nil
}

func (p *Plugins) GetPlugin(name string) Plugin {
	return p.plugins[name]
}

func (p *Plugins) PrepareInsertAccountBlocks(blocks []*vm_db.VmAccountBlock) error {
	// for recover
	accountBlocks := make([]*ledger.AccountBlock, 0, len(blocks))
	for _, block := range blocks {
		accountBlocks = append(accountBlocks, block.AccountBlock)
	}

	for _, plugin := range p.plugins {
		if err := plugin.InsertAccountBlocks(accountBlocks); err != nil {
			return err
		}
	}
	return nil
}

func (p *Plugins) PrepareInsertSnapshotBlocks(snapshotBlocks []*ledger.SnapshotBlock) error {

	for _, plugin := range p.plugins {
		if err := plugin.InsertSnapshotBlocks(snapshotBlocks); err != nil {
			return err
		}
	}

	return nil
}

func (p *Plugins) PrepareDeleteAccountBlocks(blocks []*ledger.AccountBlock) error {
	for _, plugin := range p.plugins {
		if err := plugin.DeleteChunks([]*ledger.SnapshotChunk{{
			AccountBlocks: blocks,
		}}); err != nil {
			return err
		}
	}
	return nil
}

func (p *Plugins) PrepareDeleteSnapshotBlocks(chunks []*ledger.SnapshotChunk) error {
	for _, plugin := range p.plugins {
		if err := plugin.DeleteChunks(chunks); err != nil {
			return err
		}
	}
	return nil
}

func (p *Plugins) InsertAccountBlocks(blocks []*vm_db.VmAccountBlock) error {
	return nil
}
func (p *Plugins) InsertSnapshotBlocks(snapshotBlocks []*ledger.SnapshotBlock) error {
	return nil
}
func (p *Plugins) DeleteAccountBlocks(blocks []*ledger.AccountBlock) error {
	return nil
}
func (p *Plugins) DeleteChunksByHash([]hashChunk) error {
	return nil
}

func (p *Plugins) checkAndRecover() (*chain_db.Store, error) {
	return nil, nil
}
