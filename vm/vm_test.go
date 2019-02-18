package vm

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/common/fork"
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/contracts/abi"
	"github.com/vitelabs/go-vite/vm/quota"
	"github.com/vitelabs/go-vite/vm/util"
	"github.com/vitelabs/go-vite/vm_context"
	"io/ioutil"
	"math/big"
	"os"
	"strconv"
	"testing"
	"time"
)

func init() {
	InitVmConfig(false, false, true, common.HomeDir())
	initFork()
}

func initFork() {
	fork.SetForkPoints(&config.ForkPoints{Smart: &config.ForkPoint{Height: 2}, Mint: &config.ForkPoint{Height: 20}})
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
	data13, _ := hex.DecodeString("0000000000000000000201608060405260858060116000396000f300608060405260043610603e5763ffffffff7c0100000000000000000000000000000000000000000000000000000000600035041663f021ab8f81146043575b600080fd5b604c600435604e565b005b6000805490910190555600a165627a7a72305820b8d8d60a46c6ac6569047b17b012aa1ea458271f9bc8078ef0cff9208999d0900029")
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
		Hash:           hash13,
	}
	vm := NewVM()
	//vm.Debug = true
	db.addr = addr1
	sendCreateBlockList, isRetry, err := vm.Run(db, block13, nil)
	balance1.Sub(balance1, block13.Amount)
	balance1.Sub(balance1, createContractFee)
	if len(sendCreateBlockList) != 1 ||
		isRetry ||
		err != nil ||
		sendCreateBlockList[0].AccountBlock.Quota != 29004 ||
		sendCreateBlockList[0].AccountBlock.Fee.Cmp(createContractFee) != 0 ||
		db.balanceMap[addr1][ledger.ViteTokenId].Cmp(balance1) != 0 {
		t.Fatalf("send create transaction error")
	}
	db.accountBlockMap[addr1][hash13] = sendCreateBlockList[0].AccountBlock

	// receive create
	addr2 := sendCreateBlockList[0].AccountBlock.ToAddress
	db.storageMap[types.AddressPledge][string(abi.GetPledgeBeneficialKey(addr2))], _ = abi.ABIPledge.PackVariable(abi.VariableNamePledgeBeneficial, new(big.Int).Mul(big.NewInt(1e9), big.NewInt(1e18)))
	balance2 := big.NewInt(0)

	hash21 := types.DataHash([]byte{2, 1})
	block21 := &ledger.AccountBlock{
		Height:         1,
		AccountAddress: addr2,
		FromBlockHash:  hash13,
		BlockType:      ledger.BlockTypeReceive,
		SnapshotHash:   snapshot.Hash,
		Timestamp:      &blockTime,
		Hash:           hash21,
	}
	vm = NewVM()
	//vm.Debug = true
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
		Hash:           hash14,
	}
	vm = NewVM()
	//vm.Debug = true
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
		Hash:           hash22,
	}
	vm = NewVM()
	//vm.Debug = true
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
	hash15 := types.DataHash([]byte{1, 5})
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
		Hash:           hash15,
	}
	vm = NewVM()
	//vm.Debug = true
	db.addr = addr1
	sendCallBlockList2, isRetry, err := vm.Run(db, block15, nil)
	if len(sendCallBlockList2) != 0 || err != util.ErrInsufficientBalance {
		t.Fatalf("send call transaction 2 error")
	}
	// receive call error, execution revert
	data15, _ = hex.DecodeString("")
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
		Hash:           hash15,
	}
	vm = NewVM()
	//vm.Debug = true
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
	//vm.Debug = true
	db.addr = addr2
	receiveCallBlockList2, isRetry, err := vm.Run(db, block23, sendCallBlockList2[0].AccountBlock)
	if len(receiveCallBlockList2) != 2 || isRetry || err != util.ErrExecutionReverted ||
		receiveCallBlockList2[0].AccountBlock.Quota != 21046 ||
		len(receiveCallBlockList2[0].AccountBlock.Data) != 33 ||
		receiveCallBlockList2[0].AccountBlock.Data[32] != 1 ||
		receiveCallBlockList2[1].AccountBlock.BlockType != ledger.BlockTypeSendRefund ||
		receiveCallBlockList2[1].AccountBlock.Height != 4 ||
		receiveCallBlockList2[1].AccountBlock.AccountAddress != addr2 ||
		receiveCallBlockList2[1].AccountBlock.ToAddress != addr1 ||
		receiveCallBlockList2[1].AccountBlock.Amount.Cmp(block15.Amount) != 0 ||
		receiveCallBlockList2[1].AccountBlock.TokenId != ledger.ViteTokenId ||
		receiveCallBlockList2[1].AccountBlock.Quota != 0 ||
		receiveCallBlockList2[1].AccountBlock.Fee.Sign() != 0 ||
		len(receiveCallBlockList2[1].AccountBlock.Data) != 0 {
		t.Fatalf("receive call transaction error")
	}
	db.accountBlockMap[addr2][hash23] = receiveCallBlockList2[0].AccountBlock
}

