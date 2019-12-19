package vm

import (
	"encoding/hex"
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/util"
)

func opStop(pc *uint64, vm *VM, c *contract, mem *memory, stack *stack) ([]byte, error) {
	return nil, nil
}

func opAdd(pc *uint64, vm *VM, c *contract, mem *memory, stack *stack) ([]byte, error) {
	x, y := stack.pop(), stack.peek()
	helper.U256(y.Add(x, y))

	c.intPool.Put(x)
	return nil, nil
}

func opMul(pc *uint64, vm *VM, c *contract, mem *memory, stack *stack) ([]byte, error) {
	x, y := stack.pop(), stack.peek()
	helper.U256(y.Mul(x, y))

	c.intPool.Put(x)
	return nil, nil
}

func opSub(pc *uint64, vm *VM, c *contract, mem *memory, stack *stack) ([]byte, error) {
	x, y := stack.pop(), stack.peek()
	helper.U256(y.Sub(x, y))

	c.intPool.Put(x)
	return nil, nil
}

func opDiv(pc *uint64, vm *VM, c *contract, mem *memory, stack *stack) ([]byte, error) {
	x, y := stack.pop(), stack.peek()
	if y.Sign() != 0 {
		helper.U256(y.Div(x, y))
	} else {
		y.SetUint64(0)
	}

	c.intPool.Put(x)
	return nil, nil
}

func opSdiv(pc *uint64, vm *VM, c *contract, mem *memory, stack *stack) ([]byte, error) {
	x, y := helper.S256(stack.pop()), helper.S256(stack.pop())
	res := c.intPool.GetZero()

	if y.Sign() == 0 || x.Sign() == 0 {
		stack.push(res)
	} else {
		if x.Sign() != y.Sign() {
			res.Div(x.Abs(x), y.Abs(y))
			res.Neg(res)
		} else {
			res.Div(x.Abs(x), y.Abs(y))
		}
		stack.push(helper.U256(res))
	}

	c.intPool.Put(x, y)
	return nil, nil
}

func opMod(pc *uint64, vm *VM, c *contract, mem *memory, stack *stack) ([]byte, error) {
	x, y := stack.pop(), stack.pop()
	if y.Sign() == 0 {
		stack.push(x.SetUint64(0))
	} else {
		stack.push(helper.U256(x.Mod(x, y)))
	}

	c.intPool.Put(y)
	return nil, nil
}

func opSmod(pc *uint64, vm *VM, c *contract, mem *memory, stack *stack) ([]byte, error) {
	x, y := helper.S256(stack.pop()), helper.S256(stack.pop())
	res := c.intPool.GetZero()

	if y.Sign() == 0 {
		stack.push(res)
	} else {
		if x.Sign() < 0 {
			res.Mod(x.Abs(x), y.Abs(y))
			res.Neg(res)
		} else {
			res.Mod(x.Abs(x), y.Abs(y))
		}
		stack.push(helper.U256(res))
	}

	c.intPool.Put(x, y)
	return nil, nil
}

func opAddmod(pc *uint64, vm *VM, c *contract, mem *memory, stack *stack) ([]byte, error) {
	x, y, z := stack.pop(), stack.pop(), stack.pop()
	if z.Cmp(helper.Big0) > 0 {
		x.Add(x, y)
		x.Mod(x, z)
		stack.push(helper.U256(x))
	} else {
		stack.push(x.SetUint64(0))
	}

	c.intPool.Put(y, z)
	return nil, nil
}

func opMulmod(pc *uint64, vm *VM, c *contract, mem *memory, stack *stack) ([]byte, error) {
	x, y, z := stack.pop(), stack.pop(), stack.pop()
	if z.Cmp(helper.Big0) > 0 {
		x.Mul(x, y)
		x.Mod(x, z)
		stack.push(helper.U256(x))
	} else {
		stack.push(x.SetUint64(0))
	}

	c.intPool.Put(y, z)
	return nil, nil
}

func opExp(pc *uint64, vm *VM, c *contract, mem *memory, stack *stack) ([]byte, error) {
	base, exponent := stack.pop(), stack.pop()
	stack.push(helper.Exp(base, exponent))

	c.intPool.Put(base, exponent)
	return nil, nil
}

