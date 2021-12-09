package metadata

import (
	"bytes"
	"encoding/binary"
	"errors"
	"github.com/vitelabs/go-vite/common/types"
	ledger "github.com/vitelabs/go-vite/interfaces/core"
	"math/big"
)
type SendType int64

const (
	AsyncCall SendType = 0
	SyncCall = 1
	Callback = 2
)

var (
	// HeadIdentifier defines the identifier of metadata
	HeadIdentifier = []byte{0, 0, 0, 0, 0x76, 0x01}
)

type ExecutionContext struct {
	Stack  []*big.Int
	Memory []byte
}

/*
 *	SyncCall data layout:
 *
 *  |<----------------------------------------- metadata ----------------------------------------------->|<--- calldata --->|
 *  +-----------------+----------+----------------+-----------------+---------------+--------------------+------------------+
 *  | Head Identifier |   Type   | Context Length |  Ref Send Hash  |  Callback Id  |      Context       |    Call Data     |
 *  |     6 bytes     |  1 byte  |   1~33 bytes   |     32 bytes    |    4 bytes    |                    |                  |
 *  |  0x000000007601 |   0x01   |  (packed word) |                 |               |                    |                  |
 *	+-----------------+----------+----------------+-----------------+---------------+--------------------+------------------+
 *  |<--------------- head length --------------->|                                 |<--context length-->|
 *
 *
 *  context layout:
 *
 *	|<--------------------- stack  ----------------------->|<------------------------- memory -------------------------->|
 *  +------------------+-------------+-------+-------------+------------------+-----------------+------+-----------------+
 *  |  Stack Dump Size |   Stack 1   |       |   Stack N   | Memory Dump Size |  Memory Slot 1  |      |  Memory Slot N  |
 *  |    1~33 bytes    |  1~33 bytes |  ...  |  1~33 bytes |   1~33 bytes     |    1~33 bytes   |  ... |    1~33 bytes   |
 *	|  (packed word)   |(packed word)|       |(packed word)|  (packed word)   |  (packed word)  |      |  (packed word)  |
 *  +------------------+-------------+-------+-------------+------------------+-----------------+------+-----------------+
 *	                   |<-------- stack dump size -------->|                  |<----------- memory dump size ----------->|
 *
 *
 *	Callback data layout:
 *
 *  |<----------------- metadata ----------------->|<----------- calldata ----------->|
 *  +-----------------+----------+-----------------+---------------+------------------+
 *  | Head Identifier |   Type   |  Ref Send Hash  |  Callback Id  |    Return Data   |
 *  |     6 bytes     |  1 byte  |     32 bytes    |    4 bytes    |                  |
 *  |  0x000000007601 |   0x02   |                 |               |                  |
 *	+-----------------+----------+-----------------+---------------+------------------+
 *  |<------- head length ------>|
 *
 */


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

// HasMetaData checks whether raw calldata has metadata
func HasMetaData(data []byte) bool {
	if data == nil || len(data) < 11 {
		return false
	}
	return bytes.Equal(HeadIdentifier, data[:6])
}

// GetSendType parse send transaction type
func GetSendType(data []byte) SendType {
	if !HasMetaData(data) {
		return AsyncCall
	}
	if data[6] == byte(SyncCall) {
		return SyncCall
	}
	if  data[6] == byte(Callback) {
		return Callback
	}
	return AsyncCall
}

// GetHeadLength returns the length of metadata.
//  - SyncCall: [head][type][context_length]
//  - Callback: [head][type]
func GetHeadLength(data []byte) int {
	sendType := GetSendType(data)
	if sendType == SyncCall {
		return len(HeadIdentifier) + 1 + 1 + int(data[7])
	} else if sendType == Callback {
		return len(HeadIdentifier) + 1
	} else {
		return 0
	}
}

// GetReferencedSendHash parse referenced send-transaction hash from metadata
func GetReferencedSendHash(data []byte) (types.Hash, error) {
	offset := GetHeadLength(data)
	if offset > 0 && offset + 32 < len(data) {
		return types.BytesToHash(data[offset:offset + 32])
	}

	return types.Hash{0}, errors.New("invalid metadata")
}

// GetMetaDataLength parse the length of metadata
func GetMetaDataLength(data []byte) int {
	sendType := GetSendType(data)
	if sendType == SyncCall {
		word, length := UnpackWord(data[7:])
		if length == 0 {
			return 0
		}
		contextLength := int(word.Uint64())
		metaDataLength := GetHeadLength(data) + 32 + 4 + contextLength
		return metaDataLength
	} else if sendType == Callback {
		return GetHeadLength(data) + 32
	} else {
		return 0
	}
}

