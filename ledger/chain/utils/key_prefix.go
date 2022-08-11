package chain_utils

// index db
const (
	// hash -> address + height
	AccountBlockHashKeyPrefix = byte(1)

	// address + height -> hash
	AccountBlockHeightKeyPrefix = byte(2)

	// send block hash -> receive block hash
	ReceiveKeyPrefix = byte(3)

	// address + height -> snapshot block height
	ConfirmHeightKeyPrefix = byte(4)

	// toAddress + hash -> empty
	OnRoadKeyPrefix = byte(5)

	// hash -> height
	SnapshotBlockHashKeyPrefix = byte(7)

	// height -> hash
	SnapshotBlockHeightKeyPrefix = byte(8)

	// address -> account id
	AccountAddressKeyPrefix = byte(9)
	// account id -> address
	AccountIdKeyPrefix = byte(10)
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

	GidContractKeyPrefix = byte(9)

	VmLogListKeyPrefix = byte(10)

	CallDepthKeyPrefix = byte(11)
)

// state redo db
const (
	SnapshotKeyPrefix = byte(1)
)

// onroad db

const (
	// toAddress + fromAddress + fromHeight + fromHash -> fromIndex
	OnRoadAddressHeightKeyPrefix = byte(1)
)