func opSignExtend(pc *uint64, vm *VM, c *contract, mem *memory, stack *stack) ([]byte, error) {
	back := stack.pop()
	if back.Cmp(helper.Big31) < 0 {
		bit := uint(back.Uint64()*8 + 7)
		num := stack.pop()
		mask := back.Lsh(helper.Big1, bit)
		mask.Sub(mask, helper.Big1)
		if num.Bit(int(bit)) > 0 {
			num.Or(num, mask.Not(mask))
		} else {
			num.And(num, mask)
		}
		stack.push(helper.U256(num))
	}

	c.intPool.Put(back)
	return nil, nil
}

func opLt(pc *uint64, vm *VM, c *contract, mem *memory, stack *stack) ([]byte, error) {
	x, y := stack.pop(), stack.peek()
	if x.Cmp(y) < 0 {
		y.SetUint64(1)
	} else {
		y.SetUint64(0)
	}
	c.intPool.Put(x)
	return nil, nil
}

func opGt(pc *uint64, vm *VM, c *contract, mem *memory, stack *stack) ([]byte, error) {
	x, y := stack.pop(), stack.peek()
	if x.Cmp(y) > 0 {
		y.SetUint64(1)
	} else {
		y.SetUint64(0)
	}
	c.intPool.Put(x)
	return nil, nil
}

func opSlt(pc *uint64, vm *VM, c *contract, mem *memory, stack *stack) ([]byte, error) {
	x, y := stack.pop(), stack.peek()

	xSign := x.Cmp(helper.Tt255)
	ySign := y.Cmp(helper.Tt255)

	switch {
	case xSign >= 0 && ySign < 0:
		y.SetUint64(1)

	case xSign < 0 && ySign >= 0:
		y.SetUint64(0)

	default:
		if x.Cmp(y) < 0 {
			y.SetUint64(1)
		} else {
			y.SetUint64(0)
		}
	}
	c.intPool.Put(x)
	return nil, nil
}

func opSgt(pc *uint64, vm *VM, c *contract, mem *memory, stack *stack) ([]byte, error) {
	x, y := stack.pop(), stack.peek()

	xSign := x.Cmp(helper.Tt255)
	ySign := y.Cmp(helper.Tt255)

	switch {
	case xSign >= 0 && ySign < 0:
		y.SetUint64(0)

	case xSign < 0 && ySign >= 0:
		y.SetUint64(1)

	default:
		if x.Cmp(y) > 0 {
			y.SetUint64(1)
		} else {
			y.SetUint64(0)
		}
	}
	c.intPool.Put(x)
	return nil, nil
}

func opEq(pc *uint64, vm *VM, c *contract, mem *memory, stack *stack) ([]byte, error) {
	x, y := stack.pop(), stack.peek()
	if x.Cmp(y) == 0 {
		y.SetUint64(1)
	} else {
		y.SetUint64(0)
	}
	c.intPool.Put(x)
	return nil, nil
}

func opIszero(pc *uint64, vm *VM, c *contract, mem *memory, stack *stack) ([]byte, error) {
	x := stack.peek()
	if x.Sign() > 0 {
		x.SetUint64(0)
	} else {
		x.SetUint64(1)
	}
	return nil, nil
}

func opAnd(pc *uint64, vm *VM, c *contract, mem *memory, stack *stack) ([]byte, error) {
	x, y := stack.pop(), stack.peek()
	y.And(x, y)

	c.intPool.Put(x)
	return nil, nil
}

func opOr(pc *uint64, vm *VM, c *contract, mem *memory, stack *stack) ([]byte, error) {
	x, y := stack.pop(), stack.peek()
	y.Or(x, y)

	c.intPool.Put(x)
	return nil, nil
}

func opXor(pc *uint64, vm *VM, c *contract, mem *memory, stack *stack) ([]byte, error) {
	x, y := stack.pop(), stack.peek()
	y.Xor(x, y)

	c.intPool.Put(x)
	return nil, nil
}

func opNot(pc *uint64, vm *VM, c *contract, mem *memory, stack *stack) ([]byte, error) {
	x := stack.peek()
	helper.U256(x.Not(x))
	return nil, nil
}

func opByte(pc *uint64, vm *VM, c *contract, mem *memory, stack *stack) ([]byte, error) {
	th, val := stack.pop(), stack.peek()
	if th.Cmp(helper.Big32) < 0 {
		b := helper.Byte(val, helper.WordSize, int(th.Int64()))
		val.SetUint64(uint64(b))
	} else {
		val.SetUint64(0)
	}

	c.intPool.Put(th)
	return nil, nil
}

