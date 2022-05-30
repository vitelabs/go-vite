package generator

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/vitelabs/go-vite/v2/common/types"
	"github.com/vitelabs/go-vite/v2/common/upgrade"
	"github.com/vitelabs/go-vite/v2/interfaces"
	ledger "github.com/vitelabs/go-vite/v2/interfaces/core"
	"github.com/vitelabs/go-vite/v2/ledger/consensus/core"
	"github.com/vitelabs/go-vite/v2/log15"
	"github.com/vitelabs/go-vite/v2/pow"
	"github.com/vitelabs/go-vite/v2/vm"
	"github.com/vitelabs/go-vite/v2/vm/util"
	"github.com/vitelabs/go-vite/v2/vm_db"
)

// Consensus is for Vm to read the SBP information
type Consensus interface {
	SBPReader() core.SBPStatReader
}

type chain interface {
	GetAccountBlockByHash(blockHash types.Hash) (*ledger.AccountBlock, error)
	GetSnapshotBlockByContractMeta(addr types.Address, fromHash types.Hash) (*ledger.SnapshotBlock, error)
	GetSeedConfirmedSnapshotBlock(addr types.Address, fromHash types.Hash) (*ledger.SnapshotBlock, error)
	GetSeed(limitSb *ledger.SnapshotBlock, fromHash types.Hash) (uint64, error)
}

// Generator implements the logic to generate a account transaction block.
type generator struct {
	chain chain

	vmDb interfaces.VmDb
	vm   *vm.VM

	log log15.Logger
}

// NewGenerator needs to new a interfaces.VmDb with state of the world and SBP information for Vm,
//
// the third "addr" needs to be filled with the address of the account chain to be blocked,
// and the last needs to be filled with the previous/latest block's hash on the account chain.
func NewGenerator(chain vm_db.Chain, sbpStatReader core.SBPStatReader, addr types.Address, latestSnapshotBlockHash, prevBlockHash *types.Hash) (interfaces.Generator, error) {
	gen := &generator{
		log: log15.New("module", "Generator"),
	}
	gen.chain = chain

	gen.vm = vm.NewVM(util.NewVMConsensusReader(sbpStatReader))

	vmDb, err := vm_db.NewVmDb(chain, &addr, latestSnapshotBlockHash, prevBlockHash)
	if err != nil {
		return nil, err
	}
	gen.vmDb = vmDb

	return gen, nil
}

// GenerateWithBlock implements the method to generate a transaction with VM execution results
// from a block which contains the complete transaction info.
func (gen *generator) GenerateWithBlock(block *ledger.AccountBlock, fromBlock *ledger.AccountBlock) (*interfaces.GenResult, error) {
	genResult, err := gen.generateBlock(block, fromBlock, nil, nil)
	if err != nil {
		return nil, err
	}
	return genResult, nil
}

// GenerateWithMessage implements the method to generate a transaction with VM execution results
// from a IncomingMessage which contains the necessary transaction info.
func (gen *generator) GenerateWithMessage(message *interfaces.IncomingMessage, producer *types.Address, signFunc interfaces.SignFunc) (*interfaces.GenResult, error) {
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

// GenerateWithOnRoad implements the method to generate a transaction with VM execution results
// from a sendBlock(onroad block).
func (gen *generator) GenerateWithOnRoad(sendBlock *ledger.AccountBlock, producer *types.Address, signFunc interfaces.SignFunc, difficulty *big.Int) (*interfaces.GenResult, error) {
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

func (gen *generator) generateBlock(block *ledger.AccountBlock, fromBlock *ledger.AccountBlock, producer *types.Address, signFunc interfaces.SignFunc) (result *interfaces.GenResult, resultErr error) {
	defer func() {
		if err := recover(); err != nil {
			// debug.PrintStack()
			errDetail := fmt.Sprintf("block(addr:%v prevHash:%v)", block.AccountAddress, block.PrevHash)
			if fromBlock != nil {
				errDetail += fmt.Sprintf("fromBlock(addr:%v hash:%v)", fromBlock.AccountAddress, fromBlock.Hash)
			}
			gen.log.Error(fmt.Sprintf("generator_vm panic error %v", err), "detail", errDetail)
			result = &interfaces.GenResult{}
			resultErr = types.ErrVmRunPanic
		}
	}()
	var state *VMGlobalStatus
	if block.IsReceiveBlock() {
		if fromBlock == nil {
			return nil, errors.New("need to pass in sendBlock when generate receiveBlock")
		}
		latestSb, _ := gen.GetVMDB().LatestSnapshotBlock()
		if latestSb == nil {
			return nil, fmt.Errorf("vmDb's latestSnapshotBlock is nil")
		}

		limitSb, err := gen.chain.GetSnapshotBlockByContractMeta(block.AccountAddress, fromBlock.Hash)
		if err != nil {
			return nil, fmt.Errorf("GetSnapshotBlockByContractMeta failed, %+v", err)
		}
		if upgrade.IsSeedUpgrade(latestSb.Height) {
			limitSeedSb, err := gen.chain.GetSeedConfirmedSnapshotBlock(block.AccountAddress, fromBlock.Hash)
			if err != nil {
				return nil, fmt.Errorf("GetSeedConfirmedSnapshotBlock failed, %+v", err)
			}
			if limitSb == nil {
				if limitSeedSb != nil {
					limitSb = limitSeedSb
				}
			} else {
				if limitSeedSb != nil && limitSb.Height < limitSeedSb.Height {
					limitSb = limitSeedSb
				}
			}
		}
		if limitSb != nil {
			state = NewVMGlobalStatus(gen.chain, limitSb, fromBlock.Hash)
			gen.log.Info("gen GlobalStatus", "hash", limitSb.Hash, "fromHash", fromBlock.Hash)
		}
	}

	vmBlock, isRetry, err := gen.vm.RunV2(gen.vmDb, block, fromBlock, state)
	if err != nil {
		bDetail := fmt.Sprintf("block(addr:%v prevHash:%v)", block.AccountAddress, block.PrevHash)
		if fromBlock != nil {
			bDetail += fmt.Sprintf("fromBlock(addr:%v hash:%v)", fromBlock.AccountAddress, fromBlock.Hash)
		}
		gen.log.Info(fmt.Sprintf("vm Run err %v", err), "detail", bDetail)
	}
	if vmBlock != nil {
		vb := vmBlock.AccountBlock
		if vb.IsReceiveBlock() && types.IsContractAddr(vb.AccountAddress) && len(vb.SendBlockList) > 0 {
			for idx, v := range vb.SendBlockList {
				v.Hash = v.ComputeSendHash(vb, uint8(idx))
			}
		}
		vb.Hash = vb.ComputeHash()
		if signFunc != nil {
			if producer == nil {
				return nil, errors.New("producer address is uncertain, can't sign")
			}
			signature, publicKey, e := signFunc(vb.Hash.Bytes())
			if e != nil {
				return nil, e
			}
			vb.Signature = signature
			vb.PublicKey = publicKey
		}
	}

	return &interfaces.GenResult{
		VMBlock: vmBlock,
		IsRetry: isRetry,
		Err:     err,
	}, nil
}

func (gen *generator) packReceiveBlockWithSend(sendBlock *ledger.AccountBlock, difficulty *big.Int) (*ledger.AccountBlock, error) {
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
			QuotaUsed:         0,
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
	var preHeight uint64
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

// GetVMDB returns the interfaces.VmDb the current Generator used.
func (gen *generator) GetVMDB() interfaces.VmDb {
	return gen.vmDb
}
