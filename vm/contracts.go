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
	createFee(vm *VM, block VmAccountBlock) *big.Int
	doSend(vm *VM, block VmAccountBlock, quotaLeft uint64) (uint64, error)
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
	DataRegister       = []byte{0xde, 0x6e, 0xef, 0x70}
	DataCancelRegister = []byte{0x74, 0x46, 0xae, 0x9b}
	DataReward         = []byte{0xc7, 0x45, 0xe1, 0x5c}
)

func (p *register) createFee(vm *VM, block VmAccountBlock) *big.Int {
	return big.NewInt(0)
}
func (p *register) doSend(vm *VM, block VmAccountBlock, quotaLeft uint64) (uint64, error) {
	if len(block.Data()) == 36 && bytes.Equal(block.Data()[0:4], DataRegister) {
		return p.doSendRegister(vm, block, quotaLeft)
	} else if len(block.Data()) == 36 && bytes.Equal(block.Data()[0:4], DataCancelRegister) {
		return p.doSendCancelRegister(vm, block, quotaLeft)
	} else if len(block.Data()) >= 36 && bytes.Equal(block.Data()[0:4], DataReward) {
		return p.doSendReward(vm, block, quotaLeft)
	}
	return quotaLeft, ErrInvalidData
}

// register to become a super node of a consensus group, lock 100w ViteToken for 3 month
func (p *register) doSendRegister(vm *VM, block VmAccountBlock, quotaLeft uint64) (uint64, error) {
	quotaLeft, err := useQuota(quotaLeft, registerGas)
	if err != nil {
		return quotaLeft, err
	}
	quotaLeft, err = useQuotaForData(block.Data(), quotaLeft)
	if err != nil {
		return quotaLeft, err
	}
	if block.Amount().Cmp(registerAmount) != 0 ||
		!isViteToken(block.TokenId()) ||
		!isUserAccount(vm.Db, block.AccountAddress()) {
		return quotaLeft, ErrInvalidData
	}
	// data: methodSelector(0:4) + gid(4:36)
	gid, err := BigToGid(new(big.Int).SetBytes(block.Data()[4:36]))
	if err != nil || !isExistGid(vm.Db, gid) {
		return quotaLeft, ErrInvalidData
	}
	return quotaLeft, nil
}

// cancel register to become a super node of a consensus group after registered for 3 month, get 100w ViteToken back
func (p *register) doSendCancelRegister(vm *VM, block VmAccountBlock, quotaLeft uint64) (uint64, error) {
	quotaLeft, err := useQuota(quotaLeft, cancelRegisterGas)
	if err != nil {
		return quotaLeft, err
	}
	quotaLeft, err = useQuotaForData(block.Data(), quotaLeft)
	if err != nil {
		return quotaLeft, err
	}
	if block.Amount().Sign() != 0 ||
		!isUserAccount(vm.Db, block.AccountAddress()) {
		return quotaLeft, ErrInvalidData
	}
	// data:  methodSelector(0:4) + gid(4:36)
	gid, err := BigToGid(new(big.Int).SetBytes(block.Data()[4:36]))
	if err != nil || !isExistGid(vm.Db, gid) {
		return quotaLeft, ErrInvalidData
	}

	locHash := getKey(block.AccountAddress(), gid)
	old := vm.Db.Storage(block.ToAddress(), locHash)
	// storage value: lock ViteToken amount(0:32) + lock start timestamp(32:64) + start reward snapshot height(64:96) + cancel snapshot height, 0 for default(96:128)
	if len(old) < 96 || allZero(old[0:32]) {
		return quotaLeft, ErrInvalidData
	}
	lockStartTime := new(big.Int).SetBytes(old[32:64]).Int64()
	if lockStartTime+registerLockTime < vm.Db.SnapshotBlock(block.SnapshotHash()).Timestamp() {
		return quotaLeft, ErrInvalidData
	}
	return quotaLeft, nil
}