func opSHL(pc *uint64, vm *VM, c *contract, mem *memory, stack *stack) ([]byte, error) {
	shift, value := helper.U256(stack.pop()), helper.U256(stack.peek())
	if shift.Cmp(helper.Big256) >= 0 {
		value.SetUint64(0)
	} else {
		helper.U256(value.Lsh(value, uint(shift.Uint64())))
	}

	c.intPool.Put(shift)
	return nil, nil
}

func opSHR(pc *uint64, vm *VM, c *contract, mem *memory, stack *stack) ([]byte, error) {
	shift, value := helper.U256(stack.pop()), helper.U256(stack.peek())
	if shift.Cmp(helper.Big256) >= 0 {
		value.SetUint64(0)
	} else {
		helper.U256(value.Rsh(value, uint(shift.Uint64())))
	}

	c.intPool.Put(shift)
	return nil, nil
}

func opSAR(pc *uint64, vm *VM, c *contract, mem *memory, stack *stack) ([]byte, error) {
	shift, value := helper.U256(stack.pop()), helper.S256(stack.pop())
	if shift.Cmp(helper.Big256) >= 0 {
		if value.Sign() > 0 {
			value.SetUint64(0)
		} else {
			value.SetInt64(-1)
		}
		stack.push(helper.U256(value))
	} else {
		stack.push(helper.U256(value.Rsh(value, uint(shift.Uint64()))))
	}

	c.intPool.Put(shift)
	return nil, nil
}

func opBlake2b(pc *uint64, vm *VM, c *contract, mem *memory, stack *stack) ([]byte, error) {
	offset, size := stack.pop(), stack.pop()
	data := mem.get(offset.Int64(), size.Int64())
	stack.push(c.intPool.Get().SetBytes(crypto.Hash256(data)))

	c.intPool.Put(offset, size)
	return nil, nil
}

func opAddress(pc *uint64, vm *VM, c *contract, mem *memory, stack *stack) ([]byte, error) {
	stack.push(c.intPool.Get().SetBytes(c.block.AccountAddress.Bytes()))
	return nil, nil
}

func opBalance(pc *uint64, vm *VM, c *contract, mem *memory, stack *stack) ([]byte, error) {
	tokenTypeIDBig := stack.pop()
	tokenTypeID, _ := types.BigToTokenTypeId(tokenTypeIDBig)
	b, err := c.db.GetBalance(&tokenTypeID)
	util.DealWithErr(err)
	stack.push(c.intPool.Get().Set(b))

	c.intPool.Put(tokenTypeIDBig, b)
	return nil, nil
}

func opOffchainBalance(pc *uint64, vm *VM, c *contract, mem *memory, stack *stack) ([]byte, error) {
	c.intPool.Put(stack.pop())
	stack.push(c.intPool.GetZero())
	return nil, nil
}

func opCaller(pc *uint64, vm *VM, c *contract, mem *memory, stack *stack) ([]byte, error) {
	stack.push(c.intPool.Get().SetBytes(c.sendBlock.AccountAddress.Bytes()))
	return nil, nil
}

func opOffchainCaller(pc *uint64, vm *VM, c *contract, mem *memory, stack *stack) ([]byte, error) {
	stack.push(c.intPool.GetZero())
	return nil, nil
}

func opCallValue(pc *uint64, vm *VM, c *contract, mem *memory, stack *stack) ([]byte, error) {
	stack.push(c.intPool.Get().Set(c.sendBlock.Amount))
	return nil, nil
}

func opOffchainCallValue(pc *uint64, vm *VM, c *contract, mem *memory, stack *stack) ([]byte, error) {
	stack.push(c.intPool.GetZero())
	return nil, nil
}

func opCallDataLoad(pc *uint64, vm *VM, c *contract, mem *memory, stack *stack) ([]byte, error) {
	offset := stack.pop()
	stack.push(c.intPool.Get().SetBytes(helper.GetDataBig(c.data, offset, helper.Big32)))

	c.intPool.Put(offset)
	return nil, nil
}

func opCallDataSize(pc *uint64, vm *VM, c *contract, mem *memory, stack *stack) ([]byte, error) {
	stack.push(c.intPool.Get().SetInt64(int64(len(c.data))))
	return nil, nil
}

