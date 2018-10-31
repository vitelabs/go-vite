package vm

import (
	"bytes"
	"encoding/hex"
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/contracts"
	"github.com/vitelabs/go-vite/vm/quota"
	"github.com/vitelabs/go-vite/vm/util"
	"github.com/vitelabs/go-vite/vm_context"
	"math/big"
	"strconv"
	"testing"
	"time"
)

func init() {
	InitVmConfig(false, false)
}

func TestVmRun(t *testing.T) {
	// prepare db
	viteTotalSupply := new(big.Int).Mul(big.NewInt(1e9), util.AttovPerVite)
	db, addr1, _, hash12, snapshot, _ := prepareDb(viteTotalSupply)
	blockTime := time.Now()

	/*
	* contract code
	* pragma solidity ^0.4.18;
	* contract MyContract {
	* 	uint256 v;
	* 	constructor() payable public {}
	* 	function AddV(uint256 addition) payable public {
	* 	   v = v + addition;
	* 	}
	* }
	 */
	balance1 := new(big.Int).Set(viteTotalSupply)
	// send create
	data13, _ := hex.DecodeString("00000000000000000002608060405260858060116000396000f300608060405260043610603e5763ffffffff7c0100000000000000000000000000000000000000000000000000000000600035041663f021ab8f81146043575b600080fd5b604c600435604e565b005b6000805490910190555600a165627a7a72305820b8d8d60a46c6ac6569047b17b012aa1ea458271f9bc8078ef0cff9208999d0900029")
	hash13 := types.DataHash([]byte{1, 3})
	block13 := &ledger.AccountBlock{
		Height:         3,
		AccountAddress: addr1,
		BlockType:      ledger.BlockTypeSendCreate,
		PrevHash:       hash12,
		Amount:         big.NewInt(1e18),
		Fee:            big.NewInt(0),
		TokenId:        ledger.ViteTokenId,
		SnapshotHash:   snapshot.Hash,
		Data:           data13,
		Timestamp:      &blockTime,
	}
	vm := NewVM()
	vm.Debug = true
	db.addr = addr1
	sendCreateBlockList, isRetry, err := vm.Run(db, block13, nil)
	balance1.Sub(balance1, block13.Amount)
	balance1.Sub(balance1, createContractFee)
	if len(sendCreateBlockList) != 1 ||
		isRetry ||
		err != nil ||
		sendCreateBlockList[0].AccountBlock.Quota != 28936 ||
		sendCreateBlockList[0].AccountBlock.Fee.Cmp(createContractFee) != 0 ||
		db.balanceMap[addr1][ledger.ViteTokenId].Cmp(balance1) != 0 {
		t.Fatalf("send create transaction error")
	}
	db.accountBlockMap[addr1][hash13] = sendCreateBlockList[0].AccountBlock

	// receive create
	addr2 := sendCreateBlockList[0].AccountBlock.ToAddress
	db.storageMap[contracts.AddressPledge][string(contracts.GetPledgeBeneficialKey(addr2))], _ = contracts.ABIPledge.PackVariable(contracts.VariableNamePledgeBeneficial, new(big.Int).Mul(big.NewInt(1e9), big.NewInt(1e18)))
	balance2 := big.NewInt(0)

	hash21 := types.DataHash([]byte{2, 1})
	block21 := &ledger.AccountBlock{
		Height:         1,
		AccountAddress: addr2,
		FromBlockHash:  hash13,
		BlockType:      ledger.BlockTypeReceive,
		SnapshotHash:   snapshot.Hash,
		Timestamp:      &blockTime,
	}
	vm = NewVM()
	vm.Debug = true
	db.addr = addr2
	receiveCreateBlockList, isRetry, err := vm.Run(db, block21, sendCreateBlockList[0].AccountBlock)
	balance2.Add(balance2, block13.Amount)
	if len(receiveCreateBlockList) != 1 || isRetry || err != nil ||
		receiveCreateBlockList[0].AccountBlock.Quota != 0 ||
		*db.contractGidMap[addr1] != types.DELEGATE_GID ||
		db.balanceMap[addr2][ledger.ViteTokenId].Cmp(balance2) != 0 {
		t.Fatalf("receive create transaction error")
	}
	db.accountBlockMap[addr2] = make(map[types.Hash]*ledger.AccountBlock)
	db.accountBlockMap[addr2][hash21] = receiveCreateBlockList[0].AccountBlock

	// send call
	data14, _ := hex.DecodeString("f021ab8f0000000000000000000000000000000000000000000000000000000000000005")
	hash14 := types.DataHash([]byte{1, 4})
	block14 := &ledger.AccountBlock{
		Height:         4,
		AccountAddress: addr1,
		ToAddress:      addr2,
		BlockType:      ledger.BlockTypeSendCall,
		Fee:            big.NewInt(0),
		PrevHash:       hash13,
		Amount:         big.NewInt(1e18),
		TokenId:        ledger.ViteTokenId,
		SnapshotHash:   snapshot.Hash,
		Data:           data14,
		Timestamp:      &blockTime,
	}
	vm = NewVM()
	vm.Debug = true
	db.addr = addr1
	sendCallBlockList, isRetry, err := vm.Run(db, block14, nil)
	balance1.Sub(balance1, block14.Amount)
	if len(sendCallBlockList) != 1 || isRetry || err != nil ||
		sendCallBlockList[0].AccountBlock.Quota != 21464 ||
		db.balanceMap[addr1][ledger.ViteTokenId].Cmp(balance1) != 0 {
		t.Fatalf("send call transaction error")
	}
	db.accountBlockMap[addr1][hash14] = sendCallBlockList[0].AccountBlock

	snapshot = &ledger.SnapshotBlock{Height: 3, Timestamp: snapshot.Timestamp, Hash: types.DataHash([]byte{10, 3})}
	db.snapshotBlockList = append(db.snapshotBlockList, snapshot)

	// receive call
	hash22 := types.DataHash([]byte{2, 2})
	block22 := &ledger.AccountBlock{
		Height:         2,
		AccountAddress: addr2,
		FromBlockHash:  hash14,
		PrevHash:       hash21,
		BlockType:      ledger.BlockTypeReceive,
		SnapshotHash:   snapshot.Hash,
		Timestamp:      &blockTime,
	}
	vm = NewVM()
	vm.Debug = true
	db.addr = addr2
	receiveCallBlockList, isRetry, err := vm.Run(db, block22, sendCallBlockList[0].AccountBlock)
	balance2.Add(balance2, block14.Amount)
	if len(receiveCallBlockList) != 1 || isRetry || err != nil ||
		receiveCallBlockList[0].AccountBlock.Quota != 41330 ||
		db.balanceMap[addr2][ledger.ViteTokenId].Cmp(big.NewInt(2e18)) != 0 {
		t.Fatalf("receive call transaction error")
	}
	db.accountBlockMap[addr2][hash22] = receiveCallBlockList[0].AccountBlock

	// send call error, insufficient balance
	data15, _ := hex.DecodeString("f021ab8f0000000000000000000000000000000000000000000000000000000000000005")
	block15 := &ledger.AccountBlock{
		Height:         5,
		AccountAddress: addr1,
		ToAddress:      addr2,
		BlockType:      ledger.BlockTypeSendCall,
		Fee:            big.NewInt(0),
		PrevHash:       hash14,
		Amount:         viteTotalSupply,
		TokenId:        ledger.ViteTokenId,
		SnapshotHash:   snapshot.Hash,
		Data:           data15,
		Timestamp:      &blockTime,
	}
	vm = NewVM()
	vm.Debug = true
	db.addr = addr1
	sendCallBlockList2, isRetry, err := vm.Run(db, block15, nil)
	if len(sendCallBlockList2) != 0 || err != util.ErrInsufficientBalance {
		t.Fatalf("send call transaction 2 error")
	}
	// receive call error, execution revert
	data15, _ = hex.DecodeString("")
	hash15 := types.DataHash([]byte{1, 5})
	block15 = &ledger.AccountBlock{
		Height:         5,
		AccountAddress: addr1,
		ToAddress:      addr2,
		BlockType:      ledger.BlockTypeSendCall,
		Fee:            big.NewInt(0),
		PrevHash:       hash14,
		Amount:         big.NewInt(50),
		TokenId:        ledger.ViteTokenId,
		SnapshotHash:   snapshot.Hash,
		Data:           data15,
		Timestamp:      &blockTime,
	}
	vm = NewVM()
	vm.Debug = true
	db.addr = addr1
	sendCallBlockList2, isRetry, err = vm.Run(db, block15, nil)
	db.accountBlockMap[addr1][hash15] = sendCallBlockList2[0].AccountBlock
	// receive call
	hash23 := types.DataHash([]byte{2, 3})
	block23 := &ledger.AccountBlock{
		Height:         3,
		AccountAddress: addr2,
		FromBlockHash:  hash15,
		PrevHash:       hash22,
		BlockType:      ledger.BlockTypeReceive,
		SnapshotHash:   snapshot.Hash,
		Timestamp:      &blockTime,
	}
	vm = NewVM()
	vm.Debug = true
	db.addr = addr2
	receiveCallBlockList2, isRetry, err := vm.Run(db, block23, sendCallBlockList2[0].AccountBlock)
	if len(receiveCallBlockList2) != 1 || isRetry || err != util.ErrExecutionReverted ||
		receiveCallBlockList2[0].AccountBlock.Quota != 21046 {
		t.Fatalf("receive call transaction error")
	}
	db.accountBlockMap[addr2][hash23] = receiveCallBlockList2[0].AccountBlock
}

