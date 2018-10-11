package generator

import (
	"errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/pow"
	"github.com/vitelabs/go-vite/vm"
	"github.com/vitelabs/go-vite/vm_context"
	"github.com/vitelabs/go-vite/vm_context/vmctxt_interface"
	"math/big"
	"time"
)

type SignFunc func(addr types.Address, data []byte) (signedData, pubkey []byte, err error)

type Generator struct {
	chain Chain

	vm        vm.VM
	vmContext vmctxt_interface.VmDatabase

	log log15.Logger
}

type GenResult struct {
	BlockGenList []*vm_context.VmAccountBlock
	IsRetry      bool
	Err          error
}

func NewGenerator(chain Chain, snapshotBlockHash, prevBlockHash *types.Hash, addr *types.Address) (*Generator, error) {
	gen := &Generator{
		chain: chain,
		log:   log15.New("module", "Generator"),
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
	var genResult *GenResult
	var errGenMsg error

	if message.BlockType != ledger.BlockTypeSendCall && message.BlockType != ledger.BlockTypeSendCreate {
		sendBlock := gen.vmContext.GetAccountBlockByHash(message.FromBlockHash)
		genResult, errGenMsg = gen.GenerateWithOnroad(*sendBlock, nil, signFunc)
	} else {
		block, err := gen.packBlockWithMessage(message)
		if err != nil {
			return nil, err
		}
		genResult, errGenMsg = gen.generateBlock(block, nil, signFunc)
	}

	if errGenMsg != nil {
		return nil, errGenMsg
	}
	return genResult, nil
}

func (gen *Generator) GenerateWithOnroad(sendBlock ledger.AccountBlock, consensusMsg *ConsensusMessage, signFunc SignFunc) (*GenResult, error) {
	block, err := gen.packBlockWithSendBlock(&sendBlock, consensusMsg)
	if err != nil {
		return nil, err
	}
	genResult, err := gen.generateBlock(block, &sendBlock, signFunc)
	if err != nil {
		return nil, err
	}
	return genResult, nil
}

func (gen *Generator) GenerateWithBlock(block *ledger.AccountBlock, signFunc SignFunc) (*GenResult, error) {
	var sendBlock *ledger.AccountBlock = nil
	if block.BlockType != ledger.BlockTypeSendCall && block.BlockType != ledger.BlockTypeSendCreate {
		sendBlock = gen.vmContext.GetAccountBlockByHash(&block.FromBlockHash)
	}
	genResult, err := gen.generateBlock(block, sendBlock, signFunc)
	if err != nil {
		return nil, err
	}
	return genResult, nil
}

func (gen *Generator) generateBlock(block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, signFunc SignFunc) (*GenResult, error) {
	gen.log.Info("generateBlock", "BlockType", block.BlockType)

	blockList, isRetry, err := gen.vm.Run(gen.vmContext, block, sendBlock)

	if len(blockList) > 0 {
		for k, v := range blockList {
			v.AccountBlock.Hash = v.AccountBlock.ComputeHash()

			if k == 0 {
				accountBlock := blockList[0].AccountBlock
				if signFunc != nil {
					signature, publicKey, e := signFunc(accountBlock.AccountAddress, accountBlock.Hash.Bytes())
					if e != nil {
						return nil, e
					}
					accountBlock.Signature = signature
					accountBlock.PublicKey = publicKey
				}
			} else {
				v.AccountBlock.PrevHash = blockList[k-1].AccountBlock.Hash
			}
		}
	}

	return &GenResult{
		BlockGenList: blockList,
		IsRetry:      isRetry,
		Err:          err,
	}, nil
}

func (gen *Generator) packBlockWithMessage(message *IncomingMessage) (blockPacked *ledger.AccountBlock, err error) {
	blockPacked, err = message.ToBlock()
	if err != nil {
		return nil, err
	}

	latestBlock := gen.vmContext.PrevAccountBlock()
	if latestBlock == nil {
		blockPacked.Height = 1
		blockPacked.PrevHash = types.ZERO_HASH
	} else {
		blockPacked.Height = latestBlock.Height + 1
		blockPacked.PrevHash = latestBlock.Hash
	}

	latestSnapshotBlock := gen.vmContext.CurrentSnapshotBlock()
	blockPacked.SnapshotHash = latestSnapshotBlock.Hash

	st := time.Now()
	blockPacked.Timestamp = &st

	return blockPacked, nil
}

func (gen *Generator) packBlockWithSendBlock(sendBlock *ledger.AccountBlock, consensusMsg *ConsensusMessage) (blockPacked *ledger.AccountBlock, err error) {
	gen.log.Info("PackReceiveBlock", "sendBlock.Hash", sendBlock.Hash, "sendBlock.To", sendBlock.ToAddress)
	blockPacked = &ledger.AccountBlock{
		BlockType:      ledger.BlockTypeReceive,
		AccountAddress: sendBlock.ToAddress,
		FromBlockHash:  sendBlock.Hash,
	}

	if sendBlock.Amount == nil {
		blockPacked.Amount = big.NewInt(0)
	} else {
		blockPacked.Amount = sendBlock.Amount
	}
	blockPacked.TokenId = sendBlock.TokenId

	if sendBlock.Fee == nil {
		blockPacked.Fee = big.NewInt(0)
	} else {
		blockPacked.Fee = sendBlock.Fee
	}

	preBlock := gen.vmContext.PrevAccountBlock()
	if preBlock == nil {
		blockPacked.Height = 1
		blockPacked.PrevHash = types.Hash{}
	} else {
		blockPacked.PrevHash = preBlock.Hash
		blockPacked.Height = preBlock.Height + 1
	}

	if consensusMsg != nil {
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
		nonce := pow.GetPowNonce(nil, types.DataHash(append(blockPacked.AccountAddress.Bytes(), blockPacked.PrevHash.Bytes()...)))
		blockPacked.Nonce = nonce[:]
	}
	return blockPacked, nil
}