// get reward of generating snapshot block
func (p *register) doSendReward(vm *VM, block VmAccountBlock, quotaLeft uint64) (uint64, error) {
	quotaLeft, err := useQuota(quotaLeft, rewardGas)
	if err != nil {
		return quotaLeft, err
	}
	if block.Amount().Sign() != 0 ||
		!isUserAccount(vm.Db, block.AccountAddress()) {
		return quotaLeft, ErrInvalidData
	}
	// data: methodSelector(0:4) + gid(4:36) + end reward height(36:68, optional) + start reward height(68:100, optional) + rewardAmount(100:132, optional)
	// only generating snapshot block creates reward officially
	gid, err := BigToGid(new(big.Int).SetBytes(block.Data()[4:36]))
	if err != nil || !isSnapshotGid(gid) {
		return quotaLeft, ErrInvalidData
	}
	locHash := getKey(block.AccountAddress(), gid)
	old := vm.Db.Storage(block.ToAddress(), locHash)
	if len(old) < 96 {
		return quotaLeft, ErrInvalidData
	}
	intPool := poolOfIntPools.get()
	defer poolOfIntPools.put(intPool)
	// newRewardHeight := min(currentSnapshotHeight-50, userDefined, cancelSnapshotHeight)
	newRewardHeight := intPool.get().Sub(vm.Db.SnapshotBlock(block.SnapshotHash()).Height(), rewardHeightLimit)
	defer intPool.put(newRewardHeight)
	if len(block.Data()) >= 68 {
		userDefined := intPool.get().SetBytes(block.Data()[36:68])
		defer intPool.put(userDefined)
		newRewardHeight = BigMin(newRewardHeight, userDefined)
	}
	if len(old) >= 128 && !allZero(old[96:128]) {
		cancelSnapshotHeight := intPool.get().SetBytes(old[96:128])
		defer intPool.put(cancelSnapshotHeight)
		newRewardHeight = BigMin(newRewardHeight, cancelSnapshotHeight)
	}
	oldRewardHeight := intPool.get().SetBytes(old[64:96])
	if newRewardHeight.Cmp(oldRewardHeight) <= 0 {
		return quotaLeft, ErrInvalidData
	}
	heightGap := intPool.get().Sub(newRewardHeight, oldRewardHeight)
	defer intPool.put(heightGap)
	if heightGap.Cmp(rewardGapLimit) > 0 {
		return quotaLeft, ErrInvalidData
	}

	count := heightGap.Uint64()
	quotaLeft, err = useQuota(quotaLeft, ((count+dbPageSize-1)/dbPageSize)*calcRewardGasPerPage)
	if err != nil {
		return quotaLeft, err
	}

	reward := intPool.getZero()
	calcReward(vm, block.AccountAddress().Bytes(), oldRewardHeight, count, reward)
	block.SetData(joinBytes(block.Data()[0:36], LeftPadBytes(newRewardHeight.Bytes(), 32), old[64:96], LeftPadBytes(reward.Bytes(), 32)))
	intPool.put(reward)
	quotaLeft, err = useQuotaForData(block.Data(), quotaLeft)
	if err != nil {
		return quotaLeft, err
	}
	return quotaLeft, nil
}

func calcReward(vm *VM, producer []byte, startHeight *big.Int, count uint64, reward *big.Int) {
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
	reward.SetUint64(rewardCount)
	reward.Mul(rewardPerBlock, reward)
}

