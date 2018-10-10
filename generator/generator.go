package generator

import (
	"errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/vm"
	"github.com/vitelabs/go-vite/vm_context"
	"github.com/vitelabs/go-vite/vm_context/vmctxt_interface"
	"time"
)

type SignFunc func(addr types.Address, data []byte) (signedData, pubkey []byte, err error)

type Generator struct {
	chain  Chain
	signer Signer

	vm        vm.VM
	vmContext vmctxt_interface.VmDatabase

	log log15.Logger
}

type GenResult struct {
	BlockGenList []*vm_context.VmAccountBlock
	IsRetry      bool
	Err          error
}

func NewGenerator(chain Chain, signer Signer, snapshotBlockHash, prevBlockHash *types.Hash, addr *types.Address) (*Generator, error) {
	gen := &Generator{
		chain:  chain,
		signer: signer,

		log: log15.New("module", "Generator"),
	}
	vmContext, err := vm_context.NewVmContext(chain, snapshotBlockHash, prevBlockHash, addr)
	if err != nil {
		return nil, err
	}

	gen.vmContext = vmContext
	gen.vm = *vm.NewVM()
	return gen, nil
}

func (gen *Generator) GenerateWithMessage(message *IncomingMessage, signFunc SignFunc) (*GenResult, error) {
	block, err := gen.PackBlockWithMessage(message)
	if err != nil {
		return nil, err
	}
	var sendBlock *ledger.AccountBlock = nil
	if block.BlockType != ledger.BlockTypeSendCall && block.BlockType != ledger.BlockTypeSendCreate {
		sendBlock = gen.vmContext.GetAccountBlockByHash(&block.FromBlockHash)
	}
	return gen.generateBlock(block, sendBlock, signFunc), nil
}

func (gen *Generator) GenerateWithOnroad(sendBlock ledger.AccountBlock, consensusMsg *ConsensusMessage, signFunc SignFunc) (*GenResult, error) {
	block, err := gen.packBlockWithSendBlock(&sendBlock, consensusMsg)
	if err != nil {
		return nil, err
	}
	return gen.generateBlock(block, &sendBlock, signFunc), nil
}

func (gen *Generator) GenerateWithBlock(block *ledger.AccountBlock, signFunc SignFunc) (*GenResult, error) {
	var sendBlock *ledger.AccountBlock = nil
	if block.BlockType != ledger.BlockTypeSendCall && block.BlockType != ledger.BlockTypeSendCreate {
		sendBlock = gen.vmContext.GetAccountBlockByHash(&block.FromBlockHash)
	}
	return gen.generateBlock(block, sendBlock, signFunc), nil
}

func (gen *Generator) generateBlock(block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, signFunc SignFunc) *GenResult {
	gen.log.Info("generateBlock", "BlockType", block.BlockType)

	blockList, isRetry, err := gen.vm.Run(gen.vmContext, block, sendBlock)
	for k, v := range blockList {
		v.AccountBlock.Hash = v.AccountBlock.ComputeHash()
		if k > 0 {
			v.AccountBlock.PrevHash = blockList[k-1].AccountBlock.Hash
		}
	}

	accountBlock := blockList[0].AccountBlock
	if signFunc != nil {
		accountBlock.Signature, accountBlock.PublicKey, err = signFunc(accountBlock.AccountAddress, accountBlock.Hash.Bytes())
		if err != nil {
			gen.log.Error("generate.Sign()", "Error", err)
		}
	}

	return &GenResult{
		BlockGenList: blockList,
		IsRetry:      isRetry,
		Err:          err,
	}
}

func (gen *Generator) PackBlockWithMessage(message *IncomingMessage) (blockPacked *ledger.AccountBlock, err error) {
	blockPacked, err = message.ToBlock()
	if err != nil {
		return nil, err
	}

	latestBlock := gen.vmContext.PrevAccountBlock()
	blockPacked.Height = latestBlock.Height + 1
	blockPacked.PrevHash = latestBlock.Hash

	latestSnapshotBlock := gen.vmContext.CurrentSnapshotBlock()
	blockPacked.SnapshotHash = latestSnapshotBlock.Hash

	st := time.Now()
	blockPacked.Timestamp = &st

	return blockPacked, nil
}

func (gen *Generator) packBlockWithSendBlock(sendBlock *ledger.AccountBlock, consensusMsg *ConsensusMessage) (blockPacked *ledger.AccountBlock, err error) {
	gen.log.Info("PackReceiveBlock", "sendBlock.Hash", sendBlock.Hash, "sendBlock.To", sendBlock.ToAddress)
	blockPacked = &ledger.AccountBlock{
		AccountAddress: sendBlock.ToAddress,
		FromBlockHash:  sendBlock.Hash,

		Amount:  sendBlock.Amount,
		TokenId: sendBlock.TokenId,
		Fee:     sendBlock.Fee,
		Nonce:   sendBlock.Nonce,
		Data:    sendBlock.Data,
	}

	preBlock := gen.vmContext.PrevAccountBlock()
	if preBlock == nil {
		blockPacked.Height = 1
		blockPacked.PrevHash = types.Hash{}
	} else {
		blockPacked.PrevHash = preBlock.Hash
		blockPacked.Height = preBlock.Height + 1
	}

	if consensusMsg == nil {
		blockPacked.Timestamp = &consensusMsg.Timestamp
		blockPacked.SnapshotHash = consensusMsg.SnapshotHash
	} else {
		st := time.Now()
		blockPacked.Timestamp = &st
		snapshotBlock := gen.vmContext.CurrentSnapshotBlock()
		if snapshotBlock == nil {
			return nil, errors.New("CurrentSnapshotBlock can't be nil")
		}
		blockPacked.SnapshotHash = snapshotBlock.Hash
	}
	return blockPacked, nil
}

func (gen *Generator) Sign(addr types.Address, passphrase *string, data []byte) (signedData, pubkey []byte, err error) {
	if passphrase == nil {
		return gen.signer.SignData(addr, data)
	} else {
		return gen.signer.SignDataWithPassphrase(addr, *passphrase, data)
	}
}
