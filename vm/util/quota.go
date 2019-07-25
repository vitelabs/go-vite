package util

import (
	"github.com/vitelabs/go-vite/common/fork"
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/ledger"
)

const (
	CommonQuotaRatio   uint8  = 10
	QuotaRatioDivision uint64 = 10
	OneRound           uint64 = 75
)

func MultipleCost(cost uint64, quotaRatio uint8) (uint64, error) {
	if quotaRatio < CommonQuotaRatio {
		return 0, ErrInvalidQuotaRatio
	}
	if quotaRatio == CommonQuotaRatio {
		return cost, nil
	}
	ratioUint64 := uint64(quotaRatio)
	if cost > helper.MaxUint64/ratioUint64 {
		return 0, ErrGasUintOverflow
	}
	return cost * ratioUint64 / QuotaRatioDivision, nil
}

func UseQuota(quotaLeft, cost uint64) (uint64, error) {
	if quotaLeft < cost {
		return 0, ErrOutOfQuota
	}
	quotaLeft = quotaLeft - cost
	return quotaLeft, nil
}

func UseQuotaWithFlag(quotaLeft, cost uint64, flag bool) (uint64, error) {
	if flag {
		return UseQuota(quotaLeft, cost)
	}
	return quotaLeft + cost, nil
}

func IntrinsicGasCost(data []byte, baseGas uint64, confirmTime uint8, gasTable *GasTable) (uint64, error) {
	var gas uint64
	gas = baseGas
	gasData, err := DataGasCost(data, gasTable)
	if err != nil || helper.MaxUint64-gas < gasData {
		return 0, ErrGasUintOverflow
	}
	gas = gas + gasData
	if confirmTime == 0 {
		return gas, nil
	}
	confirmGas := uint64(confirmTime) * gasTable.ConfirmTimeGas
	if helper.MaxUint64-gas < confirmGas {
		return 0, ErrGasUintOverflow
	}
	return gas + confirmGas, nil
}

func DataGasCost(data []byte, gasTable *GasTable) (uint64, error) {
	var gas uint64
	if l := uint64(len(data)); l > 0 {
		if helper.MaxUint64/gasTable.TxDataGas < l {
			return 0, ErrGasUintOverflow
		}
		gas = l * gasTable.TxDataGas
	}
	return gas, nil
}

func TxGasCost(data []byte, gasTable *GasTable) (uint64, error) {
	dataCost, err := DataGasCost(data, gasTable)
	if err != nil {
		return 0, err
	}
	totalCost, overflow := helper.SafeAdd(gasTable.TxGas, dataCost)
	if overflow {
		return 0, err
	}
	return totalCost, nil
}

func CalcQuotaUsed(useQuota bool, quotaTotal, quotaAddition, quotaLeft uint64, err error) (q uint64, qUsed uint64) {
	if !useQuota {
		return 0, 0
	}
	if err == ErrOutOfQuota {
		return 0, 0
	} else {
		qUsed = quotaTotal - quotaLeft
		if qUsed < quotaAddition {
			return 0, qUsed
		} else {
			return qUsed - quotaAddition, qUsed
		}
	}
}

func IsPoW(block *ledger.AccountBlock) bool {
	return len(block.Nonce) > 0
}

