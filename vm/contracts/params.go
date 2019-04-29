package contracts

import (
	"github.com/vitelabs/go-vite/vm/util"
	"math/big"
)

const (
	RegisterGas               uint64 = 62200
	UpdateRegistrationGas     uint64 = 62200
	CancelRegisterGas         uint64 = 83200
	RewardGas                 uint64 = 68200
	VoteGas                   uint64 = 62000
	CancelVoteGas             uint64 = 62000
	PledgeGas                 uint64 = 82000
	CancelPledgeGas           uint64 = 73000
	AgentPledgeGas            uint64 = 82000
	AgentCancelPledgeGas      uint64 = 73000
	CreateConsensusGroupGas   uint64 = 62200
	CancelConsensusGroupGas   uint64 = 83200
	ReCreateConsensusGroupGas uint64 = 62200

	dexFundDepositGas                     uint64 = 300
	dexFundDepositReceiveGas              uint64 = 1
	dexFundWithdrawGas                    uint64 = 300
	dexFundWithdrawReceiveGas             uint64 = 1
	dexFundNewOrderGas                    uint64 = 300
	dexFundNewOrderReceiveGas             uint64 = 1
	dexFundSettleOrdersGas                uint64 = 300
	dexFundSettleOrdersReceiveGas         uint64 = 1
	dexFundFeeDividendGas                 uint64 = 300
	dexFundFeeDividendReceiveGas          uint64 = 1
	dexFundMinedVxDividendGas             uint64 = 300
	dexFundMinedVxDividendReceiveGas      uint64 = 1
	dexFundNewMarketGas                   uint64 = 300
	dexFundNewMarketReceiveGas            uint64 = 1
	dexFundSetOwnerGas                    uint64 = 300
	dexFundSetOwnerReceiveGas             uint64 = 1
	dexFundConfigMineMarketGas            uint64 = 300
	dexFundConfigMineMarketReceiveGas     uint64 = 1
	dexFundPledgeForVxGas                 uint64 = 300
	dexFundPledgeForVxReceiveGas          uint64 = 1
	dexFundPledgeForVipGas                uint64 = 300
	dexFundPledgeForVipReceiveGas         uint64 = 1
	dexFundPledgeCallbackGas              uint64 = 300
	dexFundPledgeCallbackReceiveGas       uint64 = 1
	dexFundCancelPledgeCallbackGas        uint64 = 300
	dexFundCancelPledgeCallbackReceiveGas uint64 = 1
	dexFundGetTokenInfoCallbackGas        uint64 = 300
	dexFundGetTokenInfoCallbackReceiveGas uint64 = 1

	dexTradeNewOrderGas    uint64 = 300
	dexTradeCancelOrderGas uint64 = 300

	MintGas                uint64 = 104525
	MintageCancelPledgeGas uint64 = 83200
	IssueGas               uint64 = 69325
	BurnGas                uint64 = 48837
	TransferOwnerGas       uint64 = 58981
	ChangeTokenTypeGas     uint64 = 63125
	GetTokenInfoGas        uint64 = 63200

	cgNodeCountMin   uint8 = 3       // Minimum node count of consensus group
	cgNodeCountMax   uint8 = 101     // Maximum node count of consensus group
	cgIntervalMin    int64 = 1       // Minimum interval of consensus group in second
	cgIntervalMax    int64 = 10 * 60 // Maximum interval of consensus group in second
	cgPerCountMin    int64 = 1
	cgPerCountMax    int64 = 10 * 60
	cgPerIntervalMin int64 = 1
	cgPerIntervalMax int64 = 10 * 60

	registrationNameLengthMax int = 40

	tokenNameLengthMax   int = 40 // Maximum length of a token name(include)
	tokenSymbolLengthMax int = 10 // Maximum length of a token symbol(include)

	tokenNameIndexMax uint16 = 1000
)

var (
	rewardPerBlock                   = big.NewInt(951293759512937595) // Reward pre snapshot block, rewardPreBlock * blockNumPerYear / viteTotalSupply = 3%
	pledgeAmountMin                  = new(big.Int).Mul(big.NewInt(134), util.AttovPerVite)
	mintageFee                       = new(big.Int).Mul(big.NewInt(1e3), util.AttovPerVite) // Mintage cost choice 1, destroy ViteToken
	mintagePledgeAmount              = new(big.Int).Mul(big.NewInt(1e5), util.AttovPerVite) // Mintage cost choice 2, pledge ViteToken for 3 month
	createConsensusGroupPledgeAmount = new(big.Int).Mul(big.NewInt(1000), util.AttovPerVite)
)

type ContractsParams struct {
	RegisterMinPledgeHeight          uint64 // Minimum pledge height for register
	PledgeHeight                     uint64 // pledge height for stake
	CreateConsensusGroupPledgeHeight uint64 // Pledge height for registering to be a super node of snapshot group and common delegate group
	MintPledgeHeight                 uint64 // Pledge height for mintage if choose to pledge instead of destroy vite token
	GetRewardTimeLimit               int64  // Cannot get snapshot block reward of current few blocks, for latest snapshot block could be reverted
}

var (
	ContractsParamsTest = ContractsParams{
		RegisterMinPledgeHeight:          1,
		PledgeHeight:                     1,
		CreateConsensusGroupPledgeHeight: 1,
		MintPledgeHeight:                 1,
		GetRewardTimeLimit:               75,
	}
	ContractsParamsMainNet = ContractsParams{
		RegisterMinPledgeHeight:          3600 * 24 * 3,
		PledgeHeight:                     3600 * 24 * 3,
		CreateConsensusGroupPledgeHeight: 3600 * 24 * 3,
		MintPledgeHeight:                 3600 * 24 * 30 * 3,
		GetRewardTimeLimit:               3600,
	}
)
