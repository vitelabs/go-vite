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
	params contractsParams
}

var nodeConfig NodeConfig

// InitContractsConfig inits params of built-in contracts.
func InitContractsConfig(isTestParam bool) {
	if isTestParam {
		nodeConfig.params = contractsParamsTest
	} else {
		nodeConfig.params = contractsParamsMainNet
	}
}

type vmEnvironment interface {
	GlobalStatus() util.GlobalStatus
	ConsensusReader() util.ConsensusReader
}

// BuiltinContractMethod defines interfaces of built-in contract method
type BuiltinContractMethod interface {
	GetFee(block *ledger.AccountBlock) (*big.Int, error)
	// calc and use quota, check tx data
	DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error
	// quota for doSend block
	GetSendQuota(data []byte, gasTable *util.QuotaTable) (uint64, error)
	// check status, update state
	DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error)
	// receive block quota
	GetReceiveQuota(gasTable *util.QuotaTable) uint64
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
		types.AddressQuota: {
			map[string]BuiltinContractMethod{
				cabi.MethodNameStake:       &MethodPledge{cabi.MethodNameStake},
				cabi.MethodNameCancelStake: &MethodCancelPledge{cabi.MethodNameCancelStake},
			},
			cabi.ABIQuota,
		},
		types.AddressGovernance: {
			map[string]BuiltinContractMethod{
				cabi.MethodNameRegister:                    &MethodRegister{cabi.MethodNameRegister},
				cabi.MethodNameRevoke:                      &MethodCancelRegister{cabi.MethodNameRevoke},
				cabi.MethodNameWithdrawReward:              &MethodReward{cabi.MethodNameWithdrawReward},
				cabi.MethodNameUpdateBlockProducingAddress: &MethodUpdateRegistration{cabi.MethodNameUpdateBlockProducingAddress},
				cabi.MethodNameVote:                        &MethodVote{cabi.MethodNameVote},
				cabi.MethodNameCancelVote:                  &MethodCancelVote{cabi.MethodNameCancelVote},
			},
			cabi.ABIGovernance,
		},
		types.AddressAssert: {
			map[string]BuiltinContractMethod{
				cabi.MethodNameIssue:             &MethodMint{cabi.MethodNameIssue},
				cabi.MethodNameReIssue:           &MethodIssue{cabi.MethodNameReIssue},
				cabi.MethodNameBurn:              &MethodBurn{cabi.MethodNameBurn},
				cabi.MethodNameTransferOwnership: &MethodTransferOwner{cabi.MethodNameTransferOwnership},
				cabi.MethodNameDisableReIssue:    &MethodChangeTokenType{cabi.MethodNameDisableReIssue},
			},
			cabi.ABIAssert,
		},
	}
}
func newDexContracts() map[types.Address]*builtinContract {
	contracts := newSimpleContracts()
	contracts[types.AddressQuota].m[cabi.MethodNameDelegateStake] = &MethodAgentPledge{cabi.MethodNameDelegateStake}
	contracts[types.AddressQuota].m[cabi.MethodNameCancelDelegateStake] = &MethodAgentCancelPledge{cabi.MethodNameCancelDelegateStake}
	contracts[types.AddressAssert].m[cabi.MethodNameGetTokenInfo] = &MethodGetTokenInfo{cabi.MethodNameGetTokenInfo}
	contracts[types.AddressDexFund] = &builtinContract{
		map[string]BuiltinContractMethod{
			cabi.MethodNameDexFundUserDeposit:          &MethodDexFundDeposit{cabi.MethodNameDexFundUserDeposit},
			cabi.MethodNameDexFundUserWithdraw:         &MethodDexFundWithdraw{cabi.MethodNameDexFundUserWithdraw},
			cabi.MethodNameDexFundNewMarket:            &MethodDexFundOpenNewMarket{cabi.MethodNameDexFundNewMarket},
			cabi.MethodNameDexFundNewOrder:             &MethodDexFundPlaceOrder{cabi.MethodNameDexFundNewOrder},
			cabi.MethodNameDexFundSettleOrders:         &MethodDexFundSettleOrders{cabi.MethodNameDexFundSettleOrders},
			cabi.MethodNameDexFundPeriodJob:            &MethodDexFundTriggerPeriodJob{cabi.MethodNameDexFundPeriodJob},
			cabi.MethodNameDexFundPledgeForVx:          &MethodDexFundStakeForMining{cabi.MethodNameDexFundPledgeForVx},
			cabi.MethodNameDexFundPledgeForVip:         &MethodDexFundStakeForVIP{cabi.MethodNameDexFundPledgeForVip},
			cabi.MethodNameDexFundPledgeCallback:       &MethodDexFundDelegateStakeCallback{cabi.MethodNameDexFundPledgeCallback},
			cabi.MethodNameDexFundCancelPledgeCallback: &MethodDexFundCancelDelegateStakeCallback{cabi.MethodNameDexFundCancelPledgeCallback},
			cabi.MethodNameDexFundGetTokenInfoCallback: &MethodDexFundGetTokenInfoCallback{cabi.MethodNameDexFundGetTokenInfoCallback},
			cabi.MethodNameDexFundOwnerConfig:          &MethodDexFundDexAdminConfig{cabi.MethodNameDexFundOwnerConfig},
			cabi.MethodNameDexFundOwnerConfigTrade:     &MethodDexFundTradeAdminConfig{cabi.MethodNameDexFundOwnerConfigTrade},
			cabi.MethodNameDexFundMarketOwnerConfig:    &MethodDexFundMarketAdminConfig{cabi.MethodNameDexFundMarketOwnerConfig},
			cabi.MethodNameDexFundTransferTokenOwner:   &MethodDexFundTransferTokenOwnership{cabi.MethodNameDexFundTransferTokenOwner},
			cabi.MethodNameDexFundNotifyTime:           &MethodDexFundNotifyTime{cabi.MethodNameDexFundNotifyTime},
			cabi.MethodNameDexFundNewInviter:           &MethodDexFundCreateNewInviter{cabi.MethodNameDexFundNewInviter},
			cabi.MethodNameDexFundBindInviteCode:       &MethodDexFundBindInviteCode{cabi.MethodNameDexFundBindInviteCode},
			cabi.MethodNameDexFundEndorseVxMinePool:    &MethodDexFundEndorseVx{cabi.MethodNameDexFundEndorseVxMinePool},
			cabi.MethodNameDexFundSettleMakerMinedVx:   &MethodDexFundSettleMakerMinedVx{cabi.MethodNameDexFundSettleMakerMinedVx},
		},
		cabi.ABIDexFund,
	}
	contracts[types.AddressDexTrade] = &builtinContract{
		map[string]BuiltinContractMethod{
			cabi.MethodNameDexTradeNewOrder:          &MethodDexTradePlaceOrder{cabi.MethodNameDexTradeNewOrder},
			cabi.MethodNameDexTradeCancelOrder:       &MethodDexTradeCancelOrder{cabi.MethodNameDexTradeCancelOrder},
			cabi.MethodNameDexTradeNotifyNewMarket:   &MethodDexTradeSyncNewMarket{cabi.MethodNameDexTradeNotifyNewMarket},
			cabi.MethodNameDexTradeCleanExpireOrders: &MethodDexTradeClearExpiredOrders{cabi.MethodNameDexTradeCleanExpireOrders},
		},
		cabi.ABIDexTrade,
	}
	return contracts
}