func TestDelegateCall(t *testing.T) {
	// prepare db, add account1, add account2 with code, add account3 with code
	db := NewNoDatabase()
	// code1 return 1+2
	addr1, _, _ := types.CreateAddress()
	code1 := []byte{byte(PUSH1), 1, byte(PUSH1), 2, byte(ADD), byte(PUSH1), 32, byte(DUP1), byte(SWAP2), byte(SWAP1), byte(MSTORE), byte(PUSH1), 32, byte(SWAP1), byte(RETURN)}
	db.codeMap = make(map[types.Address][]byte)
	db.codeMap[addr1] = code1

	addr2, _, _ := types.CreateAddress()
	code2 := helper.JoinBytes([]byte{byte(PUSH1), 32, byte(PUSH1), 0, byte(PUSH1), 0, byte(PUSH1), 0, byte(PUSH20)}, addr1.Bytes(), []byte{byte(DELEGATECALL), byte(PUSH1), 32, byte(PUSH1), 0, byte(RETURN)})
	db.codeMap[addr2] = code2
	blockTime := time.Now()

	vm := NewVM()
	vm.Debug = true
	sendCallBlock := ledger.AccountBlock{
		AccountAddress: addr1,
		ToAddress:      addr2,
		BlockType:      ledger.BlockTypeSendCall,
		Amount:         big.NewInt(10),
		Fee:            big.NewInt(0),
		TokenId:        ledger.ViteTokenId,
	}
	receiveCallBlock := &ledger.AccountBlock{
		AccountAddress: addr2,
		BlockType:      ledger.BlockTypeReceive,
		Timestamp:      &blockTime,
	}
	c := newContract(
		addr2,
		addr2,
		&vm_context.VmAccountBlock{receiveCallBlock, db},
		&sendCallBlock,
		1000000,
		0)
	c.setCallCode(addr2, code2)
	ret, err := c.run(vm)
	if err != nil || !bytes.Equal(ret, helper.LeftPadBytes([]byte{3}, 32)) {
		t.Fatalf("delegate call error")
	}
}

