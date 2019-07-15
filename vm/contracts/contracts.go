package contracts

import (
	"github.com/vitelabs/go-vite/common/fork"
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
	GetSendQuota(data []byte, gasTable *util.GasTable) (uint64, error)
	// check status, update state
	DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error)
	// receive block quota
	GetReceiveQuota() uint64
	// refund data at receive error
	GetRefundData(sendBlock *ledger.AccountBlock) ([]byte, bool)
}

type builtinContract struct {
	m   map[string]BuiltinContractMethod
	abi abi.ABIContract
}

var (
	simpleContracts = newSimpleContracts()
	dexContracts    = newDexContracts()
)

func newSimpleContracts() map[types.Address]*builtinContract {
	return map[types.Address]*builtinContract{
		types.AddressPledge: {
			map[string]BuiltinContractMethod{
				cabi.MethodNamePledge:       &MethodPledge{},
				cabi.MethodNameCancelPledge: &MethodCancelPledge{},
			},
			cabi.ABIPledge,
		},
		types.AddressConsensusGroup: {
			map[string]BuiltinContractMethod{
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
			},
			cabi.ABIMintage,
		},
	}
}
func newDexContracts() map[types.Address]*builtinContract {
	contracts := newSimpleContracts()
	contracts[types.AddressPledge].m[cabi.MethodNameAgentPledge] = &MethodAgentPledge{}
	contracts[types.AddressPledge].m[cabi.MethodNameAgentCancelPledge] = &MethodAgentCancelPledge{}
	contracts[types.AddressDexFund] = &builtinContract{
		map[string]BuiltinContractMethod{
			cabi.MethodNameDexFundUserDeposit:          &MethodDexFundUserDeposit{},
			cabi.MethodNameDexFundUserWithdraw:         &MethodDexFundUserWithdraw{},
			cabi.MethodNameDexFundNewMarket:            &MethodDexFundNewMarket{},
			cabi.MethodNameDexFundNewOrder:             &MethodDexFundNewOrder{},
			cabi.MethodNameDexFundSettleOrders:         &MethodDexFundSettleOrders{},
			cabi.MethodNameDexFundPeriodJob:            &MethodDexFundPeriodJob{},
			cabi.MethodNameDexFundPledgeForVx:          &MethodDexFundPledgeForVx{},
			cabi.MethodNameDexFundPledgeForVip:         &MethodDexFundPledgeForVip{},
			cabi.MethodNameDexFundPledgeCallback:       &MethodDexFundPledgeCallback{},
			cabi.MethodNameDexFundCancelPledgeCallback: &MethodDexFundCancelPledgeCallback{},
			cabi.MethodNameDexFundGetTokenInfoCallback: &MethodDexFundGetTokenInfoCallback{},
			cabi.MethodNameDexFundOwnerConfig:          &MethodDexFundOwnerConfig{},
			cabi.MethodNameDexFundOwnerConfigTrade:     &MethodDexFundOwnerConfigTrade{},
			cabi.MethodNameDexFundMarketOwnerConfig:    &MethodDexFundMarketOwnerConfig{},
			cabi.MethodNameDexFundTransferTokenOwner:   &MethodDexFundTransferTokenOwner{},
			cabi.MethodNameDexFundNotifyTime:           &MethodDexFundNotifyTime{},
			cabi.MethodNameDexFundNewInviter:           &MethodDexFundNewInviter{},
			cabi.MethodNameDexFundBindInviteCode:       &MethodDexFundBindInviteCode{},
			cabi.MethodNameDexFundEndorseVxMinePool:    &MethodDexFundEndorseVxMinePool{},
			cabi.MethodNameDexFunSettleMakerMinedVx:    &MethodDexFundSettleMakerMinedVx{},
		},
		cabi.ABIDexFund,
	}
	contracts[types.AddressDexTrade] = &builtinContract{
		map[string]BuiltinContractMethod{
			cabi.MethodNameDexTradeNewOrder:          &MethodDexTradeNewOrder{},
			cabi.MethodNameDexTradeCancelOrder:       &MethodDexTradeCancelOrder{},
			cabi.MethodNameDexTradeNotifyNewMarket:   &MethodDexTradeNotifyNewMarket{},
			cabi.MethodNameDexTradeCleanExpireOrders: &MethodDexTradeCleanExpireOrders{},
		},
		cabi.ABIDexTrade,
	}
	return contracts
}

func GetBuiltinContractMethod(addr types.Address, methodSelector []byte, sbHeight uint64) (BuiltinContractMethod, bool, error) {
	var contractsMap map[types.Address]*builtinContract
	if !fork.IsDexFork(sbHeight) {
		contractsMap = simpleContracts
	} else {
		contractsMap = dexContracts
	}
	p, ok := contractsMap[addr]
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