func opCallDataCopy(pc *uint64, vm *VM, c *contract, mem *memory, stack *stack) ([]byte, error) {
	var (
		memOffset  = stack.pop()
		dataOffset = stack.pop()
		length     = stack.pop()
	)
	mem.set(memOffset.Uint64(), length.Uint64(), helper.GetDataBig(c.data, dataOffset, length))

	c.intPool.Put(memOffset, dataOffset, length)
	return nil, nil
}

func opCodeSize(pc *uint64, vm *VM, c *contract, mem *memory, stack *stack) ([]byte, error) {
	stack.push(c.intPool.Get().SetInt64(int64(len(c.code))))
	return nil, nil
}

func opCodeCopy(pc *uint64, vm *VM, c *contract, mem *memory, stack *stack) ([]byte, error) {
	var (
		memOffset  = stack.pop()
		codeOffset = stack.pop()
		length     = stack.pop()
	)
	mem.set(memOffset.Uint64(), length.Uint64(), helper.GetDataBig(c.code, codeOffset, length))

	c.intPool.Put(memOffset, codeOffset, length)
	return nil, nil
}

func opExtCodeSize(pc *uint64, vm *VM, c *contract, mem *memory, stack *stack) ([]byte, error) {
	addrBig := stack.peek()
	contractAddress, _ := types.BigToAddress(addrBig)
	_, code := util.GetContractCode(c.db, &contractAddress, vm.globalStatus)
	addrBig.SetInt64(int64(len(code)))
	return nil, nil
}

func opExtCodeCopy(pc *uint64, vm *VM, c *contract, mem *memory, stack *stack) ([]byte, error) {
	var (
		addrBig    = stack.pop()
		memOffset  = stack.pop()
		codeOffset = stack.pop()
		length     = stack.pop()
	)
	contractAddress, _ := types.BigToAddress(addrBig)
	_, code := util.GetContractCode(c.db, &contractAddress, vm.globalStatus)
	codeCopy := helper.GetDataBig(code, codeOffset, length)
	mem.set(memOffset.Uint64(), length.Uint64(), codeCopy)

	c.intPool.Put(addrBig, memOffset, codeOffset, length)
	return nil, nil
}

func opReturnDataSize(pc *uint64, vm *VM, c *contract, mem *memory, stack *stack) ([]byte, error) {
	stack.push(c.intPool.Get().SetUint64(uint64(len(c.returnData))))
	return nil, nil
}

func opReturnDataCopy(pc *uint64, vm *VM, c *contract, mem *memory, stack *stack) ([]byte, error) {
	var (
		memOffset  = stack.pop()
		dataOffset = stack.pop()
		length     = stack.pop()

		end = c.intPool.Get().Add(dataOffset, length)
	)
	defer c.intPool.Put(memOffset, dataOffset, length, end)

	if end.BitLen() > 64 || uint64(len(c.returnData)) < end.Uint64() {
		return nil, util.ErrReturnDataOutOfBounds
	}
	mem.set(memOffset.Uint64(), length.Uint64(), c.returnData[dataOffset.Uint64():end.Uint64()])

	return nil, nil
}

func opTimestamp(pc *uint64, vm *VM, c *contract, mem *memory, stack *stack) ([]byte, error) {
	stack.push(helper.U256(c.intPool.Get().SetInt64(vm.globalStatus.SnapshotBlock().Timestamp.Unix())))
	return nil, nil
}

func opOffchainTimestamp(pc *uint64, vm *VM, c *contract, mem *memory, stack *stack) ([]byte, error) {
	stack.push(c.intPool.GetZero())
	return nil, nil
}

func opHeight(pc *uint64, vm *VM, c *contract, mem *memory, stack *stack) ([]byte, error) {
	stack.push(helper.U256(c.intPool.Get().SetUint64(vm.globalStatus.SnapshotBlock().Height)))
	return nil, nil
}

func opOffchainHeight(pc *uint64, vm *VM, c *contract, mem *memory, stack *stack) ([]byte, error) {
	stack.push(c.intPool.GetZero())
	return nil, nil
}

func opTokenID(pc *uint64, vm *VM, c *contract, mem *memory, stack *stack) ([]byte, error) {
	stack.push(c.intPool.Get().SetBytes(c.sendBlock.TokenId.Bytes()))
	return nil, nil
}

func opOffchainTokenID(pc *uint64, vm *VM, c *contract, mem *memory, stack *stack) ([]byte, error) {
	stack.push(c.intPool.GetZero())
	return nil, nil
}

