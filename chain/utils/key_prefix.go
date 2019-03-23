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

// state db
const (
	BalanceKeyPrefix = byte(1)

	StorageKeyPrefix = byte(2)

	CodeKeyPrefix = byte(3)

	ContractMetaKeyPrefix = byte(4)

	VmLogListKeyPrefix = byte(5)
)

// mv db
const (
	KeyIdKeyPrefix = byte(1)

	ValueIdKeyPrefix = byte(2)

	LatestValueKeyPrefix = byte(3)

	UndoKeyPrefix = byte(4)

	MvDbLatestLocationKeyPrefix = byte(5)
)
