package vm

import (
	"bytes"
	"github.com/vitelabs/go-vite/common/types"
	"math/big"
)

var (
	AddressRegister, _       = types.BytesToAddress([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1})
	AddressVote, _           = types.BytesToAddress([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2})
	AddressMortgage, _       = types.BytesToAddress([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 3})
	AddressConsensusGroup, _ = types.BytesToAddress([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4})
)

type precompiledContract interface {
	doSend(vm *VM, block VmAccountBlock, quotaLeft, quotaRefund uint64) (uint64, uint64, error)
	doReceive(vm *VM, block VmAccountBlock) error
}

var simpleContracts = map[types.Address]precompiledContract{
	AddressRegister:       &register{},
	AddressVote:           &vote{},
	AddressMortgage:       &mortgage{},
	AddressConsensusGroup: &consensusGroup{},
}

func getPrecompiledContract(address types.Address) (precompiledContract, bool) {
	p, ok := simpleContracts[address]
	return p, ok
}

type register struct{}

var (
	DataRegister       = byte(1)
	DataCancelRegister = byte(2)
	DataReward         = byte(3)
)

func (p *register) doSend(vm *VM, block VmAccountBlock, quotaLeft, quotaRefund uint64) (uint64, uint64, error) {
	if len(block.Data()) == 11 && block.Data()[0] == DataRegister {
		return p.doSendRegister(vm, block, quotaLeft, quotaRefund)
	} else if len(block.Data()) == 11 && block.Data()[0] == DataCancelRegister {
		return p.doSendCancelRegister(vm, block, quotaLeft, quotaRefund)
	} else if len(block.Data()) >= 11 && block.Data()[0] == DataReward {
		return p.doSendReward(vm, block, quotaLeft, quotaRefund)
	}
	return quotaLeft, quotaRefund, ErrInvalidData
}

func (p *register) doSendRegister(vm *VM, block VmAccountBlock, quotaLeft, quotaRefund uint64) (uint64, uint64, error) {
	if block.Amount().Cmp(big1e24) != 0 ||
		bytes.Equal(block.TokenId().Bytes(), viteTokenTypeId.Bytes()) ||
		!isUserAccount(vm.Db, block.AccountAddress()) {
		return quotaLeft, quotaRefund, ErrInvalidData
	}
	gid, _ := BytesToGid(block.Data()[1:11])
	if !vm.Db.IsExistGid(gid) {
		return quotaLeft, quotaRefund, ErrInvalidData
	}
	locHash := getKey(block.AccountAddress(), gid)
	old := vm.Db.Storage(block.ToAddress(), locHash)
	if len(old) >= 72 && !allZero(old[0:32]) {
		return quotaLeft, quotaRefund, ErrInvalidData
	}
	quotaLeft, err := useQuota(quotaLeft, registerGas)
	if err != nil {
		return quotaLeft, quotaRefund, err
	}
	return quotaLeft, quotaRefund, nil
}

func (p *register) doSendCancelRegister(vm *VM, block VmAccountBlock, quotaLeft, quotaRefund uint64) (uint64, uint64, error) {
	if block.Amount().Cmp(big0) != 0 ||
		!isUserAccount(vm.Db, block.AccountAddress()) {
		return quotaLeft, quotaRefund, ErrInvalidData
	}
	gid, _ := BytesToGid(block.Data()[1:11])
	if !vm.Db.IsExistGid(gid) {
		return quotaLeft, quotaRefund, ErrInvalidData
	}
	locHash := getKey(block.AccountAddress(), gid)
	old := vm.Db.Storage(block.ToAddress(), locHash)
	if len(old) < 72 || allZero(old[0:32]) ||
		vm.Db.SnapshotBlock(block.SnapshotHash()).Timestamp()-new(big.Int).SetBytes(old[32:40]).Int64() < registerLockTime {
		return quotaLeft, quotaRefund, ErrInvalidData
	}

	quotaLeft, err := useQuota(quotaLeft, cancelRegisterGas)
	if err != nil {
		return quotaLeft, quotaRefund, err
	}
	return quotaLeft, quotaRefund, nil
}

