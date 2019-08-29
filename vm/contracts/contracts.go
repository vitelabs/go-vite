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
			cabi.MethodNameDexFundUserDeposit:          &MethodDexFundDeposit{cabi.MethodNameDexFundUserDeposit},
			cabi.MethodNameDexFundUserWithdraw:         &MethodDexFundWithdraw{cabi.MethodNameDexFundUserWithdraw},
			cabi.MethodNameDexFundNewMarket:            &MethodDexFundOpenNewMarket{cabi.MethodNameDexFundNewMarket},
			cabi.MethodNameDexFundNewOrder:             &MethodDexFundPlaceOrder{cabi.MethodNameDexFundNewOrder},
			cabi.MethodNameDexFundSettleOrders:         &MethodDexFundSettleOrders{cabi.MethodNameDexFundSettleOrders},
			cabi.MethodNameDexFundPeriodJob:            &MethodDexFundTriggerPeriodJob{cabi.MethodNameDexFundPeriodJob},
			cabi.MethodNameDexFundPledgeForVx:          &MethodDexFundStakeForMining{cabi.MethodNameDexFundPledgeForVx},
			cabi.MethodNameDexFundPledgeForVip:         &MethodDexFundStakeForVIP{cabi.MethodNameDexFundPledgeForVip},
			cabi.MethodNameDexFundPledgeCallback:       &MethodDexFundDelegateStakingCallback{cabi.MethodNameDexFundPledgeCallback},
			cabi.MethodNameDexFundCancelPledgeCallback: &MethodDexFundCancelDelegatedStakingCallback{cabi.MethodNameDexFundCancelPledgeCallback},
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

	contracts[types.AddressPledge].m[cabi.MethodNamePledgeV2] = &MethodPledge{cabi.MethodNamePledgeV2}
	contracts[types.AddressPledge].m[cabi.MethodNameCancelPledgeV2] = &MethodCancelPledge{cabi.MethodNameCancelPledgeV2}
	contracts[types.AddressPledge].m[cabi.MethodNameAgentPledgeV2] = &MethodAgentPledge{cabi.MethodNameAgentPledgeV2}
	contracts[types.AddressPledge].m[cabi.MethodNameAgentCancelPledgeV2] = &MethodAgentCancelPledge{cabi.MethodNameAgentCancelPledgeV2}

	contracts[types.AddressConsensusGroup].m[cabi.MethodNameCancelRegisterV2] = &MethodCancelRegister{cabi.MethodNameCancelRegisterV2}
	contracts[types.AddressConsensusGroup].m[cabi.MethodNameRewardV2] = &MethodReward{cabi.MethodNameRewardV2}
	contracts[types.AddressConsensusGroup].m[cabi.MethodNameCancelVoteV2] = &MethodCancelVote{cabi.MethodNameCancelVoteV2}

	contracts[types.AddressMintage].m[cabi.MethodNameMintV2] = &MethodMint{cabi.MethodNameMintV2}
	contracts[types.AddressMintage].m[cabi.MethodNameIssueV2] = &MethodIssue{cabi.MethodNameIssueV2}
	contracts[types.AddressMintage].m[cabi.MethodNameTransferOwnerV2] = &MethodTransferOwner{cabi.MethodNameTransferOwnerV2}

	contracts[types.AddressDexFund].m[cabi.MethodNameDexFundDeposit] = &MethodDexFundDeposit{cabi.MethodNameDexFundDeposit}
	contracts[types.AddressDexFund].m[cabi.MethodNameDexFundWithdraw] = &MethodDexFundWithdraw{cabi.MethodNameDexFundWithdraw}
	contracts[types.AddressDexFund].m[cabi.MethodNameDexFundOpenNewMarket] = &MethodDexFundOpenNewMarket{cabi.MethodNameDexFundOpenNewMarket}
	contracts[types.AddressDexFund].m[cabi.MethodNameDexFundPlaceOrder] = &MethodDexFundPlaceOrder{cabi.MethodNameDexFundPlaceOrder}
	contracts[types.AddressDexFund].m[cabi.MethodNameDexFundSettleOrdersV2] = &MethodDexFundSettleOrders{cabi.MethodNameDexFundSettleOrdersV2}
	contracts[types.AddressDexFund].m[cabi.MethodNameDexFundTriggerPeriodJob] = &MethodDexFundTriggerPeriodJob{cabi.MethodNameDexFundTriggerPeriodJob}
	contracts[types.AddressDexFund].m[cabi.MethodNameDexFundStakeForMining] = &MethodDexFundStakeForMining{cabi.MethodNameDexFundStakeForMining}
	contracts[types.AddressDexFund].m[cabi.MethodNameDexFundStakeForVIP] = &MethodDexFundStakeForVIP{cabi.MethodNameDexFundStakeForVIP}
	contracts[types.AddressDexFund].m[cabi.MethodNameDexFundDelegateStakingCallback] = &MethodDexFundDelegateStakingCallback{cabi.MethodNameDexFundDelegateStakingCallback}
	contracts[types.AddressDexFund].m[cabi.MethodNameDexFundCancelDelegatedStakingCallback] = &MethodDexFundCancelDelegatedStakingCallback{cabi.MethodNameDexFundCancelDelegatedStakingCallback}
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
