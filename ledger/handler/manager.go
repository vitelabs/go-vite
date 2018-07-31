package handler

import (
	"github.com/vitelabs/go-vite/ledger/handler_interface"
	"github.com/vitelabs/go-vite/vitedb"
)

type Manager struct {
	vite Vite
	ac   *AccountChain
	sc   *SnapshotChain
}

func NewManager(vite Vite, dataDir string) *Manager {
	// Fixme
	vitedb.SetDataDir(dataDir)

	manager := &Manager{
		vite: vite,

		ac: NewAccountChain(vite),
		sc: NewSnapshotChain(vite),
	}

	// Check if the genesis blocks exists and if it doesn't, create the genesis blocks
	manager.sc.scAccess.CheckAndCreateGenesisBlocks()

	return manager
}

func (m *Manager) Ac() handler_interface.AccountChain {
	return m.ac
}

func (m *Manager) Sc() handler_interface.SnapshotChain {
	return m.sc
}

func (m *Manager) RegisterFirstSyncDown(firstSyncDownChan chan<- int) {
	m.sc.registerFirstSyncDown(firstSyncDownChan)
}
