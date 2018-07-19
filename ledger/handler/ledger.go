package handler


type Ledger struct {
	vite Vite
	Ac *AccountChain
	Sc *SnapshotChain
}

func NewLedger (vite handler.Vite) (*Ledger)  {
	return &Ledger{
		vite: vite,

		Ac: NewAccountChain(vite),
		Sc: NewSnapshotChain(vite),
	}
}