func TestDelegateCall(t *testing.T) {
	// prepare db, add account1, add account2 with code, add account3 with code
	db := NewNoDatabase()
	// code1 return 1+2
	addr1, _, _ := types.CreateAddress()
	code1 := []byte{1, byte(PUSH1), 1, byte(PUSH1), 2, byte(ADD), byte(PUSH1), 32, byte(DUP1), byte(SWAP2), byte(SWAP1), byte(MSTORE), byte(PUSH1), 32, byte(SWAP1), byte(RETURN)}
	db.codeMap = make(map[types.Address][]byte)
	db.codeMap[addr1] = code1

	addr2, _, _ := types.CreateAddress()
	code2 := helper.JoinBytes([]byte{1, byte(PUSH1), 32, byte(PUSH1), 0, byte(PUSH1), 0, byte(PUSH1), 0, byte(PUSH20)}, addr1.Bytes(), []byte{byte(DELEGATECALL), byte(PUSH1), 32, byte(PUSH1), 0, byte(RETURN)})
	db.codeMap[addr2] = code2
	blockTime := time.Now()

	vm := NewVM()
	vm.i = NewInterpreter(1)
	//vm.Debug = true
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
		&vm_context.VmAccountBlock{receiveCallBlock, db},
		&sendCallBlock,
		nil,
		1000000,
		0)
	c.setCallCode(addr2, code2[1:])
	ret, err := c.run(vm)
	if err != nil || !bytes.Equal(ret, helper.LeftPadBytes([]byte{3}, 32)) {
		t.Fatalf("delegate call error")
	}
}

