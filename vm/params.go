package vm

import (
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/vm/util"
	"math/big"
)

const (
	callDepth  uint16 = 512  // Maximum Depth of call.
	stackLimit uint64 = 1024 // Maximum size of VM stack allowed.

	getAccountBlockByHeightLimit uint64 = 256

	maxCodeSize       = 24575 // Maximum bytecode to permit for a contract
	offChainReaderGas = 1000000

	confirmTimeMin = 0
	confirmTimeMax = 75
	seedCountMin   = 0
	seedCountMax   = 75
)

var (
	createContractFee = new(big.Int).Mul(helper.Big10, util.AttovPerVite)
)
