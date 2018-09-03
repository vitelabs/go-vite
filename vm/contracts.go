package vm

import (
	"bytes"
	"github.com/vitelabs/go-vite/common/types"
	"math/big"
)

var (
	AddressRegister, _ = types.BytesToAddress([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1})
	AddressVote, _     = types.BytesToAddress([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2})
	AddressMortgage, _ = types.BytesToAddress([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 3})
)

type precompiledContract interface {
	doSend(vm *VM, block VmAccountBlock, quotaLeft uint64) (uint64, error)
	doReceive(vm *VM, block VmAccountBlock) error
}

var simpleContracts = map[types.Address]precompiledContract{
	AddressRegister: &register{},
	AddressVote:     &vote{},
	AddressMortgage: &mortgage{},
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

func (p *register) doSend(vm *VM, block VmAccountBlock, quotaLeft uint64) (uint64, error) {
	if len(block.Data()) == 11 && block.Data()[0] == DataRegister {
		return p.doSendRegister(vm, block, quotaLeft)
	} else if len(block.Data()) == 11 && block.Data()[0] == DataCancelRegister {
		return p.doSendCancelRegister(vm, block, quotaLeft)
	} else if len(block.Data()) >= 11 && block.Data()[0] == DataReward {
		return p.doSendReward(vm, block, quotaLeft)
	}
	return quotaLeft, ErrInvalidData
}

// register to become a super node of a consensus group, lock 100w ViteToken for 3 month
func (p *register) doSendRegister(vm *VM, block VmAccountBlock, quotaLeft uint64) (uint64, error) {
	if block.Amount().Cmp(big1e24) != 0 ||
		!isViteToken(block.TokenId()) ||
		!isUserAccount(vm.Db, block.AccountAddress()) {
		return quotaLeft, ErrInvalidData
	}
	// data: 0x01(0) + gid(1:11)
	gid, _ := BytesToGid(block.Data()[1:11])
	if !vm.Db.IsExistGid(gid) {
		return quotaLeft, ErrInvalidData
	}
	quotaLeft, err := useQuota(quotaLeft, registerGas)
	if err != nil {
		return quotaLeft, err
	}
	return quotaLeft, nil
}

// cancel register to become a super node of a consensus group after registered for 3 month, get 100w ViteToken back
func (p *register) doSendCancelRegister(vm *VM, block VmAccountBlock, quotaLeft uint64) (uint64, error) {
	if block.Amount().Sign() != 0 ||
		!isUserAccount(vm.Db, block.AccountAddress()) {
		return quotaLeft, ErrInvalidData
	}
	// data:  0x02(0) + gid(1:11)
	gid, _ := BytesToGid(block.Data()[1:11])
	if !vm.Db.IsExistGid(gid) {
		return quotaLeft, ErrInvalidData
	}

	quotaLeft, err := useQuota(quotaLeft, cancelRegisterGas)
	if err != nil {
		return quotaLeft, err
	}
	return quotaLeft, nil
}

// get reward of generating snapshot block
func (p *register) doSendReward(vm *VM, block VmAccountBlock, quotaLeft uint64) (uint64, error) {
	if block.Amount().Sign() != 0 ||
		!isUserAccount(vm.Db, block.AccountAddress()) {
		return quotaLeft, ErrInvalidData
	}
	// data: 0x03(0) + gid(1:11) + end reward height(11:43, optional) + start reward height(43:75, optional) + rewardAmount(75:107, optional)
	// only generating snapshot block creates reward officially
	gid, _ := BytesToGid(block.Data()[1:11])
	if !isSnapshotGid(gid) {
		return quotaLeft, ErrInvalidData
	}
	locHash := getKey(block.AccountAddress(), gid)
	old := vm.Db.Storage(block.ToAddress(), locHash)
	if len(old) < 72 {
		return quotaLeft, ErrInvalidData
	}
	// newRewardHeight := min(currentSnapshotHeight-50, userDefined, cancelSnapshotHeight)
	newRewardHeight := new(big.Int).Sub(vm.Db.SnapshotBlock(block.SnapshotHash()).Height(), rewardHeightLimit)
	if len(block.Data()) >= 43 {
		newRewardHeight = BigMin(newRewardHeight, new(big.Int).SetBytes(block.Data()[11:43]))
	}
	if len(old) >= 104 && !allZero(old[72:104]) {
		newRewardHeight = BigMin(newRewardHeight, new(big.Int).SetBytes(old[72:104]))
	}
	oldRewardHeight := new(big.Int).SetBytes(old[40:72])
	if newRewardHeight.Cmp(oldRewardHeight) <= 0 {
		return quotaLeft, ErrInvalidData
	}
	heightGap := new(big.Int).Sub(newRewardHeight, oldRewardHeight)
	if heightGap.Cmp(rewardGapLimit) > 0 {
		return quotaLeft, ErrInvalidData
	}

	count := heightGap.Uint64()
	quotaLeft, err := useQuota(quotaLeft, rewardGas+count*calcRewardGasPerBlock)
	if err != nil {
		return quotaLeft, err
	}

	reward := calcReward(vm, block.AccountAddress().Bytes(), new(big.Int).Set(oldRewardHeight), count)
	block.SetData(joinBytes(block.Data()[0:11], leftPadBytes(newRewardHeight.Bytes(), 32), old[40:72], leftPadBytes(reward.Bytes(), 32)))
	return quotaLeft, nil
}

func calcReward(vm *VM, producer []byte, startHeight *big.Int, count uint64) *big.Int {
	var rewardCount uint64
	for count > 0 {
		var list []VmSnapshotBlock
		if count < dbPageSize {
			list = vm.Db.SnapshotBlockList(startHeight, count, true)
			count = 0
		} else {
			list = vm.Db.SnapshotBlockList(startHeight, dbPageSize, true)
			count = count - dbPageSize
			startHeight.Add(startHeight, dbPageSizeBig)
		}
		for _, block := range list {
			if bytes.Equal(block.Producer().Bytes(), producer) {
				rewardCount++
			}
		}
	}
	reward := new(big.Int).SetUint64(rewardCount)
	reward.Mul(rewardPerBlock, reward)
	return reward
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
	// data: 0x01(0) + gid(1:11)
	gid, _ := BytesToGid(block.Data()[1:11])
	// storage key: 00(0:2) + gid(2:12) + address(12:32)
	locHash := getKey(block.AccountAddress(), gid)
	// storage value: lock ViteToken amount(0:32) + lock start timestamp(32:40) + start reward snapshot height(40:72) + cancel snapshot height, 0 for default(72:104)
	old := vm.Db.Storage(block.ToAddress(), locHash)
	if len(old) >= 72 && !allZero(old[0:32]) {
		// duplicate register
		return ErrInvalidData
	}
	snapshotBlock := vm.Db.SnapshotBlock(block.SnapshotHash())
	rewardHeight := leftPadBytes(snapshotBlock.Height().Bytes(), 32)
	if len(old) >= 72 && !allZero(old[40:72]) {
		// reward of last being a super node is not drained
		rewardHeight = old[40:72]
	}
	registerInfo := joinBytes(leftPadBytes(block.Amount().Bytes(), 32),
		leftPadBytes(new(big.Int).SetInt64(snapshotBlock.Timestamp()).Bytes(), 8),
		rewardHeight,
		emptyWord)
	vm.Db.SetStorage(block.ToAddress(), locHash, registerInfo)
	return nil
}
func (p *register) doReceiveCancelRegister(vm *VM, block VmAccountBlock) error {
	// data:  0x02(0) + gid(1:11)
	gid, _ := BytesToGid(block.Data()[1:11])
	locHash := getKey(block.AccountAddress(), gid)
	old := vm.Db.Storage(block.ToAddress(), locHash)
	if len(old) < 72 || allZero(old[0:32]) {
		return ErrInvalidData
	}
	// update lock amount and loc start timestamp
	amount := new(big.Int).SetBytes(old[0:32])
	snapshotBlock := vm.Db.SnapshotBlock(block.SnapshotHash())
	registerInfo := joinBytes(emptyWord,
		emptyTimestamp,
		old[40:72],
		leftPadBytes(snapshotBlock.Height().Bytes(), 32))
	vm.Db.SetStorage(block.ToAddress(), locHash, registerInfo)
	// return locked ViteToken
	refundBlock := vm.createBlock(block.ToAddress(), block.AccountAddress(), BlockTypeSendCall, block.Depth()+1)
	refundBlock.SetAmount(amount)
	refundBlock.SetTokenId(viteTokenTypeId)
	refundBlock.SetHeight(new(big.Int).Add(block.Height(), big1))
	vm.blockList = append(vm.blockList, refundBlock)
	return nil
}
func (p *register) doReceiveReward(vm *VM, block VmAccountBlock) error {
	// data: 0x03(0) + gid(1:11) + end reward height(11:43) + start reward height(43:75) + rewardAmount(75:107)
	gid, _ := BytesToGid(block.Data()[1:11])
	locHash := getKey(block.AccountAddress(), gid)
	old := vm.Db.Storage(block.ToAddress(), locHash)
	if len(old) < 72 || !bytes.Equal(old[40:72], block.Data()[43:75]) {
		return ErrInvalidData
	}
	if len(old) >= 104 {
		cancelTime := new(big.Int).SetBytes(old[72:104])
		newRewardTime := new(big.Int).SetBytes(block.Data()[11:43])
		switch newRewardTime.Cmp(cancelTime) {
		case 1:
			return ErrInvalidData
		case 0:
			// delete storage when register canceled and reward drained
			vm.Db.SetStorage(block.ToAddress(), locHash, nil)
		case -1:
			vm.Db.SetStorage(block.ToAddress(), locHash, joinBytes(old[0:40], block.Data()[11:43], old[72:104]))
		}
	} else {
		vm.Db.SetStorage(block.ToAddress(), locHash, joinBytes(old[0:40], block.Data()[11:43]))
	}
	// create reward and return
	refundBlock := vm.createBlock(block.ToAddress(), block.AccountAddress(), BlockTypeSendReward, block.Depth()+1)
	refundBlock.SetAmount(new(big.Int).SetBytes(block.Data()[75:107]))
	refundBlock.SetTokenId(viteTokenTypeId)
	refundBlock.SetHeight(new(big.Int).Add(block.Height(), big1))
	vm.blockList = append(vm.blockList, refundBlock)
	return nil
}

type vote struct{}

var (
	DataVote       = byte(1)
	DataCancelVote = byte(2)
)

func (p *vote) doSend(vm *VM, block VmAccountBlock, quotaLeft uint64) (uint64, error) {
	if len(block.Data()) == 31 && block.Data()[0] == DataVote {
		return p.doSendVote(vm, block, quotaLeft)
	} else if len(block.Data()) == 11 && block.Data()[0] == DataCancelVote {
		return p.doSendCancelVote(vm, block, quotaLeft)
	}
	return quotaLeft, ErrInvalidData
}

// vote for a super node of a consensus group
func (p *vote) doSendVote(vm *VM, block VmAccountBlock, quotaLeft uint64) (uint64, error) {
	if block.Amount().Sign() != 0 ||
		!isUserAccount(vm.Db, block.AccountAddress()) {
		return quotaLeft, ErrInvalidData
	}
	// data: 0x01(0) + gid(1:11) + super node address(11:31)
	gid, _ := BytesToGid(block.Data()[1:11])
	if !vm.Db.IsExistGid(gid) {
		return quotaLeft, ErrInvalidData
	}
	address, _ := types.BytesToAddress(block.Data()[11:31])
	if !vm.Db.IsExistAddress(address) {
		return quotaLeft, ErrInvalidData
	}
	quotaLeft, err := useQuota(quotaLeft, voteGas)
	if err != nil {
		return quotaLeft, err
	}
	return quotaLeft, nil
}

// cancel vote for a super node of a consensus group
func (p *vote) doSendCancelVote(vm *VM, block VmAccountBlock, quotaLeft uint64) (uint64, error) {
	if block.Amount().Sign() != 0 ||
		!isUserAccount(vm.Db, block.AccountAddress()) {
		return quotaLeft, ErrInvalidData
	}
	// data: 0x02(0) + gid(1:11)
	gid, _ := BytesToGid(block.Data()[1:11])
	if !vm.Db.IsExistGid(gid) {
		return quotaLeft, ErrInvalidData
	}
	quotaLeft, err := useQuota(quotaLeft, cancelVoteGas)
	if err != nil {
		return quotaLeft, err
	}
	return quotaLeft, nil
}
func (p *vote) doReceive(vm *VM, block VmAccountBlock) error {
	if len(block.Data()) == 31 && block.Data()[0] == DataVote {
		return p.doReceiveVote(vm, block)
	} else if len(block.Data()) == 11 && block.Data()[0] == DataCancelVote {
		return p.doReceiveCancelVote(vm, block)
	}
	return nil
}
func (p *vote) doReceiveVote(vm *VM, block VmAccountBlock) error {
	gid, _ := BytesToGid(block.Data()[1:11])
	// storage key: 00(0:2) + gid(2:12) + voter address(12:32)
	locHash := getKey(block.AccountAddress(), gid)
	// storage value: superNodeAddress(0:20)
	vm.Db.SetStorage(block.ToAddress(), locHash, block.Data()[11:31])
	return nil
}
func (p *vote) doReceiveCancelVote(vm *VM, block VmAccountBlock) error {
	gid, _ := BytesToGid(block.Data()[1:11])
	locHash := getKey(block.AccountAddress(), gid)
	vm.Db.SetStorage(block.ToAddress(), locHash, nil)
	return nil
}

type mortgage struct{}

var (
	DataMortgage       = byte(1)
	DataCancelMortgage = byte(2)
)

func (p *mortgage) doSend(vm *VM, block VmAccountBlock, quotaLeft uint64) (uint64, error) {
	if len(block.Data()) == 29 && block.Data()[0] == DataMortgage {
		return p.doSendMortgage(vm, block, quotaLeft)
	} else if len(block.Data()) == 53 && block.Data()[0] == DataCancelMortgage {
		return p.doSendCancelMortgage(vm, block, quotaLeft)
	}
	return quotaLeft, ErrInvalidData
}

// mortgage ViteToken for a beneficial to get quota
func (p *mortgage) doSendMortgage(vm *VM, block VmAccountBlock, quotaLeft uint64) (uint64, error) {
	if block.Amount().Sign() == 0 ||
		!isViteToken(block.TokenId()) ||
		!isUserAccount(vm.Db, block.AccountAddress()) {
		return quotaLeft, ErrInvalidData
	}
	// data: 0x01(0) + beneficial address(1:21) + withdrawTime(21:29)
	address, _ := types.BytesToAddress(block.Data()[1:21])
	if !vm.Db.IsExistAddress(address) {
		return quotaLeft, ErrInvalidData
	}
	withdrawTime := new(big.Int).SetBytes(block.Data()[21:29]).Int64()
	if withdrawTime < vm.Db.SnapshotBlock(block.SnapshotHash()).Timestamp()+mortgageTime {
		return quotaLeft, ErrInvalidData
	}
	quotaLeft, err := useQuota(quotaLeft, mortgageGas)
	if err != nil {
		return quotaLeft, err
	}
	return quotaLeft, nil
}

// cancel mortgage ViteToken
func (p *mortgage) doSendCancelMortgage(vm *VM, block VmAccountBlock, quotaLeft uint64) (uint64, error) {
	if block.Amount().Sign() > 0 ||
		allZero(block.Data()[21:53]) ||
		!isUserAccount(vm.Db, block.AccountAddress()) {
		return quotaLeft, ErrInvalidData
	}
	// data: 0x02(0) + beneficial address(1:21) + amount(21:53)
	address, _ := types.BytesToAddress(block.Data()[1:21])
	if !vm.Db.IsExistAddress(address) {
		return quotaLeft, ErrInvalidData
	}
	quotaLeft, err := useQuota(quotaLeft, cancelMortgageGas)
	if err != nil {
		return quotaLeft, err
	}
	return quotaLeft, nil
}

func (p *mortgage) doReceive(vm *VM, block VmAccountBlock) error {
	if len(block.Data()) == 29 && block.Data()[0] == DataMortgage {
		return p.doReceiveMortgage(vm, block)
	} else if len(block.Data()) == 53 && block.Data()[0] == DataCancelMortgage {
		return p.doReceiveCancelMortgage(vm, block)
	}
	return nil
}

func (p *mortgage) doReceiveMortgage(vm *VM, block VmAccountBlock) error {
	// data: 0x01(0) + beneficial address(1:21) + withdrawTime(21:29)
	// storage key for quota: hash(beneficial)
	locHashQuotaAmount := types.DataHash(block.Data()[1:21])
	// storage key for mortgage: hash(owner, hash(beneficial))
	locHashMortgage := types.DataHash(append(block.AccountAddress().Bytes(), locHashQuotaAmount.Bytes()...))
	// storage value for mortgage: mortgage amount(0:32) + withdrawTime(32:40)
	old := vm.Db.Storage(block.ToAddress(), locHashMortgage)
	withdrawTime := new(big.Int).SetBytes(block.Data()[21:29]).Int64()
	amount := new(big.Int).SetUint64(0)
	if len(old) >= 40 {
		if withdrawTime < new(big.Int).SetBytes(old[32:40]).Int64() {
			return ErrInvalidData
		}
		amount.SetBytes(old[0:32])
	}
	amount.Add(amount, block.Amount())
	vm.Db.SetStorage(block.ToAddress(), locHashMortgage, joinBytes(leftPadBytes(amount.Bytes(), 32), leftPadBytes(new(big.Int).SetInt64(withdrawTime).Bytes(), 8)))

	// storage value for quota: quota amount(0:32)
	oldQuotaAmount := vm.Db.Storage(block.ToAddress(), locHashQuotaAmount)
	quotaAmount := new(big.Int).SetUint64(0)
	if len(oldQuotaAmount) >= 32 {
		quotaAmount.SetBytes(oldQuotaAmount[0:32])
	}
	quotaAmount.Add(quotaAmount, block.Amount())
	vm.Db.SetStorage(block.ToAddress(), locHashQuotaAmount, leftPadBytes(quotaAmount.Bytes(), 32))
	return nil
}
func (p *mortgage) doReceiveCancelMortgage(vm *VM, block VmAccountBlock) error {
	// data: 0x02(0) + beneficial address(1:21) + amount(21:53)
	locHashQuotaAmount := types.DataHash(block.Data()[1:21])
	locHashMortgage := types.DataHash(append(block.AccountAddress().Bytes(), locHashQuotaAmount.Bytes()...))
	old := vm.Db.Storage(block.ToAddress(), locHashMortgage)
	if len(old) < 40 {
		return ErrInvalidData
	}
	withdrawTime := new(big.Int).SetBytes(old[32:40]).Int64()
	if withdrawTime > vm.Db.SnapshotBlock(block.SnapshotHash()).Timestamp() {
		return ErrInvalidData
	}
	amount := new(big.Int).SetBytes(old[0:32])
	withdrawAmount := new(big.Int).SetBytes(block.Data()[21:53])
	if amount.Cmp(withdrawAmount) < 0 {
		return ErrInvalidData
	}
	amount.Sub(amount, withdrawAmount)

	oldQuota := vm.Db.Storage(block.ToAddress(), locHashQuotaAmount)
	if len(oldQuota) < 32 {
		return ErrInvalidData
	}
	quotaAmount := new(big.Int).SetBytes(oldQuota[0:32])
	if quotaAmount.Cmp(withdrawAmount) < 0 {
		return ErrInvalidData
	}
	quotaAmount.Sub(quotaAmount, withdrawAmount)

	if amount.Sign() == 0 {
		vm.Db.SetStorage(block.ToAddress(), locHashMortgage, nil)
	} else {
		vm.Db.SetStorage(block.ToAddress(), locHashMortgage, joinBytes(leftPadBytes(amount.Bytes(), 32), old[32:40]))
	}

	if quotaAmount.Sign() == 0 {
		vm.Db.SetStorage(block.ToAddress(), locHashQuotaAmount, nil)
	} else {
		vm.Db.SetStorage(block.ToAddress(), locHashQuotaAmount, leftPadBytes(quotaAmount.Bytes(), 32))
	}

	// append refund block
	refundBlock := vm.createBlock(block.ToAddress(), block.AccountAddress(), BlockTypeSendCall, block.Depth()+1)
	refundBlock.SetAmount(withdrawAmount)
	refundBlock.SetTokenId(viteTokenTypeId)
	refundBlock.SetHeight(new(big.Int).Add(block.Height(), big1))
	vm.blockList = append(vm.blockList, refundBlock)
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