func TestCall(t *testing.T) {
	// prepare db, add account1, add account2 with code, add account3 with code
	viteTotalSupply := new(big.Int).Mul(big.NewInt(1e9), util.AttovPerVite)
	db, addr1, _, hash12, snapshot, _ := prepareDb(viteTotalSupply)
	blockTime := time.Now()

	// code2 calls addr1 with data=100 and amount=10
	addr2, _, _ := types.CreateAddress()
	code2 := []byte{
		1,
		byte(PUSH1), 32, byte(PUSH1), 100, byte(PUSH1), 0, byte(DUP1), byte(SWAP2), byte(SWAP1), byte(MSTORE),
		byte(PUSH1), 10, byte(PUSH10), 'V', 'I', 'T', 'E', ' ', 'T', 'O', 'K', 'E', 'N', byte(PUSH20), 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, byte(CALL)}
	db.codeMap[addr2] = code2

	// code3 return amount+data
	addr3 := types.Address{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1}
	code3 := []byte{1, byte(CALLVALUE), byte(PUSH1), 0, byte(CALLDATALOAD), byte(ADD), byte(PUSH1), 32, byte(DUP1), byte(SWAP2), byte(SWAP1), byte(MSTORE), byte(PUSH1), 32, byte(SWAP1), byte(RETURN)}
	db.codeMap[addr3] = code3

	db.accountBlockMap[addr2] = make(map[types.Hash]*ledger.AccountBlock)
	db.storageMap[types.AddressPledge][string(abi.GetPledgeBeneficialKey(addr2))], _ = abi.ABIPledge.PackVariable(abi.VariableNamePledgeBeneficial, new(big.Int).Mul(big.NewInt(1e9), big.NewInt(1e18)))

	db.accountBlockMap[addr3] = make(map[types.Hash]*ledger.AccountBlock)
	db.storageMap[types.AddressPledge][string(abi.GetPledgeBeneficialKey(addr3))], _ = abi.ABIPledge.PackVariable(abi.VariableNamePledgeBeneficial, new(big.Int).Mul(big.NewInt(1e9), big.NewInt(1e18)))

	vm := NewVM()
	//vm.Debug = true
	// call contract
	balance1 := db.balanceMap[addr1][ledger.ViteTokenId]
	hash13 := types.DataHash([]byte{1, 3})
	block13 := &ledger.AccountBlock{
		Height:         3,
		AccountAddress: addr1,
		BlockType:      ledger.BlockTypeSendCall,
		PrevHash:       hash12,
		Amount:         big.NewInt(10),
		Fee:            big.NewInt(0),
		TokenId:        ledger.ViteTokenId,
		SnapshotHash:   snapshot.Hash,
		Timestamp:      &blockTime,
		Hash:           hash13,
	}
	db.addr = addr1
	sendCallBlockList, isRetry, err := vm.Run(db, block13, nil)
	balance1.Sub(balance1, block13.Amount)
	if len(sendCallBlockList) != 1 ||
		isRetry ||
		err != nil ||
		sendCallBlockList[0].AccountBlock.Quota != 21000 ||
		db.balanceMap[addr1][ledger.ViteTokenId].Cmp(balance1) != 0 {
		t.Fatalf("send call transaction error")
	}
	db.accountBlockMap[addr1][hash13] = sendCallBlockList[0].AccountBlock

	// contract2 receive call
	balance2 := big.NewInt(0)
	hash21 := types.DataHash([]byte{2, 1})
	block21 := &ledger.AccountBlock{
		Height:         1,
		AccountAddress: addr2,
		FromBlockHash:  hash13,
		BlockType:      ledger.BlockTypeReceive,
		SnapshotHash:   snapshot.Hash,
		Timestamp:      &blockTime,
		Hash:           hash21,
	}
	vm = NewVM()
	//vm.Debug = true
	db.addr = addr2
	receiveCallBlockList, isRetry, err := vm.Run(db, block21, sendCallBlockList[0].AccountBlock)
	if len(receiveCallBlockList) != 2 || isRetry || err != nil ||
		receiveCallBlockList[0].AccountBlock.Quota != 21733 ||
		len(receiveCallBlockList[0].AccountBlock.Data) != 33 ||
		receiveCallBlockList[0].AccountBlock.Data[32] != 0 ||
		receiveCallBlockList[1].AccountBlock.BlockType != ledger.BlockTypeSendCall ||
		receiveCallBlockList[1].AccountBlock.Height != 2 ||
		receiveCallBlockList[1].AccountBlock.AccountAddress != addr2 ||
		receiveCallBlockList[1].AccountBlock.ToAddress != addr3 ||
		receiveCallBlockList[1].AccountBlock.Amount.Cmp(big.NewInt(10)) != 0 ||
		receiveCallBlockList[1].AccountBlock.Quota != 21192 ||
		receiveCallBlockList[1].AccountBlock.Fee.Sign() != 0 ||
		!bytes.Equal(receiveCallBlockList[1].AccountBlock.Data, helper.LeftPadBytes([]byte{100}, 32)) ||
		db.balanceMap[addr2][ledger.ViteTokenId].Cmp(balance2) != 0 {
		t.Fatalf("contract receive call transaction error")
	}
	db.accountBlockMap[addr2][hash21] = receiveCallBlockList[0].AccountBlock
	hash22 := types.DataHash([]byte{2, 2})
	receiveCallBlockList[1].AccountBlock.PrevHash = hash21
	receiveCallBlockList[1].AccountBlock.Hash = hash22
	db.accountBlockMap[addr2][hash22] = receiveCallBlockList[1].AccountBlock

	// contract3 receive call
	balance3 := new(big.Int).Set(block13.Amount)
	hash31 := types.DataHash([]byte{3, 1})
	block31 := &ledger.AccountBlock{
		Height:         1,
		AccountAddress: addr3,
		FromBlockHash:  hash22,
		BlockType:      ledger.BlockTypeReceive,
		SnapshotHash:   snapshot.Hash,
		Timestamp:      &blockTime,
		Hash:           hash31,
	}
	vm = NewVM()
	//vm.Debug = true
	db.addr = addr3
	receiveCallBlockList2, isRetry, err := vm.Run(db, block31, receiveCallBlockList[1].AccountBlock)
	if len(receiveCallBlockList2) != 1 || isRetry || err != nil ||
		receiveCallBlockList2[0].AccountBlock.Quota != 21038 ||
		len(receiveCallBlockList2[0].AccountBlock.Data) != 33 ||
		receiveCallBlockList2[0].AccountBlock.Data[32] != 0 ||
		db.balanceMap[addr3][ledger.ViteTokenId].Cmp(balance3) != 0 {
		t.Fatalf("contract receive call transaction error")
	}
	db.accountBlockMap[addr3][hash31] = receiveCallBlockList2[0].AccountBlock
}