func (p *register) doSendReward(vm *VM, block VmAccountBlock, quotaLeft, quotaRefund uint64) (uint64, uint64, error) {
	if block.Amount().Cmp(big0) != 0 ||
		!isUserAccount(vm.Db, block.AccountAddress()) {
		return quotaLeft, quotaRefund, ErrInvalidData
	}
	if !bytes.Equal(block.Data()[1:11], snapshotGid.Bytes()) {
		return quotaLeft, quotaRefund, ErrInvalidData
	}
	gid, _ := BytesToGid(block.Data()[1:11])
	locHash := getKey(block.AccountAddress(), gid)
	old := vm.Db.Storage(block.ToAddress(), locHash)
	if len(old) < 72 {
		return quotaLeft, quotaRefund, ErrInvalidData
	}
	// newRewardHeight := min(userDefined, currentSnapshotHeight-50, cancelSnapshotHeight)
	newRewardHeight := new(big.Int)
	if len(block.Data()) >= 43 {
		newRewardHeight.SetBytes(block.Data()[11:43])
	} else {
		newRewardHeight = BigMin(newRewardHeight.Sub(vm.Db.SnapshotBlock(block.SnapshotHash()).Height(), rewardHeightLimit), new(big.Int).SetBytes(old[40:72]))
	}
	if len(old) >= 104 && !allZero(old[72:104]) {
		newRewardHeight = BigMin(newRewardHeight, new(big.Int).SetBytes(old[72:104]))
	}
	oldRewardHeight := new(big.Int).SetBytes(old[40:72])
	if newRewardHeight.Cmp(oldRewardHeight) <= 0 {
		return quotaLeft, quotaRefund, ErrInvalidData
	}
	heightGap := new(big.Int).Sub(newRewardHeight, oldRewardHeight)
	if heightGap.Cmp(rewardGapLimit) > 0 {
		return quotaLeft, quotaRefund, ErrInvalidData
	}

	count := heightGap.Uint64()
	quotaLeft, err := useQuota(quotaLeft, rewardGas+count*calcRewardGasPerBlock)
	if err != nil {
		return quotaLeft, quotaRefund, err
	}

	reward := calcReward(vm, block.AccountAddress().Bytes(), oldRewardHeight, count)
	block.SetData(joinBytes(block.Data()[0:11], leftPadBytes(newRewardHeight.Bytes(), 32), leftPadBytes(oldRewardHeight.Bytes(), 32), leftPadBytes(reward.Bytes(), 32)))
	return quotaLeft, quotaRefund, nil
}

func calcReward(vm *VM, producer []byte, startHeight *big.Int, count uint64) *big.Int {
	var rewardCount uint64
	for count >= 0 {
		var list []VmSnapshotBlock
		if count < dbPageSize {
			list = vm.Db.SnapshotBlockList(startHeight, count, true)
			count = 0
		} else {
			list = vm.Db.SnapshotBlockList(startHeight, dbPageSize, true)
			count = count - dbPageSize
		}
		for _, block := range list {
			if bytes.Equal(block.Producer().Bytes(), producer) {
			}
			rewardCount++
		}
	}
	return new(big.Int).Mul(rewardPerBlock, new(big.Int).SetUint64(rewardCount))
}

