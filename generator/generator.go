package generator

import (
	"errors"
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/consensus/core"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/pow"
	"github.com/vitelabs/go-vite/vm"
	"github.com/vitelabs/go-vite/vm/util"
	"github.com/vitelabs/go-vite/vm_db"
	"math/big"
)

type Chain interface {
	GetAccountBlockByHash(blockHash types.Hash) (*ledger.AccountBlock, error)
	GetSnapshotBlockByContractMeta(addr *types.Address, fromHash *types.Hash) (*ledger.SnapshotBlock, error)
	GetSeed(limitSb *ledger.SnapshotBlock, fromHash types.Hash) (uint64, error)
}

type Consensus interface {
	SBPReader() core.SBPStatReader
}

type SignFunc func(addr types.Address, data []byte) (signedData, pubkey []byte, err error)

type Generator struct {
	chain Chain

	vmDb vm_db.VmDb
	vm   *vm.VM

	log log15.Logger
}

type GenResult struct {
	VmBlock *vm_db.VmAccountBlock
	IsRetry bool
	Err     error
}

func NewGenerator(chain vm_db.Chain, consensus Consensus, addr types.Address, latestSnapshotBlockHash, prevBlockHash *types.Hash) (*Generator, error) {
	gen := &Generator{
		log: log15.New("module", "Generator"),
	}
	gen.chain = chain

	gen.vm = vm.NewVM(util.NewVmConsensusReader(consensus.SBPReader()))

	vmDb, err := vm_db.NewVmDb(chain, &addr, latestSnapshotBlockHash, prevBlockHash)
	if err != nil {
		return nil, err
	}
	gen.vmDb = vmDb

	return gen, nil
}

func (gen *Generator) GenerateWithBlock(block *ledger.AccountBlock, fromBlock *ledger.AccountBlock) (*GenResult, error) {
	genResult, err := gen.generateBlock(block, fromBlock, nil, nil)
	if err != nil {
		return nil, err
	}
	return genResult, nil
}

func (gen *Generator) GenerateWithMessage(message *IncomingMessage, producer *types.Address, signFunc SignFunc) (*GenResult, error) {
	block, err := IncomingMessageToBlock(gen.vmDb, message)
	if err != nil {
		return nil, err
	}
	var fromBlock *ledger.AccountBlock
	if block.IsReceiveBlock() {
		var fromErr error
		fromBlock, fromErr = gen.chain.GetAccountBlockByHash(block.FromBlockHash)
		if fromErr != nil {
			return nil, fromErr
		}
		if fromBlock == nil {
			return nil, errors.New("generate recvBlock failed, cause failed to find its sendBlock")
		}
	}
	return gen.generateBlock(block, fromBlock, producer, signFunc)
}

func (gen *Generator) GenerateWithOnRoad(sendBlock *ledger.AccountBlock, producer *types.Address, signFunc SignFunc, difficulty *big.Int) (*GenResult, error) {
	block, err := gen.packReceiveBlockWithSend(sendBlock, difficulty)
	if err != nil {
		return nil, err
	}
	genResult, err := gen.generateBlock(block, sendBlock, producer, signFunc)
	if err != nil {
		return nil, err
	}
	return genResult, nil
}

func (gen *Generator) generateBlock(block *ledger.AccountBlock, fromBlock *ledger.AccountBlock, producer *types.Address, signFunc SignFunc) (result *GenResult, resultErr error) {
	defer func() {
		if err := recover(); err != nil {
			errDetail := fmt.Sprintf("block(addr:%v prevHash:%v)", block.AccountAddress, block.PrevHash)
			if fromBlock != nil {
				errDetail += fmt.Sprintf("fromBlock(addr:%v hash:%v)", fromBlock.AccountAddress, fromBlock.Hash)
			}

			gen.log.Error(fmt.Sprintf("generator_vm panic error %v", err), "detail", errDetail)

			result = &GenResult{}
			resultErr = errors.New("generator_vm panic error")
		}
	}()
	var state *VMGlobalStatus
	if block.IsReceiveBlock() {
		if fromBlock == nil {
			return nil, errors.New("need to pass in sendBlock when generate receiveBlock")
		}
		sb, stateErr := gen.chain.GetSnapshotBlockByContractMeta(&block.AccountAddress, &fromBlock.Hash)
		if stateErr != nil {
			return nil, errors.New(fmt.Sprintf("GetSnapshotBlockByContractMeta failed, err:%v", stateErr))
		}
		if sb != nil {
			state = NewVMGlobalStatus(gen.chain, sb, fromBlock.Hash)
		}
	}

	vmBlock, isRetry, err := gen.vm.RunV2(gen.vmDb, block, fromBlock, state)
	if vmBlock != nil {
		vb := vmBlock.AccountBlock
		if vb.IsReceiveBlock() && vb.SendBlockList != nil && len(vb.SendBlockList) > 0 {
			for idx, v := range vb.SendBlockList {
				v.Hash = v.ComputeSendHash(vb, uint8(idx))
			}
		}
		vb.Hash = vb.ComputeHash()
		if signFunc != nil {
			if producer == nil {
				return nil, errors.New("producer address is uncertain, can't sign")
			}
			signature, publicKey, e := signFunc(*producer, vb.Hash.Bytes())
			if e != nil {
				return nil, e
			}
			vb.Signature = signature
			vb.PublicKey = publicKey
		}
	}

	return &GenResult{
		VmBlock: vmBlock,
		IsRetry: isRetry,
		Err:     err,
	}, nil
}

func (gen *Generator) packReceiveBlockWithSend(sendBlock *ledger.AccountBlock, difficulty *big.Int) (*ledger.AccountBlock, error) {
	recvBlock := &ledger.AccountBlock{
		BlockType:      ledger.BlockTypeReceive,
		AccountAddress: sendBlock.ToAddress,
		FromBlockHash:  sendBlock.Hash,

		/*	//recv don't need
			ToAddress: types.Address{},
			TokenId:   types.TokenTypeId{},
			Amount:    nil,
			Fee:       nil,

			// after vm
			Data:          nil,
			Quota:         0,
			SendBlockList: nil,
			LogHash:       nil,
			PublicKey:     nil,
			Signature:     nil,
			Hash:          types.Hash{},*/
	}
	// PrevHash, Height, Nonce, Difficulty
	prevBlock, err := gen.vmDb.PrevAccountBlock()
	if err != nil {
		return nil, err
	}
	var prevHash types.Hash
	var preHeight uint64 = 0
	if prevBlock != nil {
		prevHash = prevBlock.Hash
		preHeight = prevBlock.Height
	}
	recvBlock.PrevHash = prevHash
	recvBlock.Height = preHeight + 1

	if difficulty != nil {
		nonce, err := pow.GetPowNonce(difficulty, types.DataHash(append(sendBlock.ToAddress.Bytes(), prevHash.Bytes()...)))
		if err != nil {
			return nil, err
		}
		recvBlock.Nonce = nonce
		recvBlock.Difficulty = difficulty
	}

	return recvBlock, nil
}

func (gen *Generator) GetVmDb() vm_db.VmDb {
	return gen.vmDb
}