func (p *register) doReceive(vm *VM, block VmAccountBlock) error {
	if len(block.Data()) == 36 && bytes.Equal(block.Data()[0:4], DataRegister) {
		return p.doReceiveRegister(vm, block)
	} else if len(block.Data()) == 36 && bytes.Equal(block.Data()[0:4], DataCancelRegister) {
		return p.doReceiveCancelRegister(vm, block)
	} else if len(block.Data()) == 132 && bytes.Equal(block.Data()[0:4], DataReward) {
		return p.doReceiveReward(vm, block)
	}
	return ErrInvalidData
}
func (p *register) doReceiveRegister(vm *VM, block VmAccountBlock) error {
	// data: methodSelector(0:4) + gid(4:36)
	gid, _ := BigToGid(new(big.Int).SetBytes(block.Data()[4:36]))
	// storage key: gid(2:12) + address(12:32)
	locHash := getKey(block.AccountAddress(), gid)
	// storage value: lock ViteToken amount(0:32) + lock start timestamp(32:64) + start reward snapshot height(64:96) + cancel snapshot height, 0 for default(96:128)
	old := vm.Db.Storage(block.ToAddress(), locHash)
	if len(old) >= 32 && !allZero(old[0:32]) {
		// duplicate register
		return ErrInvalidData
	}
	intPool := poolOfIntPools.get()
	defer poolOfIntPools.put(intPool)
	snapshotBlock := vm.Db.SnapshotBlock(block.SnapshotHash())
	rewardHeight := LeftPadBytes(snapshotBlock.Height().Bytes(), 32)
	if len(old) >= 96 && !allZero(old[64:96]) {
		// reward of last being a super node is not drained
		rewardHeight = old[64:96]
	}
	startTimestamp := intPool.get().SetInt64(snapshotBlock.Timestamp())
	registerInfo := joinBytes(LeftPadBytes(block.Amount().Bytes(), 32),
		LeftPadBytes(startTimestamp.Bytes(), 32),
		rewardHeight,
		emptyWord)
	intPool.put(startTimestamp)
	vm.Db.SetStorage(block.ToAddress(), locHash, registerInfo)
	return nil
}
func (p *register) doReceiveCancelRegister(vm *VM, block VmAccountBlock) error {
	// data:  methodSelector(0:4) + gid(4:36)
	gid, _ := BytesToGid(block.Data()[26:36])
	locHash := getKey(block.AccountAddress(), gid)
	old := vm.Db.Storage(block.ToAddress(), locHash)
	if len(old) < 96 || allZero(old[0:32]) {
		return ErrInvalidData
	}
	intPool := poolOfIntPools.get()
	defer poolOfIntPools.put(intPool)
	// update lock amount and loc start timestamp
	amount := intPool.get().SetBytes(old[0:32])
	snapshotBlock := vm.Db.SnapshotBlock(block.SnapshotHash())
	registerInfo := joinBytes(emptyWord,
		emptyWord,
		old[64:96],
		LeftPadBytes(snapshotBlock.Height().Bytes(), 32))
	vm.Db.SetStorage(block.ToAddress(), locHash, registerInfo)
	// return locked ViteToken
	refundBlock := vm.createBlock(block.ToAddress(), block.AccountAddress(), BlockTypeSendCall, block.Depth()+1)
	refundBlock.SetAmount(amount)
	refundBlock.SetTokenId(viteTokenTypeId)
	refundBlock.SetHeight(intPool.get().Add(block.Height(), Big1))
	vm.blockList = append(vm.blockList, refundBlock)
	return nil
}
func (p *register) doReceiveReward(vm *VM, block VmAccountBlock) error {
	// data: methodSelector(0:4) + gid(4:36) + end reward height(36:68) + start reward height(68:100) + rewardAmount(100:132)
	gid, _ := BytesToGid(block.Data()[26:36])
	locHash := getKey(block.AccountAddress(), gid)
	old := vm.Db.Storage(block.ToAddress(), locHash)
	if len(old) < 96 || !bytes.Equal(old[64:96], block.Data()[68:100]) {
		return ErrInvalidData
	}
	intPool := poolOfIntPools.get()
	defer poolOfIntPools.put(intPool)
	if len(old) >= 128 {
		cancelTime := intPool.get().SetBytes(old[96:128])
		newRewardTime := intPool.get().SetBytes(block.Data()[36:68])
		defer intPool.put(cancelTime, newRewardTime)
		switch newRewardTime.Cmp(cancelTime) {
		case 1:
			return ErrInvalidData
		case 0:
			// delete storage when register canceled and reward drained
			vm.Db.SetStorage(block.ToAddress(), locHash, nil)
		case -1:
			vm.Db.SetStorage(block.ToAddress(), locHash, joinBytes(old[0:64], block.Data()[36:68], old[96:128]))
		}
	} else {
		vm.Db.SetStorage(block.ToAddress(), locHash, joinBytes(old[0:64], block.Data()[36:68]))
	}
	// create reward and return
	rewardAmount := intPool.get().SetBytes(block.Data()[100:132])
	if rewardAmount.Sign() > 0 {
		refundBlock := vm.createBlock(block.ToAddress(), block.AccountAddress(), BlockTypeSendReward, block.Depth()+1)
		refundBlock.SetAmount(intPool.get().SetBytes(block.Data()[100:132]))
		refundBlock.SetTokenId(viteTokenTypeId)
		refundBlock.SetHeight(intPool.get().Add(block.Height(), Big1))
		vm.blockList = append(vm.blockList, refundBlock)
	} else {
		intPool.put(rewardAmount)
	}
	return nil
}

type vote struct{}

func (p *vote) createFee(vm *VM, block VmAccountBlock) *big.Int {
	return big.NewInt(0)
}

