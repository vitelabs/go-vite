package util

import (
	"github.com/vitelabs/go-vite/common/fork"
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/ledger"
)

const (
	// CommonQuotaMultiplier defines base quota multiplier for all accounts
	CommonQuotaMultiplier   uint8  = 10
	quotaMultiplierDivision uint64 = 10
	// QuotaAccumulationBlockCount defines max quota accumulation count
	QuotaAccumulationBlockCount uint64 = 75
)

// MultipleCost multiply quota
func MultipleCost(cost uint64, quotaMultiplier uint8) (uint64, error) {
	if quotaMultiplier < CommonQuotaMultiplier {
		return 0, ErrInvalidQuotaMultiplier
	}
	if quotaMultiplier == CommonQuotaMultiplier {
		return cost, nil
	}
	ratioUint64 := uint64(quotaMultiplier)
	if cost > helper.MaxUint64/ratioUint64 {
		return 0, ErrGasUintOverflow
	}
	return cost * ratioUint64 / quotaMultiplierDivision, nil
}

// UseQuota check out of quota and return quota left
func UseQuota(quotaLeft, cost uint64) (uint64, error) {
	if quotaLeft < cost {
		return 0, ErrOutOfQuota
	}
	quotaLeft = quotaLeft - cost
	return quotaLeft, nil
}

// UseQuotaWithFlag check out of quota and return quota left
func UseQuotaWithFlag(quotaLeft, cost uint64, flag bool) (uint64, error) {
	if flag {
		return UseQuota(quotaLeft, cost)
	}
	return quotaLeft + cost, nil
}

// BlockGasCost calculate base quota cost of a block
func BlockGasCost(data []byte, baseGas uint64, snapshotCount uint8, quotaTable *QuotaTable) (uint64, error) {
	var gas uint64
	gas = baseGas
	gasData, err := DataQuotaCost(data, quotaTable)
	if err != nil || helper.MaxUint64-gas < gasData {
		return 0, ErrGasUintOverflow
	}
	gas = gas + gasData
	if snapshotCount == 0 {
		return gas, nil
	}
	confirmGas := uint64(snapshotCount) * quotaTable.SnapshotQuota
	if helper.MaxUint64-gas < confirmGas {
		return 0, ErrGasUintOverflow
	}
	return gas + confirmGas, nil
}

// DataQuotaCost calculate quota cost by request block data
func DataQuotaCost(data []byte, quotaTable *QuotaTable) (uint64, error) {
	var gas uint64
	if l := uint64(len(data)); l > 0 {
		if helper.MaxUint64/quotaTable.TxDataQuota < l {
			return 0, ErrGasUintOverflow
		}
		gas = l * quotaTable.TxDataQuota
	}
	return gas, nil
}

// RequestQuotaCost calculate quota cost by a request block
func RequestQuotaCost(data []byte, quotaTable *QuotaTable) (uint64, error) {
	dataCost, err := DataQuotaCost(data, quotaTable)
	if err != nil {
		return 0, err
	}
	totalCost, overflow := helper.SafeAdd(quotaTable.TxQuota, dataCost)
	if overflow {
		return 0, err
	}
	return totalCost, nil
}

// CalcQuotaUsed calculate stake quota and total quota used by a block
func CalcQuotaUsed(useQuota bool, quotaTotal, quotaAddition, quotaLeft uint64, err error) (qStakeUsed uint64, qUsed uint64) {
	if !useQuota {
		return 0, 0
	}
	if err == ErrOutOfQuota {
		return 0, 0
	}
	qUsed = quotaTotal - quotaLeft
	if qUsed < quotaAddition {
		return 0, qUsed
	}
	return qUsed - quotaAddition, qUsed
}

// IsPoW check whether a block calculated pow
func IsPoW(block *ledger.AccountBlock) bool {
	return len(block.Nonce) > 0
}

