package vm

import (
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/vm/util"
	"math/big"
)

const (
	callDepth  uint16 = 512  // Maximum Depth of call.
	stackLimit uint64 = 1024 // Maximum size of VM stack allowed.

	maxCodeSize       int    = 24575 // Maximum bytecode to permit for a contract
	offChainReaderGas uint64 = 1000000

	snapshotCountMin         uint8 = 0
	snapshotCountMax         uint8 = 75
	snapshotWithSeedCountMin uint8 = 0
	snapshotWithSeedCountMax uint8 = 75

	contractModifyStorageMax int = 100

	retry   = true
	noRetry = false
)

var (
	createContractFee = new(big.Int).Mul(helper.Big10, util.AttovPerVite)
)
