package vmctxt_interface

import (
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/trie"
)

type UnsavedCache interface {
	Copy() UnsavedCache
	Trie() *trie.Trie
	SetStorage(key []byte, value []byte)
	GetStorage(key []byte) []byte
	ContractGidList() []ContractGid
	LogList() ledger.VmLogList
	Storage() map[string][]byte
}
