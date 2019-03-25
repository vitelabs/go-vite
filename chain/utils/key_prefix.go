package chain_utils

// index db
const (
	AccountBlockHashKeyPrefix = byte(1)

	ReceiveKeyPrefix = byte(2)

	OnRoadKeyPrefix = byte(3)

	SnapshotBlockHashKeyPrefix = byte(4)

	SnapshotBlockHeightKeyPrefix = byte(5)

	AccountAddressKeyPrefix = byte(6)

	AccountIdKeyPrefix = byte(7)

	ConfirmHeightKeyPrefix = byte(8)

	AccountBlockHeightKeyPrefix = byte(9)

	OnRoadReverseKeyPrefix = byte(10)

	LatestOnRoadIdKeyPrefix = byte(11)

	IndexDbLatestLocationKeyPrefix = byte(12)
)

// state_bak db
const (
	StorageKeyPrefix = byte(1)

	StorageHistoryKeyPrefix = byte(2)

	BalanceKeyPrefix = byte(3)

	BalanceHistoryKeyPrefix = byte(4)

	CodeKeyPrefix = byte(5)

	ContractMetaKeyPrefix = byte(6)

	VmLogListKeyPrefix = byte(7)

	UndoLocationKeyPrefix = byte(8)

	StateDbLocationKeyPrefix = byte(9)
)

// mv db
const (
//KeyIdKeyPrefix = byte(1)
//
//ValueIdKeyPrefix = byte(2)
//
//LatestValueKeyPrefix = byte(3)
//
//UndoKeyPrefix = byte(4)
//
//MvDbLatestLocationKeyPrefix = byte(5)
)
