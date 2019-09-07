package vm

import (
	"math/big"
)

type (
	executionFunc       func(pc *uint64, vm *VM, contract *contract, memory *memory, stack *stack) ([]byte, error)
	gasFunc             func(*VM, *contract, *stack, *memory, uint64) (uint64, bool, error) // last parameter is the requested memory size as a uint64
	stackValidationFunc func(*stack) error
	memorySizeFunc      func(*stack) *big.Int
)

type operation struct {
	// execute is the operation function
	execute executionFunc
	// gasCost is the gas function and returns the gas required for execution
	gasCost gasFunc
	// validateStack validates the stack (size) for the operation
	validateStack stackValidationFunc
	// memorySize returns the memory size required for the operation
	memorySize memorySizeFunc

	halts   bool // indicates whether the operation should halt further execution
	jumps   bool // indicates whether the program counter should not increment
	writes  bool // determines whether this a state modifying operation
	valid   bool // indication whether the retrieved operation is valid and known
	reverts bool // determines whether the operation reverts state (implicitly halts)
	returns bool // determines whether the operations sets the return data content
}

var (
	simpleInstructionSet         = newSimpleInstructionSet()
	offchainSimpleInstructionSet = newOffchainSimpleInstructionSet()
	randInstructionSet           = newRandInstructionSet()
	offchainRandInstructionSet   = newRandOffchainInstructionSet()
)

func newRandInstructionSet() [256]operation {
	instructionSet := newSimpleInstructionSet()
	instructionSet[RANDOM] = operation{
		execute:       opRandom,
		gasCost:       gasRandom,
		validateStack: makeStackFunc(0, 1),
		valid:         true,
	}
	return instructionSet
}

func newRandOffchainInstructionSet() [256]operation {
	instructionSet := newOffchainSimpleInstructionSet()
	instructionSet[RANDOM] = operation{
		execute:       opOffchainRandom,
		gasCost:       gasRandom,
		validateStack: makeStackFunc(0, 1),
		valid:         true,
	}
	return instructionSet
}

func newSimpleInstructionSet() [256]operation {
	instructionSet := newBaseInstructionSet()
	instructionSet[FROMHASH] = operation{
		execute:       opFromHash,
		gasCost:       gasFromhash,
		validateStack: makeStackFunc(0, 1),
		valid:         true,
	}
	instructionSet[SEED] = operation{
		execute:       opSeed,
		gasCost:       gasSeed,
		validateStack: makeStackFunc(0, 1),
		valid:         true,
	}
	instructionSet[CALLER] = operation{
		execute:       opCaller,
		gasCost:       gasCaller,
		validateStack: makeStackFunc(0, 1),
		valid:         true,
	}
	instructionSet[CALLVALUE] = operation{
		execute:       opCallValue,
		gasCost:       gasCallvalue,
		validateStack: makeStackFunc(0, 1),
		valid:         true,
	}
	instructionSet[TIMESTAMP] = operation{
		execute:       opTimestamp,
		gasCost:       gasTimestamp,
		validateStack: makeStackFunc(0, 1),
		valid:         true,
	}
	instructionSet[HEIGHT] = operation{
		execute:       opHeight,
		gasCost:       gasHeight,
		validateStack: makeStackFunc(0, 1),
		valid:         true,
	}
	instructionSet[TOKENID] = operation{
		execute:       opTokenID,
		gasCost:       gasTokenid,
		validateStack: makeStackFunc(0, 1),
		valid:         true,
	}
	instructionSet[SSTORE] = operation{
		execute:       opSStore,
		gasCost:       gasSStore,
		validateStack: makeStackFunc(2, 0),
		valid:         true,
		writes:        true,
	}
	instructionSet[LOG0] = operation{
		execute:       makeLog(0),
		gasCost:       makeGasLog(0),
		validateStack: makeStackFunc(2, 0),
		memorySize:    memoryLog,
		valid:         true,
		writes:        true,
	}
	instructionSet[LOG1] = operation{
		execute:       makeLog(1),
		gasCost:       makeGasLog(1),
		validateStack: makeStackFunc(3, 0),
		memorySize:    memoryLog,
		valid:         true,
		writes:        true,
	}
	instructionSet[LOG2] = operation{
		execute:       makeLog(2),
		gasCost:       makeGasLog(2),
		validateStack: makeStackFunc(4, 0),
		memorySize:    memoryLog,
		valid:         true,
		writes:        true,
	}
	instructionSet[LOG3] = operation{
		execute:       makeLog(3),
		gasCost:       makeGasLog(3),
		validateStack: makeStackFunc(5, 0),
		memorySize:    memoryLog,
		valid:         true,
		writes:        true,
	}
	instructionSet[LOG4] = operation{
		execute:       makeLog(4),
		gasCost:       makeGasLog(4),
		validateStack: makeStackFunc(6, 0),
		memorySize:    memoryLog,
		valid:         true,
		writes:        true,
	}
	instructionSet[CALL] = operation{
		execute:       opCall,
		gasCost:       gasCall,
		validateStack: makeStackFunc(5, 0),
		memorySize:    memoryCall,
		valid:         true,
	}
	instructionSet[REVERT] = operation{
		execute:       opRevert,
		gasCost:       gasRevert,
		validateStack: makeStackFunc(2, 0),
		memorySize:    memoryRevert,
		valid:         true,
		reverts:       true,
		returns:       true,
	}
	return instructionSet
}

