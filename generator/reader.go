package generator

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm_db"
)

type chainReader interface {
	GetAccountBlockByHash(blockHash types.Hash) (*ledger.AccountBlock, error)
}

func NewChain(c vm_db.Chain) chainReader {
	return c
}
