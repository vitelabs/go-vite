package interfaces

import ledgerHandler "github.com/vitelabs/go-vite/ledger/handler_interface"

type Vite interface {
	Ledger() ledgerHandler.Manager
}
