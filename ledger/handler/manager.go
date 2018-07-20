package handler

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

func (m *Manager) Ac () *AccountChain{
	return m.ac
}
func (m *Manager) Sc () *SnapshotChain{
	return m.sc
}