package generator

import (
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

type Generator struct {
	Vm        vm.VM
	VmContext vmctxt_interface.VmDatabase

	chain  vm_context.Chain
	signer SignManager

	log log15.Logger
}

type SignerFunc func(a types.Address, data []byte, passphrase string) (signedData, pubkey []byte, err error)

func (gen *Generator) GenerateWithBlock(sourceType byte, block *ledger.AccountBlock, sigFunc SignerFunc) *GenResult {
	select {
	case sourceType == SourceTypeP2P:
		return gen.generateP2PTx(block, sigFunc)
	case sourceType == SourceTypeUnconfirmed:
		return gen.generateUnconfirmedTx(block, sigFunc)
	case sourceType == SourceTypeUserInitiate:
		return gen.generateInitiateTx(block, sigFunc)
	}
	return nil
}

//func (gen *Generator) GenerateWithMessage(message *IncomingMessage, passphrase string) *GenResult {
//	block := gen.PackBlockWithMessage(message)
//	return gen.GenerateBlockWithPassphrase(SourceTypeUserInitiate, block, passphrase)
//}
//
//func (gen *Generator) GenerateBlockWithPassphrase(sourceType byte, block *ledger.AccountBlock, passphrase string) *GenResult {
//	select {
//	case sourceType == SourceTypeUserInitiate:
//		return gen.generateInitiateTx(block, passphrase)
//	}
//	return nil
//}

func (gen *Generator) generateP2PTx(block *ledger.AccountBlock, sigFunc SignerFunc) *GenResult {
	gen.log.Info("generateP2PTx", "BlockType", block.BlockType)

	var blockList []*vm_context.VmAccountBlock
	var isRetry bool
	var err error

	if block.BlockType != ledger.BlockTypeSendCall && block.BlockType != ledger.BlockTypeSendCreate {
		sendBlock := gen.VmContext.GetAccountBlockByHash(&block.FromBlockHash)
		blockList, isRetry, err = gen.Vm.Run(gen.VmContext, block, sendBlock)
	} else {
		blockList, isRetry, err = gen.Vm.Run(gen.VmContext, block, nil)
	}

	blockList[0].AccountBlock.Hash = blockList[0].AccountBlock.GetComputeHash()

	return &GenResult{
		BlockGenList: blockList,
		IsRetry:      isRetry,
		Err:          err,
	}
}

// generateInitiateTx: currently only support to accept commonTx, and passphrase is in necessary
func (gen *Generator) generateInitiateTx(block *ledger.AccountBlock, sigFunc SignerFunc) *GenResult {
	gen.log.Info("generateInitiateTx", "BlockType", block.BlockType)

	var blockList []*vm_context.VmAccountBlock
	var isRetry bool
	var err error

	if block.BlockType != ledger.BlockTypeSendCall && block.BlockType != ledger.BlockTypeSendCreate {
		sendBlock := gen.VmContext.GetAccountBlockByHash(&block.FromBlockHash)
		blockList, isRetry, err = gen.Vm.Run(gen.VmContext, block, sendBlock)
	} else {
		blockList, isRetry, err = gen.Vm.Run(gen.VmContext, block, nil)
	}

	blockList[0].AccountBlock.Hash = blockList[0].AccountBlock.GetComputeHash()

	//var signErr error
	//if blockList[0].AccountBlock.Signature, blockList[0].AccountBlock.PublicKey, signErr =
	//	gen.signer.SignDataWithPassphrase(blockList[0].AccountBlock.AccountAddress, passphrase,
	//		blockList[0].AccountBlock.Hash.Bytes()); signErr != nil {
	//	gen.log.Error("SignData Error", signErr)
	//	return nil
	//}

	return &GenResult{
		BlockGenList: blockList,
		IsRetry:      isRetry,
		Err:          err,
	}
}

// generateUnconfirmedTx: only handle receiveBlock
func (gen *Generator) generateUnconfirmedTx(block *ledger.AccountBlock, sigFunc SignerFunc) *GenResult {
	gen.log.Info("generateUnconfirmedTx", "BlockType", block.BlockType)

	sendBlock := gen.VmContext.GetAccountBlockByHash(&block.FromBlockHash)
	blockList, isRetry, err := gen.Vm.Run(gen.VmContext, block, sendBlock)

	blockList[0].AccountBlock.Hash = blockList[0].AccountBlock.GetComputeHash()

	//var signErr error
	//if blockList[0].AccountBlock.Signature, blockList[0].AccountBlock.PublicKey, signErr =
	//	gen.signer.SignData(blockList[0].AccountBlock.AccountAddress, blockList[0].AccountBlock.Hash.Bytes()); signErr != nil {
	//	gen.log.Error("SignData Error", signErr)
	//	return nil
	//}

	return &GenResult{
		BlockGenList: blockList,
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
		Quota:          message.Quota,
		PublicKey:      nil, // at the time to sign after vm generate
	}
	if block.BlockType != ledger.BlockTypeSendCall && block.BlockType != ledger.BlockTypeSendCreate {
		block.ToAddress = types.Address{}
		block.FromBlockHash = *message.FromBlockHash
	} else {
		block.FromBlockHash = types.Hash{}
		block.ToAddress = *message.ToAddress
	}

	latestBlock := gen.VmContext.PrevAccountBlock()
	block.Height = latestBlock.Height + 1
	block.PrevHash = latestBlock.Hash

	latestSnapshotBlock := gen.VmContext.CurrentSnapshotBlock()
	block.SnapshotHash = latestSnapshotBlock.Hash

	return block
}

func (gen *Generator) PackUnconfirmedReceiveBlock(sendBlock *ledger.AccountBlock, snapshotHash *types.Hash, timestamp *time.Time) *ledger.AccountBlock {
	gen.log.Info("PackReceiveBlock", gen.log.New("sendBlock.Hash", sendBlock.Hash),
		gen.log.New("sendBlock.To", sendBlock.ToAddress))

	block := &ledger.AccountBlock{
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

	preBlock := gen.VmContext.PrevAccountBlock()
	if preBlock == nil {
		return nil
	} else {
		block.Hash = preBlock.Hash
		block.Height = preBlock.Height + 1
	}

	if gid, _ := gen.chain.GetContractGid(&block.AccountAddress); gid != nil {
		block.Timestamp = timestamp
		block.SnapshotHash = *snapshotHash
	} else {
		st := time.Now()
		block.Timestamp = &st

		snapshotBlock := gen.VmContext.CurrentSnapshotBlock()
		block.SnapshotHash = snapshotBlock.Hash
	}

	return block
}

type GenResult struct {
	BlockGenList []*vm_context.VmAccountBlock
	IsRetry      bool
	Err          error
}