var (
	DataVote       = []byte{0x70, 0xfe, 0xcc, 0x42}
	DataCancelVote = []byte{0x4e, 0x0b, 0x53, 0x02}
)

func (p *vote) doSend(vm *VM, block VmAccountBlock, quotaLeft uint64) (uint64, error) {
	if len(block.Data()) == 68 && bytes.Equal(block.Data()[0:4], DataVote) {
		return p.doSendVote(vm, block, quotaLeft)
	} else if len(block.Data()) == 36 && bytes.Equal(block.Data()[0:4], DataCancelVote) {
		return p.doSendCancelVote(vm, block, quotaLeft)
	}
	return quotaLeft, ErrInvalidData
}

// vote for a super node of a consensus group
func (p *vote) doSendVote(vm *VM, block VmAccountBlock, quotaLeft uint64) (uint64, error) {
	quotaLeft, err := useQuota(quotaLeft, voteGas)
	if err != nil {
		return quotaLeft, err
	}
	quotaLeft, err = useQuotaForData(block.Data(), quotaLeft)
	if err != nil {
		return quotaLeft, err
	}
	if block.Amount().Sign() != 0 ||
		!isUserAccount(vm.Db, block.AccountAddress()) {
		return quotaLeft, ErrInvalidData
	}
	// data: methodSelector(0:4) + gid(4:36) + super node address(36:68)
	gid, err := BigToGid(new(big.Int).SetBytes(block.Data()[4:36]))
	if err != nil || !isExistGid(vm.Db, gid) {
		return quotaLeft, ErrInvalidData
	}
	address, err := types.BytesToAddress(new(big.Int).SetBytes(block.Data()[36:68]).Bytes())
	if err != nil || !vm.Db.IsExistAddress(address) {
		return quotaLeft, ErrInvalidData
	}
	return quotaLeft, nil
}

// cancel vote for a super node of a consensus group
func (p *vote) doSendCancelVote(vm *VM, block VmAccountBlock, quotaLeft uint64) (uint64, error) {
	quotaLeft, err := useQuota(quotaLeft, cancelVoteGas)
	if err != nil {
		return quotaLeft, err
	}
	quotaLeft, err = useQuotaForData(block.Data(), quotaLeft)
	if err != nil {
		return quotaLeft, err
	}
	if block.Amount().Sign() != 0 ||
		!isUserAccount(vm.Db, block.AccountAddress()) {
		return quotaLeft, ErrInvalidData
	}
	// data: methodSelector(0:4) + gid(4:36)
	gid, err := BigToGid(new(big.Int).SetBytes(block.Data()[4:36]))
	if err != nil || !isExistGid(vm.Db, gid) {
		return quotaLeft, ErrInvalidData
	}
	return quotaLeft, nil
}
func (p *vote) doReceive(vm *VM, block VmAccountBlock) error {
	if len(block.Data()) == 68 && bytes.Equal(block.Data()[0:4], DataVote) {
		return p.doReceiveVote(vm, block)
	} else if len(block.Data()) == 36 && bytes.Equal(block.Data()[0:4], DataCancelVote) {
		return p.doReceiveCancelVote(vm, block)
	}
	return nil
}
func (p *vote) doReceiveVote(vm *VM, block VmAccountBlock) error {
	gid, _ := BytesToGid(block.Data()[26:36])
	// storage key: 00(0:2) + gid(2:12) + voter address(12:32)
	locHash := getKey(block.AccountAddress(), gid)
	// storage value: superNodeAddress(0:32)
	vm.Db.SetStorage(block.ToAddress(), locHash, block.Data()[36:68])
	return nil
}
func (p *vote) doReceiveCancelVote(vm *VM, block VmAccountBlock) error {
	gid, _ := BytesToGid(block.Data()[26:36])
	locHash := getKey(block.AccountAddress(), gid)
	vm.Db.SetStorage(block.ToAddress(), locHash, nil)
	return nil
}

type mortgage struct{}

func (p *mortgage) createFee(vm *VM, block VmAccountBlock) *big.Int {
	return big.NewInt(0)
}

var (
	DataMortgage       = []byte{0x9c, 0xc4, 0x4b, 0xd6}
	DataCancelMortgage = []byte{0xf8, 0x2d, 0x39, 0xca}
)

