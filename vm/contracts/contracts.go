package contracts

import (
	"errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/abi"
	"github.com/vitelabs/go-vite/vm_context"
	"github.com/vitelabs/go-vite/vm_context/vmctxt_interface"
	"math/big"
)

type NodeConfig struct {
	params ContractsParams
}

var nodeConfig NodeConfig

func InitContractsConfig(isTestParam bool) {
	if isTestParam {
		nodeConfig.params = ContractsParamsTest
	} else {
		nodeConfig.params = ContractsParamsMainNet
	}
}

var (
	AddressRegister, _       = types.BytesToAddress([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1})
	AddressVote, _           = types.BytesToAddress([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2})
	AddressPledge, _         = types.BytesToAddress([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 3})
	AddressConsensusGroup, _ = types.BytesToAddress([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4})
	AddressMintage, _        = types.BytesToAddress([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 5})
)

type contractsContext interface {
	AppendBlock(block *vm_context.VmAccountBlock)
	GetNewBlockHeight(block *vm_context.VmAccountBlock) uint64
}

type PrecompiledContractMethod interface {
	GetFee(context contractsContext, block *vm_context.VmAccountBlock) (*big.Int, error)
	// calc and use quota, check tx data
	DoSend(context contractsContext, block *vm_context.VmAccountBlock, quotaLeft uint64) (uint64, error)
	// check status, update state
	DoReceive(context contractsContext, block *vm_context.VmAccountBlock, sendBlock *ledger.AccountBlock) error
}

var (
	precompiledContractsAbiMap = map[types.Address]abi.ABIContract{
		AddressRegister:       ABIRegister,
		AddressVote:           ABIVote,
		AddressPledge:         ABIPledge,
		AddressConsensusGroup: ABIConsensusGroup,
		AddressMintage:        ABIMintage,
	}

	errInvalidParam = errors.New("invalid param")
)

type StorageDatabase interface {
	GetStorage(addr *types.Address, key []byte) []byte
	NewStorageIterator(addr *types.Address, prefix []byte) vmctxt_interface.StorageIterator
}
