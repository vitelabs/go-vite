package builtin

import (
	"encoding/binary"
	"errors"
	"github.com/vitelabs/go-vite/v2/common/types"
	"github.com/vitelabs/go-vite/v2/common/upgrade"
	"github.com/vitelabs/go-vite/v2/interfaces"
	ledger "github.com/vitelabs/go-vite/v2/interfaces/core"
	"github.com/vitelabs/go-vite/v2/log15"
	"github.com/vitelabs/go-vite/v2/vm/abi"
	"github.com/vitelabs/go-vite/v2/vm/util"
	"github.com/vitelabs/go-vite/v2/vm_db"
	"math/big"
)


type vmEnvironment interface {
	GlobalStatus() 		util.GlobalStatus
	ConsensusReader() 	util.ConsensusReader
	Chain()				vm_db.Chain
	SetOrigin(origin *ledger.AccountBlock)
	GetOrigin() 		*ledger.AccountBlock
}

type ExecutionFunction func(request NativeContractRequest) (*NativeContractResult, error)

var (
	log = log15.New("module", "vm")
	NativeContracts = initNativeContracts()
	ContractAssetV3 = NewTokenContract()
	PackedTrue = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}
	PackedZero = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
)

type NativeContract struct {
	Name 	string
	Abi     *abi.ABIContract
	Methods map[uint32] *NativeContractMethod
	ReceiveFunction ExecutionFunction
	FallbackFunction ExecutionFunction
}

type NativeContractMethod struct {
	Abi abi.Method
	Execute ExecutionFunction
}

type NativeContractRequest struct {
	Abi *abi.Method
	Db           *interfaces.VmDb
	ReceiveBlock *ledger.AccountBlock
	SendBlock    *ledger.AccountBlock
	Vm *vmEnvironment
}

type NativeContractResult struct {
	Data []byte
	TriggeredBlocks []*ledger.AccountBlock
	Events []*ledger.VmLog
}

func initNativeContracts() map[types.Address]*NativeContract {
	contracts := make(map[types.Address]*NativeContract)

	contracts[types.AddressAsset] = ContractAssetV3

	return contracts
}

func Exists(addr types.Address, methodSelector []byte, sbHeight uint64) bool {
	if upgrade.IsVersion11Upgrade(sbHeight) {
		c := NativeContracts[addr]
		if c != nil {
			if len(methodSelector) == 0 {
				// block without data
				log.Debug("built-in contract: match method", "contract", c.Name, "method", "receive()")
				return true
			}
			method := c.Methods[binary.BigEndian.Uint32(methodSelector[:4])]
			if method != nil {
				log.Debug("built-in contract: match method", "contract", c.Name, "method", method.Abi.Name)
				return true
			}
		}
	}
	return false
}