func (p *mortgage) doSend(vm *VM, block VmAccountBlock, quotaLeft uint64) (uint64, error) {
	if len(block.Data()) == 68 && bytes.Equal(block.Data()[0:4], DataMortgage) {
		return p.doSendMortgage(vm, block, quotaLeft)
	} else if len(block.Data()) == 68 && bytes.Equal(block.Data()[0:4], DataCancelMortgage) {
		return p.doSendCancelMortgage(vm, block, quotaLeft)
	}
	return quotaLeft, ErrInvalidData
}

// mortgage ViteToken for a beneficial to get quota
func (p *mortgage) doSendMortgage(vm *VM, block VmAccountBlock, quotaLeft uint64) (uint64, error) {
	quotaLeft, err := useQuota(quotaLeft, mortgageGas)
	if err != nil {
		return quotaLeft, err
	}
	quotaLeft, err = useQuotaForData(block.Data(), quotaLeft)
	if err != nil {
		return quotaLeft, err
	}
	if block.Amount().Sign() == 0 ||
		!isViteToken(block.TokenId()) ||
		!isUserAccount(vm.Db, block.AccountAddress()) {
		return quotaLeft, ErrInvalidData
	}
	// data:methodSelector(0:4) + beneficial address(4:36) + withdrawTime(36:68)
	address, err := types.BytesToAddress(new(big.Int).SetBytes(block.Data()[4:36]).Bytes())
	if err != nil || !vm.Db.IsExistAddress(address) {
		return quotaLeft, ErrInvalidData
	}
	intPool := poolOfIntPools.get()
	defer poolOfIntPools.put(intPool)
	withdrawTime := intPool.get().SetBytes(block.Data()[36:68])
	defer intPool.put(withdrawTime)
	if !withdrawTime.IsInt64() || withdrawTime.Int64() < vm.Db.SnapshotBlock(block.SnapshotHash()).Timestamp()+mortgageTime {
		return quotaLeft, ErrInvalidData
	}
	return quotaLeft, nil
}

// cancel mortgage ViteToken
func (p *mortgage) doSendCancelMortgage(vm *VM, block VmAccountBlock, quotaLeft uint64) (uint64, error) {
	quotaLeft, err := useQuota(quotaLeft, cancelMortgageGas)
	if err != nil {
		return quotaLeft, err
	}
	quotaLeft, err = useQuotaForData(block.Data(), quotaLeft)
	if err != nil {
		return quotaLeft, err
	}
	if block.Amount().Sign() > 0 ||
		allZero(block.Data()[21:53]) ||
		!isUserAccount(vm.Db, block.AccountAddress()) {
		return quotaLeft, ErrInvalidData
	}
	// data: methodSelector(0:4) + beneficial address(4:36) + amount(36:68)
	address, err := types.BytesToAddress(new(big.Int).SetBytes(block.Data()[4:36]).Bytes())
	if err != nil || !vm.Db.IsExistAddress(address) {
		return quotaLeft, ErrInvalidData
	}
	return quotaLeft, nil
}

func (p *mortgage) doReceive(vm *VM, block VmAccountBlock) error {
	if len(block.Data()) == 68 && bytes.Equal(block.Data()[0:4], DataMortgage) {
		return p.doReceiveMortgage(vm, block)
	} else if len(block.Data()) == 68 && bytes.Equal(block.Data()[0:4], DataCancelMortgage) {
		return p.doReceiveCancelMortgage(vm, block)
	}
	return nil
}

