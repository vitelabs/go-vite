package chain_plugins

import (
	"errors"
	"fmt"
	"github.com/vitelabs/go-vite/chain/db"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/vm_db"
	"os"
	"path"
)

const initRoundSize = uint64(10)

type Plugins struct {
	id      types.Hash
	dataDir string

	log     log15.Logger
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

	dataDir := path.Join(chainDir, "plugins")

	store, err := chain_db.NewStore(dataDir, id)
	if err != nil {
		return nil, err
	}

	plugins := map[string]Plugin{
		"filterToken": newFilterToken(store, chain),
		"onRoadInfo":  newOnRoadInfo(store, chain),
	}

	return &Plugins{
		id:      id,
		dataDir: dataDir,
		chain:   chain,
		store:   store,
		plugins: plugins,
		log:     log15.New("module", "chain_plugins"),
	}, nil
}

func (p *Plugins) RebuildData() error {
	oLog.Info("Start rebuild plugin data")
	if err := p.store.Close(); err != nil {
		return err
	}
	flusher := p.chain.Flusher()

	// remove data
	os.RemoveAll(p.dataDir)

	// set new store
	store, err := chain_db.NewStore(p.dataDir, p.id)
	if err != nil {
		return err
	}

	for _, plugin := range p.plugins {
		plugin.SetStore(store)
	}

	// get latest snapshot block
	latestSnapshot := p.chain.GetLatestSnapshotBlock()
	if latestSnapshot == nil {
		return errors.New("GetLatestSnapshotBlock fail")
	}

	p.log.Info(fmt.Sprintf("latestSnapshot[%v %v]", latestSnapshot.Hash, latestSnapshot.Height), "method", "RebuildData")

	// build data
	round := latestSnapshot.Height / initRoundSize
	leftCount := latestSnapshot.Height % initRoundSize

	for i := uint64(0); i < round+1; i++ {
		start := i * initRoundSize
		end := i * initRoundSize
		if i < round {
			end = end + initRoundSize
		} else {
			if leftCount <= 0 {
				break
			}
			end = end + leftCount
		}
		if i == 0 {
			start += 1
		}
		chunks, err := p.chain.GetSubLedger(start, end)
		if err != nil {
			return err
		}

		for idx, chunk := range chunks {
			if idx == 0 {
				continue
			}
			// write ab
			for _, ab := range chunk.AccountBlocks {

				batch := p.store.NewBatch()

				for _, plugin := range p.plugins {
					if err := plugin.InsertAccountBlock(batch, ab); err != nil {
						return err
					}
				}
				p.store.WriteAccountBlock(batch, ab)
			}

			// write sb
			batch := p.store.NewBatch()

			for _, plugin := range p.plugins {
				if err := plugin.InsertSnapshotBlock(batch, chunk.SnapshotBlock, chunk.AccountBlocks); err != nil {
					pErr := errors.New(fmt.Sprintf("InsertSnapshotBlock fail, err:%v, sb[%v, %v,len=%v] ", err, chunk.SnapshotBlock.Height, chunk.SnapshotBlock.Hash, len(chunk.AccountBlocks)))
					p.log.Error(pErr.Error(), "method", "RebuildData")
					return pErr
				}
			}

			p.store.WriteSnapshot(batch, chunk.AccountBlocks)

		}
		// flush to disk
		flusher.Flush()
	}

	// success
	p.log.Info("Succeed rebuild plugin data")
	return nil
}

func (p *Plugins) Close() error {
	if err := p.store.Close(); err != nil {
		return err
	}
	return nil
}

func (p *Plugins) Store() *chain_db.Store {
	return p.store
}

func (p *Plugins) GetPlugin(name string) Plugin {
	return p.plugins[name]
}

func (p *Plugins) RemovePlugin(name string) {
	delete(p.plugins, name)
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
