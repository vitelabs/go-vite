package compress

import "os"

type Store struct {
}

func NewStore() *Store {
	return &Store{}
}

func (*Store) Write(file *os.File, ledger *subLedger) {

}

func StoreGet() {

}
