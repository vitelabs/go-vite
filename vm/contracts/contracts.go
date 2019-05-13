package contracts

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/abi"
	cabi "github.com/vitelabs/go-vite/vm/contracts/abi"
	"github.com/vitelabs/go-vite/vm/util"
	"github.com/vitelabs/go-vite/vm_db"
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

type vmEnvironment interface {
	GlobalStatus() util.GlobalStatus
	ConsensusReader() util.ConsensusReader
}

type BuiltinContractMethod interface {
	GetFee(block *ledger.AccountBlock) (*big.Int, error)
	// calc and use quota, check tx data
	DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error
	// quota for doSend block
	GetSendQuota(data []byte) (uint64, error)
	// check status, update state
	DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error)
	// receive block quota
	GetReceiveQuota() uint64
	// refund data at receive error
	GetRefundData() ([]byte, bool)
}

type builtinContract struct {
	m   map[string]BuiltinContractMethod
	abi abi.ABIContract
}

var simpleContracts = map[types.Address]*builtinContract{
	types.AddressPledge: {
		map[string]BuiltinContractMethod{
			cabi.MethodNamePledge:            &MethodPledge{},
			cabi.MethodNameCancelPledge:      &MethodCancelPledge{},
			cabi.MethodNameAgentPledge:       &MethodAgentPledge{},
			cabi.MethodNameAgentCancelPledge: &MethodAgentCancelPledge{},
		},
		cabi.ABIPledge,
	},
	types.AddressConsensusGroup: {
		map[string]BuiltinContractMethod{
			/*MethodNameCreateConsensusGroup:   &MethodCreateConsensusGroup{},
			MethodNameCancelConsensusGroup:   &MethodCancelConsensusGroup{},
			MethodNameReCreateConsensusGroup: &MethodReCreateConsensusGroup{},*/
			cabi.MethodNameRegister:           &MethodRegister{},
			cabi.MethodNameCancelRegister:     &MethodCancelRegister{},
			cabi.MethodNameReward:             &MethodReward{},
			cabi.MethodNameUpdateRegistration: &MethodUpdateRegistration{},
			cabi.MethodNameVote:               &MethodVote{},
			cabi.MethodNameCancelVote:         &MethodCancelVote{},
		},
		cabi.ABIConsensusGroup,
	},
	types.AddressMintage: {
		map[string]BuiltinContractMethod{
			cabi.MethodNameMint:            &MethodMint{},
			cabi.MethodNameIssue:           &MethodIssue{},
			cabi.MethodNameBurn:            &MethodBurn{},
			cabi.MethodNameTransferOwner:   &MethodTransferOwner{},
			cabi.MethodNameChangeTokenType: &MethodChangeTokenType{},
			cabi.MethodNameGetTokenInfo:    &MethodGetTokenInfo{},
		},
		cabi.ABIMintage,
	},
	types.AddressDexFund: {
		map[string]BuiltinContractMethod{
			MethodNameDexFundUserDeposit:          &MethodDexFundUserDeposit{},
			MethodNameDexFundUserWithdraw:         &MethodDexFundUserWithdraw{},
			MethodNameDexFundNewOrder:             &MethodDexFundNewOrder{},
			MethodNameDexFundSettleOrders:         &MethodDexFundSettleOrders{},
			MethodNameDexFundFeeDividend:          &MethodDexFundFeeDividend{},
			MethodNameDexFundMinedVxDividend:      &MethodDexFundMinedVxDividend{},
			MethodNameDexFundNewMarket:            &MethodDexFundNewMarket{},
			MethodNameDexFundSetOwner:             &MethodDexFundSetOwner{},
			MethodNameDexFundConfigMineMarket:     &MethodDexFundConfigMineMarket{},
			MethodNameDexFundPledgeForVx:          &MethodDexFundPledgeForVx{},
			MethodNameDexFundPledgeForVip:         &MethodDexFundPledgeForVip{},
			MethodNameDexFundPledgeCallback:       &MethodDexFundPledgeCallback{},
			MethodNameDexFundCancelPledgeCallback: &MethodDexFundCancelPledgeCallback{},
			MethodNameDexFundGetTokenInfoCallback: &MethodDexFundGetTokenInfoCallback{},
			MethodNameDexFundConfigTimerAddress:   &MethodDexFundConfigTimerAddress{},
			MethodNameDexFundNotifyTime:           &MethodDexFundNotifyTime{},
		},
		ABIDexFund,
	},
	types.AddressDexTrade: {
		map[string]BuiltinContractMethod{
			MethodNameDexTradeNewOrder:    &MethodDexTradeNewOrder{},
			MethodNameDexTradeCancelOrder: &MethodDexTradeCancelOrder{},
		},
		ABIDexTrade,
	},
}

func GetBuiltinContract(addr types.Address, methodSelector []byte) (BuiltinContractMethod, bool, error) {
	p, ok := simpleContracts[addr]
	if ok {
		if method, err := p.abi.MethodById(methodSelector); err == nil {
			c, ok := p.m[method.Name]
			return c, ok, nil
		} else {
			return nil, ok, util.ErrAbiMethodNotFound
		}
	}
	return nil, ok, nil
}

func GetOriginSendBlock(db vm_db.VmDb, sendBlockHash types.Hash) (*ledger.AccountBlock, error) {
	receiveBlock, err := db.GetCompleteBlockByHash(sendBlockHash)
	if err != nil {
		return nil, err
	}
	if receiveBlock == nil {
		return nil, util.ErrChainForked
	}
	sendBlock, err := db.GetAccountBlockByHash(receiveBlock.FromBlockHash)
	if err != nil {
		return nil, err
	}
	if sendBlock == nil {
		return nil, util.ErrChainForked
	}
	return sendBlock, nil
}