func newOffchainSimpleInstructionSet() [256]operation {
	instructionSet := newBaseInstructionSet()
	instructionSet[FROMHASH] = operation{
		execute:       opOffchainFromHash,
		gasCost:       gasFromhash,
		validateStack: makeStackFunc(0, 1),
		valid:         true,
	}
	instructionSet[SEED] = operation{
		execute:       opOffchainSeed,
		gasCost:       gasSeed,
		validateStack: makeStackFunc(0, 1),
		valid:         true,
	}
	instructionSet[CALLER] = operation{
		execute:       opOffchainCaller,
		gasCost:       gasCaller,
		validateStack: makeStackFunc(0, 1),
		valid:         true,
	}
	instructionSet[CALLVALUE] = operation{
		execute:       opOffchainCallValue,
		gasCost:       gasCallvalue,
		validateStack: makeStackFunc(0, 1),
		valid:         true,
	}
	instructionSet[TIMESTAMP] = operation{
		execute:       opOffchainTimestamp,
		gasCost:       gasTimestamp,
		validateStack: makeStackFunc(0, 1),
		valid:         true,
	}
	instructionSet[HEIGHT] = operation{
		execute:       opOffchainHeight,
		gasCost:       gasHeight,
		validateStack: makeStackFunc(0, 1),
		valid:         true,
	}
	instructionSet[TOKENID] = operation{
		execute:       opOffchainTokenID,
		gasCost:       gasTokenid,
		validateStack: makeStackFunc(0, 1),
		valid:         true,
	}
	instructionSet[SSTORE] = operation{
		execute:       opOffchainSStore,
		gasCost:       gasSStore,
		validateStack: makeStackFunc(2, 0),
		valid:         true,
		writes:        true,
	}
	instructionSet[LOG0] = operation{
		execute:       makeOffchainLog(0),
		gasCost:       makeGasLog(0),
		validateStack: makeStackFunc(2, 0),
		memorySize:    memoryLog,
		valid:         true,
		writes:        true,
	}
	instructionSet[LOG1] = operation{
		execute:       makeOffchainLog(1),
		gasCost:       makeGasLog(1),
		validateStack: makeStackFunc(3, 0),
		memorySize:    memoryLog,
		valid:         true,
		writes:        true,
	}
	instructionSet[LOG2] = operation{
		execute:       makeOffchainLog(2),
		gasCost:       makeGasLog(2),
		validateStack: makeStackFunc(4, 0),
		memorySize:    memoryLog,
		valid:         true,
		writes:        true,
	}
	instructionSet[LOG3] = operation{
		execute:       makeOffchainLog(3),
		gasCost:       makeGasLog(3),
		validateStack: makeStackFunc(5, 0),
		memorySize:    memoryLog,
		valid:         true,
		writes:        true,
	}
	instructionSet[LOG4] = operation{
		execute:       makeOffchainLog(4),
		gasCost:       makeGasLog(4),
		validateStack: makeStackFunc(6, 0),
		memorySize:    memoryLog,
		valid:         true,
		writes:        true,
	}
	instructionSet[CALL] = operation{
		execute:       opOffchainCall,
		gasCost:       gasCall,
		validateStack: makeStackFunc(5, 0),
		memorySize:    memoryCall,
		valid:         true,
	}
	instructionSet[REVERT] = operation{
		execute:       opOffchainRevert,
		gasCost:       gasRevert,
		validateStack: makeStackFunc(2, 0),
		memorySize:    memoryRevert,
		valid:         true,
		halts:         true,
	}
	return instructionSet
}