func TestCalcQuotaV2(t *testing.T) {
	quota.InitQuotaConfig(false)
	// prepare db
	addr1, _, _ := types.CreateAddress()
	db := NewNoDatabase()
	timestamp := time.Unix(1536214502, 0)
	snapshot1 := &ledger.SnapshotBlock{Height: 1, Timestamp: &timestamp, Hash: types.DataHash([]byte{10, 1})}
	db.snapshotBlockList = append(db.snapshotBlockList, snapshot1)

	difficulty := quota.DefaultDifficulty
	quotaForTx := uint64(21000)
	quotaLimit := uint64(987000)
	minPledgeAmount := new(big.Int).Mul(big.NewInt(1000), big.NewInt(1e18))
	maxPledgeAmount := new(big.Int).Mul(big.NewInt(1e9), big.NewInt(1e18))
	db.storageMap[contracts.AddressPledge] = make(map[string][]byte)
	db.addr = addr1

	// genesis account block without PoW, pledge amount reaches quota limit
	quotaTotal, quotaAddition, err := quota.CalcQuotaV2(db, addr1, maxPledgeAmount, helper.Big0)
	if quotaTotal != quotaLimit || quotaAddition != uint64(0) || err != nil {
		t.Fatalf("calc quota error, genesis account block without PoW, pledge amount reaches quota limit")
	}
	// genesis account block with PoW, pledge amount reaches quota limit
	quotaTotal, quotaAddition, err = quota.CalcQuotaV2(db, addr1, maxPledgeAmount, difficulty)
	if quotaTotal != quotaLimit || quotaAddition != uint64(0) || err != nil {
		t.Fatalf("calc quota error, genesis account block with PoW, pledge amount reaches quota limit")
	}

	// genesis account block without PoW, pledge amount reaches no quota limit
	quotaTotal, quotaAddition, err = quota.CalcQuotaV2(db, addr1, minPledgeAmount, helper.Big0)
	if quotaTotal != quotaForTx || quotaAddition != uint64(0) || err != nil {
		t.Fatalf("calc quota error, genesis account block without PoW, pledge amount reaches no quota limit")
	}
	// genesis account block without PoW, pledge amount reaches no quota limit
	quotaTotal, quotaAddition, err = quota.CalcQuotaV2(db, addr1, minPledgeAmount, difficulty)
	if quotaTotal != quotaForTx || quotaAddition != uint64(0) || err != nil {
		t.Fatalf("calc quota error, genesis account block without PoW, pledge amount reaches no quota limit")
	}

	// genesis account block with PoW, pledge amount reaches quota limit
	quotaTotal, quotaAddition, err = quota.CalcQuota(db, addr1, maxPledgeAmount, nil)
	if quotaTotal != quotaLimit || quotaAddition != uint64(0) || err != nil {
		t.Fatalf("calc quota error, genesis account block with PoW, pledge amount reaches quota limit")
	}
	// genesis account block without PoW, pledge amount reaches no quota limit
	quotaTotal, quotaAddition, err = quota.CalcQuota(db, addr1, minPledgeAmount, nil)
	if quotaTotal != quotaForTx || quotaAddition != uint64(0) || err != nil {
		t.Fatalf("calc quota error, genesis account block without PoW, pledge amount reaches no quota limit")
	}

	blockTime := time.Now()

	// prepare db
	hash11 := types.DataHash([]byte{1, 1})
	block11 := &ledger.AccountBlock{
		Height:         1,
		ToAddress:      addr1,
		AccountAddress: addr1,
		BlockType:      ledger.BlockTypeSendCall,
		Fee:            big.NewInt(0),
		Amount:         big.NewInt(1),
		TokenId:        ledger.ViteTokenId,
		SnapshotHash:   snapshot1.Hash,
		Timestamp:      &blockTime,
		Quota:          quotaForTx,
	}
	db.accountBlockMap[addr1] = make(map[types.Hash]*ledger.AccountBlock)
	db.accountBlockMap[addr1][hash11] = block11

	snapshot2 := &ledger.SnapshotBlock{Height: 2, Timestamp: &timestamp, Hash: types.DataHash([]byte{10, 2})}
	db.snapshotBlockList = append(db.snapshotBlockList, snapshot2)

	// first account block without PoW, pledge amount reaches quota limit, snapshot height gap=1
	quotaTotal, quotaAddition, err = quota.CalcQuotaV2(db, addr1, maxPledgeAmount, helper.Big0)
	if quotaTotal != quotaLimit || quotaAddition != uint64(0) || err != nil {
		t.Fatalf("calc quota error")
	}
	// first account block with PoW, pledge amount reaches quota limit, snapshot height gap=1
	quotaTotal, quotaAddition, err = quota.CalcQuotaV2(db, addr1, maxPledgeAmount, difficulty)
	if quotaTotal != quotaLimit || quotaAddition != uint64(0) || err != nil {
		t.Fatalf("calc quota error")
	}

	// first account block without PoW, pledge amount reaches no quota limit, snapshot height gap=1
	quotaTotal, quotaAddition, err = quota.CalcQuotaV2(db, addr1, minPledgeAmount, helper.Big0)
	if quotaTotal != quotaForTx || quotaAddition != uint64(0) || err != nil {
		t.Fatalf("calc quota error")
	}
	// first account block without PoW, pledge amount reaches no quota limit, snapshot height gap=1
	quotaTotal, quotaAddition, err = quota.CalcQuotaV2(db, addr1, minPledgeAmount, difficulty)
	if quotaTotal != quotaForTx || quotaAddition != uint64(0) || err != nil {
		t.Fatalf("calc quota error")
	}
	// first account block with PoW, pledge amount reaches no quota limit, snapshot height gap=1
	quotaTotal, quotaAddition, err = quota.CalcQuotaV2(db, addr1, new(big.Int).Mul(big.NewInt(1500), big.NewInt(1e18)), difficulty)
	if quotaTotal != quotaForTx*2 || quotaAddition != quotaForTx || err != nil {
		t.Fatalf("calc quota error")
	}

	// prepare db
	snapshot3 := &ledger.SnapshotBlock{Height: 3, Timestamp: &timestamp, Hash: types.DataHash([]byte{10, 3})}
	db.snapshotBlockList = append(db.snapshotBlockList, snapshot3)

	// first account block without PoW, pledge amount reaches quota limit, snapshot height gap=2
	quotaTotal, quotaAddition, err = quota.CalcQuotaV2(db, addr1, maxPledgeAmount, helper.Big0)
	if quotaTotal != quotaLimit || quotaAddition != uint64(0) || err != nil {
		t.Fatalf("calc quota error")
	}
	// first account block with PoW, pledge amount reaches quota limit, snapshot height gap=2
	quotaTotal, quotaAddition, err = quota.CalcQuotaV2(db, addr1, maxPledgeAmount, difficulty)
	if quotaTotal != quotaLimit || quotaAddition != uint64(0) || err != nil {
		t.Fatalf("calc quota error")
	}

	// first account block without PoW, pledge amount reaches no quota limit, snapshot height gap=2
	quotaTotal, quotaAddition, err = quota.CalcQuotaV2(db, addr1, new(big.Int).Mul(big.NewInt(1500), big.NewInt(1e18)), helper.Big0)
	if quotaTotal != quotaForTx*2 || quotaAddition != uint64(0) || err != nil {
		t.Fatalf("calc quota error")
	}
	// first account block without PoW, pledge amount reaches no quota limit, snapshot height gap=2
	quotaTotal, quotaAddition, err = quota.CalcQuotaV2(db, addr1, new(big.Int).Mul(big.NewInt(700), big.NewInt(1e18)), difficulty)
	if quotaTotal != quotaForTx*2 || quotaAddition != uint64(21000) || err != nil {
		t.Fatalf("calc quota error")
	}

	// prepare db
	hash12 := types.DataHash([]byte{1, 2})
	block12 := &ledger.AccountBlock{
		Height:         2,
		ToAddress:      addr1,
		AccountAddress: addr1,
		BlockType:      ledger.BlockTypeSendCall,
		Fee:            big.NewInt(0),
		Amount:         big.NewInt(1),
		TokenId:        ledger.ViteTokenId,
		SnapshotHash:   snapshot3.Hash,
		PrevHash:       hash11,
		Quota:          quotaForTx,
		Timestamp:      &blockTime,
	}
	db.accountBlockMap[addr1][hash12] = block12

	// second account block referring to same snapshotBlock without PoW, pledge amount reaches quota limit, snapshot height gap=2
	quotaTotal, quotaAddition, err = quota.CalcQuotaV2(db, addr1, maxPledgeAmount, helper.Big0)
	if quotaTotal != quotaLimit-quotaForTx || quotaAddition != uint64(0) || err != nil {
		t.Fatalf("calc quota error")
	}
	// second account block referring to same snapshotBlock with PoW, pledge amount reaches quota limit, snapshot height gap=2
	quotaTotal, quotaAddition, err = quota.CalcQuotaV2(db, addr1, maxPledgeAmount, difficulty)
	if quotaTotal != quotaLimit-quotaForTx || quotaAddition != uint64(0) || err != nil {
		t.Fatalf("calc quota error")
	}

	// second account block referring to same snapshotBlock without PoW, pledge amount reaches no quota limit, snapshot height gap=2
	// error case, quotaUsed > quotaInit
	quotaTotal, quotaAddition, err = quota.CalcQuotaV2(db, addr1, big.NewInt(10), helper.Big0)
	if quotaTotal != uint64(0) || quotaAddition != uint64(0) || err != nil {
		t.Fatalf("calc quota error")
	}
	// second account block referring to same snapshotBlock without PoW, pledge amount reaches no quota limit, snapshot height gap=2
	quotaTotal, quotaAddition, err = quota.CalcQuotaV2(db, addr1, new(big.Int).Mul(big.NewInt(700), big.NewInt(1e18)), difficulty)
	if quotaTotal != uint64(21000) || quotaAddition != uint64(21000) || err != nil {
		t.Fatalf("calc quota error")
	}

	// prepare db
	hash13 := types.DataHash([]byte{1, 3})
	block13 := &ledger.AccountBlock{
		Height:         3,
		ToAddress:      addr1,
		AccountAddress: addr1,
		BlockType:      ledger.BlockTypeReceiveError,
		Fee:            big.NewInt(0),
		Amount:         big.NewInt(1),
		TokenId:        ledger.ViteTokenId,
		SnapshotHash:   snapshot3.Hash,
		PrevHash:       hash12,
		Quota:          uint64(0),
		Timestamp:      &blockTime,
	}
	db.accountBlockMap[addr1][hash13] = block13
	// second account block referring to same snapshotBlock with PoW, first block receive error
	quotaTotal, quotaAddition, err = quota.CalcQuotaV2(db, addr1, maxPledgeAmount, difficulty)
	if quotaTotal != uint64(0) || quotaAddition != uint64(0) || err != nil {
		t.Fatalf("calc quota error")
	}
	// second account block referring to same snapshotBlock without PoW, first block receive error
	quotaTotal, quotaAddition, err = quota.CalcQuotaV2(db, addr1, maxPledgeAmount, helper.Big0)
	if quotaTotal != uint64(0) || quotaAddition != uint64(0) || err != nil {
		t.Fatalf("calc quota error")
	}

	snapshot4 := &ledger.SnapshotBlock{Height: 4, Timestamp: &timestamp, Hash: types.DataHash([]byte{10, 4})}
	db.snapshotBlockList = append(db.snapshotBlockList, snapshot4)
	hash14 := types.DataHash([]byte{1, 4})
	block14 := &ledger.AccountBlock{
		Height:         4,
		ToAddress:      addr1,
		AccountAddress: addr1,
		BlockType:      ledger.BlockTypeSendCall,
		Fee:            big.NewInt(0),
		Amount:         big.NewInt(1),
		TokenId:        ledger.ViteTokenId,
		SnapshotHash:   snapshot4.Hash,
		PrevHash:       hash13,
		Quota:          uint64(0),
		Timestamp:      &blockTime,
		Nonce:          []byte{1},
	}
	db.accountBlockMap[addr1][hash14] = block14

	// second account block referring to same snapshotBlock without PoW, first block calc PoW, pledge amount reaches quota limit, snapshot height gap=1
	quotaTotal, quotaAddition, err = quota.CalcQuotaV2(db, addr1, maxPledgeAmount, helper.Big0)
	if quotaTotal != quotaLimit || quotaAddition != uint64(0) || err != nil {
		t.Fatalf("calc quota error")
	}
	// second account block referring to same snapshotBlock with PoW, first block calc PoW, pledge amount reaches quota limit, snapshot height gap=1
	// error case
	quotaTotal, quotaAddition, err = quota.CalcQuotaV2(db, addr1, maxPledgeAmount, difficulty)
	if quotaTotal != 0 || quotaAddition != uint64(0) || err == nil {
		t.Fatalf("calc quota error")
	}

	// second account block referring to same snapshotBlock without PoW, first block calc PoW, pledge amount reaches no quota limit, snapshot height gap=1
	quotaTotal, quotaAddition, err = quota.CalcQuotaV2(db, addr1, minPledgeAmount, helper.Big0)
	if quotaTotal != quotaForTx || quotaAddition != uint64(0) || err != nil {
		t.Fatalf("calc quota error")
	}
	// second account block referring to same snapshotBlock without PoW, first block calc PoW, pledge amount reaches no quota limit, snapshot height gap=1
	// error case
	quotaTotal, quotaAddition, err = quota.CalcQuotaV2(db, addr1, new(big.Int).Mul(big.NewInt(700), big.NewInt(1e18)), difficulty)
	if quotaTotal != uint64(0) || quotaAddition != uint64(0) || err == nil {
		t.Fatalf("calc quota error")
	}
}

