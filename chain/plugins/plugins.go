package chain_plugins

import (
	"github.com/vitelabs/go-vite/chain/db"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/vm_db"
	"path"
	"sync"
	"sync/atomic"
)

const roundSize = uint64(10)

const (
	stop  = 0
	start = 1
)

const (
	PluginKeyFilterToken = "plugin_filter_token"
	PluginKeyOnRoadInfo  = "plugin_onroad_info"
)

type Plugins struct {
	log   log15.Logger
	chain Chain

	plugins map[string]Plugin

	writeStatus uint32
	mu          sync.RWMutex
}

func NewPlugins(chainDir string, chain Chain, pluginsName []string) (*Plugins, error) {

	// default open PluginKeyOnRoadInfo
	onroadInfoStore, onroadInfoErr := chain_db.NewStore(path.Join(chainDir, PluginKeyOnRoadInfo), PluginKeyOnRoadInfo)
	if onroadInfoErr != nil {
		return nil, onroadInfoErr
	}
	plugins := map[string]Plugin{PluginKeyOnRoadInfo: newOnRoadInfo(onroadInfoStore, chain)}

	for _, v := range pluginsName {
		switch v {
		case PluginKeyFilterToken:
			store, err := chain_db.NewStore(path.Join(chainDir, "plugins"), PluginKeyFilterToken)
			if err != nil {
				return nil, err
			}
			plugins[PluginKeyFilterToken] = newFilterToken(store, chain)
		default:
		}
	}

	return &Plugins{
		chain:       chain,
		plugins:     plugins,
		writeStatus: start,
		log:         log15.New("module", "chain_plugins"),
	}, nil
}

func (p *Plugins) StopWrite() {
	if !atomic.CompareAndSwapUint32(&p.writeStatus, start, stop) {
		return
	}

	p.mu.Lock()
}

func (p *Plugins) StartWrite() {
	if !atomic.CompareAndSwapUint32(&p.writeStatus, stop, start) {
		return
	}

	p.mu.Unlock()
}

func (p *Plugins) RebuildData() error {
	p.StopWrite()
	defer p.StartWrite()

	p.log.Info("Start rebuild plugin data")

	for _, plugin := range p.plugins {
		if err := plugin.RebuildData(p.chain.Flusher()); err != nil {
			return err
		}
	}
	// success
	p.log.Info("Succeed rebuild plugin data")
	return nil
}

func (p *Plugins) Close() error {
	for _, v := range p.plugins {
		if err := v.GetStore().Close(); err != nil {
			return err
		}
	}
	return nil
}

func (p *Plugins) Stores() []*chain_db.Store {
	storeList := make([]*chain_db.Store, 0)
	for _, v := range p.plugins {
		storeList = append(storeList, v.GetStore())
	}
	return storeList
}

func (p *Plugins) GetPlugin(name string) Plugin {
	return p.plugins[name]
}

func (p *Plugins) RemovePlugin(name string) {
	delete(p.plugins, name)
}

func (p *Plugins) PrepareInsertAccountBlocks(vmBlocks []*vm_db.VmAccountBlock) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// for recover
	for _, vmBlock := range vmBlocks {

		for _, plugin := range p.plugins {

			batch := plugin.GetStore().NewBatch()

			if err := plugin.InsertAccountBlock(batch, vmBlock.AccountBlock); err != nil {
				return err
			}
			plugin.GetStore().WriteAccountBlock(batch, vmBlock.AccountBlock)

		}
	}

	return nil
}

func (p *Plugins) PrepareInsertSnapshotBlocks(chunks []*ledger.SnapshotChunk) error {
	p.mu.RLock()
	defer p.mu.RUnlock()
	for _, chunk := range chunks {

		for _, plugin := range p.plugins {

			batch := plugin.GetStore().NewBatch()

			if err := plugin.InsertSnapshotBlock(batch, chunk.SnapshotBlock, chunk.AccountBlocks); err != nil {
				return err
			}

			plugin.GetStore().WriteSnapshot(batch, chunk.AccountBlocks)
		}
	}

	return nil
}

func (p *Plugins) PrepareDeleteAccountBlocks(blocks []*ledger.AccountBlock) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for _, plugin := range p.plugins {

		batch := plugin.GetStore().NewBatch()

		if err := plugin.DeleteAccountBlocks(batch, blocks); err != nil {
			return err
		}

		plugin.GetStore().RollbackAccountBlocks(batch, blocks)
	}

	return nil
}

func (p *Plugins) PrepareDeleteSnapshotBlocks(chunks []*ledger.SnapshotChunk) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for _, plugin := range p.plugins {

		batch := plugin.GetStore().NewBatch()

		if err := plugin.DeleteSnapshotBlocks(batch, chunks); err != nil {
			return err
		}

		plugin.GetStore().RollbackSnapshot(batch)

	}

	return nil
}

func (p *Plugins) DeleteSnapshotBlocks(chunks []*ledger.SnapshotChunk) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	allUnconfirmedBlocks := p.chain.GetAllUnconfirmedBlocks()

	for _, plugin := range p.plugins {
		rollbackBatch := plugin.GetStore().NewBatch()

		if err := plugin.RemoveNewUnconfirmed(rollbackBatch, allUnconfirmedBlocks); err != nil {
			return err
		}
		plugin.GetStore().RollbackSnapshot(rollbackBatch)

		for _, unconfirmedBlock := range allUnconfirmedBlocks {
			batch := plugin.GetStore().NewBatch()
			if err := plugin.InsertAccountBlock(batch, unconfirmedBlock); err != nil {
				return err
			}
			plugin.GetStore().WriteAccountBlock(batch, unconfirmedBlock)
		}
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