func newBaseInstructionSet() [256]operation {
	return [256]operation{
		STOP: {
			execute:       opStop,
			gasCost:       constGasFunc(0),
			validateStack: makeStackFunc(0, 0),
			halts:         true,
			valid:         true,
		},
		ADD: {
			execute:       opAdd,
			gasCost:       gasAdd,
			validateStack: makeStackFunc(2, 1),
			valid:         true,
		},
		MUL: {
			execute:       opMul,
			gasCost:       gasMul,
			validateStack: makeStackFunc(2, 1),
			valid:         true,
		},
		SUB: {
			execute:       opSub,
			gasCost:       gasSub,
			validateStack: makeStackFunc(2, 1),
			valid:         true,
		},
		DIV: {
			execute:       opDiv,
			gasCost:       gasDiv,
			validateStack: makeStackFunc(2, 1),
			valid:         true,
		},
		SDIV: {
			execute:       opSdiv,
			gasCost:       gasSdiv,
			validateStack: makeStackFunc(2, 1),
			valid:         true,
		},
		MOD: {
			execute:       opMod,
			gasCost:       gasMod,
			validateStack: makeStackFunc(2, 1),
			valid:         true,
		},
		SMOD: {
			execute:       opSmod,
			gasCost:       gasSmod,
			validateStack: makeStackFunc(2, 1),
			valid:         true,
		},
		ADDMOD: {
			execute:       opAddmod,
			gasCost:       gasAddmod,
			validateStack: makeStackFunc(3, 1),
			valid:         true,
		},
		MULMOD: {
			execute:       opMulmod,
			gasCost:       gasMulmod,
			validateStack: makeStackFunc(3, 1),
			valid:         true,
		},
		EXP: {
			execute:       opExp,
			gasCost:       gasExp,
			validateStack: makeStackFunc(2, 1),
			valid:         true,
		},
		SIGNEXTEND: {
			execute:       opSignExtend,
			gasCost:       gasSignextend,
			validateStack: makeStackFunc(2, 1),
			valid:         true,
		},
		LT: {
			execute:       opLt,
			gasCost:       gasLt,
			validateStack: makeStackFunc(2, 1),
			valid:         true,
		},
		GT: {
			execute:       opGt,
			gasCost:       gasGt,
			validateStack: makeStackFunc(2, 1),
			valid:         true,
		},
		SLT: {
			execute:       opSlt,
			gasCost:       gasSlt,
			validateStack: makeStackFunc(2, 1),
			valid:         true,
		},
		SGT: {
			execute:       opSgt,
			gasCost:       gasSgt,
			validateStack: makeStackFunc(2, 1),
			valid:         true,
		},
		EQ: {
			execute:       opEq,
			gasCost:       gasEq,
			validateStack: makeStackFunc(2, 1),
			valid:         true,
		},
		ISZERO: {
			execute:       opIszero,
			gasCost:       gasIszero,
			validateStack: makeStackFunc(1, 1),
			valid:         true,
		},
		AND: {
			execute:       opAnd,
			gasCost:       gasAnd,
			validateStack: makeStackFunc(2, 1),
			valid:         true,
		},
		OR: {
			execute:       opOr,
			gasCost:       gasOr,
			validateStack: makeStackFunc(2, 1),
			valid:         true,
		},
		XOR: {
			execute:       opXor,
			gasCost:       gasXor,
			validateStack: makeStackFunc(2, 1),
			valid:         true,
		},
		NOT: {
			execute:       opNot,
			gasCost:       gasNot,
			validateStack: makeStackFunc(1, 1),
			valid:         true,
		},
		BYTE: {
			execute:       opByte,
			gasCost:       gasByte,
			validateStack: makeStackFunc(2, 1),
			valid:         true,
		},
		SHL: {
			execute:       opSHL,
			gasCost:       gasShl,
			validateStack: makeStackFunc(2, 1),
			valid:         true,
		},
		SHR: {
			execute:       opSHR,
			gasCost:       gasShr,
			validateStack: makeStackFunc(2, 1),
			valid:         true,
		},
		SAR: {
			execute:       opSAR,
			gasCost:       gasSar,
			validateStack: makeStackFunc(2, 1),
			valid:         true,
		},
		BLAKE2B: {
			execute:       opBlake2b,
			gasCost:       gasBlake2b,
			validateStack: makeStackFunc(2, 1),
			memorySize:    memoryBlake2b,
			valid:         true,
		},
		ADDRESS: {
			execute:       opAddress,
			gasCost:       gasAddress,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		CALLDATALOAD: {
			execute:       opCallDataLoad,
			gasCost:       gasCalldataload,
			validateStack: makeStackFunc(1, 1),
			valid:         true,
		},
		CALLDATASIZE: {
			execute:       opCallDataSize,
			gasCost:       gasCalldatasize,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		CALLDATACOPY: {
			execute:       opCallDataCopy,
			gasCost:       gasCallDataCopy,
			validateStack: makeStackFunc(3, 0),
			memorySize:    memoryCallDataCopy,
			valid:         true,
		},
		CODESIZE: {
			execute:       opCodeSize,
			gasCost:       gasCodesize,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		CODECOPY: {
			execute:       opCodeCopy,
			gasCost:       gasCodeCopy,
			validateStack: makeStackFunc(3, 0),
			memorySize:    memoryCodeCopy,
			valid:         true,
		},
		RETURNDATASIZE: {
			execute:       opReturnDataSize,
			gasCost:       gasReturndatasize,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		RETURNDATACOPY: {
			execute:       opReturnDataCopy,
			gasCost:       gasReturnDataCopy,
			validateStack: makeStackFunc(3, 0),
			memorySize:    memoryReturnDataCopy,
			valid:         true,
		},
		POP: {
			execute:       opPop,
			gasCost:       gasPop,
			validateStack: makeStackFunc(1, 0),
			valid:         true,
		},
		MLOAD: {
			execute:       opMload,
			gasCost:       gasMLoad,
			validateStack: makeStackFunc(1, 1),
			memorySize:    memoryMLoad,
			valid:         true,
		},
		MSTORE: {
			execute:       opMstore,
			gasCost:       gasMStore,
			validateStack: makeStackFunc(2, 0),
			memorySize:    memoryMStore,
			valid:         true,
		},
		MSTORE8: {
			execute:       opMstore8,
			gasCost:       gasMStore8,
			memorySize:    memoryMStore8,
			validateStack: makeStackFunc(2, 0),
			valid:         true,
		},
		SLOAD: {
			execute:       opSLoad,
			gasCost:       gasSload,
			validateStack: makeStackFunc(1, 1),
			valid:         true,
		},
		JUMP: {
			execute:       opJump,
			gasCost:       gasJump,
			validateStack: makeStackFunc(1, 0),
			jumps:         true,
			valid:         true,
		},
		JUMPI: {
			execute:       opJumpi,
			gasCost:       gasJumpi,
			validateStack: makeStackFunc(2, 0),
			jumps:         true,
			valid:         true,
		},
		PC: {
			execute:       opPc,
			gasCost:       gasPc,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		MSIZE: {
			execute:       opMsize,
			gasCost:       gasMsize,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		JUMPDEST: {
			execute:       opJumpdest,
			gasCost:       gasJumpdest,
			validateStack: makeStackFunc(0, 0),
			valid:         true,
		},
		PUSH1: {
			execute:       makePush(1, 1),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		PUSH2: {
			execute:       makePush(2, 2),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		PUSH3: {
			execute:       makePush(3, 3),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		PUSH4: {
			execute:       makePush(4, 4),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		PUSH5: {
			execute:       makePush(5, 5),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		PUSH6: {
			execute:       makePush(6, 6),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		PUSH7: {
			execute:       makePush(7, 7),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		PUSH8: {
			execute:       makePush(8, 8),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		PUSH9: {
			execute:       makePush(9, 9),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		PUSH10: {
			execute:       makePush(10, 10),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		PUSH11: {
			execute:       makePush(11, 11),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		PUSH12: {
			execute:       makePush(12, 12),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		PUSH13: {
			execute:       makePush(13, 13),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		PUSH14: {
			execute:       makePush(14, 14),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		PUSH15: {
			execute:       makePush(15, 15),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		PUSH16: {
			execute:       makePush(16, 16),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		PUSH17: {
			execute:       makePush(17, 17),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		PUSH18: {
			execute:       makePush(18, 18),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		PUSH19: {
			execute:       makePush(19, 19),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		PUSH20: {
			execute:       makePush(20, 20),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		PUSH21: {
			execute:       makePush(21, 21),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		PUSH22: {
			execute:       makePush(22, 22),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		PUSH23: {
			execute:       makePush(23, 23),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		PUSH24: {
			execute:       makePush(24, 24),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		PUSH25: {
			execute:       makePush(25, 25),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		PUSH26: {
			execute:       makePush(26, 26),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		PUSH27: {
			execute:       makePush(27, 27),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		PUSH28: {
			execute:       makePush(28, 28),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		PUSH29: {
			execute:       makePush(29, 29),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		PUSH30: {
			execute:       makePush(30, 30),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		PUSH31: {
			execute:       makePush(31, 31),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		PUSH32: {
			execute:       makePush(32, 32),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		DUP1: {
			execute:       makeDup(1),
			gasCost:       gasDup,
			validateStack: makeDupStackFunc(1),
			valid:         true,
		},
		DUP2: {
			execute:       makeDup(2),
			gasCost:       gasDup,
			validateStack: makeDupStackFunc(2),
			valid:         true,
		},
		DUP3: {
			execute:       makeDup(3),
			gasCost:       gasDup,
			validateStack: makeDupStackFunc(3),
			valid:         true,
		},
		DUP4: {
			execute:       makeDup(4),
			gasCost:       gasDup,
			validateStack: makeDupStackFunc(4),
			valid:         true,
		},
		DUP5: {
			execute:       makeDup(5),
			gasCost:       gasDup,
			validateStack: makeDupStackFunc(5),
			valid:         true,
		},
		DUP6: {
			execute:       makeDup(6),
			gasCost:       gasDup,
			validateStack: makeDupStackFunc(6),
			valid:         true,
		},
		DUP7: {
			execute:       makeDup(7),
			gasCost:       gasDup,
			validateStack: makeDupStackFunc(7),
			valid:         true,
		},
		DUP8: {
			execute:       makeDup(8),
			gasCost:       gasDup,
			validateStack: makeDupStackFunc(8),
			valid:         true,
		},
		DUP9: {
			execute:       makeDup(9),
			gasCost:       gasDup,
			validateStack: makeDupStackFunc(9),
			valid:         true,
		},
		DUP10: {
			execute:       makeDup(10),
			gasCost:       gasDup,
			validateStack: makeDupStackFunc(10),
			valid:         true,
		},
		DUP11: {
			execute:       makeDup(11),
			gasCost:       gasDup,
			validateStack: makeDupStackFunc(11),
			valid:         true,
		},
		DUP12: {
			execute:       makeDup(12),
			gasCost:       gasDup,
			validateStack: makeDupStackFunc(12),
			valid:         true,
		},
		DUP13: {
			execute:       makeDup(13),
			gasCost:       gasDup,
			validateStack: makeDupStackFunc(13),
			valid:         true,
		},
		DUP14: {
			execute:       makeDup(14),
			gasCost:       gasDup,
			validateStack: makeDupStackFunc(14),
			valid:         true,
		},
		DUP15: {
			execute:       makeDup(15),
			gasCost:       gasDup,
			validateStack: makeDupStackFunc(15),
			valid:         true,
		},
		DUP16: {
			execute:       makeDup(16),
			gasCost:       gasDup,
			validateStack: makeDupStackFunc(16),
			valid:         true,
		},
		SWAP1: {
			execute:       makeSwap(1),
			gasCost:       gasSwap,
			validateStack: makeSwapStackFunc(2),
			valid:         true,
		},
		SWAP2: {
			execute:       makeSwap(2),
			gasCost:       gasSwap,
			validateStack: makeSwapStackFunc(3),
			valid:         true,
		},
		SWAP3: {
			execute:       makeSwap(3),
			gasCost:       gasSwap,
			validateStack: makeSwapStackFunc(4),
			valid:         true,
		},
		SWAP4: {
			execute:       makeSwap(4),
			gasCost:       gasSwap,
			validateStack: makeSwapStackFunc(5),
			valid:         true,
		},
		SWAP5: {
			execute:       makeSwap(5),
			gasCost:       gasSwap,
			validateStack: makeSwapStackFunc(6),
			valid:         true,
		},
		SWAP6: {
			execute:       makeSwap(6),
			gasCost:       gasSwap,
			validateStack: makeSwapStackFunc(7),
			valid:         true,
		},
		SWAP7: {
			execute:       makeSwap(7),
			gasCost:       gasSwap,
			validateStack: makeSwapStackFunc(8),
			valid:         true,
		},
		SWAP8: {
			execute:       makeSwap(8),
			gasCost:       gasSwap,
			validateStack: makeSwapStackFunc(9),
			valid:         true,
		},
		SWAP9: {
			execute:       makeSwap(9),
			gasCost:       gasSwap,
			validateStack: makeSwapStackFunc(10),
			valid:         true,
		},
		SWAP10: {
			execute:       makeSwap(10),
			gasCost:       gasSwap,
			validateStack: makeSwapStackFunc(11),
			valid:         true,
		},
		SWAP11: {
			execute:       makeSwap(11),
			gasCost:       gasSwap,
			validateStack: makeSwapStackFunc(12),
			valid:         true,
		},
		SWAP12: {
			execute:       makeSwap(12),
			gasCost:       gasSwap,
			validateStack: makeSwapStackFunc(13),
			valid:         true,
		},
		SWAP13: {
			execute:       makeSwap(13),
			gasCost:       gasSwap,
			validateStack: makeSwapStackFunc(14),
			valid:         true,
		},
		SWAP14: {
			execute:       makeSwap(14),
			gasCost:       gasSwap,
			validateStack: makeSwapStackFunc(15),
			valid:         true,
		},
		SWAP15: {
			execute:       makeSwap(15),
			gasCost:       gasSwap,
			validateStack: makeSwapStackFunc(16),
			valid:         true,
		},
		SWAP16: {
			execute:       makeSwap(16),
			gasCost:       gasSwap,
			validateStack: makeSwapStackFunc(17),
			valid:         true,
		},
		RETURN: {
			execute:       opReturn,
			gasCost:       gasReturn,
			validateStack: makeStackFunc(2, 0),
			memorySize:    memoryReturn,
			halts:         true,
			valid:         true,
		},
		BALANCE: {
			execute:       opBalance,
			gasCost:       gasBalance,
			validateStack: makeStackFunc(1, 1),
			valid:         true,
		},
		ACCOUNTHEIGHT: {
			execute:       opAccountHeight,
			gasCost:       gasAccountheight,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		PREVHASH: {
			execute:       opAccountHash,
			gasCost:       gasPrevhash,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
	}
}
