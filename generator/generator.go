package generator

import (
	"errors"
	"fmt"
	"github.com/vitelabs/go-vite/common/fork"

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

const DefaultHeightDifference uint64 = 10

type SignFunc func(addr types.Address, data []byte) (signedData, pubkey []byte, err error)

type Generator struct {
	vmContext vmctxt_interface.VmDatabase
	vm        vm.VM
	sbHeight  uint64

	log log15.Logger
}

type GenResult struct {
	BlockGenList []*vm_context.VmAccountBlock
	IsRetry      bool
	Err          error
}

func NewGenerator(chain vm_context.Chain, snapshotBlockHash, prevBlockHash *types.Hash, addr *types.Address) (*Generator, error) {
	gen := &Generator{
		log:      log15.New("module", "Generator"),
		sbHeight: 2,
	}

	gen.vm = *vm.NewVM()

	vmContext, err := vm_context.NewVmContext(chain, snapshotBlockHash, prevBlockHash, addr)
	if err != nil {
		return nil, err
	}
	gen.vmContext = vmContext

	if sb := gen.vmContext.CurrentSnapshotBlock(); sb != nil {
		gen.sbHeight = sb.Height
	} else {
		return nil, errors.New("failed to new generator, cause current snapshotblock is nil")
	}
	return gen, nil
}

func (gen *Generator) GenerateWithMessage(message *IncomingMessage, signFunc SignFunc) (*GenResult, error) {
	var genResult *GenResult
	var errGenMsg error

	switch message.BlockType {
	case ledger.BlockTypeReceiveError:
		return nil, errors.New("block type error")
	case ledger.BlockTypeReceive:
		if message.FromBlockHash == nil {
			return nil, errors.New("fromblockhash can't be nil when create receive block")
		}
		sendBlock := gen.vmContext.GetAccountBlockByHash(message.FromBlockHash)
		if sendBlock == nil {
			return nil, ErrGetVmContextValueFailed
		}
		//genResult, errGenMsg = gen.GenerateWithOnroad(*sendBlock, nil, signFunc, message.Difficulty)
		genResult, errGenMsg = gen.GenerateWithOnroad(*sendBlock, nil, signFunc, message.Difficulty)
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

func (gen *Generator) GenerateWithOnroad(sendBlock ledger.AccountBlock, consensusMsg *ConsensusMessage, signFunc SignFunc, difficulty *big.Int) (*GenResult, error) {
	var producer types.Address
	if consensusMsg == nil {
		producer = sendBlock.ToAddress
	} else {
		producer = consensusMsg.Producer
	}

	block, err := gen.packBlockWithSendBlock(&sendBlock, consensusMsg, difficulty)
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
	if block.IsReceiveBlock() {
		if sendBlock = gen.vmContext.GetAccountBlockByHash(&block.FromBlockHash); sendBlock == nil {
			return nil, ErrGetVmContextValueFailed
		}
	}
	genResult, err := gen.generateBlock(block, sendBlock, block.AccountAddress, signFunc)
	if err != nil {
		return nil, err
	}
	return genResult, nil
}

func (gen *Generator) generateBlock(block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, producer types.Address, signFunc SignFunc) (result *GenResult, resultErr error) {
	gen.log.Info("generateBlock", "BlockType", block.BlockType)
	defer func() {
		if err := recover(); err != nil {
			errDetail := fmt.Sprintf("block(addr:%v prevHash:%v sbHash:%v )", block.AccountAddress, block.PrevHash, block.SnapshotHash)
			if sendBlock != nil {
				errDetail += fmt.Sprintf("sendBlock(addr:%v hash:%v)", block.AccountAddress, block.Hash)
			}

			gen.log.Error(fmt.Sprintf("generator_vm panic error %v", err), "detail", errDetail)

			result = &GenResult{}
			resultErr = errors.New("generator_vm panic error")
		}
	}()

	blockList, isRetry, err := gen.vm.Run(gen.vmContext, block, sendBlock)
	if len(blockList) > 0 {
		for k, v := range blockList {
			if k == 0 {
				v.AccountBlock.Hash = v.AccountBlock.ComputeHash()
				if signFunc != nil {
					signature, publicKey, e := signFunc(producer, v.AccountBlock.Hash.Bytes())
					if e != nil {
						return nil, e
					}
					v.AccountBlock.Signature = signature
					v.AccountBlock.PublicKey = publicKey
				}
			} else {
				v.AccountBlock.PrevHash = blockList[k-1].AccountBlock.Hash
				v.AccountBlock.Hash = v.AccountBlock.ComputeHash()
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
		return nil, errors.New("account address doesn't exist")
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

	if message.Difficulty != nil {
		// currently, default mode of GenerateWithOnroad is to calc pow
		nonce, err := pow.GetPowNonce(message.Difficulty, types.DataHash(append(blockPacked.AccountAddress.Bytes(), blockPacked.PrevHash.Bytes()...)))
		if err != nil {
			return nil, err
		}
		blockPacked.Nonce = nonce[:]
		blockPacked.Difficulty = message.Difficulty
	}

	latestSnapshotBlock := gen.vmContext.CurrentSnapshotBlock()
	blockPacked.SnapshotHash = latestSnapshotBlock.Hash
	st := time.Now()
	blockPacked.Timestamp = &st

	return blockPacked, nil
}

func (gen *Generator) packBlockWithSendBlock(sendBlock *ledger.AccountBlock, consensusMsg *ConsensusMessage, difficulty *big.Int) (blockPacked *ledger.AccountBlock, err error) {
	gen.log.Info("PackReceiveBlock", "sendBlock.Hash", sendBlock.Hash, "sendBlock.To", sendBlock.ToAddress)

	blockPacked = &ledger.AccountBlock{BlockType: ledger.BlockTypeReceive}

	gen.getDatasFromSendBlock(blockPacked, sendBlock)

	preBlockReferredSbHeight := uint64(0)
	preBlock := gen.vmContext.PrevAccountBlock()
	if preBlock == nil {
		blockPacked.Height = 1
		blockPacked.PrevHash = types.ZERO_HASH
	} else {
		blockPacked.PrevHash = preBlock.Hash
		blockPacked.Height = preBlock.Height + 1
		if sb := gen.vmContext.GetSnapshotBlockByHash(&preBlock.SnapshotHash); sb == nil {
			return nil, ErrGetVmContextValueFailed
		} else {
			preBlockReferredSbHeight = sb.Height
		}
	}

	if consensusMsg == nil {
		st := time.Now()
		blockPacked.Timestamp = &st

		snapshotBlock := gen.vmContext.CurrentSnapshotBlock()
		if snapshotBlock == nil {
			return nil, errors.New("current snapshotblock can't be nil")
		}
		if snapshotBlock.Height > preBlockReferredSbHeight && difficulty != nil {
			// currently, default mode of GenerateWithOnroad is to calc pow
			//difficulty = pow.defaultDifficulty
			nonce, err := pow.GetPowNonce(difficulty, types.DataHash(append(blockPacked.AccountAddress.Bytes(), blockPacked.PrevHash.Bytes()...)))
			if err != nil {
				return nil, err
			}
			blockPacked.Nonce = nonce[:]
			blockPacked.Difficulty = difficulty
		}
		blockPacked.SnapshotHash = snapshotBlock.Hash
	} else {
		blockPacked.Timestamp = &consensusMsg.Timestamp
		blockPacked.SnapshotHash = consensusMsg.SnapshotHash
	}
	return blockPacked, nil
}

// includes fork logic
func (gen *Generator) getDatasFromSendBlock(blockPacked, sendBlock *ledger.AccountBlock) {
	blockPacked.AccountAddress = sendBlock.ToAddress
	blockPacked.FromBlockHash = sendBlock.Hash
	if fork.IsSmartFork(gen.sbHeight) {
		blockPacked.Amount = big.NewInt(0)
		blockPacked.Fee = big.NewInt(0)
		blockPacked.TokenId = types.ZERO_TOKENID
		return
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
}
