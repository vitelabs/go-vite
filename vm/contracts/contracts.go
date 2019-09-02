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
	GetReceiveQuota(gasTable *util.GasTable) uint64
	// refund data at receive error
	GetRefundData(sendBlock *ledger.AccountBlock, sbHeight uint64) ([]byte, bool)
}

type builtinContract struct {
	m   map[string]BuiltinContractMethod
	abi abi.ABIContract
}

var (
	simpleContracts   = newSimpleContracts()
	dexContracts      = newDexContracts()
	dexAgentContracts = newDexAgentContracts()
	leafContracts     = newLeafContracts()
)

func newSimpleContracts() map[types.Address]*builtinContract {
	return map[types.Address]*builtinContract{
		types.AddressPledge: {
			map[string]BuiltinContractMethod{
				cabi.MethodNamePledge:       &MethodPledge{cabi.MethodNamePledge},
				cabi.MethodNameCancelPledge: &MethodCancelPledge{cabi.MethodNameCancelPledge},
			},
			cabi.ABIPledge,
		},
		types.AddressConsensusGroup: {
			map[string]BuiltinContractMethod{
				cabi.MethodNameRegister:           &MethodRegister{cabi.MethodNameRegister},
				cabi.MethodNameCancelRegister:     &MethodCancelRegister{cabi.MethodNameCancelRegister},
				cabi.MethodNameReward:             &MethodReward{cabi.MethodNameReward},
				cabi.MethodNameUpdateRegistration: &MethodUpdateRegistration{cabi.MethodNameUpdateRegistration},
				cabi.MethodNameVote:               &MethodVote{cabi.MethodNameVote},
				cabi.MethodNameCancelVote:         &MethodCancelVote{cabi.MethodNameCancelVote},
			},
			cabi.ABIConsensusGroup,
		},
		types.AddressMintage: {
			map[string]BuiltinContractMethod{
				cabi.MethodNameMint:            &MethodMint{cabi.MethodNameMint},
				cabi.MethodNameIssue:           &MethodIssue{cabi.MethodNameIssue},
				cabi.MethodNameBurn:            &MethodBurn{cabi.MethodNameBurn},
				cabi.MethodNameTransferOwner:   &MethodTransferOwner{cabi.MethodNameTransferOwner},
				cabi.MethodNameChangeTokenType: &MethodChangeTokenType{cabi.MethodNameChangeTokenType},
			},
			cabi.ABIMintage,
		},
	}
}
func newDexContracts() map[types.Address]*builtinContract {
	contracts := newSimpleContracts()
	contracts[types.AddressPledge].m[cabi.MethodNameAgentPledge] = &MethodAgentPledge{cabi.MethodNameAgentPledge}
	contracts[types.AddressPledge].m[cabi.MethodNameAgentCancelPledge] = &MethodAgentCancelPledge{cabi.MethodNameAgentCancelPledge}
	contracts[types.AddressMintage].m[cabi.MethodNameGetTokenInfo] = &MethodGetTokenInfo{cabi.MethodNameGetTokenInfo}
	contracts[types.AddressDexFund] = &builtinContract{
		map[string]BuiltinContractMethod{
			cabi.MethodNameDexFundUserDeposit:          &MethodDexFundUserDeposit{cabi.MethodNameDexFundUserDeposit},
			cabi.MethodNameDexFundUserWithdraw:         &MethodDexFundUserWithdraw{cabi.MethodNameDexFundUserWithdraw},
			cabi.MethodNameDexFundNewMarket:            &MethodDexFundNewMarket{cabi.MethodNameDexFundNewMarket},
			cabi.MethodNameDexFundNewOrder:             &MethodDexFundNewOrder{cabi.MethodNameDexFundNewOrder},
			cabi.MethodNameDexFundSettleOrders:         &MethodDexFundSettleOrders{cabi.MethodNameDexFundSettleOrders},
			cabi.MethodNameDexFundPeriodJob:            &MethodDexFundPeriodJob{cabi.MethodNameDexFundPeriodJob},
			cabi.MethodNameDexFundPledgeForVx:          &MethodDexFundPledgeForVx{cabi.MethodNameDexFundPledgeForVx},
			cabi.MethodNameDexFundPledgeForVip:         &MethodDexFundPledgeForVip{cabi.MethodNameDexFundPledgeForVip},
			cabi.MethodNameDexFundPledgeCallback:       &MethodDexFundPledgeCallback{cabi.MethodNameDexFundPledgeCallback},
			cabi.MethodNameDexFundCancelPledgeCallback: &MethodDexFundCancelPledgeCallback{cabi.MethodNameDexFundCancelPledgeCallback},
			cabi.MethodNameDexFundGetTokenInfoCallback: &MethodDexFundGetTokenInfoCallback{cabi.MethodNameDexFundGetTokenInfoCallback},
			cabi.MethodNameDexFundOwnerConfig:          &MethodDexFundOwnerConfig{cabi.MethodNameDexFundOwnerConfig},
			cabi.MethodNameDexFundOwnerConfigTrade:     &MethodDexFundOwnerConfigTrade{cabi.MethodNameDexFundOwnerConfigTrade},
			cabi.MethodNameDexFundMarketOwnerConfig:    &MethodDexFundMarketOwnerConfig{cabi.MethodNameDexFundMarketOwnerConfig},
			cabi.MethodNameDexFundTransferTokenOwner:   &MethodDexFundTransferTokenOwner{cabi.MethodNameDexFundTransferTokenOwner},
			cabi.MethodNameDexFundNotifyTime:           &MethodDexFundNotifyTime{cabi.MethodNameDexFundNotifyTime},
			cabi.MethodNameDexFundNewInviter:           &MethodDexFundNewInviter{cabi.MethodNameDexFundNewInviter},
			cabi.MethodNameDexFundBindInviteCode:       &MethodDexFundBindInviteCode{cabi.MethodNameDexFundBindInviteCode},
			cabi.MethodNameDexFundEndorseVxMinePool:    &MethodDexFundEndorseVxMinePool{cabi.MethodNameDexFundEndorseVxMinePool},
			cabi.MethodNameDexFundSettleMakerMinedVx:   &MethodDexFundSettleMakerMinedVx{cabi.MethodNameDexFundSettleMakerMinedVx},
		},
		cabi.ABIDexFund,
	}
	contracts[types.AddressDexTrade] = &builtinContract{
		map[string]BuiltinContractMethod{
			cabi.MethodNameDexTradeNewOrder:          &MethodDexTradeNewOrder{cabi.MethodNameDexTradeNewOrder},
			cabi.MethodNameDexTradeCancelOrder:       &MethodDexTradeCancelOrder{cabi.MethodNameDexTradeCancelOrder},
			cabi.MethodNameDexTradeNotifyNewMarket:   &MethodDexTradeNotifyNewMarket{cabi.MethodNameDexTradeNotifyNewMarket},
			cabi.MethodNameDexTradeCleanExpireOrders: &MethodDexTradeCleanExpireOrders{cabi.MethodNameDexTradeCleanExpireOrders},
		},
		cabi.ABIDexTrade,
	}
	return contracts
}