type GasTable struct {
	AddGas            uint64
	MulGas            uint64
	SubGas            uint64
	DivGas            uint64
	SdivGas           uint64
	ModGas            uint64
	SmodGas           uint64
	AddmodGas         uint64
	MulmodGas         uint64
	ExpGas            uint64
	ExpByteGas        uint64
	SignextendGas     uint64
	LtGas             uint64
	GtGas             uint64
	SltGas            uint64
	SgtGas            uint64
	EqGas             uint64
	IszeroGas         uint64
	AndGas            uint64
	OrGas             uint64
	XorGas            uint64
	NotGas            uint64
	ByteGas           uint64
	ShlGas            uint64
	ShrGas            uint64
	SarGas            uint64
	Blake2bGas        uint64
	Blake2bWordGas    uint64
	AddressGas        uint64
	BalanceGas        uint64
	CallerGas         uint64
	CallvalueGas      uint64
	CalldataloadGas   uint64
	CalldatasizeGas   uint64
	CalldatacopyGas   uint64
	MemcopyWordGas    uint64
	CodesizeGas       uint64
	CodeCopyGas       uint64
	ReturndatasizeGas uint64
	ReturndatacopyGas uint64
	TimestampGas      uint64
	HeightGas         uint64
	TokenidGas        uint64
	AccountheightGas  uint64
	PrevhashGas       uint64
	FromhashGas       uint64
	SeedGas           uint64
	RandomGas         uint64
	PopGas            uint64
	MloadGas          uint64
	MstoreGas         uint64
	Mstore8Gas        uint64
	SloadGas          uint64
	SstoreResetGas    uint64
	SstoreInitGas     uint64
	SstoreCleanGas    uint64
	SstoreNoopGas     uint64
	SstoreMemGas      uint64
	JumpGas           uint64
	JumpiGas          uint64
	PcGas             uint64
	MsizeGas          uint64
	JumpdestGas       uint64
	PushGas           uint64
	DupGas            uint64
	SwapGas           uint64
	LogGas            uint64
	LogTopicGas       uint64
	LogDataGas        uint64
	CallMinusGas      uint64
	MemGasDivision    uint64
	ConfirmTimeGas    uint64
	CodeGas           uint64
	MemGas            uint64

	TxGas               uint64
	TxDataGas           uint64
	CreateTxRequestGas  uint64
	CreateTxResponseGas uint64

	RegisterGas                    uint64
	UpdateRegistrationGas          uint64
	CancelRegisterGas              uint64
	RewardGas                      uint64
	VoteGas                        uint64
	CancelVoteGas                  uint64
	PledgeGas                      uint64
	CancelPledgeGas                uint64
	AgentPledgeGas                 uint64
	AgentCancelPledgeGas           uint64
	MintGas                        uint64
	IssueGas                       uint64
	BurnGas                        uint64
	TransferOwnerGas               uint64
	ChangeTokenTypeGas             uint64
	GetTokenInfoGas                uint64
	DexFundDepositGas              uint64
	DexFundWithdrawGas             uint64
	DexFundNewMarketGas            uint64
	DexFundNewOrderGas             uint64
	DexFundSettleOrdersGas         uint64
	DexFundPeriodJobGas            uint64
	DexFundPledgeForVxGas          uint64
	DexFundPledgeForVipGas         uint64
	DexFundPledgeCallbackGas       uint64
	DexFundCancelPledgeCallbackGas uint64
	DexFundGetTokenInfoCallbackGas uint64
	DexFundOwnerConfigGas          uint64
	DexFundOwnerConfigTradeGas     uint64
	DexFundMarketOwnerConfigGas    uint64
	DexFundTransferTokenOwnerGas   uint64
	DexFundNotifyTimeGas           uint64
	DexFundNewInviterGas           uint64
	DexFundBindInviteCodeGas       uint64
	DexFundEndorseVxMinePoolGas    uint64
	DexFundSettleMakerMinedVxGas   uint64
}