func newDexAgentContracts() map[types.Address]*builtinContract {
	contracts := newDexContracts()
	contracts[types.AddressDexFund].m[cabi.MethodNameDexFundStakeForSuperVip] = &MethodDexFundStakeForSVIP{cabi.MethodNameDexFundStakeForSuperVip}
	contracts[types.AddressDexFund].m[cabi.MethodNameDexFundConfigMarketsAgent] = &MethodDexFundConfigMarketAgents{cabi.MethodNameDexFundConfigMarketsAgent}
	contracts[types.AddressDexFund].m[cabi.MethodNameDexFundNewAgentOrder] = &MethodDexFundPlaceAgentOrder{cabi.MethodNameDexFundNewAgentOrder}
	contracts[types.AddressDexTrade].m[cabi.MethodNameDexTradeCancelOrderByHash] = &MethodDexTradeCancelOrderByTransactionHash{cabi.MethodNameDexTradeCancelOrderByHash}
	return contracts
}

func newLeafContracts() map[types.Address]*builtinContract {
	contracts := newDexAgentContracts()

	contracts[types.AddressQuota].m[cabi.MethodNameStakeV2] = &MethodPledge{cabi.MethodNameStakeV2}
	contracts[types.AddressQuota].m[cabi.MethodNameCancelStakeV2] = &MethodCancelPledge{cabi.MethodNameCancelStakeV2}
	contracts[types.AddressQuota].m[cabi.MethodNameDelegateStakeV2] = &MethodAgentPledge{cabi.MethodNameDelegateStakeV2}
	contracts[types.AddressQuota].m[cabi.MethodNameCancelDelegateStakeV2] = &MethodAgentCancelPledge{cabi.MethodNameCancelDelegateStakeV2}

	contracts[types.AddressGovernance].m[cabi.MethodNameUpdateBlockProducintAddressV2] = &MethodUpdateRegistration{cabi.MethodNameUpdateBlockProducintAddressV2}
	contracts[types.AddressGovernance].m[cabi.MethodNameRevokeV2] = &MethodCancelRegister{cabi.MethodNameRevokeV2}
	contracts[types.AddressGovernance].m[cabi.MethodNameWithdrawRewardV2] = &MethodReward{cabi.MethodNameWithdrawRewardV2}

	contracts[types.AddressAssert].m[cabi.MethodNameIssueV2] = &MethodMint{cabi.MethodNameIssueV2}
	contracts[types.AddressAssert].m[cabi.MethodNameReIssueV2] = &MethodIssue{cabi.MethodNameReIssueV2}
	contracts[types.AddressAssert].m[cabi.MethodNameDisableReIssueV2] = &MethodChangeTokenType{cabi.MethodNameDisableReIssueV2}
	contracts[types.AddressAssert].m[cabi.MethodNameTransferOwnershipV2] = &MethodTransferOwner{cabi.MethodNameTransferOwnershipV2}

	contracts[types.AddressDexFund].m[cabi.MethodNameDexFundDeposit] = &MethodDexFundDeposit{cabi.MethodNameDexFundDeposit}
	contracts[types.AddressDexFund].m[cabi.MethodNameDexFundWithdraw] = &MethodDexFundWithdraw{cabi.MethodNameDexFundWithdraw}
	contracts[types.AddressDexFund].m[cabi.MethodNameDexFundOpenNewMarket] = &MethodDexFundOpenNewMarket{cabi.MethodNameDexFundOpenNewMarket}
	contracts[types.AddressDexFund].m[cabi.MethodNameDexFundPlaceOrder] = &MethodDexFundPlaceOrder{cabi.MethodNameDexFundPlaceOrder}
	contracts[types.AddressDexFund].m[cabi.MethodNameDexFundSettleOrdersV2] = &MethodDexFundSettleOrders{cabi.MethodNameDexFundSettleOrdersV2}
	contracts[types.AddressDexFund].m[cabi.MethodNameDexFundTriggerPeriodJob] = &MethodDexFundTriggerPeriodJob{cabi.MethodNameDexFundTriggerPeriodJob}
	contracts[types.AddressDexFund].m[cabi.MethodNameDexFundStakeForMining] = &MethodDexFundStakeForMining{cabi.MethodNameDexFundStakeForMining}
	contracts[types.AddressDexFund].m[cabi.MethodNameDexFundStakeForVIP] = &MethodDexFundStakeForVIP{cabi.MethodNameDexFundStakeForVIP}
	contracts[types.AddressDexFund].m[cabi.MethodNameDexFundDelegateStakeCallback] = &MethodDexFundDelegateStakeCallback{cabi.MethodNameDexFundDelegateStakeCallback}
	contracts[types.AddressDexFund].m[cabi.MethodNameDexFundCancelDelegateStakeCallback] = &MethodDexFundCancelDelegateStakeCallback{cabi.MethodNameDexFundCancelDelegateStakeCallback}
	contracts[types.AddressDexFund].m[cabi.MethodNameDexFundDexAdminConfig] = &MethodDexFundDexAdminConfig{cabi.MethodNameDexFundDexAdminConfig}
	contracts[types.AddressDexFund].m[cabi.MethodNameDexFundTradeAdminConfig] = &MethodDexFundTradeAdminConfig{cabi.MethodNameDexFundTradeAdminConfig}
	contracts[types.AddressDexFund].m[cabi.MethodNameDexFundMarketAdminConfig] = &MethodDexFundMarketAdminConfig{cabi.MethodNameDexFundMarketAdminConfig}
	contracts[types.AddressDexFund].m[cabi.MethodNameDexFundTransferTokenOwnership] = &MethodDexFundTransferTokenOwnership{cabi.MethodNameDexFundTransferTokenOwnership}
	contracts[types.AddressDexFund].m[cabi.MethodNameDexFundCreateNewInviter] = &MethodDexFundCreateNewInviter{cabi.MethodNameDexFundCreateNewInviter}
	contracts[types.AddressDexFund].m[cabi.MethodNameDexFundBindInviteCodeV2] = &MethodDexFundBindInviteCode{cabi.MethodNameDexFundBindInviteCodeV2}
	contracts[types.AddressDexFund].m[cabi.MethodNameDexFundEndorseVxV2] = &MethodDexFundEndorseVx{cabi.MethodNameDexFundEndorseVxV2}
	contracts[types.AddressDexFund].m[cabi.MethodNameDexFundSettleMakerMinedVxV2] = &MethodDexFundSettleMakerMinedVx{cabi.MethodNameDexFundSettleMakerMinedVxV2}

	contracts[types.AddressDexTrade].m[cabi.MethodNameDexTradePlaceOrder] = &MethodDexTradePlaceOrder{cabi.MethodNameDexTradePlaceOrder}
	contracts[types.AddressDexTrade].m[cabi.MethodNameDexTradeCancelOrderV2] = &MethodDexTradeCancelOrder{cabi.MethodNameDexTradeCancelOrderV2}
	contracts[types.AddressDexTrade].m[cabi.MethodNameDexTradeSyncNewMarket] = &MethodDexTradeSyncNewMarket{cabi.MethodNameDexTradeSyncNewMarket}
	contracts[types.AddressDexTrade].m[cabi.MethodNameDexTradeClearExpiredOrders] = &MethodDexTradeClearExpiredOrders{cabi.MethodNameDexTradeClearExpiredOrders}

	contracts[types.AddressDexFund].m[cabi.MethodNameDexFundStakeForSVIP] = &MethodDexFundStakeForSVIP{cabi.MethodNameDexFundStakeForSVIP}
	contracts[types.AddressDexFund].m[cabi.MethodNameDexFundConfigMarketAgents] = &MethodDexFundConfigMarketAgents{cabi.MethodNameDexFundConfigMarketAgents}
	contracts[types.AddressDexFund].m[cabi.MethodNameDexFundPlaceAgentOrder] = &MethodDexFundPlaceAgentOrder{cabi.MethodNameDexFundPlaceAgentOrder}
	contracts[types.AddressDexTrade].m[cabi.MethodNameDexTradeCancelOrderByTransactionHash] = &MethodDexTradeCancelOrderByTransactionHash{cabi.MethodNameDexTradeCancelOrderByTransactionHash}

	return contracts
}

// GetBuiltinContractMethod finds method instance of built-in contract method by address and method id
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
