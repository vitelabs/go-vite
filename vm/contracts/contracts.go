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

type nodeConfigParams struct {
	params contractsParams
}

var nodeConfig nodeConfigParams

// InitContractsConfig init params of built-in contracts.This method is
// supposed be called when the node started.
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
	earthContracts    = newEarthContracts()
)

func newSimpleContracts() map[types.Address]*builtinContract {
	return map[types.Address]*builtinContract{
		types.AddressQuota: {
			map[string]BuiltinContractMethod{
				cabi.MethodNameStake:       &MethodStake{cabi.MethodNameStake},
				cabi.MethodNameCancelStake: &MethodCancelStake{cabi.MethodNameCancelStake},
			},
			cabi.ABIQuota,
		},
		types.AddressGovernance: {
			map[string]BuiltinContractMethod{
				cabi.MethodNameRegister:                    &MethodRegister{cabi.MethodNameRegister},
				cabi.MethodNameRevoke:                      &MethodRevoke{cabi.MethodNameRevoke},
				cabi.MethodNameWithdrawReward:              &MethodWithdrawReward{cabi.MethodNameWithdrawReward},
				cabi.MethodNameUpdateBlockProducingAddress: &MethodUpdateBlockProducingAddress{cabi.MethodNameUpdateBlockProducingAddress},
				cabi.MethodNameVote:                        &MethodVote{cabi.MethodNameVote},
				cabi.MethodNameCancelVote:                  &MethodCancelVote{cabi.MethodNameCancelVote},
			},
			cabi.ABIGovernance,
		},
		types.AddressAsset: {
			map[string]BuiltinContractMethod{
				cabi.MethodNameIssue:             &MethodIssue{cabi.MethodNameIssue},
				cabi.MethodNameReIssue:           &MethodReIssue{cabi.MethodNameReIssue},
				cabi.MethodNameBurn:              &MethodBurn{cabi.MethodNameBurn},
				cabi.MethodNameTransferOwnership: &MethodTransferOwnership{cabi.MethodNameTransferOwnership},
				cabi.MethodNameDisableReIssue:    &MethodDisableReIssue{cabi.MethodNameDisableReIssue},
			},
			cabi.ABIAsset,
		},
	}
}
func newDexContracts() map[types.Address]*builtinContract {
	contracts := newSimpleContracts()
	contracts[types.AddressQuota].m[cabi.MethodNameDelegateStake] = &MethodDelegateStake{cabi.MethodNameDelegateStake}
	contracts[types.AddressQuota].m[cabi.MethodNameCancelDelegateStake] = &MethodCancelDelegateStake{cabi.MethodNameCancelDelegateStake}
	contracts[types.AddressAsset].m[cabi.MethodNameGetTokenInfo] = &MethodGetTokenInfo{cabi.MethodNameGetTokenInfo}
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

	contracts[types.AddressQuota].m[cabi.MethodNameStakeV2] = &MethodStake{cabi.MethodNameStakeV2}
	contracts[types.AddressQuota].m[cabi.MethodNameCancelStakeV2] = &MethodCancelStake{cabi.MethodNameCancelStakeV2}
	contracts[types.AddressQuota].m[cabi.MethodNameDelegateStakeV2] = &MethodDelegateStake{cabi.MethodNameDelegateStakeV2}
	contracts[types.AddressQuota].m[cabi.MethodNameCancelDelegateStakeV2] = &MethodCancelDelegateStake{cabi.MethodNameCancelDelegateStakeV2}

	contracts[types.AddressGovernance].m[cabi.MethodNameUpdateBlockProducintAddressV2] = &MethodUpdateBlockProducingAddress{cabi.MethodNameUpdateBlockProducintAddressV2}
	contracts[types.AddressGovernance].m[cabi.MethodNameRevokeV2] = &MethodRevoke{cabi.MethodNameRevokeV2}
	contracts[types.AddressGovernance].m[cabi.MethodNameWithdrawRewardV2] = &MethodWithdrawReward{cabi.MethodNameWithdrawRewardV2}

	contracts[types.AddressAsset].m[cabi.MethodNameIssueV2] = &MethodIssue{cabi.MethodNameIssueV2}
	contracts[types.AddressAsset].m[cabi.MethodNameReIssueV2] = &MethodReIssue{cabi.MethodNameReIssueV2}
	contracts[types.AddressAsset].m[cabi.MethodNameDisableReIssueV2] = &MethodDisableReIssue{cabi.MethodNameDisableReIssueV2}
	contracts[types.AddressAsset].m[cabi.MethodNameTransferOwnershipV2] = &MethodTransferOwnership{cabi.MethodNameTransferOwnershipV2}

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

func newEarthContracts() map[types.Address]*builtinContract {
	contracts := newLeafContracts()
	contracts[types.AddressAsset].m[cabi.MethodNameGetTokenInfoV3] = &MethodGetTokenInfo{cabi.MethodNameGetTokenInfoV3}
	contracts[types.AddressGovernance].m[cabi.MethodNameRegisterV3] = &MethodRegister{cabi.MethodNameRegisterV3}
	contracts[types.AddressGovernance].m[cabi.MethodNameUpdateBlockProducintAddressV3] = &MethodUpdateBlockProducingAddress{cabi.MethodNameUpdateBlockProducintAddressV3}
	contracts[types.AddressGovernance].m[cabi.MethodNameUpdateSBPRewardWithdrawAddress] = &MethodUpdateRewardWithdrawAddress{cabi.MethodNameUpdateSBPRewardWithdrawAddress}
	contracts[types.AddressGovernance].m[cabi.MethodNameRevokeV3] = &MethodRevoke{cabi.MethodNameRevokeV3}
	contracts[types.AddressGovernance].m[cabi.MethodNameWithdrawRewardV3] = &MethodWithdrawReward{cabi.MethodNameWithdrawRewardV3}
	contracts[types.AddressGovernance].m[cabi.MethodNameVoteV3] = &MethodVote{cabi.MethodNameVoteV3}
	contracts[types.AddressGovernance].m[cabi.MethodNameCancelVoteV3] = &MethodCancelVote{cabi.MethodNameCancelVoteV3}

	contracts[types.AddressDexFund].m[cabi.MethodNameDexFundLockVxForDividend] = &MethodDexFundLockVxForDividend{cabi.MethodNameDexFundLockVxForDividend}
	contracts[types.AddressDexFund].m[cabi.MethodNameDexFundSwitchConfig] = &MethodDexFundSwitchConfig{cabi.MethodNameDexFundSwitchConfig}
	contracts[types.AddressDexFund].m[cabi.MethodNameDexFundStakeForPrincipalSVIP] = &MethodDexFundStakeForPrincipalSVIP{cabi.MethodNameDexFundStakeForPrincipalSVIP}
	contracts[types.AddressDexFund].m[cabi.MethodNameDexFundCancelStakeById] = &MethodDexFundCancelStakeById{cabi.MethodNameDexFundCancelStakeById}
	contracts[types.AddressDexFund].m[cabi.MethodNameDexFundDelegateStakeCallbackV2] = &MethodDexFundDelegateStakeCallbackV2{cabi.MethodNameDexFundDelegateStakeCallbackV2}
	contracts[types.AddressDexFund].m[cabi.MethodNameDexFundCancelDelegateStakeCallbackV2] = &MethodDexFundCancelDelegateStakeCallbackV2{cabi.MethodNameDexFundCancelDelegateStakeCallbackV2}

	contracts[types.AddressQuota].m[cabi.MethodNameStakeV3] = &MethodStakeV3{cabi.MethodNameStakeV3}
	contracts[types.AddressQuota].m[cabi.MethodNameCancelStakeV3] = &MethodCancelStakeV3{cabi.MethodNameCancelStakeV3}
	contracts[types.AddressQuota].m[cabi.MethodNameStakeWithCallback] = &MethodStakeV3{cabi.MethodNameStakeWithCallback}
	contracts[types.AddressQuota].m[cabi.MethodNameCancelStakeWithCallback] = &MethodCancelStakeV3{cabi.MethodNameCancelStakeWithCallback}
	return contracts
}

// GetBuiltinContractMethod finds method instance of built-in contract method by address and method id
func GetBuiltinContractMethod(addr types.Address, methodSelector []byte, sbHeight uint64) (BuiltinContractMethod, bool, error) {
	var contractsMap map[types.Address]*builtinContract
	if fork.IsEarthFork(sbHeight) {
		contractsMap = earthContracts
	} else if fork.IsLeafFork(sbHeight) {
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

// NewLog generate vm log
func NewLog(c abi.ABIContract, name string, params ...interface{}) *ledger.VmLog {
	topics, data, _ := c.PackEvent(name, params...)
	return &ledger.VmLog{Topics: topics, Data: data}
}