// QuotaTable is used to query quota used by op code and transactions
type QuotaTable struct {
	AddQuota            uint64
	MulQuota            uint64
	SubQuota            uint64
	DivQuota            uint64
	SDivQuota           uint64
	ModQuota            uint64
	SModQuota           uint64
	AddModQuota         uint64
	MulModQuota         uint64
	ExpQuota            uint64
	ExpByteQuota        uint64
	SignExtendQuota     uint64
	LtQuota             uint64
	GtQuota             uint64
	SltQuota            uint64
	SgtQuota            uint64
	EqQuota             uint64
	IsZeroQuota         uint64
	AndQuota            uint64
	OrQuota             uint64
	XorQuota            uint64
	NotQuota            uint64
	ByteQuota           uint64
	ShlQuota            uint64
	ShrQuota            uint64
	SarQuota            uint64
	Blake2bQuota        uint64
	Blake2bWordQuota    uint64
	AddressQuota        uint64
	BalanceQuota        uint64
	CallerQuota         uint64
	CallValueQuota      uint64
	CallDataLoadQuota   uint64
	CallDataSizeQuota   uint64
	CallDataCopyQuota   uint64
	MemCopyWordQuota    uint64
	CodeSizeQuota       uint64
	CodeCopyQuota       uint64
	ReturnDataSizeQuota uint64
	ReturnDataCopyQuota uint64
	TimestampQuota      uint64
	HeightQuota         uint64
	TokenIDQuota        uint64
	AccountHeightQuota  uint64
	PreviousHashQuota   uint64
	FromBlockHashQuota  uint64
	SeedQuota           uint64
	RandomQuota         uint64
	PopQuota            uint64
	MloadQuota          uint64
	MstoreQuota         uint64
	Mstore8Quota        uint64
	SloadQuota          uint64
	SstoreResetQuota    uint64
	SstoreInitQuota     uint64
	SstoreCleanQuota    uint64
	SstoreNoopQuota     uint64
	SstoreMemQuota      uint64
	JumpQuota           uint64
	JumpiQuota          uint64
	PcQuota             uint64
	MsizeQuota          uint64
	JumpdestQuota       uint64
	PushQuota           uint64
	DupQuota            uint64
	SwapQuota           uint64
	LogQuota            uint64
	LogTopicQuota       uint64
	LogDataQuota        uint64
	CallMinusQuota      uint64
	MemQuotaDivision    uint64
	SnapshotQuota       uint64
	CodeQuota           uint64
	MemQuota            uint64

	TxQuota               uint64
	TxDataQuota           uint64
	CreateTxRequestQuota  uint64
	CreateTxResponseQuota uint64

	RegisterQuota                             uint64
	UpdateBlockProducingAddressQuota          uint64
	UpdateRewardWithdrawAddressQuota          uint64
	RevokeQuota                               uint64
	WithdrawRewardQuota                       uint64
	VoteQuota                                 uint64
	CancelVoteQuota                           uint64
	StakeQuota                                uint64
	CancelStakeQuota                          uint64
	DelegateStakeQuota                        uint64
	CancelDelegateStakeQuota                  uint64
	IssueQuota                                uint64
	ReIssueQuota                              uint64
	BurnQuota                                 uint64
	TransferOwnershipQuota                    uint64
	DisableReIssueQuota                       uint64
	GetTokenInfoQuota                         uint64
	DexFundDepositQuota                       uint64
	DexFundWithdrawQuota                      uint64
	DexFundOpenNewMarketQuota                 uint64
	DexFundPlaceOrderQuota                    uint64
	DexFundSettleOrdersQuota                  uint64
	DexFundTriggerPeriodJobQuota              uint64
	DexFundStakeForMiningQuota                uint64
	DexFundStakeForVipQuota                   uint64
	DexFundStakeForSuperVIPQuota              uint64
	DexFundDelegateStakeCallbackQuota         uint64
	DexFundCancelDelegateStakeCallbackQuota   uint64
	DexFundGetTokenInfoCallbackQuota          uint64
	DexFundAdminConfigQuota                   uint64
	DexFundTradeAdminConfigQuota              uint64
	DexFundMarketAdminConfigQuota             uint64
	DexFundTransferTokenOwnershipQuota        uint64
	DexFundNotifyTimeQuota                    uint64
	DexFundCreateNewInviterQuota              uint64
	DexFundBindInviteCodeQuota                uint64
	DexFundEndorseVxQuota                     uint64
	DexFundSettleMakerMinedVxQuota            uint64
	DexFundConfigMarketAgentsQuota            uint64
	DexFunPlaceAgentOrderQuota                uint64
	DexFunLockVxForDividendQuota              uint64
	DexFunSwitchConfigQuota                   uint64
	DexFundStakeForPrincipalSuperVIPQuota     uint64
	DexFundCancelStakeByIdQuota               uint64
	DexFundDelegateStakeCallbackV2Quota       uint64
	DexFundDelegateCancelStakeCallbackV2Quota uint64
}

// QuotaTableByHeight returns different quota table by hard fork version
func QuotaTableByHeight(sbHeight uint64) *QuotaTable {
	if fork.IsEarthFork(sbHeight) {
		return &earthQuotaTable
	} else if fork.IsStemFork(sbHeight) {
		return &dexAgentQuotaTable
	} else if fork.IsDexFork(sbHeight) {
		return &viteQuotaTable
	}
	return &initQuotaTable
}