func opAccountHeight(pc *uint64, vm *VM, c *contract, mem *memory, stack *stack) ([]byte, error) {
	ab, err := c.db.PrevAccountBlock()
	util.DealWithErr(err)
	if ab != nil {
		stack.push(helper.U256(c.intPool.Get().SetUint64(ab.Height)))
	} else {
		stack.push(c.intPool.GetZero())
	}
	return nil, nil
}

func opOffchainAccountHeight(pc *uint64, vm *VM, c *contract, mem *memory, stack *stack) ([]byte, error) {
	stack.push(c.intPool.GetZero())
	return nil, nil
}

func opAccountHash(pc *uint64, vm *VM, c *contract, mem *memory, stack *stack) ([]byte, error) {
	prevAccountBlock, err := c.db.PrevAccountBlock()
	util.DealWithErr(err)
	if prevAccountBlock == nil {
		stack.push(c.intPool.GetZero())
	} else {
		stack.push(c.intPool.Get().SetBytes(prevAccountBlock.Hash.Bytes()))
	}
	return nil, nil
}

func opOffchainAccountHash(pc *uint64, vm *VM, c *contract, mem *memory, stack *stack) ([]byte, error) {
	c.intPool.Put(stack.pop())
	stack.push(c.intPool.GetZero())
	return nil, nil
}

func opFromHash(pc *uint64, vm *VM, c *contract, mem *memory, stack *stack) ([]byte, error) {
	stack.push(c.intPool.Get().SetBytes(c.block.FromBlockHash.Bytes()))
	return nil, nil
}

func opOffchainFromHash(pc *uint64, vm *VM, c *contract, mem *memory, stack *stack) ([]byte, error) {
	stack.push(c.intPool.GetZero())
	return nil, nil
}

func opSeed(pc *uint64, vm *VM, c *contract, mem *memory, stack *stack) ([]byte, error) {
	seed, err := vm.globalStatus.Seed()
	util.DealWithErr(err)
	stack.push(c.intPool.Get().SetUint64(seed))
	return nil, nil
}

func opOffchainSeed(pc *uint64, vm *VM, c *contract, mem *memory, stack *stack) ([]byte, error) {
	stack.push(c.intPool.GetZero())
	return nil, nil
}

func opRandom(pc *uint64, vm *VM, c *contract, mem *memory, stack *stack) ([]byte, error) {
	seed, err := vm.globalStatus.Random()
	util.DealWithErr(err)
	stack.push(c.intPool.Get().SetUint64(seed))
	return nil, nil
}

func opOffchainRandom(pc *uint64, vm *VM, c *contract, mem *memory, stack *stack) ([]byte, error) {
	stack.push(c.intPool.GetZero())
	return nil, nil
}

func opPop(pc *uint64, vm *VM, c *contract, mem *memory, stack *stack) ([]byte, error) {
	c.intPool.Put(stack.pop())
	return nil, nil
}

func opMload(pc *uint64, vm *VM, c *contract, mem *memory, stack *stack) ([]byte, error) {
	offset := stack.pop()
	val := c.intPool.Get().SetBytes(mem.get(offset.Int64(), helper.WordSize))
	stack.push(val)

	c.intPool.Put(offset)
	return nil, nil
}

func opMstore(pc *uint64, vm *VM, c *contract, mem *memory, stack *stack) ([]byte, error) {
	// pop amount of the stack
	mStart, val := stack.pop(), stack.pop()
	mem.set32(mStart.Uint64(), val)

	c.intPool.Put(mStart, val)
	return nil, nil
}

func opMstore8(pc *uint64, vm *VM, c *contract, mem *memory, stack *stack) ([]byte, error) {
	off, val := stack.pop().Int64(), stack.pop().Int64()
	mem.store[off] = byte(val & 0xff)

	return nil, nil
}

func opSLoad(pc *uint64, vm *VM, c *contract, mem *memory, stack *stack) ([]byte, error) {
	loc := stack.peek()
	locHash, _ := types.BigToHash(loc)
	val := util.GetValue(c.db, locHash.Bytes())
	loc.SetBytes(val)
	return nil, nil
}

func opSStore(pc *uint64, vm *VM, c *contract, mem *memory, stack *stack) ([]byte, error) {
	loc, val := stack.pop(), stack.pop()
	locHash, _ := types.BigToHash(loc)
	util.SetValue(c.db, locHash.Bytes(), val.Bytes())

	c.intPool.Put(loc, val)
	return nil, nil
}

