package generator

import (
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/vm"
	"github.com/vitelabs/go-vite/vm_context"
	"github.com/vitelabs/go-vite/vm_context/vmctxt_interface"
	"time"
)

const (
	SourceTypeP2P = iota
	SourceTypeUserInitiate
	SourceTypeUnconfirmed
)

type SignFunc func(addr types.Address, data []byte) (signedData, pubkey []byte, err error)

type Generator struct {
	vm        vm.VM
	vmContext vmctxt_interface.VmDatabase

	chain  Chain
	signer Signer

	log log15.Logger
}

type GenResult struct {
	BlockGenList []*vm_context.VmAccountBlock
	IsRetry      bool
	Err          error
}

func NewGenerator(chain Chain, wSigner Signer) *Generator {
	return &Generator{
		chain:  chain,
		signer: wSigner,
		log:    log15.New("module", "Generator"),
	}
}

func (gen *Generator) PrepareVm(snapshotBlockHash, preBlockHash *types.Hash, addr *types.Address) error {
	//fixme @gx 和炎达交流一下

	var sbHash, pbHash *types.Hash
	if snapshotBlockHash == nil {
		snapshotBlock, err := gen.chain.GetLatestSnapshotBlock()
		if err != nil || snapshotBlock == nil {
			return errors.New("PrepareVm.GetLatestSnapshotBlock, Error:" + err.Error())
		}
		sbHash = &snapshotBlock.Hash
	} else {
		sbHash = snapshotBlockHash
	}
	if preBlockHash == nil {
		preBlock, err := gen.chain.GetLatestAccountBlock(addr)
		if err != nil || preBlock == nil {
			return errors.New("PrepareVm.GetLatestAccountBlock, Error:" + err.Error())
		}
		pbHash = &preBlock.Hash
	} else {
		pbHash = preBlockHash
	}

	vmContext, err := vm_context.NewVmContext(gen.chain, sbHash, pbHash, addr)
	if err != nil {
		return err
	}
	gen.vmContext = vmContext
	gen.vm = *vm.NewVM()
	return nil
}

//  todo maintain the sendCreate Block's gidToAddrList
func (gen *Generator) GenerateWithMessage(message *IncomingMessage, signFunc SignFunc) (*GenResult, error) {
	block, err := gen.PackBlockWithMessage(message)
	if err != nil {
		return nil, err
	}
	var sendBlock *ledger.AccountBlock = nil
	if block.BlockType != ledger.BlockTypeSendCall && block.BlockType != ledger.BlockTypeSendCreate {
		sendBlock = gen.vmContext.GetAccountBlockByHash(&block.FromBlockHash)
	}
	return gen.generateBlock(SourceTypeUserInitiate, block, sendBlock, signFunc), nil

}

func (gen *Generator) GenerateWithOnroad(sendBlock ledger.AccountBlock, consensusMsg *ConsensusMessage, signFunc SignFunc) (*GenResult, error) {
	block, err := gen.PackBlockWithSendBlock(&sendBlock, consensusMsg)
	if err != nil {
		return nil, err
	}
	return gen.generateBlock(SourceTypeUnconfirmed, block, &sendBlock, signFunc), nil
}

//  todo maintain the sendCreate Block's gidToAddrList
func (gen *Generator) GenerateWithP2PBlock(block *ledger.AccountBlock, signFunc SignFunc) *GenResult {
	var sendBlock *ledger.AccountBlock = nil
	if block.BlockType != ledger.BlockTypeSendCall && block.BlockType != ledger.BlockTypeSendCreate {
		sendBlock = gen.vmContext.GetAccountBlockByHash(&block.FromBlockHash)
	}
	return gen.generateBlock(SourceTypeP2P, block, sendBlock, signFunc)
}

func (gen *Generator) generateBlock(sourceType byte, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, signFunc SignFunc) *GenResult {
	gen.log.Info("generateBlock", "SourceType", sourceType, "BlockType", block.BlockType)

	blockList, isRetry, err := gen.vm.Run(gen.vmContext, block, sendBlock)
	accountBlock := blockList[0].AccountBlock

	accountBlock.Hash = accountBlock.ComputeHash()
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

func (gen *Generator) PackBlockWithSendBlock(sendBlock *ledger.AccountBlock, consensusMsg *ConsensusMessage) (blockPacked *ledger.AccountBlock, err error) {
	gen.log.Info("PackReceiveBlock", "sendBlock.Hash", sendBlock.Hash, "sendBlock.To", sendBlock.ToAddress)
	blockPacked = &ledger.AccountBlock{
		BlockType:      0,
		AccountAddress: sendBlock.ToAddress,
		FromBlockHash:  sendBlock.Hash,
		Amount:         sendBlock.Amount,
		TokenId:        sendBlock.TokenId,
		Quota:          sendBlock.Quota,
		Fee:            sendBlock.Fee,
		Nonce:          sendBlock.Nonce,
		Data:           sendBlock.Data,

		ToAddress: types.Address{},
		StateHash: types.Hash{},
		LogHash:   nil,
		PublicKey: nil, // contractAddress's receiveBlock's publicKey is from consensus node
		Signature: nil,
	}

	preBlock := gen.vmContext.PrevAccountBlock()
	if preBlock == nil {
		return nil, errors.New("PackBlockWithSendBlock.PrevAccountBlock failed")
	}
	blockPacked.Hash = preBlock.Hash
	blockPacked.Height = preBlock.Height + 1

	if gid := gen.vmContext.GetGid(); gid != nil {
		if consensusMsg == nil {
			return nil, errors.New("contractAddress must enter ConsensusMessages")
		}
		blockPacked.Timestamp = &consensusMsg.Timestamp
		blockPacked.SnapshotHash = consensusMsg.SnapshotHash
	} else {
		st := time.Now()
		blockPacked.Timestamp = &st
		snapshotBlock := gen.vmContext.CurrentSnapshotBlock()
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
