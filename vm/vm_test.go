package vm

import (
	"encoding/hex"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/contracts"
	"math/big"
	"strconv"
	"testing"
	"time"
)

func TestVmRun(t *testing.T) {
	// prepare db
	viteTotalSupply := viteTotalSupply
	db, addr1, hash12, snapshot2, _ := prepareDb(viteTotalSupply)

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
		SnapshotHash:   snapshot2.Hash,
		Data:           data13,
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
	db.storageMap[contracts.AddressPledge][types.DataHash(addr2.Bytes())], _ = contracts.ABIPledge.PackVariable(contracts.VariableNamePledgeBeneficial, big.NewInt(1e18))
	balance2 := big.NewInt(0)

	hash21 := types.DataHash([]byte{2, 1})
	block21 := &ledger.AccountBlock{
		Height:         1,
		AccountAddress: addr2,
		FromBlockHash:  hash13,
		BlockType:      ledger.BlockTypeReceive,
		SnapshotHash:   snapshot2.Hash,
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
		SnapshotHash:   snapshot2.Hash,
		Data:           data14,
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

	// receive call
	hash22 := types.DataHash([]byte{2, 2})
	block22 := &ledger.AccountBlock{
		Height:         2,
		AccountAddress: addr2,
		FromBlockHash:  hash14,
		PrevHash:       hash21,
		BlockType:      ledger.BlockTypeReceive,
		SnapshotHash:   snapshot2.Hash,
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

	// TODO error case
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
		SnapshotHash:   snapshot2.Hash,
		Data:           data15,
	}
	vm = NewVM()
	vm.Debug = true
	db.addr = addr1
	sendCallBlockList2, isRetry, err := vm.Run(db, block15, nil)
	if len(sendCallBlockList2) != 0 || err != ErrInsufficientBalance {
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
		SnapshotHash:   snapshot2.Hash,
		Data:           data15,
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
		SnapshotHash:   snapshot2.Hash,
	}
	vm = NewVM()
	vm.Debug = true
	db.addr = addr2
	receiveCallBlockList2, isRetry, err := vm.Run(db, block23, sendCallBlockList2[0].AccountBlock)
	if len(receiveCallBlockList2) != 1 || isRetry || err != ErrExecutionReverted ||
		receiveCallBlockList2[0].AccountBlock.Quota != 21046 {
		t.Fatalf("receive call transaction error")
	}
	db.accountBlockMap[addr2][hash23] = receiveCallBlockList2[0].AccountBlock
}

func TestDelegateCall(t *testing.T) {
	// TODO test delegate call
	// prepare db, add account1, add account2 with code, add account3 with code
	// send call
	// receive call
}

func BenchmarkVMTransfer(b *testing.B) {
	viteTotalSupply := viteTotalSupply
	db, addr1, hash12, _, timestamp := prepareDb(viteTotalSupply)
	for i := 3; i < b.N+3; i++ {
		timestamp = timestamp + 1
		ti := time.Unix(timestamp, 0)
		snapshoti := &ledger.SnapshotBlock{Height: uint64(i), Timestamp: &ti, Hash: types.DataHash([]byte(strconv.Itoa(i)))}
		db.snapshotBlockList = append(db.snapshotBlockList, snapshoti)
	}

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