func opOffchainSStore(pc *uint64, vm *VM, c *contract, mem *memory, stack *stack) ([]byte, error) {
	c.intPool.Put(stack.pop(), stack.pop())
	return nil, nil
}

func opJump(pc *uint64, vm *VM, c *contract, mem *memory, stack *stack) ([]byte, error) {
	pos := stack.pop()
	if !c.jumpdests.has(c.codeAddr, c.code, pos) {
		nop := c.getOp(pos.Uint64())
		nodeConfig.log.Error("invalid jump destination", nop, pos)
		return nil, util.ErrInvalidJumpDestination
	}
	*pc = pos.Uint64()

	c.intPool.Put(pos)
	return nil, nil
}

func opJumpi(pc *uint64, vm *VM, c *contract, mem *memory, stack *stack) ([]byte, error) {
	pos, cond := stack.pop(), stack.pop()
	if cond.Sign() != 0 {
		if !c.jumpdests.has(c.codeAddr, c.code, pos) {
			nop := c.getOp(pos.Uint64())
			nodeConfig.log.Error("invalid jump destination", nop, pos)
			return nil, util.ErrInvalidJumpDestination
		}
		*pc = pos.Uint64()
	} else {
		*pc++
	}

	c.intPool.Put(pos, cond)
	return nil, nil
}

func opPc(pc *uint64, vm *VM, c *contract, mem *memory, stack *stack) ([]byte, error) {
	stack.push(c.intPool.Get().SetUint64(*pc))
	return nil, nil
}

func opMsize(pc *uint64, vm *VM, c *contract, mem *memory, stack *stack) ([]byte, error) {
	stack.push(c.intPool.Get().SetInt64(int64(mem.len())))
	return nil, nil
}

func opJumpdest(pc *uint64, vm *VM, c *contract, mem *memory, stack *stack) ([]byte, error) {
	return nil, nil
}

// make push instruction function
func makePush(size uint64, pushByteSize int) executionFunc {
	return func(pc *uint64, vm *VM, c *contract, mem *memory, stack *stack) ([]byte, error) {
		codeLen := len(c.code)

		startMin := int(*pc + 1)
		if codeLen < startMin {
			startMin = codeLen
		}

		endMin := startMin + pushByteSize
		if codeLen < endMin {
			endMin = codeLen
		}

		integer := c.intPool.Get()
		stack.push(integer.SetBytes(helper.RightPadBytes(c.code[startMin:endMin], pushByteSize)))

		*pc += size
		return nil, nil
	}
}

// make dup instruction function
func makeDup(size int64) executionFunc {
	return func(pc *uint64, vm *VM, c *contract, mem *memory, stack *stack) ([]byte, error) {
		stack.dup(c.intPool, int(size))
		return nil, nil
	}
}

// make swap instruction function
func makeSwap(size int64) executionFunc {
	// switch n + 1 otherwise n would be swapped with n
	size++
	return func(pc *uint64, vm *VM, c *contract, mem *memory, stack *stack) ([]byte, error) {
		stack.swap(int(size))
		return nil, nil
	}
}

func makeLog(size int) executionFunc {
	return func(pc *uint64, vm *VM, c *contract, mem *memory, stack *stack) ([]byte, error) {
		topics := make([]types.Hash, size)
		mStart, mSize := stack.pop(), stack.pop()
		for i := 0; i < size; i++ {
			topic := stack.pop()
			topics[i], _ = types.BigToHash(topic)
			c.intPool.Put(topic)
		}

		d := mem.get(mStart.Int64(), mSize.Int64())
		c.db.AddLog(&ledger.VmLog{Topics: topics, Data: d})

		if nodeConfig.IsDebug {
			topicsStr := ""
			if size > 0 {
				for _, t := range topics {
					topicsStr = topicsStr + t.String() + ","
				}
				topicsStr = topicsStr[:len(topicsStr)-1]
			}
			var params []interface{}
			var name string
			var err error
			if contractAbi, ok := GetContractABI(c.block.AccountAddress); ok {
				name, params, err = contractAbi.DirectUnpackEvent(topics, d)
				if err != nil {
					nodeConfig.log.Warn("unpack event failed", "topics", topics, "data", d, "err", err)
				}
			}
			if len(name) > 0 {
				nodeConfig.log.Info("vm log",
					"blockType", c.block.BlockType,
					"address", c.block.AccountAddress.String(),
					"height", c.block.Height,
					"fromHash", c.block.FromBlockHash.String(),
					"topics", topicsStr,
					"data", hex.EncodeToString(d),
					"logName", name,
					"params", params)
			} else {
				nodeConfig.log.Info("vm log",
					"blockType", c.block.BlockType,
					"address", c.block.AccountAddress.String(),
					"height", c.block.Height,
					"fromHash", c.block.FromBlockHash.String(),
					"topics", topicsStr,
					"data", hex.EncodeToString(d))
			}
		}

		c.intPool.Put(mStart, mSize)
		return nil, nil
	}
}

