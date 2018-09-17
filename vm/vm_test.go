package vm

import (
	"bytes"
	"encoding/hex"
	"errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/contracts"
	"github.com/vitelabs/go-vite/ledger"
	"math/big"
	"testing"
)

func TestVmRun(t *testing.T) {
	// prepare db
	viteTotalSupply := big.NewInt(6e18)
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
	data13, _ := hex.DecodeString("00000000000000000001608060405260858060116000396000f300608060405260043610603e5763ffffffff7c0100000000000000000000000000000000000000000000000000000000600035041663f021ab8f81146043575b600080fd5b604c600435604e565b005b6000805490910190555600a165627a7a72305820b8d8d60a46c6ac6569047b17b012aa1ea458271f9bc8078ef0cff9208999d0900029")
	hash13 := types.DataHash([]byte{1, 3})
	block13 := &ledger.AccountBlock{
		Height:         3,
		AccountAddress: addr1,
		BlockType:      ledger.BlockTypeSendCreate,
		PrevHash:       hash12,
		Amount:         big.NewInt(1e18),
		TokenId:        *ledger.ViteTokenId(),
		SnapshotHash:   snapshot2.Hash,
		Data:           data13,
	}
	vm := NewVM()
	vm.Debug = true
	db.addr = addr1
	sendCreateBlockList, isRetry, err := vm.Run(db, block13, nil)
	balance1.Sub(balance1, block13.Amount)
	balance1.Sub(balance1, contractFee)
	if len(sendCreateBlockList) != 1 ||
		isRetry ||
		err != nil ||
		sendCreateBlockList[0].AccountBlock.Quota != 28936 ||
		sendCreateBlockList[0].AccountBlock.Fee.Cmp(contractFee) != 0 ||
		db.balanceMap[addr1][*ledger.ViteTokenId()].Cmp(balance1) != 0 {
		t.Fatalf("send create transaction error")
	}
	db.accountBlockMap[addr1][hash13] = sendCreateBlockList[0].AccountBlock

	// receive create
	addr2 := sendCreateBlockList[0].AccountBlock.ToAddress
	db.storageMap[contracts.AddressPledge][types.DataHash(addr2.Bytes())], _ = contracts.ABI_pledge.PackVariable(contracts.VariableNamePledgeBeneficial, big.NewInt(1e18))
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
	updateReveiceBlockBySendBlock(block21, block13)
	receiveCreateBlockList, isRetry, err := vm.Run(db, block21, block13)
	balance2.Add(balance2, block13.Amount)
	if len(receiveCreateBlockList) != 1 || isRetry || err != nil ||
		receiveCreateBlockList[0].AccountBlock.Quota != 0 ||
		!bytes.Equal(db.contractGidMap[addr1].Bytes(), ledger.CommonGid().Bytes()) ||
		db.balanceMap[addr2][*ledger.ViteTokenId()].Cmp(balance2) != 0 {
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
		PrevHash:       hash13,
		Amount:         big.NewInt(1e18),
		TokenId:        *ledger.ViteTokenId(),
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
		db.balanceMap[addr1][*ledger.ViteTokenId()].Cmp(balance1) != 0 {
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
	updateReveiceBlockBySendBlock(block22, block14)
	receiveCallBlockList, isRetry, err := vm.Run(db, block22, block14)
	balance2.Add(balance2, block14.Amount)
	if len(receiveCallBlockList) != 1 || isRetry || err != nil ||
		receiveCallBlockList[0].AccountBlock.Quota != 41330 ||
		db.balanceMap[addr2][*ledger.ViteTokenId()].Cmp(big.NewInt(2e18)) != 0 {
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
		PrevHash:       hash14,
		Amount:         big.NewInt(4e18),
		TokenId:        *ledger.ViteTokenId(),
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
		PrevHash:       hash14,
		Amount:         big.NewInt(50),
		TokenId:        *ledger.ViteTokenId(),
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
	updateReveiceBlockBySendBlock(block23, block15)
	receiveCallBlockList2, isRetry, err := vm.Run(db, block23, block15)
	if len(receiveCallBlockList2) != 1 || isRetry || err != ErrExecutionReverted ||
		receiveCallBlockList2[0].AccountBlock.Quota != 21046 {
		t.Fatalf("receive call transaction error")
	}
	db.accountBlockMap[addr2][hash23] = receiveCallBlockList2[0].AccountBlock
}

func TestDelegateCall(t *testing.T) {
	// TODO
	// prepare db, add account1, add account2 with code, add account3 with code
	// send call
	// receive call
}

func TestQuotaLeft(t *testing.T) {
	// TODO
	// prepare db
	// calc quota
}

func TestQuotaUsed(t *testing.T) {
	tests := []struct {
		quotaTotal, quotaAddition, quotaLeft, quotaRefund, quotaUsed uint64
		err                                                          error
	}{
		{15000, 5000, 10001, 0, 0, nil},
		{15000, 5000, 9999, 0, 1, nil},
		{10000, 0, 9999, 0, 1, nil},
		{10000, 0, 5000, 1000, 4000, nil},
		{10000, 0, 5000, 5000, 2500, nil},
		{15000, 5000, 5000, 5000, 2500, nil},
		{15000, 5000, 10001, 0, 10000, ErrOutOfQuota},
		{15000, 5000, 9999, 0, 10000, ErrOutOfQuota},
		{10000, 0, 9999, 0, 10000, ErrOutOfQuota},
		{10000, 0, 5000, 1000, 10000, ErrOutOfQuota},
		{10000, 0, 5000, 5000, 10000, ErrOutOfQuota},
		{15000, 5000, 5000, 5000, 10000, ErrOutOfQuota},
		{15000, 5000, 10001, 0, 0, errors.New("")},
		{15000, 5000, 9999, 0, 1, errors.New("")},
		{10000, 0, 9999, 0, 1, errors.New("")},
		{10000, 0, 5000, 1000, 5000, errors.New("")},
		{15000, 5000, 5000, 5000, 5000, errors.New("")},
	}
	for i, test := range tests {
		quotaUsed := quotaUsed(test.quotaTotal, test.quotaAddition, test.quotaLeft, test.quotaRefund, test.err)
		if quotaUsed != test.quotaUsed {
			t.Fatalf("%v th calculate quota used failed, expected %v, got %v", i, test.quotaUsed, quotaUsed)
		}
	}
}

func updateReveiceBlockBySendBlock(receiveBlock, sendBlock *ledger.AccountBlock) {
	receiveBlock.Data = sendBlock.Data
	receiveBlock.Fee = sendBlock.Fee
	receiveBlock.Amount = sendBlock.Amount
	receiveBlock.TokenId = sendBlock.TokenId
}
