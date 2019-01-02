package index

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm_context"
)

type FilterTokenIndex struct {
}

func NewFilterTokenIndex() {
	// register
}

func (fti *FilterTokenIndex) build() {
	// scan
}

func (fti *FilterTokenIndex) addBlocks([]*vm_context.VmAccountBlock) {
	// add
}

func (fti *FilterTokenIndex) deleteBlocks([]*vm_context.VmAccountBlock) {
	// delete
}

func (fti *FilterTokenIndex) GetBlockHashList(account *ledger.Account, originBlockHash *types.Hash, count uint64) []types.Hash {
	return nil
}