func makeOffchainLog(size int) executionFunc {
	return func(pc *uint64, vm *VM, c *contract, mem *memory, stack *stack) ([]byte, error) {
		c.intPool.Put(stack.pop(), stack.pop())
		for i := 0; i < size; i++ {
			c.intPool.Put(stack.pop())
		}
		return nil, nil
	}
}

func opDelegateCall(pc *uint64, vm *VM, c *contract, mem *memory, stack *stack) ([]byte, error) {
	addrBig, inOffset, inSize, outOffset, outSize := stack.pop(), stack.pop(), stack.pop(), stack.pop(), stack.pop()
	contractAddress, _ := types.BigToAddress(addrBig)
	data := mem.get(inOffset.Int64(), inSize.Int64())
	ret, err := vm.delegateCall(contractAddress, data, c)
	if err == nil || err == util.ErrExecutionReverted {
		mem.set(outOffset.Uint64(), outSize.Uint64(), ret)
	}
	if err != nil {
		stack.push(c.intPool.GetZero())
	} else {
		stack.push(c.intPool.Get().SetUint64(1))
	}

	c.intPool.Put(addrBig, inOffset, inSize, outOffset, outSize)
	return ret, nil
}

func opCall(pc *uint64, vm *VM, c *contract, mem *memory, stack *stack) ([]byte, error) {
	toAddrBig, tokenIDBig, amount, inOffset, inSize := stack.pop(), stack.pop(), stack.pop(), stack.pop(), stack.pop()
	toAddress, _ := types.BigToAddress(toAddrBig)
	tokenID, _ := types.BigToTokenTypeId(tokenIDBig)
	data := mem.get(inOffset.Int64(), inSize.Int64())
	vm.AppendBlock(
		util.MakeRequestBlock(
			c.block.AccountAddress,
			toAddress,
			ledger.BlockTypeSendCall,
			amount,
			tokenID,
			data))
	return nil, nil
}

func opOffchainCall(pc *uint64, vm *VM, c *contract, mem *memory, stack *stack) ([]byte, error) {
	c.intPool.Put(stack.pop(), stack.pop(), stack.pop(), stack.pop(), stack.pop())
	return nil, nil
}

func opCall2(pc *uint64, vm *VM, c *contract, mem *memory, stack *stack) ([]byte, error) {
	returnData, err := opCall(pc, vm, c, mem, stack)
	if err != nil {
		return returnData, err
	}
	h := util.ComputeSendBlockHash(c.block, vm.sendBlockList[len(vm.sendBlockList)-1], uint8(len(vm.sendBlockList)-1))
	stack.push(c.intPool.Get().SetBytes(h.Bytes()))
	return returnData, nil
}

func opOffchainCall2(pc *uint64, vm *VM, c *contract, mem *memory, stack *stack) ([]byte, error) {
	opOffchainCall(pc, vm, c, mem, stack)
	stack.push(c.intPool.GetZero())
	return nil, nil
}

func opReturn(pc *uint64, vm *VM, c *contract, mem *memory, stack *stack) ([]byte, error) {
	offset, size := stack.pop(), stack.pop()
	ret := mem.getPtr(offset.Int64(), size.Int64())

	c.intPool.Put(offset, size)
	return ret, nil
}

func opRevert(pc *uint64, vm *VM, c *contract, mem *memory, stack *stack) ([]byte, error) {
	offset, size := stack.pop(), stack.pop()
	ret := mem.getPtr(offset.Int64(), size.Int64())

	c.intPool.Put(offset, size)
	return ret, nil
}

func opOffchainRevert(pc *uint64, vm *VM, c *contract, mem *memory, stack *stack) ([]byte, error) {
	c.intPool.Put(stack.pop(), stack.pop())
	return nil, nil
}
