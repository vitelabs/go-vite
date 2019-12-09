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
	// SbpStakeAmountPreMainnet defines SBP register stake amount before leaf fork
	SbpStakeAmountPreMainnet = new(big.Int).Mul(big.NewInt(5e5), util.AttovPerVite)
	// SbpStakeAmountMainnet defines SBP register stake amount after leaf fork
	SbpStakeAmountMainnet = new(big.Int).Mul(big.NewInt(1e6), util.AttovPerVite)
	// Reward pre snapshot block, rewardPreBlock * blockNumPerYear / viteTotalSupply = 3%
	rewardPerBlock = big.NewInt(951293759512937595)
	stakeAmountMin = new(big.Int).Mul(big.NewInt(134), util.AttovPerVite)
	issueFee       = new(big.Int).Mul(big.NewInt(1e3), util.AttovPerVite)
)

type contractsParams struct {
	StakeHeight            uint64 // locking height for stake
	DexVipStakeHeight      uint64 // locking height for dex_fund contract, in order to upgrade to dex vip
	DexSuperVipStakeHeight uint64 // locking height for dex_fund contract, in order to upgrade to dex super vip
}

var (
	contractsParamsTest = contractsParams{
		StakeHeight:            600,
		DexVipStakeHeight:      1,
		DexSuperVipStakeHeight: 1,
	}
	contractsParamsMainNet = contractsParams{
		StakeHeight:            3600 * 24 * 3,
		DexVipStakeHeight:      3600 * 24 * 30,
		DexSuperVipStakeHeight: 3600 * 24 * 30,
	}
)
