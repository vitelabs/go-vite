package handler

import "github.com/vitelabs/go-vite/ledger/handler_interface"

type Manager struct {
	vite Vite
	ac *AccountChain
	sc *SnapshotChain

}

func NewManager(vite Vite) (*Manager)  {
	manager := &Manager{
		vite: vite,

		ac: NewAccountChain(vite),
		sc: NewSnapshotChain(vite),
	}

	// Check if the genesis blocks exists and if it doesn't, create the genesis blocks
	manager.sc.scAccess.CheckAndCreateGenesisBlock()
	return manager
}

func (m *Manager) Ac () handler_interface.AccountChain{
	return m.ac
}
func (m *Manager) Sc () handler_interface.SnapshotChain{
	return m.sc
}