func (p *mortgage) doReceiveMortgage(vm *VM, block VmAccountBlock) error {
	intPool := poolOfIntPools.get()
	defer poolOfIntPools.put(intPool)
	// data: methodSelector(0:4) + beneficial address(4:36) + withdrawTime(36:68)
	// storage key for quota: hash(beneficial)
	locHashQuotaAmount := types.DataHash(block.Data()[16:36])
	// storage key for mortgage: hash(owner, hash(beneficial))
	locHashMortgage := types.DataHash(append(block.AccountAddress().Bytes(), locHashQuotaAmount.Bytes()...))
	// storage value for mortgage: mortgage amount(0:32) + withdrawTime(32:64)
	old := vm.Db.Storage(block.ToAddress(), locHashMortgage)
	withdrawTime := intPool.get().SetBytes(block.Data()[36:68])
	defer intPool.put(withdrawTime)
	amount := intPool.getZero()
	defer intPool.put(amount)
	if len(old) >= 64 {
		oldWithdrawTime := intPool.get().SetBytes(old[32:64])
		defer intPool.put(oldWithdrawTime)
		if withdrawTime.Int64() < oldWithdrawTime.Int64() {
			return ErrInvalidData
		}
		amount.SetBytes(old[0:32])
	}
	amount.Add(amount, block.Amount())
	vm.Db.SetStorage(block.ToAddress(), locHashMortgage, joinBytes(LeftPadBytes(amount.Bytes(), 32), LeftPadBytes(withdrawTime.Bytes(), 32)))

	// storage value for quota: quota amount(0:32)
	oldQuotaAmount := vm.Db.Storage(block.ToAddress(), locHashQuotaAmount)
	quotaAmount := intPool.getZero()
	if len(oldQuotaAmount) >= 32 {
		quotaAmount.SetBytes(oldQuotaAmount[0:32])
	}
	quotaAmount.Add(quotaAmount, block.Amount())
	vm.Db.SetStorage(block.ToAddress(), locHashQuotaAmount, LeftPadBytes(quotaAmount.Bytes(), 32))
	intPool.put(quotaAmount)
	return nil
}
func (p *mortgage) doReceiveCancelMortgage(vm *VM, block VmAccountBlock) error {
	// data: methodSelector(0:4) + beneficial address(4:36) + amount(36:68)
	locHashQuotaAmount := types.DataHash(block.Data()[16:36])
	locHashMortgage := types.DataHash(append(block.AccountAddress().Bytes(), locHashQuotaAmount.Bytes()...))
	old := vm.Db.Storage(block.ToAddress(), locHashMortgage)
	if len(old) < 64 {
		return ErrInvalidData
	}
	intPool := poolOfIntPools.get()
	defer poolOfIntPools.put(intPool)
	withdrawTime := intPool.get().SetBytes(old[32:64])
	defer intPool.put(withdrawTime)
	if withdrawTime.Int64() > vm.Db.SnapshotBlock(block.SnapshotHash()).Timestamp() {
		return ErrInvalidData
	}
	amount := intPool.get().SetBytes(old[0:32])
	defer intPool.put(amount)
	withdrawAmount := intPool.get().SetBytes(block.Data()[36:68])
	if amount.Cmp(withdrawAmount) < 0 {
		return ErrInvalidData
	}
	amount.Sub(amount, withdrawAmount)

	oldQuota := vm.Db.Storage(block.ToAddress(), locHashQuotaAmount)
	if len(oldQuota) < 32 {
		return ErrInvalidData
	}
	quotaAmount := intPool.get().SetBytes(oldQuota[0:32])
	defer intPool.put(quotaAmount)
	if quotaAmount.Cmp(withdrawAmount) < 0 {
		return ErrInvalidData
	}
	quotaAmount.Sub(quotaAmount, withdrawAmount)

	if amount.Sign() == 0 {
		vm.Db.SetStorage(block.ToAddress(), locHashMortgage, nil)
	} else {
		vm.Db.SetStorage(block.ToAddress(), locHashMortgage, joinBytes(LeftPadBytes(amount.Bytes(), 32), old[32:64]))
	}

	if quotaAmount.Sign() == 0 {
		vm.Db.SetStorage(block.ToAddress(), locHashQuotaAmount, nil)
	} else {
		vm.Db.SetStorage(block.ToAddress(), locHashQuotaAmount, LeftPadBytes(quotaAmount.Bytes(), 32))
	}

	// append refund block
	refundBlock := vm.createBlock(block.ToAddress(), block.AccountAddress(), BlockTypeSendCall, block.Depth()+1)
	refundBlock.SetAmount(withdrawAmount)
	refundBlock.SetTokenId(viteTokenTypeId)
	refundBlock.SetHeight(intPool.get().Add(block.Height(), Big1))
	vm.blockList = append(vm.blockList, refundBlock)
	return nil
}

type consensusGroup struct{}

func (p *consensusGroup) createFee(vm *VM, block VmAccountBlock) *big.Int {
	return new(big.Int).Set(createConsensusGroupFee)
}

var (
	DataCreateConsensusGroup = []byte{0xce, 0xe9, 0xbd, 0xc9}
)

