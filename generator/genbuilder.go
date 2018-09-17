package generator

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/vm"
	"github.com/vitelabs/go-vite/vm_context"
)

type Builder interface {
	SetDependentModule(chain vm_context.Chain, wManager SignManager) Builder
	PrepareVm(snapshotBlockHash *types.Hash, prevAccountBlockHash *types.Hash, addr *types.Address) (Builder, error)
	Build() Generator
}

type GenBuilder struct {
	generator Generator
}

func NewGenBuilder() *GenBuilder {
	return &GenBuilder{}
}

func (builder *GenBuilder) SetDependentModule(chain vm_context.Chain, wManager SignManager) Builder {
	builder.generator.chain = chain
	builder.generator.signer = wManager
	return builder
}

func (builder *GenBuilder) PrepareVm(snapshotBlockHash *types.Hash, prevAccountBlockHash *types.Hash, addr *types.Address) (Builder, error) {
	vmContext, err := vm_context.NewVmContext(builder.generator.chain, snapshotBlockHash, prevAccountBlockHash, addr)
	if err != nil {
		return builder, err
	}
	builder.generator.VmContext = vmContext
	builder.generator.Vm = *vm.NewVM()
	return builder, nil
}

func (builder *GenBuilder) Build() Generator {
	return builder.generator
}
