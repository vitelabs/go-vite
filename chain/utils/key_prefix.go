package chain_utils

// index db
const (
	AccountBlockHashKeyPrefix = byte(1)

	AccountBlockHeightKeyPrefix = byte(2)

	ReceiveKeyPrefix = byte(3)

	ConfirmHeightKeyPrefix = byte(4)

	OnRoadKeyPrefix = byte(5)

	OnRoadReverseKeyPrefix = byte(6)

	SnapshotBlockHashKeyPrefix = byte(7)

	SnapshotBlockHeightKeyPrefix = byte(8)

	AccountAddressKeyPrefix = byte(9)

	AccountIdKeyPrefix = byte(10)

	LatestOnRoadIdKeyPrefix = byte(11)

	IndexDbLatestLocationKeyPrefix = byte(12)
)

// state db
const (
	StorageKeyPrefix = byte(1)

	StorageHistoryKeyPrefix = byte(2)

	BalanceKeyPrefix = byte(3)

	BalanceHistoryKeyPrefix = byte(4)

	CodeKeyPrefix = byte(5)

	// CodeHistoryKeyPrefix = byte(6)

	ContractMetaKeyPrefix = byte(7)

	// ContractMetaHistoryKeyPrefix = byte(8)

	VmLogListKeyPrefix = byte(9)

	CallDepthKeyPrefix = byte(10)

	UndoLocationKeyPrefix = byte(11)

	StateDbLocationKeyPrefix = byte(12)
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
