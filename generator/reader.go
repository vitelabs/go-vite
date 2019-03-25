package generator

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/util"
	"github.com/vitelabs/go-vite/vm_db"
)

type chainReader interface {
	GetAccountBlockByHash(blockHash *types.Hash) (*ledger.AccountBlock, error)
}

func NewChain(c vm_db.Chain) chainReader {
	return c
}

type vmReader interface {
	Run(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, status *util.GlobalStatus) (vmBlock *vm_db.VmAccountBlock, isRetry bool, err error)
}

func NewVM() vmReader {
	return nil
}
