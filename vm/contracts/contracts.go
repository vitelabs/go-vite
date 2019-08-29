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
	GetRefundData(sendBlock *ledger.AccountBlock) ([]byte, bool)
}

type builtinContract struct {
	m   map[string]BuiltinContractMethod
	abi abi.ABIContract
}

var (
	simpleContracts   = newSimpleContracts()
	dexContracts      = newDexContracts()
	dexAgentContracts = newDexAgentContracts()
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
	contracts[types.AddressMintage].m[cabi.MethodNameGetTokenInfo] = &MethodGetTokenInfo{}
	contracts[types.AddressDexFund] = &builtinContract{
		map[string]BuiltinContractMethod{
			cabi.MethodNameDexFundUserDeposit:          &MethodDexFundDeposit{},
			cabi.MethodNameDexFundUserWithdraw:         &MethodDexFundWithdraw{},
			cabi.MethodNameDexFundNewMarket:            &MethodDexFundOpenNewMarket{},
			cabi.MethodNameDexFundNewOrder:             &MethodDexFundPlaceOrder{},
			cabi.MethodNameDexFundSettleOrders:         &MethodDexFundSettleOrders{},
			cabi.MethodNameDexFundPeriodJob:            &MethodDexFundTriggerPeriodJob{},
			cabi.MethodNameDexFundPledgeForVx:          &MethodDexFundStakeForMining{},
			cabi.MethodNameDexFundPledgeForVip:         &MethodDexFundStakeForVIP{},
			cabi.MethodNameDexFundPledgeCallback:       &MethodDexFundDelegateStakingCallback{},
			cabi.MethodNameDexFundCancelPledgeCallback: &MethodDexFundCancelDelegatedStakingCallback{},
			cabi.MethodNameDexFundGetTokenInfoCallback: &MethodDexFundGetTokenInfoCallback{},
			cabi.MethodNameDexFundOwnerConfig:          &MethodDexFundDexAdminConfig{},
			cabi.MethodNameDexFundOwnerConfigTrade:     &MethodDexFundTradeAdminConfig{},
			cabi.MethodNameDexFundMarketOwnerConfig:    &MethodDexFundMarketAdminConfig{},
			cabi.MethodNameDexFundTransferTokenOwner:   &MethodDexFundTransferTokenOwnership{},
			cabi.MethodNameDexFundNotifyTime:           &MethodDexFundNotifyTime{},
			cabi.MethodNameDexFundNewInviter:           &MethodDexFundCreateNewInviter{},
			cabi.MethodNameDexFundBindInviteCode:       &MethodDexFundBindInviteCode{},
			cabi.MethodNameDexFundEndorseVxMinePool:    &MethodDexFundEndorseVx{},
			cabi.MethodNameDexFundSettleMakerMinedVx:   &MethodDexFundSettleMakerMinedVx{},
		},
		cabi.ABIDexFund,
	}
	contracts[types.AddressDexTrade] = &builtinContract{
		map[string]BuiltinContractMethod{
			cabi.MethodNameDexTradeNewOrder:          &MethodDexTradePlace{},
			cabi.MethodNameDexTradeCancelOrder:       &MethodDexTradeCancelOrder{},
			cabi.MethodNameDexTradeNotifyNewMarket:   &MethodDexTradeSyncNewMarket{},
			cabi.MethodNameDexTradeCleanExpireOrders: &MethodDexTradeClearExpiredOrders{},
		},
		cabi.ABIDexTrade,
	}
	return contracts
}

func newDexAgentContracts() map[types.Address]*builtinContract {
	contracts := newDexContracts()
	contracts[types.AddressDexFund].m[cabi.MethodNameDexFundStakeForSuperVip] = &MethodDexFundStakeForSVIP{}
	contracts[types.AddressDexFund].m[cabi.MethodNameDexFundConfigMarketsAgent] = &MethodDexFundConfigMarketAgents{}
	contracts[types.AddressDexFund].m[cabi.MethodNameDexFundNewAgentOrder] = &MethodDexFundPlaceAgentOrder{}
	contracts[types.AddressDexTrade].m[cabi.MethodNameDexTradeCancelOrderByHash] = &MethodDexTradeCancelOrderByTransactionHash{}
	return contracts
}

func GetBuiltinContractMethod(addr types.Address, methodSelector []byte, sbHeight uint64) (BuiltinContractMethod, bool, error) {
	var contractsMap map[types.Address]*builtinContract
	if fork.IsStemFork(sbHeight) {
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
