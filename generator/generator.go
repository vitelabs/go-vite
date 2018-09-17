package generator

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/vm"
	"github.com/vitelabs/go-vite/vm_context"
)

const (
	SourceTypeP2P = iota
	SourceTypeUserInitiate
	SourceTypeUnconfirmed
)

type Generator struct {
	Vm vm.VM

	chain  vm_context.Chain
	signer SignManager

	log log15.Logger
}

func (gen *Generator) GenerateWithBlock(sourceType byte, block *ledger.AccountBlock) *GenResult {
	select {
	case sourceType == SourceTypeP2P:
		return gen.generateP2PTx(block)
	case sourceType == SourceTypeUnconfirmed:

		return gen.generateUnconfirmedTx(block)
	}
	return nil
}

func (gen *Generator) GenerateWithMessage(message *IncomingMessage, passphrase string) *GenResult {
	block := gen.PackBlockWithMessage(message)
	return gen.GenerateBlockWithPassphrase(SourceTypeUserInitiate, block, passphrase)
}

func (gen *Generator) GenerateBlockWithPassphrase(sourceType byte, block *ledger.AccountBlock, passphrase string) *GenResult {
	select {
	case sourceType == SourceTypeUserInitiate:
		return gen.generateInitiateTx(block, passphrase)
	}
	return nil
}

func (gen *Generator) generateP2PTx(block *ledger.AccountBlock) *GenResult {
	gen.log.Info("generateP2PTx", "BlockType", block.BlockType)
	// todo  run the the complete set of verify

	var blockList []*ledger.AccountBlock
	var isRetry bool
	var err error

	if block.BlockType != ledger.BlockTypeSendCall && block.BlockType != ledger.BlockTypeSendCreate {
		sendBlock := gen.Vm.Db.GetAccountBlockByHash(&block.FromBlockHash)
		blockList, isRetry, err = gen.Vm.Run(block, sendBlock)
	} else {
		blockList, isRetry, err = gen.Vm.Run(block, nil)
	}

	blockList[0].Hash = blockList[0].GetComputeHash()

	var blockGenList []*vm_context.VmAccountBlock
	blockGen := &vm_context.VmAccountBlock{
		AccountBlock: blockList[0],
		VmContext:    nil,
	}
	blockGenList = append(blockGenList, blockGen)

	return &GenResult{
		BlockGenList: blockGenList,
		IsRetry:      isRetry,
		Err:          err,
	}
}

// generateInitiateTx: currently only support to accept commonTx, and passphrase is in necessary
func (gen *Generator) generateInitiateTx(block *ledger.AccountBlock, passphrase string) *GenResult {
	gen.log.Info("generateInitiateTx", "BlockType", block.BlockType)

	var blockList []*ledger.AccountBlock
	var isRetry bool
	var err error

	if block.BlockType != ledger.BlockTypeSendCall && block.BlockType != ledger.BlockTypeSendCreate {
		sendBlock := gen.Vm.Db.GetAccountBlockByHash(&block.FromBlockHash)
		blockList, isRetry, err = gen.Vm.Run(block, sendBlock)
	} else {
		blockList, isRetry, err = gen.Vm.Run(block, nil)
	}

	blockList[0].Hash = blockList[0].GetComputeHash()

	var signErr error
	if blockList[0].Signature, blockList[0].PublicKey, signErr =
		gen.signer.SignDataWithPassphrase(blockList[0].AccountAddress, passphrase,
			blockList[0].Hash.Bytes()); signErr != nil {
		gen.log.Error("SignData Error", signErr)
		return nil
	}

	var blockGenList []*vm_context.VmAccountBlock
	blockGen := &vm_context.VmAccountBlock{
		AccountBlock: blockList[0],
		VmContext:    nil,
	}
	blockGenList = append(blockGenList, blockGen)

	return &GenResult{
		BlockGenList: blockGenList,
		IsRetry:      isRetry,
		Err:          err,
	}
}

// generateUnconfirmedTx: only handle receiveBlock
func (gen *Generator) generateUnconfirmedTx(block *ledger.AccountBlock) *GenResult {
	gen.log.Info("generateUnconfirmedTx", "BlockType", block.BlockType)

	sendBlock := gen.Vm.Db.GetAccountBlockByHash(&block.FromBlockHash)
	blockList, isRetry, err := gen.Vm.Run(block, sendBlock)

	var blockGenList []*vm_context.VmAccountBlock
	for k, v := range blockList {
		v.Hash = v.GetComputeHash()
		blockGen := &vm_context.VmAccountBlock{
			AccountBlock: v,
			VmContext:    nil,
		}

		if k == 0 {
			blockGen.VmContext = gen.Vm.Db

			var signErr error
			if blockGen.AccountBlock.Signature, blockGen.AccountBlock.PublicKey, signErr =
				gen.signer.SignData(blockGen.AccountBlock.AccountAddress, blockGen.AccountBlock.Hash.Bytes()); signErr != nil {
				gen.log.Error("SignData Error", signErr)
				return nil
			}

		}
		blockGenList = append(blockGenList, blockGen)
	}

	return &GenResult{
		BlockGenList: blockGenList,
		IsRetry:      isRetry,
		Err:          err,
	}
}

func (gen *Generator) PackBlockWithMessage(message *IncomingMessage) *ledger.AccountBlock {
	block := &ledger.AccountBlock{
		BlockType:      message.BlockType,
		AccountAddress: message.AccountAddress,
		Amount:         message.Amount,
		TokenId:        message.TokenId,
		Data:           message.Data,
		PublicKey:      nil, // at the time to sign after vm generate
	}
	if block.BlockType != ledger.BlockTypeSendCall && block.BlockType != ledger.BlockTypeSendCreate {
		block.ToAddress = types.Address{}
		block.FromBlockHash = *message.FromBlockHash
	} else {
		block.FromBlockHash = types.Hash{}
		block.ToAddress = *message.ToAddress
	}

	latestBlock := gen.Vm.Db.PrevAccountBlock()
	block.Height = latestBlock.Height + 1
	block.PrevHash = latestBlock.Hash

	latestSnapshotBlock := gen.Vm.Db.CurrentSnapshotBlock()
	block.SnapshotHash = latestSnapshotBlock.Hash

	return block
}

func (gen *Generator) PackBlockWithSendBlock(sendBlock *ledger.AccountBlock, snapshot *types.Hash) *ledger.AccountBlock {
	return nil
}

type GenResult struct {
	BlockGenList []*vm_context.VmAccountBlock
	IsRetry      bool
	Err          error
}
