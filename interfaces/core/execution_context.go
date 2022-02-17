package core

import (
	"encoding/binary"
	"math/big"

	"github.com/vitelabs/go-vite/v2/common/types"
)

type ExecutionContext struct {
	ReferrerSendHash types.Hash
	CallbackId big.Int
	Stack  []big.Int
	Memory []byte
}

/*
 *  Execution Context Data Layout:
 *
 *	                                  |<--------------------- stack  ----------------------->|<------------------------- memory -------------------------->|
 *  +-----------------+---------------+------------------+-------------+-------+-------------+------------------+-----------------+------+-----------------+
 *  |  Ref Send Hash  |  Callback Id  |  Stack Dump Size |   Stack 1   |       |   Stack N   | Memory Dump Size |  Memory Slot 1  |      |  Memory Slot N  |
 *  |     32 bytes    |    4 bytes    |      2 bytes     |  1~33 bytes |  ...  |  1~33 bytes |   1~33 bytes     |    1~33 bytes   |  ... |    1~33 bytes   |
 *	|                 |               |  (packed word)   |(packed word)|       |(packed word)|  (packed word)   |  (packed word)  |      |  (packed word)  |
 *  +-----------------+---------------+------------------+-------------+-------+-------------+------------------+-----------------+------+-----------------+
 *	                                                     |<-------- stack dump size -------->|                  |<----------- memory dump size ----------->|
 *
 */


func (ec ExecutionContext) Serialize() ([]byte, error) {
	var data []byte
	// append referrer send hash
	data = append(data, ec.ReferrerSendHash.Bytes()...)
	// append callback id
	data = append(data, ec.CallbackId.FillBytes(make([]byte, 4))...)
	// marshal stack
	data = append(data, MarshalStack(ec.Stack)...)
	// marshal memory
	data = append(data, MarshalMemory(ec.Memory)...)

	return data, nil
}

func (ec *ExecutionContext) Deserialize(buf []byte) error {
	// unmarshal referrer send hash
	hash, err := types.BytesToHash(buf[:32])
	if err != nil {
		return err
	}
	ec.ReferrerSendHash = hash

	// unmarshal callback id
	callback := big.NewInt(0).SetBytes(buf[32:36])
	if callback.Uint64() > 0 {
		ec.CallbackId.Set(callback)
	}

	// unmarshal stack
	stackDumpSize := int(binary.BigEndian.Uint16(buf[36:38]))
	endOfStack := 36+2+stackDumpSize
	ec.Stack = UnmarshalStack(buf[36:endOfStack])

	// unmarshal memory
	ec.Memory = UnmarshalMemory(buf[endOfStack:])

	return nil
}


// PackWord packs a 32-bytes word into the dynamic size format:
//  +----------+------------+
//  |  Length  |   Payload  |
//  |  1 byte  |  0~32 byte |
//  +----------+------------+
func PackWord(word *big.Int) []byte {
	payload := word.Bytes()
	length := byte(len(payload))
	data := make([]byte, 0, 1 + length)
	data = append(data, length)
	data = append(data, payload...)
	return data
}

// UnpackWord unpacks a 32-bytes word from the dynamic size format
func UnpackWord(data []byte) (word *big.Int, length int)  {
	length = int(data[0])
	if length <= 0 || length > 32 {
		return big.NewInt(0), 0
	}
	word = big.NewInt(0).SetBytes(data[1:1 + length])
	return
}

// MarshalStack marshal VM stack
func MarshalStack(stack []big.Int) (dump []byte) {
	// A stack has a maximum size of 1024 elements and contains words of 256 bits.
	// The max length of a stack dump is 32768 bytes.
	// So we reserve 2 bytes for the length of the stack dump.
	data := make([]byte, 0, len(stack)*2)
	for _, item := range stack {
		data = append(data, PackWord(&item)...)
	}

	dump = append(dump, big.NewInt(int64(len(data))).FillBytes(make([]byte, 2))...)
	dump = append(dump, data...)

	return
}

// UnmarshalStack unmarshal VM stack
func UnmarshalStack(dump []byte) []big.Int {
	bodyLength := int(binary.BigEndian.Uint16(dump[:2]))
	if bodyLength == 0 {
		return nil
	}
	stack := make([]big.Int, 0)

	for i := 2; i < bodyLength + 2; {
		item, length := UnpackWord(dump[i:])
		stack = append(stack, *item)
		i += 1 + length
	}
	return stack
}

// MarshalMemory marshal VM memory
func MarshalMemory(mem []byte) []byte {
	memLength := len(mem)
	var data []byte
	for i := 0; i < memLength; i += 32 {
		word := big.NewInt(0)
		if memLength - i < 32 {
			w := mem[i:]
			w = append(w, make([]byte, 32 - (memLength - i))...)
			word.SetBytes(w)
		} else {
			word.SetBytes(mem[i:i+32])
		}
		packed := PackWord(word)
		data = append(data, packed...)
	}

	dump := PackWord(big.NewInt(int64(len(data))))
	dump = append(dump, data...)

	return dump
}

// UnmarshalMemory unmarshal VM memory
func UnmarshalMemory(dump []byte) []byte {
	bodyLength, offset := UnpackWord(dump)
	if bodyLength.Uint64() == 0 {
		return nil
	}
	offset += 1
	var mem []byte
	for i := offset; i < int(bodyLength.Uint64()) + offset; {
		word, wordLength := UnpackWord(dump[i:])
		mem = append(mem, word.FillBytes(make([]byte, 32))...)
		i += 1 + wordLength
	}
	// remove trailing zeros
	for i := len(mem) - 1; i >= 0; i-- {
		if mem[i] != 0 {
			return mem[:i + 1]
		}
	}
	return mem
}
