package vm

import (
	"bytes"
	"github.com/vitelabs/go-vite/common/types"
	"math/big"
)

type bitvec []byte
type destinations map[types.Address]bitvec

func (bits *bitvec) set(pos uint64) {
	(*bits)[pos/8] |= 0x80 >> (pos % 8)
}

func (bits *bitvec) set8(pos uint64) {
	(*bits)[pos/8] |= 0xFF >> (pos % 8)
	(*bits)[pos/8+1] |= ^(0xFF >> (pos % 8))
}

// codeSegment checks if the position is in a code segment.
func (bits *bitvec) codeSegment(pos uint64) bool {
	return ((*bits)[pos/8] & (0x80 >> (pos % 8))) == 0
}

// has checks whether code has a JUMPDEST at dest.
func (d destinations) has(addr types.Address, code []byte, dest *big.Int) bool {
	// PC cannot go beyond len(code) and certainly can't be bigger than 63bits.
	// Don't bother checking for JUMPDEST in that case.
	udest := dest.Uint64()
	if dest.BitLen() >= 63 || udest >= uint64(len(code)) {
		return false
	}

	m, analysed := d[addr]
	if !analysed {
		m = codeBitmap(code)
		d[addr] = m
	}
	return opCode(code[udest]) == JUMPDEST && m.codeSegment(udest)
}

// codeBitmap collects data locations in code.
func codeBitmap(code []byte) bitvec {
	// The bitmap is 4 bytes longer than necessary, in case the code
	// ends with a PUSH32, the algorithm will push zeroes onto the
	// bitvector outside the bounds of the actual code.
	bits := make(bitvec, len(code)/8+1+4)
	for pc := uint64(0); pc < uint64(len(code)); {
		op := opCode(code[pc])

		if op >= PUSH1 && op <= PUSH32 {
			numbits := op - PUSH1 + 1
			pc++
			for ; numbits >= 8; numbits -= 8 {
				bits.set8(pc) // 8
				pc += 8
			}
			for ; numbits > 0; numbits-- {
				bits.set(pc)
				pc++
			}
		} else {
			pc++
		}
	}
	return bits
}

var (
	auxCodePrefix  = []byte{0xa1, 0x65, 'b', 'z', 'z', 'r', '0', 0x58, 0x20}
	auxCodeSuffix  = []byte{0x00, 0x29}
	statusCodeList = []opCode{HEIGHT, TIMESTAMP, SEED, DELEGATECALL}
)

// Check whether code includes status reading opcode
func ContainsStatusCode(code []byte) bool {
	if containsAuxCode(code) {
		code = code[:len(code)-43]
	}
	m := codeBitmap(code)
	for i := uint64(0); i < uint64(len(code)); i++ {
		if m.codeSegment(i) {
			for _, c := range statusCodeList {
				if opCode(code[i]) == c {
					return true
				}
			}
		}
	}
	return false
}

func containsAuxCode(code []byte) bool {
	l := len(code)
	if l > 43 && bytes.Equal(code[l-43:l-34], auxCodePrefix) && bytes.Equal(code[l-2:], auxCodeSuffix) {
		return true
	}
	return false
}