var (
	initQuotaTable = QuotaTable{
		AddQuota:                         3,
		MulQuota:                         5,
		SubQuota:                         3,
		DivQuota:                         5,
		SDivQuota:                        5,
		ModQuota:                         5,
		SModQuota:                        5,
		AddModQuota:                      8,
		MulModQuota:                      8,
		ExpQuota:                         10,
		ExpByteQuota:                     50,
		SignExtendQuota:                  5,
		LtQuota:                          3,
		GtQuota:                          3,
		SltQuota:                         3,
		SgtQuota:                         3,
		EqQuota:                          3,
		IsZeroQuota:                      3,
		AndQuota:                         3,
		OrQuota:                          3,
		XorQuota:                         3,
		NotQuota:                         3,
		ByteQuota:                        3,
		ShlQuota:                         3,
		ShrQuota:                         3,
		SarQuota:                         3,
		Blake2bQuota:                     30,
		Blake2bWordQuota:                 6,
		AddressQuota:                     2,
		BalanceQuota:                     400,
		CallerQuota:                      2,
		CallValueQuota:                   2,
		CallDataLoadQuota:                3,
		CallDataSizeQuota:                2,
		CallDataCopyQuota:                3,
		MemCopyWordQuota:                 3,
		CodeSizeQuota:                    2,
		CodeCopyQuota:                    3,
		ReturnDataSizeQuota:              2,
		ReturnDataCopyQuota:              3,
		TimestampQuota:                   2,
		HeightQuota:                      2,
		TokenIDQuota:                     2,
		AccountHeightQuota:               2,
		PreviousHashQuota:                2,
		FromBlockHashQuota:               2,
		SeedQuota:                        2,
		RandomQuota:                      2,
		PopQuota:                         2,
		MloadQuota:                       3,
		MstoreQuota:                      3,
		Mstore8Quota:                     3,
		SloadQuota:                       200,
		SstoreResetQuota:                 5000,
		SstoreInitQuota:                  20000,
		SstoreCleanQuota:                 100,
		SstoreNoopQuota:                  200,
		SstoreMemQuota:                   200,
		JumpQuota:                        8,
		JumpiQuota:                       10,
		PcQuota:                          2,
		MsizeQuota:                       2,
		JumpdestQuota:                    1,
		PushQuota:                        3,
		DupQuota:                         3,
		SwapQuota:                        3,
		LogQuota:                         375,
		LogTopicQuota:                    375,
		LogDataQuota:                     8,
		CallMinusQuota:                   10000,
		MemQuotaDivision:                 512,
		SnapshotQuota:                    200,
		CodeQuota:                        200,
		MemQuota:                         3,
		TxQuota:                          21000,
		TxDataQuota:                      68,
		CreateTxRequestQuota:             21000,
		CreateTxResponseQuota:            53000,
		RegisterQuota:                    62200,
		UpdateBlockProducingAddressQuota: 62200,
		RevokeQuota:                      83200,
		WithdrawRewardQuota:              68200,
		VoteQuota:                        62000,
		CancelVoteQuota:                  62000,
		StakeQuota:                       82000,
		CancelStakeQuota:                 73000,
		DelegateStakeQuota:               82000,
		CancelDelegateStakeQuota:         73000,
		IssueQuota:                       104525,
		ReIssueQuota:                     69325,
		BurnQuota:                        48837,
		TransferOwnershipQuota:           58981,
		DisableReIssueQuota:              63125,
		GetTokenInfoQuota:                63200,
	}

	viteQuotaTable     = newViteQuotaTable()
	dexAgentQuotaTable = newDexAgentQuotaTable()
	earthQuotaTable    = newEarthQuotaTable()
)

