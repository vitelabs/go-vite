package generator

import (
	"errors"
	"math/big"
	"math/rand"
	"time"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/pow"
	"github.com/vitelabs/go-vite/vm"
	"github.com/vitelabs/go-vite/vm_context"
	"github.com/vitelabs/go-vite/vm_context/vmctxt_interface"
)

const DefaultHeightDifference uint64 = 10

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
		return nil, errors.New("block type error")
	case ledger.BlockTypeReceive:
		if message.FromBlockHash == nil {
			return nil, errors.New("fromblockhash can't be nil when create receive block")
		}
		sendBlock := gen.vmContext.GetAccountBlockByHash(message.FromBlockHash)
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
		sendBlock = gen.vmContext.GetAccountBlockByHash(&block.FromBlockHash)
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
			gen.log.Error("generator_vm panic error", "error", err)
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

func GetFittestGeneratorSnapshotHash(chain vm_context.Chain, accAddr *types.Address,
	referredSnapshotHashList []types.Hash, isRandom bool) (prevSbHash *types.Hash, fittestSbHash *types.Hash, err error) {
	var fittestSbHeight uint64
	var referredMaxSbHeight uint64
	latestSb := chain.GetLatestSnapshotBlock()
	if latestSb == nil {
		return nil, nil, errors.New("get latest snapshotblock failed")
	}
	fittestSbHeight = latestSb.Height

	var prevSbFlag = false
	var prevSb *ledger.SnapshotBlock
	if accAddr != nil {
		prevAccountBlock, err := chain.GetLatestAccountBlock(accAddr)
		if err != nil {
			return nil, nil, err
		}
		if prevAccountBlock != nil {
			referredSnapshotHashList = append(referredSnapshotHashList, prevAccountBlock.SnapshotHash)
			prevSbFlag = true
		}
	}
	referredMaxSbHeight = uint64(1)
	if len(referredSnapshotHashList) > 0 {
		// get max referredSbHeight
		for k, v := range referredSnapshotHashList {
			vSb, _ := chain.GetSnapshotBlockByHash(&v)
			if vSb == nil {
				return nil, nil, ErrGetSnapshotOfReferredBlockFailed
			} else {
				if referredMaxSbHeight < vSb.Height {
					referredMaxSbHeight = vSb.Height
				}
				if k == len(referredSnapshotHashList)-1 && prevSbFlag {
					prevSb = vSb
				}
			}
		}
		if latestSb.Height < referredMaxSbHeight {
			return nil, nil, errors.New("the height of the snapshotblock referred can't be larger than the latest")
		}
	}
	gapHeight := latestSb.Height - referredMaxSbHeight
	fittestSbHeight = latestSb.Height - minGapToLatest(gapHeight, DefaultHeightDifference)
	if isRandom && fittestSbHeight < latestSb.Height {
		fittestSbHeight = fittestSbHeight + addHeight(1)
	}

	// protect code
	if fittestSbHeight > latestSb.Height || fittestSbHeight < referredMaxSbHeight {
		fittestSbHeight = latestSb.Height
	}

	fittestSb, err := chain.GetSnapshotBlockByHeight(fittestSbHeight)
	if fittestSb == nil {
		if err != nil {
			return nil, nil, err
		}
		return nil, nil, ErrGetFittestSnapshotBlockFailed
	}
	fittestSbHash = &fittestSb.Hash

	if accAddr == nil || prevSb == nil || prevSb.Hash == *fittestSbHash || prevSb.Height+types.SnapshotHourHeight < fittestSbHeight {
		return nil, fittestSbHash, nil
	}
	return &prevSb.Hash, fittestSbHash, nil
}

func addHeight(gapHeight uint64) uint64 {
	randHeight := uint64(0)
	if gapHeight >= 1 {
		rand.Seed(time.Now().UnixNano())
		randHeight = uint64(rand.Intn(int(gapHeight + 1)))
	}
	return randHeight
}

func minGapToLatest(us ...uint64) uint64 {
	if len(us) == 0 {
		panic("zero args")
	}
	min := us[0]
	for _, u := range us {
		if u < min {
			min = u
		}
	}
	return min
}
