package vm

import (
	"github.com/vitelabs/go-vite/common/helper"
	"math/big"
)

const (
	quickStepGas    uint64 = 2
	fastestStepGas  uint64 = 3
	fastStepGas     uint64 = 5
	midStepGas      uint64 = 8
	slowStepGas     uint64 = 10
	extStepGas      uint64 = 20
	extCodeSizeGas  uint64 = 700
	extCodeCopyGas  uint64 = 700
	balanceGas      uint64 = 400
	sLoadGas        uint64 = 200
	expByteGas      uint64 = 50
	quadCoeffDiv    uint64 = 512   // Divisor for the quadratic particle of the memory cost equation.
	logGas          uint64 = 375   // Per LOG* operation.
	logTopicGas     uint64 = 375   // Multiplied by the * of the LOG*, per LOG transaction. e.g. LOG0 incurs 0 * c_txLogTopicGas, LOG4 incurs 4 * c_txLogTopicGas.
	logDataGas      uint64 = 8     // Per byte in a LOG* operation's data.
	blake2bGas      uint64 = 30    // Once per Blake2b operation.
	blake2bWordGas  uint64 = 6     // Once per word of the Blake2b operation's data.
	sstoreSetGas    uint64 = 20000 // Once per SSTORE operation
	sstoreResetGas  uint64 = 5000  // Once per SSTORE operation if the zeroness changes from zero.
	sstoreClearGas  uint64 = 5000  // Once per SSTORE operation if the zeroness doesn't change.
	sstoreRefundGas uint64 = 15000 // Once per SSTORE operation if the zeroness changes to zero.
	jumpdestGas     uint64 = 1     // Jumpdest gas cost.
	callGas         uint64 = 700   // Once per CALL operation & message call transaction.
	contractCodeGas uint64 = 200   // Per byte in contract code
	copyGas         uint64 = 3     //
	memoryGas       uint64 = 3     // Times the address of the (highest referenced byte in memory + 1). NOTE: referencing happens on read, write and in instructions such as RETURN and CALL.

	// callCreateDepth          uint64 = 1024    // Maximum Depth of call/create stack.
	stackLimit uint64 = 1024 // Maximum size of VM stack allowed.

	// precompiled contract gas
	registerGas               uint64 = 62200
	updateRegistrationGas     uint64 = 62200
	cancelRegisterGas         uint64 = 83200
	rewardGas                 uint64 = 83200
	calcRewardGasPerPage      uint64 = 200
	maxRewardCount            uint64 = 150000000
	voteGas                   uint64 = 62000
	cancelVoteGas             uint64 = 62000
	pledgeGas                 uint64 = 21000
	cancelPledgeGas           uint64 = 103400
	createConsensusGroupGas   uint64 = 62200
	cancelConsensusGroupGas   uint64 = 83200
	reCreateConsensusGroupGas uint64 = 62200
	mintageGas                uint64 = 83200
	mintageCancelPledgeGas    uint64 = 83200

	// TODO test minPledgeHeight uint64 = 3600 * 24 * 3 // Minimum pledge height
	minPledgeHeight uint64 = 1

	cgNodeCountMin   uint8 = 3       // Minimum node count of consensus group
	cgNodeCountMax   uint8 = 101     // Maximum node count of consensus group
	cgIntervalMin    int64 = 1       // Minimum interval of consensus group in second
	cgIntervalMax    int64 = 10 * 60 // Maximum interval of consensus group in second
	cgPerCountMin    int64 = 1
	cgPerCountMax    int64 = 10 * 60
	cgPerIntervalMin int64 = 1
	cgPerIntervalMax int64 = 10 * 60
	// TODO test createConsensusGroupPledgeHeight uint64 = 3600 * 24 * 3
	createConsensusGroupPledgeHeight uint64 = 1

	rewardHeightLimit     uint64 = 60 * 30 // Get snapshot block reward of 30 minutes before current
	dbPageSize            uint64 = 10000   // Batch get snapshot blocks from vm database to calc snapshot block reward
	getBlockByHeightLimit uint64 = 256

	tokenNameLengthMax   int = 40 // Maximum length of a token name(include)
	tokenSymbolLengthMax int = 10 // Maximum length of a token symbol(include)
	// TODO test mintagePledgeHeight  uint64 = 3600 * 24 * 30 * 3
	mintagePledgeHeight uint64 = 1

	//CallValueTransferGas  uint64 = 9000  // Paid for CALL when the amount transfer is non-zero.
	//CallNewAccountGas     uint64 = 25000 // Paid for CALL when the destination address didn't exist prior.
	//CallStipend           uint64 = 2300  // Free gas given at beginning of call.

	MaxCodeSize = 24576 // Maximum bytecode to permit for a contract
)

var (
	createContractFee = new(big.Int).Mul(helper.Big10, attovPerVite)

	viteTotalSupply                  = new(big.Int).Mul(big.NewInt(1e9), attovPerVite)
	rewardPerBlock                   = new(big.Int).Div(viteTotalSupply, big.NewInt(1051200000)) // Reward pre snapshot block, rewardPreBlock * blockNumPerYear / viteTotalSupply = 3%
	attovPerVite                     = big.NewInt(1e18)
	pledgeAmountMin                  = new(big.Int).Mul(big.NewInt(10), attovPerVite)
	mintageFee                       = new(big.Int).Mul(big.NewInt(1e3), attovPerVite) // Mintage cost choice 1, destroy ViteToken
	mintagePledgeAmount              = new(big.Int).Mul(big.NewInt(1e5), attovPerVite) // Mintage cost choice 2, pledge ViteToken for 3 month
	createConsensusGroupPledgeAmount = new(big.Int).Mul(big.NewInt(1000), attovPerVite)
)
