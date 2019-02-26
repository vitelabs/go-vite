package vm

import (
	"bytes"
	"errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/util"
	"math/big"
	"testing"
)

func TestRun(t *testing.T) {
	tests := []struct {
		input, result          []byte
		err                    error
		quotaLeft, quotaRefund uint64
		summary                string
	}{
		{[]byte{byte(PUSH1), 1, byte(PUSH1), 2, byte(ADD), byte(PUSH1), 32, byte(DUP1), byte(SWAP2), byte(SWAP1), byte(MSTORE), byte(PUSH1), 32, byte(SWAP1), byte(RETURN)}, []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 3}, nil, 999964, 0, "return 1+2"},
		{[]byte{0xFA}, []byte{}, errors.New(""), 1000000, 0, "invalid opcode"},
		{[]byte{byte(PUSH1), 32, byte(PUSH17), 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, byte(MLOAD)}, []byte{}, util.ErrGasUintOverflow, 999994, 0, "memory size overflow"},
		{[]byte{byte(POP)}, []byte{}, errors.New(""), 1000000, 0, "make stack error"},
		{[]byte{byte(PUSH1), 32, byte(JUMP)}, []byte{}, errors.New(""), 999989, 0, "execution error"},
		{[]byte{byte(CALLVALUE), byte(DUP1), byte(ISZERO), byte(PUSH2), 0, 11, byte(JUMPI), byte(PUSH1), 0, byte(DUP1), byte(REVERT), byte(JUMPDEST), byte(PUSH1), 32, byte(PUSH1), 0, byte(DUP2), byte(DUP2), byte(MSTORE), byte(RETURN)}, []byte{}, util.ErrExecutionReverted, 999973, 0, "execution revert"},
		{[]byte{byte(CALLVALUE), byte(DUP1), byte(ISZERO), byte(NOT), byte(PUSH2), 0, 12, byte(JUMPI), byte(PUSH1), 0, byte(DUP1), byte(REVERT), byte(JUMPDEST), byte(PUSH1), 32, byte(PUSH1), 0, byte(DUP2), byte(DUP2), byte(MSTORE), byte(RETURN)}, []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 32}, nil, 999957, 0, "jumpi"},
	}
	for _, test := range tests {
		vm := NewVM()
		vm.i = NewInterpreter(1)
		//vm.Debug = true
		sendCallBlock := ledger.AccountBlock{
			AccountAddress: types.Address{},
			ToAddress:      types.Address{},
			BlockType:      ledger.BlockTypeSendCall,
			Data:           test.input,
			Amount:         big.NewInt(10),
			Fee:            big.NewInt(0),
			TokenId:        ledger.ViteTokenId,
		}
		receiveCallBlock := &ledger.AccountBlock{
			AccountAddress: types.Address{},
			ToAddress:      types.Address{},
			BlockType:      ledger.BlockTypeReceive,
		}
		db := NewNoDatabase()
		c := newContract(
			receiveCallBlock,
			db,
			&sendCallBlock,
			sendCallBlock.Data,
			1000000,
			0)
		c.setCallCode(types.Address{}, test.input)
		ret, err := c.run(vm)
		if bytes.Compare(ret, test.result) != 0 ||
			c.quotaLeft != test.quotaLeft ||
			c.quotaRefund != test.quotaRefund ||
			(err == nil && test.err != nil) ||
			(err != nil && test.err == nil) {
			t.Fatalf("contract run failed, summary: %v", test.summary)
		}
	}
}