var DefaultDifficulty = new(big.Int).SetUint64(67108863)

func TestCalcQuotaV2(t *testing.T) {
	quota.InitQuotaConfig(false)
	// prepare db
	addr1, _, _ := types.CreateAddress()
	db := NewNoDatabase()
	timestamp := time.Unix(1536214502, 0)
	snapshot1 := &ledger.SnapshotBlock{Height: 1, Timestamp: &timestamp, Hash: types.DataHash([]byte{10, 1})}
	db.snapshotBlockList = append(db.snapshotBlockList, snapshot1)

	difficulty := DefaultDifficulty
	quotaForTx := uint64(21000)
	quotaLimit := uint64(987000)
	minPledgeAmount := new(big.Int).Mul(big.NewInt(10000), big.NewInt(1e18))
	maxPledgeAmount := new(big.Int).Mul(big.NewInt(1e9), big.NewInt(1e18))
	db.storageMap[types.AddressPledge] = make(map[string][]byte)
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
	quotaTotal, quotaAddition, err = quota.CalcQuotaV2(db, addr1, new(big.Int).Mul(big.NewInt(15000), big.NewInt(1e18)), difficulty)
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
	quotaTotal, quotaAddition, err = quota.CalcQuotaV2(db, addr1, new(big.Int).Mul(big.NewInt(15000), big.NewInt(1e18)), helper.Big0)
	if quotaTotal != quotaForTx*2 || quotaAddition != uint64(0) || err != nil {
		t.Fatalf("calc quota error")
	}
	// first account block without PoW, pledge amount reaches no quota limit, snapshot height gap=2
	quotaTotal, quotaAddition, err = quota.CalcQuotaV2(db, addr1, new(big.Int).Mul(big.NewInt(7000), big.NewInt(1e18)), difficulty)
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
		Hash:           hash12,
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
	quotaTotal, quotaAddition, err = quota.CalcQuotaV2(db, addr1, new(big.Int).Mul(big.NewInt(7000), big.NewInt(1e18)), difficulty)
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
		Hash:           hash13,
	}
	db.accountBlockMap[addr1][hash13] = block13
	// second account block referring to same snapshotBlock with PoW, first block receive error
	quotaTotal, quotaAddition, err = quota.CalcQuotaV2(db, addr1, maxPledgeAmount, difficulty)
	if quotaTotal != uint64(0) || quotaAddition != uint64(0) || err != util.ErrOutOfQuota {
		t.Fatalf("calc quota error")
	}
	// second account block referring to same snapshotBlock without PoW, first block receive error
	quotaTotal, quotaAddition, err = quota.CalcQuotaV2(db, addr1, maxPledgeAmount, helper.Big0)
	if quotaTotal != uint64(0) || quotaAddition != uint64(0) || err != util.ErrOutOfQuota {
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
		Hash:           hash14,
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
	quotaTotal, quotaAddition, err = quota.CalcQuotaV2(db, addr1, new(big.Int).Mul(big.NewInt(7000), big.NewInt(1e18)), difficulty)
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
			Hash:           hashi,
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
	InitVmConfig(true, true, false, "")
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
	//vm.Debug = true
	db.addr = addr1
	sendCallBlockList, isRetry, err := vm.Run(db, block11, nil)
	if len(sendCallBlockList) != 1 || isRetry || err != nil {
		t.Fatalf("init test vm config failed")
	}
}

