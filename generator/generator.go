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

type SignerFunc func(addr types.Address, data []byte) (signedData, pubkey []byte, err error)

type Generator struct {
	vm        vm.VM
	vmContext vmctxt_interface.VmDatabase

	chain  Chain
	signer Signer

	log log15.Logger
}

func NewGenerator(chain Chain, wSigner Signer) *Generator {
	return &Generator{
		chain:  chain,
		signer: wSigner,
		log:    log15.New("module", "Generator"),
	}
}

func (gen *Generator) PrepareVm(snapshotBlockHash, preBlockHash *types.Hash, addr *types.Address) error {
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

func (gen *Generator) GenerateWithMessage(message *IncomingMessage, sigFunc SignerFunc) (*GenResult, error) {
	block, err := gen.PackBlockWithMessage(message)
	if err != nil {
		return nil, err
	}
	if block.BlockType != ledger.BlockTypeSendCall && block.BlockType != ledger.BlockTypeSendCreate {
		sendBlock := gen.vmContext.GetAccountBlockByHash(&block.FromBlockHash)
		return gen.generateBlock(SourceTypeUserInitiate, block, sendBlock, sigFunc), nil
	} else {
		return gen.generateBlock(SourceTypeUserInitiate, block, nil, sigFunc), nil
	}
}

func (gen *Generator) GenerateWithUnconfirmed(sendBlock ledger.AccountBlock, conMessage *ConsensusMessage, sigFunc SignerFunc) (*GenResult, error) {
	block, err := gen.PackBlockWithSendBlock(&sendBlock, conMessage)
	if err != nil {
		return nil, err
	}
	return gen.generateBlock(SourceTypeUnconfirmed, block, &sendBlock, sigFunc), nil
}

func (gen *Generator) GenerateWithP2PBlock(block *ledger.AccountBlock, sigFunc SignerFunc) *GenResult {
	if block.BlockType != ledger.BlockTypeSendCall && block.BlockType != ledger.BlockTypeSendCreate {
		sendBlock := gen.vmContext.GetAccountBlockByHash(&block.FromBlockHash)
		return gen.generateBlock(SourceTypeP2P, block, sendBlock, sigFunc)
	} else {
		return gen.generateBlock(SourceTypeP2P, block, nil, sigFunc)
	}
}

func (gen *Generator) generateBlock(sourceType byte, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, sigFunc SignerFunc) *GenResult {
	gen.log.Info("generateBlock", "SourceType", sourceType, "BlockType", block.BlockType)

	blockList, isRetry, err := gen.vm.Run(gen.vmContext, block, sendBlock)

	blockList[0].AccountBlock.Hash = blockList[0].AccountBlock.GetComputeHash()

	if sigFunc != nil {
		blockList[0].AccountBlock.Signature, blockList[0].AccountBlock.PublicKey, err = sigFunc(
			blockList[0].AccountBlock.AccountAddress, blockList[0].AccountBlock.Hash.Bytes())
		if err != nil {
			gen.log.Error("generate.Sign()", "Error", err)
		}
	}

	return &GenResult{
		BlockGenList: blockList,
		IsRetry:      isRetry,
		Err:          err,
	}

	return nil
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

func (gen *Generator) PackBlockWithSendBlock(sendBlock *ledger.AccountBlock, conMessage *ConsensusMessage) (
	blockPacked *ledger.AccountBlock, err error) {

	gen.log.Info("PackReceiveBlock", gen.log.New("sendBlock.Hash", sendBlock.Hash),
		gen.log.New("sendBlock.To", sendBlock.ToAddress))

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
	} else {
		blockPacked.Hash = preBlock.Hash
		blockPacked.Height = preBlock.Height + 1
	}

	if gid := gen.vmContext.GetGid(); gid != nil {
		if conMessage != nil {
			blockPacked.Timestamp = &conMessage.Timestamp
			blockPacked.SnapshotHash = conMessage.SnapshotHash
		} else {
			return nil, errors.New("contractAddress must enter ConsensusMessages")
		}
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

type GenResult struct {
	BlockGenList []*vm_context.VmAccountBlock
	IsRetry      bool
	Err          error
}