func Execute(db interfaces.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	// router
	contract := NativeContracts[sendBlock.ToAddress]
	if contract == nil {
		return nil, util.ErrContractNotExists
	}

	// function selector
	var method *NativeContractMethod
	calldata := sendBlock.Data
	if len(calldata) == 0 {
		// receive function
		method = &NativeContractMethod{Execute: contract.ReceiveFunction}
	} else {
		method = contract.Methods[binary.BigEndian.Uint32(calldata[:4])]
	}

	if method == nil || method.Execute == nil {
		if contract.FallbackFunction == nil {
			return nil, util.ErrAbiMethodNotFound
		}
		// fallback function
		method = &NativeContractMethod{Execute: contract.FallbackFunction}
	}

	// execution context
	sendType := sendBlock.BlockType
	if sendType == ledger.BlockTypeSendCallback || sendType == ledger.BlockTypeSendFailureCallback {
		context, err := db.GetExecutionContext(&sendBlock.Hash)
		if err != nil {
			log.Error("builtin Execute(): GetExecutionContext fails", "error", err)
			return nil, err
		}
		syncCallSendHash := context.ReferrerSendHash
		syncCallSendBlock, err := vm.Chain().GetAccountBlockByHash(syncCallSendHash)
		if err != nil {
			log.Error("builtin Execute(): GetAccountBlockByHash fails", "error", err)
			return nil, err
		}

		// validate sync call send block
		if syncCallSendBlock.AccountAddress != block.AccountAddress || syncCallSendBlock.ToAddress != sendBlock.AccountAddress {
			return nil, errors.New("the callback transaction references an invalid send block")
		}

		syncCallContext, err := db.GetExecutionContext(&syncCallSendHash)
		if err != nil {
			log.Error("builtin Execute(): GetExecutionContext of send fails", "error", err)
			return nil, err
		}

		// get origin send block
		originSendHash := syncCallContext.ReferrerSendHash
		originSendBlock, err := vm.Chain().GetAccountBlockByHash(originSendHash)
		if err != nil {
			log.Error("builtin Execute(): GetAccountBlockByHash fails when get origin", "error", err)
			return nil, err
		}

		// validate origin send block
		if originSendBlock.ToAddress != block.AccountAddress {
			return nil, errors.New("invalid origin send block")
		}
		vm.SetOrigin(originSendBlock)
	}

	// execute contract method
	req := NativeContractRequest{&method.Abi, &db, block, sendBlock, &vm}
	res, err := method.Execute(req)
	if err != nil {
		return nil, err
	}

	// append VM logs
	for _, event := range res.Events {
		db.AddLog(event)
	}

	// prepare triggered block
	origin := sendBlock
	if vm.GetOrigin() != nil {
		origin = vm.GetOrigin()
	}
	if origin.BlockType == ledger.BlockTypeSendSyncCall {
		// load execution context
		originContext, err := db.GetExecutionContext(&origin.Hash)
		if err != nil {
			log.Error("builtin Execute(): GetExecutionContext fails", "error", err)
			return nil, err
		}
		if originContext == nil {
			return nil, errors.New("no execution context in the send block of sync call")
		}
		callback := originContext.CallbackId
		var data []byte
		// append callback id
		data = append(data, callback.FillBytes(make([]byte, 4))...)
		// append return data
		data = append(data, res.Data...)

		callbackBlock := util.MakeRequestBlock(
			block.AccountAddress,
			origin.AccountAddress,
			ledger.BlockTypeSendCallback,
			big.NewInt(0),
			ledger.ViteTokenId,
			data)

		executionContext := ledger.ExecutionContext{
			ReferrerSendHash: origin.Hash,
		}

		callbackBlock.Hash = callbackBlock.ComputeHash()

		// save execution context
		db.SetExecutionContext(&callbackBlock.Hash, &executionContext)

		return append(res.TriggeredBlocks, callbackBlock), nil
	}

	return res.TriggeredBlocks, nil
}

func Query(db interfaces.VmDb, address types.Address, calldata []byte) ([]byte, error) {
	// router
	contract := NativeContracts[address]
	if contract == nil {
		return nil, errors.New("native contract does not exist")
	}

	// function selector
	method := contract.Methods[binary.BigEndian.Uint32(calldata[:4])]

	if method == nil {
		return nil, util.ErrAbiMethodNotFound
	}

	sendBlock := &ledger.AccountBlock{
		Height:         1,
		ToAddress:      address,
		AccountAddress: types.ZERO_ADDRESS,
		Amount:         big.NewInt(0),
		TokenId:        ledger.ViteTokenId,
		BlockType:      ledger.BlockTypeSendCall,
		Fee:            big.NewInt(0),
		PrevHash:       types.ZERO_HASH,
		Data:           calldata,
	}

	// execute contract method
	req := NativeContractRequest{&method.Abi, &db, nil, sendBlock, nil}
	res, err := method.Execute(req)

	return res.Data, err
}

func NewLog(c abi.ABIContract, name string, params ...interface{}) *ledger.VmLog {
	topics, data, _ := c.PackEvent(name, params...)
	return &ledger.VmLog{Topics: topics, Data: data}
}
