package ledger

import (
	"github.com/vitelabs/go-vite/ledger/handler"
)



type Ledger struct {
	vite handler.Vite
	Ac *handler.AccountChain
	Sc *handler.SnapshotChain
}

func NewLedger (vite handler.Vite) (*Ledger)  {
	return &Ledger{
		vite: vite,

		Ac: handler.NewAccountChain(vite),
		Sc: handler.NewSnapshotChain(vite),
	}
}