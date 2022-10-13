package vm_db

import (
	"github.com/vitelabs/go-vite/v2/common/types"
	"github.com/vitelabs/go-vite/v2/interfaces/core"
)

func (vdb *vmDb) GetExecutionContext(blockHash *types.Hash) (*core.ExecutionContext, error) {
	if vdb.executionContext[blockHash] != nil {
		return vdb.executionContext[blockHash], nil
	}

	context, err := vdb.chain.GetExecutionContext(blockHash)
	if err != nil {
		return nil, err
	}
	vdb.executionContext[blockHash] = context

	return context, nil
}

func (vdb *vmDb) SetExecutionContext(blockHash *types.Hash, context *core.ExecutionContext) {
	vdb.executionContext[blockHash] = context
}