func newDexAgentContracts() map[types.Address]*builtinContract {
	contracts := newDexContracts()
	contracts[types.AddressDexFund].m[cabi.MethodNameDexFundPledgeForSuperVip] = &MethodDexFundPledgeForSuperVip{cabi.MethodNameDexFundPledgeForSuperVip}
	contracts[types.AddressDexFund].m[cabi.MethodNameDexFundConfigMarketsAgent] = &MethodDexFundConfigMarketsAgent{cabi.MethodNameDexFundConfigMarketsAgent}
	contracts[types.AddressDexFund].m[cabi.MethodNameDexFundNewAgentOrder] = &MethodDexFundNewAgentOrder{cabi.MethodNameDexFundNewAgentOrder}
	contracts[types.AddressDexTrade].m[cabi.MethodNameDexTradeCancelOrderByHash] = &MethodDexTradeCancelOrderByHash{cabi.MethodNameDexTradeCancelOrderByHash}
	return contracts
}

func newLeafContracts() map[types.Address]*builtinContract {
	contracts := newDexAgentContracts()

	contracts[types.AddressPledge].m[cabi.MethodNamePledgeV2] = &MethodPledge{cabi.MethodNamePledgeV2}
	contracts[types.AddressPledge].m[cabi.MethodNameCancelPledgeV2] = &MethodCancelPledge{cabi.MethodNameCancelPledgeV2}
	contracts[types.AddressPledge].m[cabi.MethodNameAgentPledgeV2] = &MethodAgentPledge{cabi.MethodNameAgentPledgeV2}
	contracts[types.AddressPledge].m[cabi.MethodNameAgentCancelPledgeV2] = &MethodAgentCancelPledge{cabi.MethodNameAgentCancelPledgeV2}

	contracts[types.AddressConsensusGroup].m[cabi.MethodNameUpdateRegistrationV2] = &MethodCancelRegister{cabi.MethodNameUpdateRegistrationV2}
	contracts[types.AddressConsensusGroup].m[cabi.MethodNameCancelRegisterV2] = &MethodCancelRegister{cabi.MethodNameCancelRegisterV2}
	contracts[types.AddressConsensusGroup].m[cabi.MethodNameRewardV2] = &MethodReward{cabi.MethodNameRewardV2}

	contracts[types.AddressMintage].m[cabi.MethodNameMintV2] = &MethodMint{cabi.MethodNameMintV2}
	contracts[types.AddressMintage].m[cabi.MethodNameIssueV2] = &MethodIssue{cabi.MethodNameIssueV2}
	contracts[types.AddressMintage].m[cabi.MethodNameChangeTokenTypeV2] = &MethodIssue{cabi.MethodNameChangeTokenTypeV2}
	contracts[types.AddressMintage].m[cabi.MethodNameTransferOwnerV2] = &MethodTransferOwner{cabi.MethodNameTransferOwnerV2}
	return contracts
}

func GetBuiltinContractMethod(addr types.Address, methodSelector []byte, sbHeight uint64) (BuiltinContractMethod, bool, error) {
	var contractsMap map[types.Address]*builtinContract
	if fork.IsLeafFork(sbHeight) {
		contractsMap = leafContracts
	} else if fork.IsStemFork(sbHeight) {
		contractsMap = dexAgentContracts
	} else if fork.IsDexFork(sbHeight) {
		contractsMap = dexContracts
	} else {
		contractsMap = simpleContracts
	}
	p, addrExists := contractsMap[addr]
	if addrExists {
		if method, err := p.abi.MethodById(methodSelector); err == nil {
			c, methodExists := p.m[method.Name]
			if methodExists {
				return c, methodExists, nil
			}
		}
		return nil, addrExists, util.ErrAbiMethodNotFound
	}
	return nil, addrExists, nil
}