// create consensus group
func (p *consensusGroup) doSend(vm *VM, block VmAccountBlock, quotaLeft uint64) (uint64, error) {
	quotaLeft, err := useQuota(quotaLeft, createConsensusGroupGas)
	if err != nil {
		return quotaLeft, err
	}
	if len(block.Data()) < 384 || (len(block.Data())-4)%32 != 0 || !bytes.Equal(block.Data()[0:4], DataCreateConsensusGroup) ||
		block.Amount().Sign() != 0 ||
		!isUserAccount(vm.Db, block.AccountAddress()) {
		return quotaLeft, ErrInvalidData
	}
	if err := checkCreateConsensusGroupData(vm, block.Data()); err != nil {
		return quotaLeft, err
	}
	// data: methodSelector(0:4) + gid(4:36) + ConsensusGroup
	gid := DataToGid(block.AccountAddress().Bytes(), block.Height().Bytes(), block.PrevHash().Bytes(), block.SnapshotHash().Bytes())
	if allZero(gid.Bytes()) || isExistGid(vm.Db, gid) {
		return quotaLeft, ErrInvalidData
	}
	copy(block.Data()[4:36], LeftPadBytes(gid.Bytes(), 32))
	quotaLeft, err = useQuotaForData(block.Data(), quotaLeft)
	if err != nil {
		return quotaLeft, err
	}
	return quotaLeft, nil
}
func checkCreateConsensusGroupData(vm *VM, data []byte) error {
	var tmp big.Int
	tmp.SetBytes(data[36:68])
	nodeCount := tmp.Uint64()
	if tmp.BitLen() > 8 || nodeCount < cgNodeCountMin || nodeCount > cgNodeCountMax {
		return ErrInvalidData
	}
	tmp.SetBytes(data[68:100])
	interval := tmp.Int64()
	if tmp.BitLen() > 64 || interval < cgIntervalMin || interval > cgIntervalMax {
		return ErrInvalidData
	}
	if _, err := checkCondition(vm, data, tmp, 100, "counting"); err != nil {
		return ErrInvalidData
	}
	if _, err := checkCondition(vm, data, tmp, 164, "register"); err != nil {
		return ErrInvalidData
	}
	if endIndex, err := checkCondition(vm, data, tmp, 228, "vote"); err != nil || uint64(len(data)) > endIndex {
		return ErrInvalidData
	}
	return nil
}
func checkCondition(vm *VM, data []byte, tmp big.Int, startIndex uint64, conditionIdPrefix string) (uint64, error) {
	conditionId := tmp.SetBytes(data[startIndex : startIndex+32])
	if conditionId.BitLen() > 8 {
		return 0, ErrInvalidData
	}
	condition, ok := SimpleCountingRuleList[CountingRuleCode(conditionIdPrefix+conditionId.String())]
	if !ok {
		return 0, ErrInvalidData
	}
	tmp.SetBytes(data[startIndex+32 : startIndex+64])
	conditionParamAddressStart := tmp.Uint64() + 4
	if tmp.BitLen() > 64 || uint64(len(data)) < conditionParamAddressStart+32 {
		return 0, ErrInvalidData
	}
	tmp.SetBytes(data[conditionParamAddressStart : conditionParamAddressStart+32])
	conditionParamLength := tmp.Uint64()
	if conditionParamLength > 0 {
		conditionParamWordCount := toWordSize(conditionParamLength)
		if tmp.Cmp(tt256m1) > 0 || uint64(len(data)) < conditionParamAddressStart+32+conditionParamWordCount*32 ||
			!allZero(data[conditionParamAddressStart+32+conditionParamLength:conditionParamAddressStart+32+conditionParamWordCount*32]) {
			return 0, ErrInvalidData
		}
		if ok := condition.checkParam(data[conditionParamAddressStart+32:conditionParamAddressStart+32+conditionParamLength], vm.Db); !ok {
			return 0, ErrInvalidData
		}
		return conditionParamAddressStart + 32 + conditionParamWordCount*32, nil
	}
	return conditionParamAddressStart + 32, nil
}

func (p *consensusGroup) doReceive(vm *VM, block VmAccountBlock) error {
	gid, _ := BigToGid(new(big.Int).SetBytes(block.Data()[4:36]))
	locHash := types.DataHash(gid.Bytes())
	if len(vm.Db.Storage(block.ToAddress(), locHash)) > 0 {
		return ErrIdCollision
	}
	vm.Db.SetStorage(block.ToAddress(), locHash, block.Data()[36:])
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