func BenchmarkVMTransfer(b *testing.B) {
	viteTotalSupply := new(big.Int).Mul(big.NewInt(1e9), util.AttovPerVite)
	db, addr1, _, hash12, _, timestamp := prepareDb(viteTotalSupply)
	for i := 3; i < b.N+3; i++ {
		timestamp = timestamp + 1
		ti := time.Unix(timestamp, 0)
		snapshoti := &ledger.SnapshotBlock{Height: uint64(i), Timestamp: &ti, Hash: types.DataHash([]byte(strconv.Itoa(i)))}
		db.snapshotBlockList = append(db.snapshotBlockList, snapshoti)
	}

	blockTime := time.Now()

	// send call
	b.ResetTimer()
	prevHash := hash12
	addr2, _, _ := types.CreateAddress()
	amount := big.NewInt(1)
	db.addr = addr1
	for i := 3; i < b.N+3; i++ {
		hashi := types.DataHash([]byte(strconv.Itoa(i)))
		blocki := &ledger.AccountBlock{
			Height:         uint64(i),
			AccountAddress: addr1,
			ToAddress:      addr2,
			BlockType:      ledger.BlockTypeSendCall,
			PrevHash:       prevHash,
			Amount:         amount,
			TokenId:        ledger.ViteTokenId,
			SnapshotHash:   types.DataHash([]byte(strconv.Itoa(i))),
			Timestamp:      &blockTime,
		}
		vm := NewVM()
		sendCallBlockList, _, err := vm.Run(db, blocki, nil)
		if err != nil {
			b.Fatal(err)
		}
		db.accountBlockMap[addr1][hashi] = sendCallBlockList[0].AccountBlock
		prevHash = hashi
	}
}

func TestVmForTest(t *testing.T) {
	InitVmConfig(true, true)
	db, _, _, _, snapshot2, _ := prepareDb(big.NewInt(0))
	blockTime := time.Now()

	addr1, _, _ := types.CreateAddress()
	//hash11 := types.DataHash([]byte{1, 1})
	block11 := &ledger.AccountBlock{
		Height:         1,
		AccountAddress: addr1,
		BlockType:      ledger.BlockTypeSendCall,
		Amount:         big.NewInt(0),
		Fee:            big.NewInt(0),
		TokenId:        ledger.ViteTokenId,
		SnapshotHash:   snapshot2.Hash,
		Timestamp:      &blockTime,
	}
	vm := NewVM()
	vm.Debug = true
	db.addr = addr1
	sendCallBlockList, isRetry, err := vm.Run(db, block11, nil)
	if len(sendCallBlockList) != 1 || isRetry || err != nil {
		t.Fatalf("init test vm config failed")
	}
}
