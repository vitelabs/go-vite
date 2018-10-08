package sender

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/chain_db"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm_context"
)

type InsertProcessorFunc func(batch *leveldb.Batch, blocks []*vm_context.VmAccountBlock) error
type InsertProcessorFuncSuccess func(blocks []*vm_context.VmAccountBlock)
type DeleteProcessorFunc func(batch *leveldb.Batch, subLedger map[types.Address][]*ledger.AccountBlock) error
type DeleteProcessorFuncSuccess func(subLedger map[types.Address][]*ledger.AccountBlock)

type Chain interface {
	ChainDb() *chain_db.ChainDb

	UnRegister(listenerId uint64)
	RegisterInsertAccountBlocks(processor InsertProcessorFunc) uint64
	RegisterInsertAccountBlocksSuccess(processor InsertProcessorFuncSuccess) uint64
	RegisterDeleteAccountBlocks(processor DeleteProcessorFunc) uint64
	RegisterDeleteAccountBlocksSuccess(processor DeleteProcessorFuncSuccess) uint64
}
