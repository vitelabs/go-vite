package handler


type Ledger struct {
	vite Vite
	Ac *AccountChain
	Sc *SnapshotChain
}

func NewLedger (vite Vite) (*Ledger)  {
	return &Ledger{
		vite: vite,

		Ac: NewAccountChain(vite),
		Sc: NewSnapshotChain(vite),
	}
}