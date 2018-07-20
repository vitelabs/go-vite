package handler

import "github.com/vitelabs/go-vite/ledger/handler_interface"

type Manager struct {
	vite Vite
	ac *AccountChain
	sc *SnapshotChain

}

func NewManager(vite Vite) (*Manager)  {
	return &Manager{
		vite: vite,

		ac: NewAccountChain(vite),
		sc: NewSnapshotChain(vite),
	}
}

func (m *Manager) Ac () handler_interface.AccountChain{
	return m.ac
}
func (m *Manager) Sc () handler_interface.SnapshotChain{
	return m.sc
}