func TestCheckDepth(t *testing.T) {
	db, addr1, _, prevHash1, snapshot, _ := prepareDb(big.NewInt(0))

	addr2, _ := types.HexToAddress("vite_0acbb1335822c8df4488f3eea6e9000eabb0f19d8802f57c87")
	addr3, _ := types.HexToAddress("vite_14edbc9214bd1e5f6082438f707d10bf43463a6d599a4f2d08")
	db.storageMap[types.AddressPledge][string(abi.GetPledgeBeneficialKey(addr2))], _ = abi.ABIPledge.PackVariable(abi.VariableNamePledgeBeneficial, new(big.Int).Mul(big.NewInt(1e9), big.NewInt(1e18)))
	db.storageMap[types.AddressPledge][string(abi.GetPledgeBeneficialKey(addr3))], _ = abi.ABIPledge.PackVariable(abi.VariableNamePledgeBeneficial, new(big.Int).Mul(big.NewInt(1e9), big.NewInt(1e18)))
	db.accountBlockMap[addr2] = make(map[types.Hash]*ledger.AccountBlock)
	db.accountBlockMap[addr3] = make(map[types.Hash]*ledger.AccountBlock)

	blockTime := time.Now()

	initSendHash := getHash(1, 3)
	initSendBlock := &ledger.AccountBlock{
		Height:         3,
		AccountAddress: addr1,
		BlockType:      ledger.BlockTypeSendCall,
		Amount:         big.NewInt(0),
		Fee:            big.NewInt(0),
		TokenId:        ledger.ViteTokenId,
		SnapshotHash:   snapshot.Hash,
		Timestamp:      &blockTime,
		Hash:           initSendHash,
		PrevHash:       prevHash1,
	}
	vm := NewVM()
	db.addr = addr1
	sendBlockList, isRetry, err := vm.Run(db, initSendBlock, nil)
	if len(sendBlockList) != 1 || isRetry || err != nil {
		t.Fatalf("init send call error")
	}
	db.accountBlockMap[addr1][initSendHash] = sendBlockList[0].AccountBlock

	initReceiveHash := getHash(2, 1)
	initReceiveBlock := &ledger.AccountBlock{
		Height:         1,
		AccountAddress: addr2,
		FromBlockHash:  initSendHash,
		BlockType:      ledger.BlockTypeReceive,
		SnapshotHash:   snapshot.Hash,
		Timestamp:      &blockTime,
		Hash:           initReceiveHash,
	}
	vm = NewVM()
	db.addr = addr2
	receiveBlockList, isRetry, err := vm.Run(db, initReceiveBlock, sendBlockList[0].AccountBlock)
	if len(receiveBlockList) != 1 || isRetry || err != nil {
		t.Fatalf("init receive call error")
	}
	db.accountBlockMap[addr2][initReceiveHash] = receiveBlockList[0].AccountBlock

	code2 := helper.JoinBytes([]byte{
		1,
		byte(PUSH1), 0,
		byte(PUSH1), 0,
		byte(PUSH1), 0,
		byte(PUSH10), 'V', 'I', 'T', 'E', ' ', 'T', 'O', 'K', 'E', 'N',
		byte(PUSH20)}, addr3.Bytes(),
		[]byte{byte(CALL)})
	db.codeMap = make(map[types.Address][]byte)
	db.codeMap[addr2] = code2
	code3 := helper.JoinBytes([]byte{
		1,
		byte(PUSH1), 0,
		byte(PUSH1), 0,
		byte(PUSH1), 0,
		byte(PUSH10), 'V', 'I', 'T', 'E', ' ', 'T', 'O', 'K', 'E', 'N',
		byte(PUSH20)}, addr2.Bytes(),
		[]byte{byte(CALL)})
	db.codeMap[addr3] = code3

	newHash := getHash(1, 4)
	prevHash := initSendHash
	sendBlock := &ledger.AccountBlock{
		Height:         4,
		AccountAddress: addr1,
		ToAddress:      addr2,
		BlockType:      ledger.BlockTypeSendCall,
		Amount:         big.NewInt(0),
		Fee:            big.NewInt(0),
		TokenId:        ledger.ViteTokenId,
		SnapshotHash:   snapshot.Hash,
		Timestamp:      &blockTime,
		Hash:           newHash,
		PrevHash:       prevHash,
	}
	vm = NewVM()
	db.addr = addr1
	sendBlockList, isRetry, err = vm.Run(db, sendBlock, nil)
	if len(sendBlockList) != 1 || isRetry || err != nil {
		t.Fatalf("send call error")
	}
	db.accountBlockMap[addr1][newHash] = sendBlockList[0].AccountBlock

	prevHash2 := initReceiveHash
	height2 := uint64(1)

	height3 := uint64(0)
	sendBlock = sendBlockList[0].AccountBlock
	prevHash3 := types.Hash{}
	depth := 1
	for {
		addr := sendBlock.ToAddress
		var height uint64
		if addr == addr2 {
			height2 = height2 + 1
			height = height2
			newHash = getHash(2, height)
			prevHash = prevHash2
		} else if addr == addr3 {
			height3 = height3 + 1
			height = height3
			newHash = getHash(3, height)
			prevHash = prevHash3
		}
		receiveBlock := &ledger.AccountBlock{
			Height:         height,
			AccountAddress: addr,
			FromBlockHash:  sendBlock.Hash,
			BlockType:      ledger.BlockTypeReceive,
			SnapshotHash:   snapshot.Hash,
			Timestamp:      &blockTime,
			PrevHash:       prevHash,
			Hash:           newHash,
		}
		db.addr = addr
		receiveBlockList, isRetry, err := vm.Run(db, receiveBlock, sendBlock)
		if len(receiveBlockList) == 1 {
			if isRetry || err != util.ErrDepth {
				t.Fatalf("receive block failed, list size = 1")
			}
			fmt.Printf("depth %v reached, height2: %v, height3: %v\n", depth, height2, height3)
			return
		} else if len(receiveBlockList) == 2 {
			if isRetry || err != nil {
				t.Fatalf("receive block failed, list size = 2")
			}
		} else {
			t.Fatalf("receive block failed, list size unknown")
		}

		db.accountBlockMap[addr][newHash] = receiveBlockList[0].AccountBlock
		receiveBlockList[1].AccountBlock.PrevHash = newHash
		if addr == addr2 {
			height2 = height2 + 1
			newHash = getHash(2, height2)
			prevHash2 = newHash
		} else if addr == addr3 {
			height3 = height3 + 1
			newHash = getHash(3, height3)
			prevHash3 = newHash
		}
		receiveBlockList[1].AccountBlock.Hash = newHash
		db.accountBlockMap[addr][newHash] = receiveBlockList[1].AccountBlock

		snapshot = &ledger.SnapshotBlock{Height: snapshot.Height + 1, Timestamp: &blockTime, Hash: getHash(10, snapshot.Height+1)}
		db.snapshotBlockList = append(db.snapshotBlockList, snapshot)
		depth = depth + 1

		sendBlock = receiveBlockList[1].AccountBlock
	}
}