// GetCalldata extract the original calldata from the encoded calldata with metadata
func GetCalldata(data []byte) []byte {
	metaDataLength := GetMetaDataLength(data)
	if metaDataLength > 0 {
		return data[metaDataLength:]
	}
	return data
}

// GetContext returns the VM context
func GetContext(data []byte) *ExecutionContext {
	if GetSendType(data) == SyncCall {
		offset := GetHeadLength(data) + 32 + 4
		metaDataLength := GetMetaDataLength(data)
		if metaDataLength < offset || metaDataLength > len(data) {
			return nil
		}
		contextData := data[offset:metaDataLength]
		// unmarshal context
		context := UnmarshalContext(contextData)
		return context
	}
	return nil
}

// MarshalStack marshal VM stack to metadata
func MarshalStack(stack []*big.Int) (dump []byte) {
	// A stack has a maximum size of 1024 elements and contains words of 256 bits.
	// The max length of a stack dump is 32768 bytes.
	// So we reserve 2 bytes for the length of the stack dump.
	data := make([]byte, 0, len(stack)*2)
	for _, item := range stack {
		data = append(data, PackWord(item)...)
	}

	dump = append(dump, big.NewInt(int64(len(data))).FillBytes(make([]byte, 2))...)
	dump = append(dump, data...)

	return
}

// UnmarshalStack unmarshal VM stack from metadata
func UnmarshalStack(dump []byte) []*big.Int {
	bodyLength := int(binary.BigEndian.Uint16(dump[:2]))
	stack := make([]*big.Int, 0)

	for i := 2; i < bodyLength + 2; {
		item, length := UnpackWord(dump[i:])
		stack = append(stack, item)
		i += 1 + length
	}
	return stack
}

// MarshalMemory marshal VM memory to metadata
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

// UnmarshalMemory unmarshal VM memory from metadata
func UnmarshalMemory(dump []byte) []byte {
	bodyLength, offset := UnpackWord(dump)
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

// MarshalContext marshal VM context to metadata
func MarshalContext(context *ExecutionContext) []byte {
	var data []byte
	data = append(data, MarshalStack(context.Stack)...)
	data = append(data, MarshalMemory(context.Memory)...)
	return data
}

// UnmarshalContext unmarshal VM context from metadata
func UnmarshalContext(dump []byte) *ExecutionContext {
	context := &ExecutionContext{}
	stackDumpLength := 2 + int(binary.BigEndian.Uint16(dump[:2]))
	context.Stack = UnmarshalStack(dump[:stackDumpLength])
	context.Memory = UnmarshalMemory(dump[stackDumpLength:])
	return context
}

// EncodeSyncCallData encode the calldata for a sync call
func EncodeSyncCallData(sendHash types.Hash, callbackId *big.Int, context *ExecutionContext, calldata []byte) ([]byte, error) {
	// marshal context
	contextData := MarshalContext(context)
	// metadata: [head_identifier][type][context_length][ref_send_hash][callback_id][context]
	contextDataLength := big.NewInt(int64(len(contextData)))
	// init data slice
	dataCapacity := len(HeadIdentifier) + 1 + 32 + 32 + 4 + int(contextDataLength.Uint64())
	data := make([]byte, 0, dataCapacity)
	// append HeadIdentifier
	data = append(data, HeadIdentifier...)
	// append SendType
	data = append(data, SyncCall)
	// append metadata length
	data = append(data, PackWord(contextDataLength)...)
	// append referenced send hash
	data = append(data, sendHash.Bytes()...)
	// append callback id
	data = append(data, callbackId.FillBytes(make([]byte, 4))...)
	// append context
	data = append(data, contextData...)
	// append calldata
	data = append(data, calldata...)

	return data, nil
}

// EncodeCallbackData encode the calldata for a callback (return transaction to a sync call)
func EncodeCallbackData(originSendBlock *ledger.AccountBlock , returnData []byte) ([]byte, error) {
	sendCallData := originSendBlock.Data
	if GetSendType(sendCallData) == SyncCall {
		// metadata: [head_identifier][type][ref_send_hash] | [callback_id][returnData]
		metaDataLength := len(HeadIdentifier) + 1 + 32
		// init data slice
		data := make([]byte, 0, metaDataLength)
		// append HeadIdentifier
		data = append(data, HeadIdentifier...)
		// append SendType
		data = append(data, Callback)
		// append referenced send hash
		data = append(data, originSendBlock.Hash.Bytes()...)
		// append callback id
		offset := GetHeadLength(sendCallData) + 32
		callbackId := sendCallData[offset:offset + 4]
		data = append(data, callbackId...)
		// append return data
		data = append(data, returnData...)

		return data, nil
	}
	return nil, errors.New("invalid origin send block")
}


