package handler


type Manager struct {
	vite Vite
	Ac *AccountChain
	Sc *SnapshotChain

}

func NewManager(vite Vite) (*Manager)  {
	return &Manager{
		vite: vite,

		Ac: NewAccountChain(vite),
		Sc: NewSnapshotChain(vite),
	}
}