func newViteQuotaTable() QuotaTable {
	return QuotaTable{
		AddQuota:            2,
		MulQuota:            2,
		SubQuota:            2,
		DivQuota:            3,
		SDivQuota:           5,
		ModQuota:            3,
		SModQuota:           4,
		AddModQuota:         4,
		MulModQuota:         5,
		ExpQuota:            10,
		ExpByteQuota:        50,
		SignExtendQuota:     2,
		LtQuota:             2,
		GtQuota:             2,
		SltQuota:            2,
		SgtQuota:            2,
		EqQuota:             2,
		IsZeroQuota:         1,
		AndQuota:            2,
		OrQuota:             2,
		XorQuota:            2,
		NotQuota:            2,
		ByteQuota:           2,
		ShlQuota:            2,
		ShrQuota:            2,
		SarQuota:            3,
		Blake2bQuota:        20,
		Blake2bWordQuota:    1,
		AddressQuota:        1,
		BalanceQuota:        150,
		CallerQuota:         1,
		CallValueQuota:      1,
		CallDataLoadQuota:   2,
		CallDataSizeQuota:   1,
		CallDataCopyQuota:   3,
		MemCopyWordQuota:    3,
		CodeSizeQuota:       1,
		CodeCopyQuota:       3,
		ReturnDataSizeQuota: 1,
		ReturnDataCopyQuota: 3,
		TimestampQuota:      1,
		HeightQuota:         1,
		TokenIDQuota:        1,
		AccountHeightQuota:  1,
		PreviousHashQuota:   1,
		FromBlockHashQuota:  1,
		SeedQuota:           200,
		RandomQuota:         250,
		PopQuota:            1,
		MloadQuota:          2,
		MstoreQuota:         1,
		Mstore8Quota:        1,
		SloadQuota:          150,
		SstoreResetQuota:    15000,
		SstoreInitQuota:     15000,
		SstoreCleanQuota:    0,
		SstoreNoopQuota:     200,
		SstoreMemQuota:      200,
		JumpQuota:           4,
		JumpiQuota:          4,
		PcQuota:             1,
		MsizeQuota:          1,
		JumpdestQuota:       1,
		PushQuota:           1,
		DupQuota:            1,
		SwapQuota:           2,
		LogQuota:            375,
		LogTopicQuota:       375,
		LogDataQuota:        12,
		CallMinusQuota:      13500,
		MemQuotaDivision:    1024,
		SnapshotQuota:       40,
		CodeQuota:           160,
		MemQuota:            1,

		TxQuota:               21000,
		TxDataQuota:           68,
		CreateTxRequestQuota:  31000,
		CreateTxResponseQuota: 31000,

		RegisterQuota:                           168000,
		UpdateBlockProducingAddressQuota:        168000,
		RevokeQuota:                             126000,
		WithdrawRewardQuota:                     147000,
		VoteQuota:                               84000,
		CancelVoteQuota:                         52500,
		StakeQuota:                              105000,
		CancelStakeQuota:                        105000,
		DelegateStakeQuota:                      115500,
		CancelDelegateStakeQuota:                115500,
		IssueQuota:                              189000,
		ReIssueQuota:                            126000,
		BurnQuota:                               115500,
		TransferOwnershipQuota:                  136500,
		DisableReIssueQuota:                     115500,
		GetTokenInfoQuota:                       31500,
		DexFundDepositQuota:                     10500,
		DexFundWithdrawQuota:                    10500,
		DexFundOpenNewMarketQuota:               31500,
		DexFundPlaceOrderQuota:                  25200,
		DexFundSettleOrdersQuota:                21000,
		DexFundTriggerPeriodJobQuota:            8400,
		DexFundStakeForMiningQuota:              31500,
		DexFundStakeForVipQuota:                 31500,
		DexFundDelegateStakeCallbackQuota:       12600,
		DexFundCancelDelegateStakeCallbackQuota: 16800,
		DexFundGetTokenInfoCallbackQuota:        10500,
		DexFundAdminConfigQuota:                 16800,
		DexFundTradeAdminConfigQuota:            10500,
		DexFundMarketAdminConfigQuota:           10500,
		DexFundTransferTokenOwnershipQuota:      8400,
		DexFundNotifyTimeQuota:                  10500,
		DexFundCreateNewInviterQuota:            18900,
		DexFundBindInviteCodeQuota:              8400,
		DexFundEndorseVxQuota:                   6300,
		DexFundSettleMakerMinedVxQuota:          25200,
	}
}

func newDexAgentQuotaTable() QuotaTable {
	gt := newViteQuotaTable()
	gt.DexFundStakeForSuperVIPQuota = 33600
	gt.DexFundConfigMarketAgentsQuota = 8400
	gt.DexFunPlaceAgentOrderQuota = 25200
	return gt
}

func newEarthQuotaTable() QuotaTable {
	gt := newDexAgentQuotaTable()
	gt.UpdateRewardWithdrawAddressQuota = 168000
	gt.DexFunLockVxForDividendQuota = 31500
	gt.DexFunSwitchConfigQuota = 31500
	gt.DexFundStakeForPrincipalSuperVIPQuota = 10500
	gt.DexFundCancelStakeByIdQuota = 10500
	gt.DexFundDelegateStakeCallbackV2Quota = 31500
	gt.DexFundDelegateCancelStakeCallbackV2Quota = 33000
	return gt
}
