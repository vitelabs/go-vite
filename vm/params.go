package vm

import (
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/vm/util"
	"math/big"
)

const (
	quickStepGas   uint64 = 2
	fastestStepGas uint64 = 3
	fastStepGas    uint64 = 5
	midStepGas     uint64 = 8
	slowStepGas    uint64 = 10
	extStepGas     uint64 = 20
	extCodeSizeGas uint64 = 700
	extCodeCopyGas uint64 = 700
	balanceGas     uint64 = 400
	sLoadGas       uint64 = 200
	expByteGas     uint64 = 50
	quadCoeffDiv   uint64 = 512 // Divisor for the quadratic particle of the memory cost equation.
	logGas         uint64 = 375 // Per LOG* operation.
	logTopicGas    uint64 = 375 // Multiplied by the * of the LOG*, per LOG transaction. e.g. LOG0 incurs 0 * c_txLogTopicGas, LOG4 incurs 4 * c_txLogTopicGas.
	logDataGas     uint64 = 8   // Per byte in a LOG* operation's data.
	blake2bGas     uint64 = 30  // Once per Blake2b operation.
	blake2bWordGas uint64 = 6   // Once per word of the Blake2b operation's data.

	sstoreNoopGas             uint64 = 200
	sstoreInitGas             uint64 = 20000
	sstoreCleanGas            uint64 = 5000
	sstoreDirtyGas            uint64 = 200
	sstoreClearRefundGas      uint64 = 15000
	sstoreResetClearRefundGas uint64 = 19800
	sstoreResetRefundGas      uint64 = 4800

	jumpdestGas     uint64 = 1 // Jumpdest gas cost.
	callMinusGas    uint64 = 10000
	contractCodeGas uint64 = 200 // Per byte in contract code
	copyGas         uint64 = 3   //
	memoryGas       uint64 = 3   // Times the address of the (highest referenced byte in memory + 1). NOTE: referencing happens on read, write and in instructions such as RETURN and CALL.

	callDepth  uint64 = 512  // Maximum Depth of call.
	stackLimit uint64 = 1024 // Maximum size of VM stack allowed.

	getAccountBlockByHeightLimit uint64 = 256

	//CallValueTransferGas  uint64 = 9000  // Paid for CALL when the amount transfer is non-zero.
	//CallNewAccountGas     uint64 = 25000 // Paid for CALL when the destination address didn't exist prior.
	//CallStipend           uint64 = 2300  // Free gas given at beginning of call.

	MaxCodeSize       = 24575 // Maximum bytecode to permit for a contract
	offChainReaderGas = 1000000

	confirmTimeMin = 0
	confirmTimeMax = 75
)

var (
	createContractFee = new(big.Int).Mul(helper.Big10, util.AttovPerVite)
)