func getHash(a, b uint64) types.Hash {
	h, _ := types.BigToHash(big.NewInt(int64(a*10000 + b)))
	return h

}

type TestCaseMap map[string]TestCase

type TestCaseSendBlock struct {
	ToAddress types.Address
	Amount    string
	TokenId   types.TokenTypeId
	Data      string
}

type TestCase struct {
	SBHeight      uint64
	SBTime        int64
	FromAddress   types.Address
	ToAddress     types.Address
	InputData     string
	Amount        string
	TokenId       types.TokenTypeId
	Code          string
	ReturnData    string
	QuotaTotal    uint64
	QuotaLeft     uint64
	QuotaRefund   uint64
	Err           string
	Storage       map[string]string
	PreStorage    map[string]string
	LogHash       string
	SendBlockList []*TestCaseSendBlock
}

func TestVm(t *testing.T) {
	testDir := "./test/"
	testFiles, ok := ioutil.ReadDir(testDir)
	if ok != nil {
		t.Fatalf("read dir failed, %v", ok)
	}
	for _, testFile := range testFiles {
		file, ok := os.Open(testDir + testFile.Name())
		if ok != nil {
			t.Fatalf("open test file failed, %v", ok)
		}
		testCaseMap := new(TestCaseMap)
		if ok := json.NewDecoder(file).Decode(testCaseMap); ok != nil {
			t.Fatalf("decode test file failed, %v", ok)
		}

		for k, testCase := range *testCaseMap {
			var sbTime time.Time
			if testCase.SBTime > 0 {
				sbTime = time.Unix(testCase.SBTime, 0)
			} else {
				sbTime = time.Now()
			}
			sb := ledger.SnapshotBlock{
				Height:    testCase.SBHeight,
				Timestamp: &sbTime,
				Hash:      types.DataHash([]byte{1, 1}),
			}
			vm := NewVM()
			vm.i = NewInterpreter(1)
			//fmt.Printf("testcase %v: %v\n", testFile.Name(), k)
			inputData, _ := hex.DecodeString(testCase.InputData)
			amount, _ := hex.DecodeString(testCase.Amount)
			sendCallBlock := ledger.AccountBlock{
				AccountAddress: testCase.FromAddress,
				ToAddress:      testCase.ToAddress,
				BlockType:      ledger.BlockTypeSendCall,
				Data:           inputData,
				Amount:         new(big.Int).SetBytes(amount),
				Fee:            big.NewInt(0),
				TokenId:        testCase.TokenId,
				SnapshotHash:   sb.Hash,
				Timestamp:      &sbTime,
			}
			receiveCallBlock := &ledger.AccountBlock{
				AccountAddress: testCase.ToAddress,
				BlockType:      ledger.BlockTypeReceive,
				SnapshotHash:   sb.Hash,
				Timestamp:      &sbTime,
			}
			db := NewMemoryDatabase(testCase.ToAddress, &sb)
			if len(testCase.PreStorage) > 0 {
				for k, v := range testCase.PreStorage {
					vByte, _ := hex.DecodeString(v)
					db.storage[k] = vByte
					db.originalStorage[k] = vByte
				}
			}
			c := newContract(
				&vm_context.VmAccountBlock{receiveCallBlock, db},
				&sendCallBlock,
				sendCallBlock.Data,
				testCase.QuotaTotal,
				0)
			code, _ := hex.DecodeString(testCase.Code)
			c.setCallCode(testCase.ToAddress, code)
			db.AddBalance(&sendCallBlock.TokenId, sendCallBlock.Amount)
			ret, err := c.run(vm)
			returnData, _ := hex.DecodeString(testCase.ReturnData)
			if (err == nil && testCase.Err != "") || (err != nil && testCase.Err != err.Error()) {
				t.Fatalf("%v: %v failed, err not match, expected %v, got %v", testFile.Name(), k, testCase.Err, err)
			}
			if err == nil || err.Error() == "execution reverted" {
				if bytes.Compare(returnData, ret) != 0 {
					t.Fatalf("%v: %v failed, return Data error, expected %v, got %v", testFile.Name(), k, returnData, ret)
				} else if c.quotaLeft != testCase.QuotaLeft {
					t.Fatalf("%v: %v failed, quota left error, expected %v, got %v", testFile.Name(), k, testCase.QuotaLeft, c.quotaLeft)
				} else if c.quotaRefund != testCase.QuotaRefund {
					t.Fatalf("%v: %v failed, quota refund error, expected %v, got %v", testFile.Name(), k, testCase.QuotaRefund, c.quotaRefund)
				} else if checkStorageResult := checkStorage(db, testCase.Storage); checkStorageResult != "" {
					t.Fatalf("%v: %v failed, storage error, %v", testFile.Name(), k, checkStorageResult)
				} else if logHash := db.GetLogListHash(); (logHash == nil && len(testCase.LogHash) != 0) || (logHash != nil && logHash.String() != testCase.LogHash) {
					t.Fatalf("%v: %v failed, log hash error, expected\n%v,\ngot\n%v", testFile.Name(), k, testCase.LogHash, logHash)
				} else if checkSendBlockListResult := checkSendBlockList(testCase.SendBlockList, vm.blockList); checkSendBlockListResult != "" {
					t.Fatalf("%v: %v failed, send block list error, %v", testFile.Name(), k, checkSendBlockListResult)
				}
			}
		}
	}
}
func checkStorage(got *memoryDatabase, expected map[string]string) string {
	if len(expected) != len(got.storage) {
		return "expected len " + strconv.Itoa(len(expected)) + ", got len" + strconv.Itoa(len(got.storage))
	}
	for k, v := range got.storage {
		if sv, ok := expected[k]; !ok || sv != hex.EncodeToString(v) {
			return "expect " + k + ": " + sv + ", got " + k + ": " + hex.EncodeToString(v)
		}
	}
	return ""
}

func checkSendBlockList(expected []*TestCaseSendBlock, got []*vm_context.VmAccountBlock) string {
	if len(got) != len(expected) {
		return "expected len " + strconv.Itoa(len(expected)) + ", got len" + strconv.Itoa(len(got))
	}
	for i, expectedSendBlock := range expected {
		gotSendBlock := got[i].AccountBlock
		if gotSendBlock.ToAddress != expectedSendBlock.ToAddress {
			return "expected toAddress " + expectedSendBlock.ToAddress.String() + ", got toAddress " + gotSendBlock.ToAddress.String()
		} else if gotAmount := hex.EncodeToString(gotSendBlock.Amount.Bytes()); gotAmount != expectedSendBlock.Amount {
			return "expected amount " + expectedSendBlock.Amount + ", got amount " + gotAmount
		} else if gotSendBlock.TokenId != expectedSendBlock.TokenId {
			return "expected tokenId " + expectedSendBlock.TokenId.String() + ", got tokenId " + gotSendBlock.TokenId.String()
		} else if gotData := hex.EncodeToString(gotSendBlock.Data); gotData != expectedSendBlock.Data {
			return "expected data " + expectedSendBlock.Data + ", got data " + gotData
		}
	}
	return ""
}