func (p *register) doReceive(vm *VM, block VmAccountBlock) error {
	if len(block.Data()) == 11 && block.Data()[0] == DataRegister {
		return p.doReceiveRegister(vm, block)
	} else if len(block.Data()) == 11 && block.Data()[0] == DataCancelRegister {
		return p.doReceiveCancelRegister(vm, block)
	} else if len(block.Data()) == 107 && block.Data()[0] == DataReward {
		return p.doReceiveReward(vm, block)
	}
	return ErrInvalidData
}
func (p *register) doReceiveRegister(vm *VM, block VmAccountBlock) error {
	gid, _ := BytesToGid(block.Data()[1:11])
	locHash := getKey(block.AccountAddress(), gid)
	old := vm.Db.Storage(block.ToAddress(), locHash)
	if len(old) >= 72 && !allZero(old[0:32]) {
		return ErrInvalidData
	}
	snapshotBlock := vm.Db.SnapshotBlock(block.SnapshotHash())
	rewardHeight := leftPadBytes(snapshotBlock.Height().Bytes(), 32)
	if len(old) >= 72 && !allZero(old[40:72]) {
		rewardHeight = old[40:72]
	}
	registerInfo := joinBytes(leftPadBytes(block.Amount().Bytes(), 32),
		leftPadBytes(new(big.Int).SetInt64(snapshotBlock.Timestamp()).Bytes(), 32),
		rewardHeight,
		emptyWord)
	vm.Db.SetStorage(block.ToAddress(), locHash, registerInfo)
	return nil
}
func (p *register) doReceiveCancelRegister(vm *VM, block VmAccountBlock) error {
	gid, _ := BytesToGid(block.Data()[1:11])
	locHash := getKey(block.AccountAddress(), gid)
	old := vm.Db.Storage(block.ToAddress(), locHash)
	if len(old) < 72 || allZero(old[0:32]) {
		return ErrInvalidData
	}
	amount := new(big.Int).SetBytes(old[0:32])
	snapshotBlock := vm.Db.SnapshotBlock(block.SnapshotHash())
	registerInfo := joinBytes(emptyWord,
		leftPadBytes(new(big.Int).SetInt64(snapshotBlock.Timestamp()).Bytes(), 8),
		old[40:72],
		leftPadBytes(snapshotBlock.Height().Bytes(), 32))
	vm.Db.SetStorage(block.ToAddress(), locHash, registerInfo)
	refundBlock := vm.createBlock(block.ToAddress(), block.AccountAddress(), BlockTypeSendCall, block.Depth()+1)
	refundBlock.SetAmount(amount)
	refundBlock.SetTokenId(viteTokenTypeId)
	refundBlock.SetHeight(new(big.Int).Add(block.Height(), big1))
	vm.blockList = append(vm.blockList, refundBlock)
	return nil
}
func (p *register) doReceiveReward(vm *VM, block VmAccountBlock) error {
	gid, _ := BytesToGid(block.Data()[1:11])
	locHash := getKey(block.AccountAddress(), gid)
	old := vm.Db.Storage(block.ToAddress(), locHash)
	if len(old) < 72 || !bytes.Equal(old[40:72], block.Data()[43:75]) {
		return ErrInvalidData
	}
	if len(old) >= 104 && bytes.Equal(old[72:104], block.Data()[11:43]) {
		vm.Db.SetStorage(block.ToAddress(), locHash, []byte{})
	} else {
		var registerInfo []byte
		if len(old) >= 104 {
			registerInfo = joinBytes(old[0:40], block.Data()[11:43], old[72:104])
		} else {
			registerInfo = joinBytes(old[0:40], block.Data()[11:43])
		}
		vm.Db.SetStorage(block.ToAddress(), locHash, registerInfo)
	}
	refundBlock := vm.createBlock(block.ToAddress(), block.AccountAddress(), BlockTypeSendReward, block.Depth()+1)
	refundBlock.SetAmount(new(big.Int).SetBytes(block.Data()[75:107]))
	refundBlock.SetTokenId(viteTokenTypeId)
	refundBlock.SetHeight(new(big.Int).Add(block.Height(), big1))
	vm.blockList = append(vm.blockList, refundBlock)
	return nil
}

type vote struct{}

var (
	DataVote       = []byte{1}
	DataCancelVote = []byte{2}
)

func (p *vote) doSend(vm *VM, block VmAccountBlock, quotaLeft, quotaRefund uint64) (uint64, uint64, error) {
	// TODO
	return 0, 0, nil
}
func (p *vote) doReceive(vm *VM, block VmAccountBlock) error {
	// TODO
	return nil
}

type mortgage struct{}

func (p *mortgage) doSend(vm *VM, block VmAccountBlock, quotaLeft, quotaRefund uint64) (uint64, uint64, error) {
	// TODO
	return 0, 0, nil
}
func (p *mortgage) doReceive(vm *VM, block VmAccountBlock) error {
	// TODO
	return nil
}

type consensusGroup struct{}

func (p *consensusGroup) doSend(vm *VM, block VmAccountBlock, quotaLeft, quotaRefund uint64) (uint64, uint64, error) {
	// TODO
	return 0, 0, nil
}
func (p *consensusGroup) doReceive(vm *VM, block VmAccountBlock) error {
	// TODO
	return nil
}

func isUserAccount(db VmDatabase, addr types.Address) bool {
	return len(db.ContractCode(addr)) == 0
}

func getKey(addr types.Address, gid Gid) types.Hash {
	var data = types.Hash{}
	copy(data[2:12], gid[:])
	copy(data[12:], addr[:])
	return data
}
