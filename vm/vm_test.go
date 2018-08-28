package vm

import (
	"encoding/hex"
	"errors"
	"github.com/vitelabs/go-vite/common/types"
	"math/big"
	"testing"
	"time"
)

func TestGetTokenInfo(t *testing.T) {
	tests := []struct {
		tokenTypeId types.TokenTypeId
		owner       types.Address
		totalSupply *big.Int
		data        string
		result      error
	}{
		{types.CreateTokenTypeId([]byte{1}), types.CreateContractAddress([]byte{1}), big.NewInt(10), "00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000012000000000000000000000000000000000000000000000000000000000000000956697465546f6b656e0000000000000000000000000000000000000000000000", nil},
		{types.CreateTokenTypeId([]byte{2}), types.CreateContractAddress([]byte{2}), big.NewInt(10), "00000000000000000000000000000000000000000000000000000000000000600000000000000000000000000000000000000000000000000000000000000012000000000000000000000000000000000000000000000000000000000000000956697465546f6b656e0000000000000000000000000000000000000000000000", ErrInvalidTokenData},
		{types.CreateTokenTypeId([]byte{3}), types.CreateContractAddress([]byte{3}), big.NewInt(10), "00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000956697465546f6b656e0000000000000000000000000000000000000000000000", ErrInvalidTokenData},
		{types.CreateTokenTypeId([]byte{4}), types.CreateContractAddress([]byte{4}), big.NewInt(10), "00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000013000000000000000000000000000000000000000000000000000000000000000956697465546f6b656e0000000000000000000000000000000000000000000000", ErrInvalidTokenData},
		{types.CreateTokenTypeId([]byte{5}), types.CreateContractAddress([]byte{5}), big.NewInt(10), "00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000012000000000000000000000000000000000000000000000000000000000000001556697465546f6b656e0000000000000000000000000000000000000000000000", ErrInvalidTokenData},
		{types.CreateTokenTypeId([]byte{6}), types.CreateContractAddress([]byte{6}), big.NewInt(10), "00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000012000000000000000000000000000000000000000000000000000000000000001456697465546f6b656e0000000000000000000000000000000000000000000000", ErrInvalidTokenData},
		{types.CreateTokenTypeId([]byte{7}), types.CreateContractAddress([]byte{7}), big.NewInt(10), "0000000000000000000000000000000000000000000000000000000000000040000000000000000000000000000000000000000000000000000000000000001200000000000000000000000000000000000000000000000000000000000000093c697465546f6b656e0000000000000000000000000000000000000000000000", ErrInvalidTokenData},
		{types.CreateTokenTypeId([]byte{1}), types.CreateContractAddress([]byte{8}), big.NewInt(10), "00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000012000000000000000000000000000000000000000000000000000000000000000956697465546f6b656e0000000000000000000000000000000000000000000000", ErrTokenIdCollision},
	}
	vm := &VM{Db: NewNoDatabase(), createBlock: CreateNoAccountBlock, instructionSet: simpleInstructionSet}
	for i, test := range tests {
		inputdata, _ := hex.DecodeString(test.data)
		err := vm.checkAndCreateToken(test.tokenTypeId, test.owner, test.totalSupply, inputdata)
		if err != test.result {
			t.Fatalf("%v th check token data fail %v %v", i, test, err)
		}
	}
}

