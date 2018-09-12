package generator

import (
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/vm"
	"github.com/vitelabs/go-vite/vm_context"
)

const (
	SourceTypeP2P         = iota
	SourceTypeInitiate
	SourceTypeUnconfirmed
)

type Generator struct {
	VmContext vm_context.VmContext
	Vm        vm.VM
	log       log15.Logger
}

func NewGenerator(chain *chain.Chain, snapshotBlockHash *types.Hash, prevAccountBlockHash *types.Hash, addr *types.Address) (*Generator, error) {
	vmContext, err := vm_context.NewVmContext(chain, snapshotBlockHash, prevAccountBlockHash, addr)
	if err != nil {
		return nil, err
	}
	vm := vm.NewVM(vmContext)
	return &Generator{
		VmContext: *vmContext,
		Vm:        *vm,
		log:       log15.New("module", "ContractTask"),
	}, nil
}

func (gen *Generator) GenerateTx(sourceType int32, block *ledger.AccountBlock) *GenResult {

	var blockList []*ledger.AccountBlock
	var isRetry bool
	var err error
	var blockGenList []*BlockGen

	select {
	case sourceType == SourceTypeP2P:
		blockList, isRetry, err = gen.generateP2PTx(block)
	case sourceType == SourceTypeInitiate:
		blockList, isRetry, err = gen.generateInitiateTx(block)
	case sourceType == SourceTypeUnconfirmed:
		blockList, isRetry, err = gen.generateUnconfirmedTx(block)
	}

	for k, v := range blockList {
		hash := block.GetComputeHash()
		block.Hash = hash
		blockGen := &BlockGen{
			Block:     v,
			VmContext: nil,
		}
		if k == 0 {
			blockGen.VmContext = &gen.VmContext
		}
		blockGenList = append(blockGenList, blockGen)
	}
	return &GenResult{
		BlockGenList: blockGenList,
		IsRetry:      isRetry,
		Err:          err,
	}
}

func (gen *Generator) generateP2PTx(block *ledger.AccountBlock) (blockList []*ledger.AccountBlock, isRetry bool, err error) {
	gen.log.Info("generateP2PTx", "BlockType", block.BlockType)
	if block.BlockType != byte(0) {
		return gen.Vm.Run(block, nil)

	} else {
		sendBlock := gen.Vm.Db.GetAccountBlockByHash(&block.FromBlockHash)
		return gen.Vm.Run(block, sendBlock)
	}
}

func (gen *Generator) generateInitiateTx(block *ledger.AccountBlock) (blockList []*ledger.AccountBlock, isRetry bool, err error) {
	gen.log.Info("generateInitiateTx", "BlockType", block.BlockType)
	if block.BlockType != byte(0) {
		return gen.Vm.Run(block, nil)
	} else {
		sendBlock := gen.Vm.Db.GetAccountBlockByHash(&block.FromBlockHash)
		return gen.Vm.Run(block, sendBlock)
	}
}

// only handle receiveBlock
func (gen *Generator) generateUnconfirmedTx(block *ledger.AccountBlock) (blockList []*ledger.AccountBlock, isRetry bool, err error) {
	gen.log.Info("generateUnconfirmedTx", "BlockType", block.BlockType)
	sendBlock := gen.Vm.Db.GetAccountBlockByHash(&block.FromBlockHash)
	return gen.Vm.Run(block, sendBlock)
}

type GenResult struct {
	BlockGenList []*BlockGen
	IsRetry      bool
	Err          error
}

type BlockGen struct {
	Block     *ledger.AccountBlock
	VmContext *vm_context.VmContext
}