var (
	initGasTable = GasTable{
		AddGas:                3,
		MulGas:                5,
		SubGas:                3,
		DivGas:                5,
		SdivGas:               5,
		ModGas:                5,
		SmodGas:               5,
		AddmodGas:             8,
		MulmodGas:             8,
		ExpGas:                10,
		ExpByteGas:            50,
		SignextendGas:         5,
		LtGas:                 3,
		GtGas:                 3,
		SltGas:                3,
		SgtGas:                3,
		EqGas:                 3,
		IszeroGas:             3,
		AndGas:                3,
		OrGas:                 3,
		XorGas:                3,
		NotGas:                3,
		ByteGas:               3,
		ShlGas:                3,
		ShrGas:                3,
		SarGas:                3,
		Blake2bGas:            30,
		Blake2bWordGas:        6,
		AddressGas:            2,
		BalanceGas:            400,
		CallerGas:             2,
		CallvalueGas:          2,
		CalldataloadGas:       3,
		CalldatasizeGas:       2,
		CalldatacopyGas:       3,
		MemcopyWordGas:        3,
		CodesizeGas:           2,
		CodeCopyGas:           3,
		ReturndatasizeGas:     2,
		ReturndatacopyGas:     3,
		TimestampGas:          2,
		HeightGas:             2,
		TokenidGas:            2,
		AccountheightGas:      2,
		PrevhashGas:           2,
		FromhashGas:           2,
		SeedGas:               2,
		RandomGas:             2,
		PopGas:                2,
		MloadGas:              3,
		MstoreGas:             3,
		Mstore8Gas:            3,
		SloadGas:              200,
		SstoreResetGas:        5000,
		SstoreInitGas:         20000,
		SstoreCleanGas:        100,
		SstoreNoopGas:         200,
		SstoreMemGas:          200,
		JumpGas:               8,
		JumpiGas:              10,
		PcGas:                 2,
		MsizeGas:              2,
		JumpdestGas:           1,
		PushGas:               3,
		DupGas:                3,
		SwapGas:               3,
		LogGas:                375,
		LogTopicGas:           375,
		LogDataGas:            8,
		CallMinusGas:          10000,
		MemGasDivision:        512,
		ConfirmTimeGas:        200,
		CodeGas:               200,
		MemGas:                3,
		TxGas:                 21000,
		TxDataGas:             68,
		CreateTxRequestGas:    21000,
		CreateTxResponseGas:   53000,
		RegisterGas:           62200,
		UpdateRegistrationGas: 62200,
		CancelRegisterGas:     83200,
		RewardGas:             68200,
		VoteGas:               62000,
		CancelVoteGas:         62000,
		PledgeGas:             82000,
		CancelPledgeGas:       73000,
		AgentPledgeGas:        82000,
		AgentCancelPledgeGas:  73000,
		MintGas:               104525,
		IssueGas:              69325,
		BurnGas:               48837,
		TransferOwnerGas:      58981,
		ChangeTokenTypeGas:    63125,
		GetTokenInfoGas:       63200,
	}

	viteGasTable = GasTable{
		AddGas:            2,
		MulGas:            2,
		SubGas:            2,
		DivGas:            3,
		SdivGas:           5,
		ModGas:            3,
		SmodGas:           4,
		AddmodGas:         4,
		MulmodGas:         5,
		ExpGas:            10,
		ExpByteGas:        50,
		SignextendGas:     2,
		LtGas:             2,
		GtGas:             2,
		SltGas:            2,
		SgtGas:            2,
		EqGas:             2,
		IszeroGas:         1,
		AndGas:            2,
		OrGas:             2,
		XorGas:            2,
		NotGas:            2,
		ByteGas:           2,
		ShlGas:            2,
		ShrGas:            2,
		SarGas:            3,
		Blake2bGas:        20,
		Blake2bWordGas:    1,
		AddressGas:        1,
		BalanceGas:        150,
		CallerGas:         1,
		CallvalueGas:      1,
		CalldataloadGas:   2,
		CalldatasizeGas:   1,
		CalldatacopyGas:   3,
		MemcopyWordGas:    3,
		CodesizeGas:       1,
		CodeCopyGas:       3,
		ReturndatasizeGas: 1,
		ReturndatacopyGas: 3,
		TimestampGas:      1,
		HeightGas:         1,
		TokenidGas:        1,
		AccountheightGas:  1,
		PrevhashGas:       1,
		FromhashGas:       1,
		SeedGas:           200,
		RandomGas:         250,
		PopGas:            1,
		MloadGas:          2,
		MstoreGas:         1,
		Mstore8Gas:        1,
		SloadGas:          150,
		SstoreResetGas:    15000,
		SstoreInitGas:     15000,
		SstoreCleanGas:    0,
		SstoreNoopGas:     200,
		SstoreMemGas:      200,
		JumpGas:           4,
		JumpiGas:          4,
		PcGas:             1,
		MsizeGas:          1,
		JumpdestGas:       1,
		PushGas:           1,
		DupGas:            1,
		SwapGas:           2,
		LogGas:            375,
		LogTopicGas:       375,
		LogDataGas:        12,
		CallMinusGas:      13500,
		MemGasDivision:    1024,
		ConfirmTimeGas:    40,
		CodeGas:           160,
		MemGas:            1,

		TxGas:               21000,
		TxDataGas:           68,
		CreateTxRequestGas:  31000,
		CreateTxResponseGas: 31000,

		RegisterGas:                    168000,
		UpdateRegistrationGas:          168000,
		CancelRegisterGas:              126000,
		RewardGas:                      147000,
		VoteGas:                        84000,
		CancelVoteGas:                  52500,
		PledgeGas:                      105000,
		CancelPledgeGas:                105000,
		AgentPledgeGas:                 115500,
		AgentCancelPledgeGas:           115500,
		MintGas:                        189000,
		IssueGas:                       126000,
		BurnGas:                        115500,
		TransferOwnerGas:               136500,
		ChangeTokenTypeGas:             115500,
		GetTokenInfoGas:                31500,
		DexFundDepositGas:              10500,
		DexFundWithdrawGas:             10500,
		DexFundNewMarketGas:            31500,
		DexFundNewOrderGas:             25200,
		DexFundSettleOrdersGas:         21000,
		DexFundPeriodJobGas:            8400,
		DexFundPledgeForVxGas:          31500,
		DexFundPledgeForVipGas:         31500,
		DexFundPledgeCallbackGas:       12600,
		DexFundCancelPledgeCallbackGas: 16800,
		DexFundGetTokenInfoCallbackGas: 10500,
		DexFundOwnerConfigGas:          16800,
		DexFundOwnerConfigTradeGas:     10500,
		DexFundMarketOwnerConfigGas:    10500,
		DexFundTransferTokenOwnerGas:   8400,
		DexFundNotifyTimeGas:           10500,
		DexFundNewInviterGas:           18900,
		DexFundBindInviteCodeGas:       8400,
		DexFundEndorseVxMinePoolGas:    6300,
		DexFundSettleMakerMinedVxGas:   25200,
	}
)

func GasTableByHeight(sbHeight uint64) *GasTable {
	if !fork.IsDexFork(sbHeight) {
		return &initGasTable
	}
	return &viteGasTable
}