func TestVmRun(t *testing.T) {
	// prepare db
	addr1, _, _ := types.CreateAddress()
	db := NewNoDatabase()
	db.tokenMap[viteTokenTypeId] = VmToken{tokenId: viteTokenTypeId, tokenName: "ViteToken", owner: addr1, totalSupply: big.NewInt(1000), decimals: 18}

	timestamp := time.Now().Unix()
	snapshot1 := &NoSnapshotBlock{height: big.NewInt(1), timestamp: timestamp - 1, hash: types.DataHash([]byte{10, 1})}
	db.snapshotBlockMap[snapshot1.hash] = snapshot1
	snapshot2 := &NoSnapshotBlock{height: big.NewInt(1), timestamp: timestamp, hash: types.DataHash([]byte{10, 2})}
	db.snapshotBlockMap[snapshot2.hash] = snapshot2
	db.currentSnapshotBlockHash = snapshot2.hash

	hash11 := types.DataHash([]byte{1, 1})
	block11 := &NoAccountBlock{
		height:         big.NewInt(1),
		toAddress:      addr1,
		accountAddress: addr1,
		blockType:      BlockTypeSendCall,
		amount:         big.NewInt(1000),
		tokenId:        viteTokenTypeId,
		snapshotHash:   snapshot1.Hash(),
		depth:          1,
	}
	db.accountBlockMap[addr1] = make(map[types.Hash]VmAccountBlock)
	db.accountBlockMap[addr1][hash11] = block11
	hash12 := types.DataHash([]byte{1, 2})
	block12 := &NoAccountBlock{
		height:         big.NewInt(2),
		toAddress:      addr1,
		accountAddress: addr1,
		fromBlockHash:  hash11,
		blockType:      BlockTypeReceive,
		prevHash:       hash11,
		amount:         big.NewInt(1000),
		tokenId:        viteTokenTypeId,
		snapshotHash:   snapshot1.Hash(),
		depth:          1,
	}
	db.accountBlockMap[addr1][hash12] = block12

	db.balanceMap[addr1] = make(map[types.TokenTypeId]*big.Int)
	db.balanceMap[addr1][viteTokenTypeId] = db.tokenMap[viteTokenTypeId].totalSupply

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
	// send create
	data13, _ := hex.DecodeString("608060405260858060116000396000f300608060405260043610603e5763ffffffff7c0100000000000000000000000000000000000000000000000000000000600035041663f021ab8f81146043575b600080fd5b604c600435604e565b005b6000805490910190555600a165627a7a72305820b8d8d60a46c6ac6569047b17b012aa1ea458271f9bc8078ef0cff9208999d0900029")
	hash13 := types.DataHash([]byte{1, 3})
	block13 := &NoAccountBlock{
		height:         big.NewInt(3),
		accountAddress: addr1,
		blockType:      BlockTypeSendCreate,
		prevHash:       hash12,
		amount:         big.NewInt(10),
		tokenId:        viteTokenTypeId,
		snapshotHash:   snapshot2.Hash(),
		createFee:      big.NewInt(10),
		depth:          1,
		data:           data13,
	}
	vm := NewVM(db, CreateNoAccountBlock)
	vm.Debug = true
	sendCreateBlockList, isRetry, err := vm.Run(block13)
	if len(sendCreateBlockList) != 1 ||
		isRetry ||
		err != nil ||
		sendCreateBlockList[0].Quota() != 28832 ||
		db.balanceMap[addr1][viteTokenTypeId].Cmp(big.NewInt(980)) != 0 {
		t.Fatalf("send create transaction error")
	}
	db.accountBlockMap[addr1][hash13] = sendCreateBlockList[0]

	// receive create
	addr2 := sendCreateBlockList[0].ToAddress()
	hash21 := types.DataHash([]byte{2, 1})
	block21 := &NoAccountBlock{
		height:         big.NewInt(1),
		accountAddress: addr1,
		toAddress:      addr2,
		fromBlockHash:  hash13,
		blockType:      BlockTypeReceive,
		snapshotHash:   snapshot2.Hash(),
		depth:          1,
	}
	vm = NewVM(db, CreateNoAccountBlock)
	vm.Debug = true
	receiveCreateBlockList, isRetry, err := vm.Run(block21)
	if len(receiveCreateBlockList) != 1 || isRetry || err != nil ||
		receiveCreateBlockList[0].Quota() != 0 ||
		db.balanceMap[addr2][viteTokenTypeId].Cmp(big.NewInt(10)) != 0 {
		t.Fatalf("receive create transaction error")
	}
	db.accountBlockMap[addr2] = make(map[types.Hash]VmAccountBlock)
	db.accountBlockMap[addr2][hash21] = receiveCreateBlockList[0]

	// send call
	data14, _ := hex.DecodeString("f021ab8f0000000000000000000000000000000000000000000000000000000000000005")
	hash14 := types.DataHash([]byte{1, 4})
	block14 := &NoAccountBlock{
		height:         big.NewInt(4),
		accountAddress: addr1,
		toAddress:      addr2,
		blockType:      BlockTypeSendCall,
		prevHash:       hash13,
		amount:         big.NewInt(20),
		tokenId:        viteTokenTypeId,
		snapshotHash:   snapshot2.Hash(),
		depth:          1,
		data:           data14,
	}
	vm = NewVM(db, CreateNoAccountBlock)
	vm.Debug = true
	sendCallBlockList, isRetry, err := vm.Run(block14)
	if len(sendCallBlockList) != 1 || isRetry || err != nil ||
		sendCallBlockList[0].Quota() != 21464 ||
		db.balanceMap[addr1][viteTokenTypeId].Cmp(big.NewInt(960)) != 0 {
		t.Fatalf("send call transaction error")
	}
	db.accountBlockMap[addr1][hash14] = sendCallBlockList[0]

	// receive call
	hash22 := types.DataHash([]byte{2, 2})
	block22 := &NoAccountBlock{
		height:         big.NewInt(2),
		accountAddress: addr1,
		toAddress:      addr2,
		fromBlockHash:  hash14,
		prevHash:       hash21,
		blockType:      BlockTypeReceive,
		snapshotHash:   snapshot2.Hash(),
		depth:          1,
	}
	vm = NewVM(db, CreateNoAccountBlock)
	vm.Debug = true
	receiveCallBlockList, isRetry, err := vm.Run(block22)
	if len(receiveCallBlockList) != 1 || isRetry || err != nil ||
		receiveCallBlockList[0].Quota() != 41330 ||
		db.balanceMap[addr2][viteTokenTypeId].Cmp(big.NewInt(30)) != 0 {
		t.Fatalf("receive call transaction error")
	}
	db.accountBlockMap[addr2][hash22] = receiveCallBlockList[0]

	// send mintage
	data15, _ := hex.DecodeString("0000000000000000000000000000000000000000000000000000000000000040000000000000000000000000000000000000000000000000000000000000001200000000000000000000000000000000000000000000000000000000000000074d79546f6b656e00000000000000000000000000000000000000000000000000")
	hash15 := types.DataHash([]byte{1, 5})
	block15 := &NoAccountBlock{
		height:         big.NewInt(5),
		accountAddress: addr1,
		toAddress:      addr2,
		blockType:      BlockTypeSendMintage,
		prevHash:       hash14,
		amount:         big.NewInt(1000),
		snapshotHash:   snapshot2.Hash(),
		depth:          1,
		data:           data15,
	}
	vm = NewVM(db, CreateNoAccountBlock)
	vm.Debug = true
	sendMintageBlockList, isRetry, err := vm.Run(block15)
	if len(sendMintageBlockList) != 1 || isRetry || err != nil ||
		sendMintageBlockList[0].Quota() != 22152 ||
		sendMintageBlockList[0].CreateFee().Cmp(big.NewInt(0)) != 0 ||
		db.balanceMap[addr1][viteTokenTypeId].Cmp(big.NewInt(960)) != 0 {
		t.Fatalf("send mintage transaction error")
	}
	db.accountBlockMap[addr1][hash15] = sendMintageBlockList[0]
	myTokenId := sendMintageBlockList[0].TokenId()

	// receive mintage
	hash23 := types.DataHash([]byte{2, 3})
	block23 := &NoAccountBlock{
		height:         big.NewInt(3),
		accountAddress: addr1,
		toAddress:      addr2,
		prevHash:       hash22,
		fromBlockHash:  hash15,
		blockType:      BlockTypeReceive,
		snapshotHash:   snapshot2.Hash(),
		depth:          1,
	}
	vm = NewVM(db, CreateNoAccountBlock)
	vm.Debug = true
	receiveMintageBlockList, isRetry, err := vm.Run(block23)
	if len(receiveMintageBlockList) != 1 || isRetry || err != nil ||
		receiveMintageBlockList[0].Quota() != 21000 ||
		db.balanceMap[addr2][viteTokenTypeId].Cmp(big.NewInt(30)) != 0 ||
		db.balanceMap[addr2][myTokenId].Cmp(big.NewInt(1000)) != 0 {
		t.Fatalf("receive mintage transaction error")
	}
	db.accountBlockMap[addr2][hash23] = receiveMintageBlockList[0]

	// TODO error case
	// send call error, insufficient balance
	data16, _ := hex.DecodeString("f021ab8f0000000000000000000000000000000000000000000000000000000000000005")
	block16 := &NoAccountBlock{
		height:         big.NewInt(6),
		accountAddress: addr1,
		toAddress:      addr2,
		blockType:      BlockTypeSendCall,
		prevHash:       hash15,
		amount:         big.NewInt(970),
		tokenId:        viteTokenTypeId,
		snapshotHash:   snapshot2.Hash(),
		depth:          1,
		data:           data16,
	}
	vm = NewVM(db, CreateNoAccountBlock)
	vm.Debug = true
	sendCallBlockList2, isRetry, err := vm.Run(block16)
	if len(sendCallBlockList2) != 0 || err != ErrInsufficientBalance {
		t.Fatalf("send call transaction 2 error")
	}
	// receive call error, execution revert
	data16, _ = hex.DecodeString("")
	hash16 := types.DataHash([]byte{1, 6})
	block16 = &NoAccountBlock{
		height:         big.NewInt(6),
		accountAddress: addr1,
		toAddress:      addr2,
		blockType:      BlockTypeSendCall,
		prevHash:       hash15,
		amount:         big.NewInt(50),
		tokenId:        viteTokenTypeId,
		snapshotHash:   snapshot2.Hash(),
		depth:          1,
		data:           data16,
	}
	vm = NewVM(db, CreateNoAccountBlock)
	vm.Debug = true
	sendCallBlockList2, isRetry, err = vm.Run(block16)
	db.accountBlockMap[addr1][hash16] = sendCallBlockList2[0]
	// receive call
	hash24 := types.DataHash([]byte{2, 4})
	block24 := &NoAccountBlock{
		height:         big.NewInt(4),
		accountAddress: addr1,
		toAddress:      addr2,
		fromBlockHash:  hash16,
		prevHash:       hash23,
		blockType:      BlockTypeReceive,
		snapshotHash:   snapshot2.Hash(),
		depth:          1,
	}
	vm = NewVM(db, CreateNoAccountBlock)
	vm.Debug = true
	receiveCallBlockList2, isRetry, err := vm.Run(block24)
	if len(receiveCallBlockList2) != 1 || isRetry || err != ErrExecutionReverted ||
		receiveCallBlockList2[0].Quota() != 21046 {
		t.Fatalf("receive call transaction error")
	}
	db.accountBlockMap[addr2][hash24] = receiveCallBlockList2[0]
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

func TestVM_CreateSend(t *testing.T) {
	inputdata, _ := hex.DecodeString("608060405260008055348015601357600080fd5b5060358060216000396000f3006080604052600080fd00a165627a7a723058207c31c74808fe0f95820eb3c48eac8e3e10ef27058dc6ca159b547fccde9290790029")
	sendCreateBlock := CreateNoAccountBlock(types.Address{}, types.Address{}, BlockTypeSendCreate, 1)
	sendCreateBlock.SetTokenId(viteTokenTypeId)
	sendCreateBlock.SetAmount(big.NewInt(0))
	sendCreateBlock.SetSnapshotHash(types.Hash{})
	sendCreateBlock.SetPrevHash(types.Hash{})
	sendCreateBlock.SetHeight(big.NewInt(1))
	sendCreateBlock.SetData(inputdata)
	sendCreateBlock.SetCreateFee(big.NewInt(0))
	// vm.Debug = true
	vm := NewVM(NewNoDatabase(), CreateNoAccountBlock)
	blockList, _, err := vm.Run(sendCreateBlock)
	if len(blockList) != 1 ||
		//blockList[0].Quota() != 58336 ||
		blockList[0].ToAddress() == emptyAddress ||
		//blockList[0].Balance() == nil ||
		blockList[0].Amount().Cmp(big.NewInt(0)) != 0 ||
		//blockList[0].StateHash() == emptyHash ||
		blockList[0].TokenId() != viteTokenTypeId {
		t.Fatalf("send create fail [%v] %v", blockList, err)
	}
}
