package contracts

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
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

type SendBlock struct {
	Block     *ledger.AccountBlock
	ToAddress types.Address
	BlockType byte
	Amount    *big.Int
	TokenId   types.TokenTypeId
	Data      []byte
}

type PrecompiledContractMethod interface {
	GetFee(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock) (*big.Int, error)
	// calc and use quota, check tx data
	DoSend(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, quotaLeft uint64) (uint64, error)
	// check status, update state
	DoReceive(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock) ([]*SendBlock, error)
	// refund data at receive error
	GetRefundData() []byte
	GetQuota() uint64
}
