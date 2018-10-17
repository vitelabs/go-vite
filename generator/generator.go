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
	vm        vm.VM
	vmContext vmctxt_interface.VmDatabase

	log log15.Logger
}

type GenResult struct {
	BlockGenList []*vm_context.VmAccountBlock
	IsRetry      bool
	Err          error
}

func NewGenerator(chain vm_context.Chain, snapshotBlockHash, prevBlockHash *types.Hash, addr *types.Address) (*Generator, error) {
	gen := &Generator{
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
	var genResult *GenResult
	var errGenMsg error

	switch message.BlockType {
	case ledger.BlockTypeReceiveError:
		return nil, errors.New("block type can't be BlockTypeReceiveError")
	case ledger.BlockTypeReceive:
		if message.FromBlockHash == nil {
			return nil, errors.New("FromBlockHash can't be nil when create ReceiveBlock")
		}
		sendBlock := gen.vmContext.GetAccountBlockByHash(message.FromBlockHash)
		genResult, errGenMsg = gen.GenerateWithOnroad(*sendBlock, nil, signFunc)
	default:
		block, err := gen.packSendBlockWithMessage(message)
		if err != nil {
			return nil, err
		}
		genResult, errGenMsg = gen.generateBlock(block, nil, block.AccountAddress, signFunc)
	}
	if errGenMsg != nil {
		return nil, errGenMsg
	}
	return genResult, nil
}

func (gen *Generator) GenerateWithOnroad(sendBlock ledger.AccountBlock, consensusMsg *ConsensusMessage, signFunc SignFunc) (*GenResult, error) {
	var producer types.Address
	if consensusMsg == nil {
		producer = sendBlock.ToAddress
	} else {
		producer = consensusMsg.Producer
	}
	block, err := gen.packBlockWithSendBlock(&sendBlock, consensusMsg)
	if err != nil {
		return nil, err
	}
	genResult, err := gen.generateBlock(block, &sendBlock, producer, signFunc)
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
	genResult, err := gen.generateBlock(block, sendBlock, block.AccountAddress, signFunc)
	if err != nil {
		return nil, err
	}
	return genResult, nil
}

func (gen *Generator) generateBlock(block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, producer types.Address, signFunc SignFunc) (*GenResult, error) {
	gen.log.Info("generateBlock", "BlockType", block.BlockType)

	blockList, isRetry, err := gen.vm.Run(gen.vmContext, block, sendBlock)

	if len(blockList) > 0 {
		for k, v := range blockList {
			v.AccountBlock.Hash = v.AccountBlock.ComputeHash()

			if k == 0 {
				accountBlock := blockList[0].AccountBlock
				if signFunc != nil {
					signature, publicKey, e := signFunc(producer, accountBlock.Hash.Bytes())
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

func (gen *Generator) packSendBlockWithMessage(message *IncomingMessage) (blockPacked *ledger.AccountBlock, err error) {
	latestBlock := gen.vmContext.PrevAccountBlock()
	if latestBlock == nil {
		return nil, errors.New("SendTx's AccountAddress doesn't exist")
	}

	blockPacked, err = message.ToSendBlock()
	if err != nil {
		return nil, err
	}

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

	preBlockReferredSbHeight := uint64(0)
	preBlock := gen.vmContext.PrevAccountBlock()
	if preBlock == nil {
		blockPacked.Height = 1
		blockPacked.PrevHash = types.ZERO_HASH
	} else {
		blockPacked.PrevHash = preBlock.Hash
		blockPacked.Height = preBlock.Height + 1
		if sb := gen.vmContext.GetSnapshotBlockByHash(&preBlock.SnapshotHash); sb != nil {
			preBlockReferredSbHeight = sb.Height
		}
	}

	if consensusMsg == nil {
		st := time.Now()
		blockPacked.Timestamp = &st

		snapshotBlock := gen.vmContext.CurrentSnapshotBlock()
		if snapshotBlock == nil {
			return nil, errors.New("CurrentSnapshotBlock can't be nil")
		}
		if snapshotBlock.Height > preBlockReferredSbHeight {
			nonce := pow.GetPowNonce(nil, types.DataHash(append(blockPacked.AccountAddress.Bytes(), blockPacked.PrevHash.Bytes()...)))
			blockPacked.Nonce = nonce[:]
		}
		blockPacked.SnapshotHash = snapshotBlock.Hash
	} else {
		blockPacked.Timestamp = &consensusMsg.Timestamp
		blockPacked.SnapshotHash = consensusMsg.SnapshotHash
	}
	return blockPacked, nil
}
