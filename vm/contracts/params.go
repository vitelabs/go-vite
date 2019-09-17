package contracts

import (
	"github.com/vitelabs/go-vite/vm/util"
	"math/big"
)

const (
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
	rewardTimeLimit   int64  = 3600 // Cannot get snapshot block reward of current few blocks, for latest snapshot block could be reverted

	stakeHeightMax uint64 = 3600 * 24 * 365
)

var (
	SbpStakeAmountPreMainnet = new(big.Int).Mul(big.NewInt(5e5), util.AttovPerVite) // SBP register stake amount before leaf fork
	SbpStakeAmountMainnet    = new(big.Int).Mul(big.NewInt(1e6), util.AttovPerVite) // SBP register stake amount after leaf fork
	rewardPerBlock           = big.NewInt(951293759512937595)                       // Reward pre snapshot block, rewardPreBlock * blockNumPerYear / viteTotalSupply = 3%
	pledgeAmountMin          = new(big.Int).Mul(big.NewInt(134), util.AttovPerVite)
	mintageFee               = new(big.Int).Mul(big.NewInt(1e3), util.AttovPerVite) // Mintage cost choice 1, destroy ViteToken
	// mintagePledgeAmount              = new(big.Int).Mul(big.NewInt(1e5), util.AttovPerVite) // Mintage cost choice 2, pledge ViteToken for 3 month
	createConsensusGroupPledgeAmount = new(big.Int).Mul(big.NewInt(1000), util.AttovPerVite)
)

type contractsParams struct {
	RegisterMinPledgeHeight          uint64 // Minimum pledge height for register
	PledgeHeight                     uint64 // pledge height for stake
	CreateConsensusGroupPledgeHeight uint64 // Pledge height for registering to be a super node of snapshot group and common delegate group
	MintPledgeHeight                 uint64 // Pledge height for mintage if choose to pledge instead of destroy vite token
	DexVipStakeHeight                uint64 // Pledge height for dex_fund contract, in order to upgrade to dex vip
	DexSuperVipStakeHeight           uint64 // Pledge height for dex_fund contract, in order to upgrade to dex super vip
}

var (
	contractsParamsTest = contractsParams{
		RegisterMinPledgeHeight:          1,
		PledgeHeight:                     1,
		CreateConsensusGroupPledgeHeight: 1,
		MintPledgeHeight:                 1,
		DexVipStakeHeight:                1,
		DexSuperVipStakeHeight:           1,
	}
	contractsParamsMainNet = contractsParams{
		RegisterMinPledgeHeight:          3600 * 24 * 3,
		PledgeHeight:                     3600 * 24 * 3,
		CreateConsensusGroupPledgeHeight: 3600 * 24 * 3,
		MintPledgeHeight:                 3600 * 24 * 30 * 3,
		DexVipStakeHeight:                3600 * 24 * 30,
		DexSuperVipStakeHeight:           3600 * 24 * 30,
	}
)
