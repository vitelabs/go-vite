package vm

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/contracts"
	"github.com/vitelabs/go-vite/vm/contracts/abi"
	"github.com/vitelabs/go-vite/vm/util"
	"math/big"
	"regexp"
	"strconv"
	"testing"
	"time"
)

func TestContractsRefund(t *testing.T) {
	// prepare db
	viteTotalSupply := new(big.Int).Mul(big.NewInt(1e9), big.NewInt(1e18))
	db, addr1, _, hash12, snapshot2, _ := prepareDb(viteTotalSupply)

	addr2 := types.AddressConsensusGroup
	nodeName := "s1"
	locHashRegister, _ := types.BytesToHash(abi.GetRegisterKey(nodeName, types.SNAPSHOT_GID))
	registrationDataOld := db.storageMap[addr2][ToKey(locHashRegister.Bytes())]
	db.addr = addr2
	contractBalance, _ := db.GetBalance(&ledger.ViteTokenId)
	// register with an existed super node name, get refund
	balance1 := new(big.Int).Set(viteTotalSupply)
	addr6, _, _ := types.CreateAddress()
	db.accountBlockMap[addr6] = make(map[types.Hash]*ledger.AccountBlock)
	block13Data, err := abi.ABIConsensusGroup.PackMethod(abi.MethodNameRegister, types.SNAPSHOT_GID, nodeName, addr6)
	if err != nil {
		panic(err)
	}
	hash13 := types.DataHash([]byte{1, 3})
	block13 := &ledger.AccountBlock{
		Height:         3,
		ToAddress:      addr2,
		AccountAddress: addr1,
		BlockType:      ledger.BlockTypeSendCall,
		PrevHash:       hash12,
		Amount:         new(big.Int).Mul(big.NewInt(1e6), big.NewInt(1e18)),
		Fee:            big.NewInt(0),
		Data:           block13Data,
		TokenId:        ledger.ViteTokenId,
		Hash:           hash13,
	}
	vm := NewVM()
	//vm.Debug = true
	db.addr = addr1
	sendRegisterBlock, isRetry, err := vm.RunV2(db, block13, nil, nil)
	balance1.Sub(balance1, block13.Amount)
	if sendRegisterBlock == nil ||
		len(sendRegisterBlock.AccountBlock.SendBlockList) != 0 || isRetry || err != nil ||
		sendRegisterBlock.AccountBlock.Quota != contracts.RegisterGas ||
		!bytes.Equal(sendRegisterBlock.AccountBlock.Data, block13Data) ||
		db.balanceMap[addr1][ledger.ViteTokenId].Cmp(balance1) != 0 {
		t.Fatalf("send register transaction error")
	}
	db.accountBlockMap[addr1][hash13] = sendRegisterBlock.AccountBlock

	hash21 := types.DataHash([]byte{2, 1})
	block21 := &ledger.AccountBlock{
		Height:         1,
		AccountAddress: addr2,
		BlockType:      ledger.BlockTypeReceive,
		FromBlockHash:  hash13,
		Hash:           hash21,
	}
	vm = NewVM()
	//vm.Debug = true
	db.addr = addr2
	receiveRegisterBlock, isRetry, err := vm.RunV2(db, block21, sendRegisterBlock.AccountBlock, NewTestGlobalStatus(0, snapshot2))
	contractBalance.Add(contractBalance, block13.Amount)
	newBalance, _ := db.GetBalance(&ledger.ViteTokenId)
	if receiveRegisterBlock == nil ||
		len(receiveRegisterBlock.AccountBlock.SendBlockList) != 1 || isRetry || err == nil ||
		db.balanceMap[addr1][ledger.ViteTokenId].Cmp(balance1) != 0 ||
		!bytes.Equal(db.storageMap[addr2][ToKey(locHashRegister.Bytes())], registrationDataOld) ||
		receiveRegisterBlock.AccountBlock.Quota != 0 ||
		len(receiveRegisterBlock.AccountBlock.Data) != 33 ||
		receiveRegisterBlock.AccountBlock.Data[32] != byte(1) ||
		receiveRegisterBlock.AccountBlock.SendBlockList[0].TokenId != block13.TokenId ||
		receiveRegisterBlock.AccountBlock.SendBlockList[0].Amount.Cmp(block13.Amount) != 0 ||
		receiveRegisterBlock.AccountBlock.SendBlockList[0].BlockType != ledger.BlockTypeSendCall ||
		receiveRegisterBlock.AccountBlock.SendBlockList[0].AccountAddress != block13.ToAddress ||
		receiveRegisterBlock.AccountBlock.SendBlockList[0].ToAddress != block13.AccountAddress ||
		newBalance.Cmp(contractBalance) != 0 ||
		!bytes.Equal(receiveRegisterBlock.AccountBlock.SendBlockList[0].Data, []byte{1}) {
		t.Fatalf("receive register transaction error")
	}
	db.accountBlockMap[addr2] = make(map[types.Hash]*ledger.AccountBlock)
	db.accountBlockMap[addr2][hash21] = receiveRegisterBlock.AccountBlock
	hash22 := types.DataHash([]byte{2, 2})
	receiveRegisterBlock.AccountBlock.SendBlockList[0].Hash = hash22
	receiveRegisterBlock.AccountBlock.SendBlockList[0].PrevHash = hash21
	db.accountBlockMap[addr2][hash22] = receiveRegisterBlock.AccountBlock.SendBlockList[0]

	hash14 := types.DataHash([]byte{1, 4})
	block14 := &ledger.AccountBlock{
		Height:         4,
		AccountAddress: addr1,
		BlockType:      ledger.BlockTypeReceive,
		FromBlockHash:  hash22,
		Hash:           hash14,
	}
	vm = NewVM()
	//vm.Debug = true
	db.addr = addr1
	receiveRegisterRefuncBlock, isRetry, err := vm.RunV2(db, block14, receiveRegisterBlock.AccountBlock.SendBlockList[0], nil)
	balance1.Add(balance1, block13.Amount)
	if receiveRegisterRefuncBlock == nil ||
		len(receiveRegisterRefuncBlock.AccountBlock.SendBlockList) != 0 || isRetry || err != nil ||
		db.balanceMap[addr1][ledger.ViteTokenId].Cmp(balance1) != 0 ||
		receiveRegisterRefuncBlock.AccountBlock.Quota != 21000 {
		t.Fatalf("receive register refund transaction error")
	}
	db.accountBlockMap[addr1][hash14] = receiveRegisterRefuncBlock.AccountBlock
}

func TestContractsRegister(t *testing.T) {
	// prepare db
	viteTotalSupply := new(big.Int).Mul(big.NewInt(1e9), big.NewInt(1e18))
	db, addr1, _, hash12, snapshot2, timestamp := prepareDb(viteTotalSupply)
	// register
	balance1 := new(big.Int).Set(viteTotalSupply)
	addr6, privateKey6, _ := types.CreateAddress()
	addr7, _, _ := types.CreateAddress()
	publicKey6 := ed25519.PublicKey(privateKey6.PubByte())
	db.accountBlockMap[addr6] = make(map[types.Hash]*ledger.AccountBlock)
	db.accountBlockMap[addr7] = make(map[types.Hash]*ledger.AccountBlock)
	addr2 := types.AddressConsensusGroup
	nodeName := "super1"
	block13Data, err := abi.ABIConsensusGroup.PackMethod(abi.MethodNameRegister, types.SNAPSHOT_GID, nodeName, addr7)
	hash13 := types.DataHash([]byte{1, 3})
	block13 := &ledger.AccountBlock{
		Height:         3,
		ToAddress:      addr2,
		AccountAddress: addr1,
		BlockType:      ledger.BlockTypeSendCall,
		PrevHash:       hash12,
		Amount:         new(big.Int).Mul(big.NewInt(1e6), big.NewInt(1e18)),
		Fee:            big.NewInt(0),
		Data:           block13Data,
		TokenId:        ledger.ViteTokenId,
		Hash:           hash13,
	}
	vm := NewVM()
	//vm.Debug = true
	db.addr = addr1
	sendRegisterBlock, isRetry, err := vm.RunV2(db, block13, nil, nil)
	balance1.Sub(balance1, block13.Amount)
	if sendRegisterBlock == nil ||
		len(sendRegisterBlock.AccountBlock.SendBlockList) != 0 || isRetry || err != nil ||
		sendRegisterBlock.AccountBlock.Quota != contracts.RegisterGas ||
		!bytes.Equal(sendRegisterBlock.AccountBlock.Data, block13Data) ||
		db.balanceMap[addr1][ledger.ViteTokenId].Cmp(balance1) != 0 {
		t.Fatalf("send register transaction error")
	}
	db.accountBlockMap[addr1][hash13] = sendRegisterBlock.AccountBlock

	hash21 := types.DataHash([]byte{2, 1})
	block21 := &ledger.AccountBlock{
		Height:         1,
		AccountAddress: addr2,
		BlockType:      ledger.BlockTypeReceive,
		FromBlockHash:  hash13,
		Hash:           hash21,
	}
	vm = NewVM()
	//vm.Debug = true
	locHashRegister := abi.GetRegisterKey(nodeName, types.SNAPSHOT_GID)
	hisAddrList := []types.Address{addr7}
	withdrawHeight := snapshot2.Height + 3600*24*90
	registrationData, _ := abi.ABIConsensusGroup.PackVariable(abi.VariableNameRegistration, nodeName, addr7, addr1, block13.Amount, withdrawHeight, snapshot2.Timestamp.Unix(), int64(0), hisAddrList)
	db.addr = addr2
	receiveRegisterBlock, isRetry, err := vm.RunV2(db, block21, sendRegisterBlock.AccountBlock, NewTestGlobalStatus(0, snapshot2))
	if receiveRegisterBlock == nil ||
		len(receiveRegisterBlock.AccountBlock.SendBlockList) != 0 || isRetry || err != nil ||
		db.balanceMap[addr1][ledger.ViteTokenId].Cmp(balance1) != 0 ||
		!bytes.Equal(db.storageMap[addr2][ToKey(locHashRegister)], registrationData) ||
		len(receiveRegisterBlock.AccountBlock.Data) != 33 ||
		receiveRegisterBlock.AccountBlock.Data[32] != byte(0) ||
		receiveRegisterBlock.AccountBlock.Quota != 0 {
		t.Fatalf("receive register transaction error")
	}
	db.accountBlockMap[addr2] = make(map[types.Hash]*ledger.AccountBlock)
	db.accountBlockMap[addr2][hash21] = receiveRegisterBlock.AccountBlock

	// update registration
	block14Data, err := abi.ABIConsensusGroup.PackMethod(abi.MethodNameUpdateRegistration, types.SNAPSHOT_GID, nodeName, addr6)
	hash14 := types.DataHash([]byte{1, 4})
	block14 := &ledger.AccountBlock{
		Height:         4,
		ToAddress:      addr2,
		AccountAddress: addr1,
		BlockType:      ledger.BlockTypeSendCall,
		PrevHash:       hash13,
		Data:           block14Data,
		Amount:         big.NewInt(0),
		Fee:            big.NewInt(0),
		TokenId:        ledger.ViteTokenId,
		Hash:           hash14,
	}
	vm = NewVM()
	//vm.Debug = true
	db.addr = addr1
	sendRegisterBlock2, isRetry, err := vm.RunV2(db, block14, nil, nil)
	if sendRegisterBlock2 == nil ||
		len(sendRegisterBlock2.AccountBlock.SendBlockList) != 0 || isRetry || err != nil ||
		sendRegisterBlock2.AccountBlock.Quota != contracts.UpdateRegistrationGas ||
		!bytes.Equal(sendRegisterBlock2.AccountBlock.Data, block14Data) ||
		db.balanceMap[addr1][ledger.ViteTokenId].Cmp(balance1) != 0 {
		t.Fatalf("send update registration transaction error")
	}
	db.accountBlockMap[addr1][hash14] = sendRegisterBlock2.AccountBlock

	hash22 := types.DataHash([]byte{2, 2})
	block22 := &ledger.AccountBlock{
		Height:         2,
		AccountAddress: addr2,
		BlockType:      ledger.BlockTypeReceive,
		FromBlockHash:  hash14,
		PrevHash:       hash21,
		Hash:           hash22,
	}
	vm = NewVM()
	//vm.Debug = true
	hisAddrList = append(hisAddrList, addr6)
	registrationData, _ = abi.ABIConsensusGroup.PackVariable(abi.VariableNameRegistration, nodeName, addr6, addr1, block13.Amount, withdrawHeight, snapshot2.Timestamp.Unix(), int64(0), hisAddrList)
	db.addr = addr2
	receiveRegisterBlock2, isRetry, err := vm.RunV2(db, block22, sendRegisterBlock2.AccountBlock, NewTestGlobalStatus(0, snapshot2))
	if receiveRegisterBlock2 == nil ||
		len(receiveRegisterBlock2.AccountBlock.SendBlockList) != 0 || isRetry || err != nil ||
		db.balanceMap[addr1][ledger.ViteTokenId].Cmp(balance1) != 0 ||
		!bytes.Equal(db.storageMap[addr2][ToKey(locHashRegister)], registrationData) ||
		len(receiveRegisterBlock2.AccountBlock.Data) != 33 ||
		receiveRegisterBlock2.AccountBlock.Data[32] != byte(0) ||
		receiveRegisterBlock2.AccountBlock.Quota != 0 {
		t.Fatalf("receive update registration transaction error")
	}
	db.accountBlockMap[addr2][hash22] = receiveRegisterBlock2.AccountBlock

	// get contracts data
	db.addr = types.AddressConsensusGroup
	if registerList, _ := abi.GetCandidateList(db, types.SNAPSHOT_GID); len(registerList) != 3 || len(registerList[0].Name) == 0 {
		t.Fatalf("get register list failed")
	}

	// cancel register
	time3 := time.Unix(timestamp+1, 0)
	snapshot3 := &ledger.SnapshotBlock{Height: 3, Timestamp: &time3, Hash: types.DataHash([]byte{10, 3}), PublicKey: publicKey6}
	db.snapshotBlockList = append(db.snapshotBlockList, snapshot3)
	time4 := time.Unix(timestamp+2, 0)
	snapshot4 := &ledger.SnapshotBlock{Height: 4, Timestamp: &time4, Hash: types.DataHash([]byte{10, 4}), PublicKey: publicKey6}
	db.snapshotBlockList = append(db.snapshotBlockList, snapshot4)
	time5 := time.Unix(timestamp+3, 0)
	snapshot5 := &ledger.SnapshotBlock{Height: 3 + 3600*24*90, Timestamp: &time5, Hash: types.DataHash([]byte{10, 5})}
	db.snapshotBlockList = append(db.snapshotBlockList, snapshot5)

	hash15 := types.DataHash([]byte{1, 5})
	block15Data, _ := abi.ABIConsensusGroup.PackMethod(abi.MethodNameCancelRegister, types.SNAPSHOT_GID, nodeName)
	block15 := &ledger.AccountBlock{
		Height:         5,
		ToAddress:      addr2,
		AccountAddress: addr1,
		Amount:         big.NewInt(0),
		TokenId:        ledger.ViteTokenId,
		BlockType:      ledger.BlockTypeSendCall,
		Fee:            big.NewInt(0),
		PrevHash:       hash13,
		Data:           block15Data,
		Hash:           hash15,
	}
	vm = NewVM()
	//vm.Debug = true
	db.addr = addr1
	sendCancelRegisterBlock, isRetry, err := vm.RunV2(db, block15, nil, nil)
	if sendCancelRegisterBlock == nil ||
		len(sendCancelRegisterBlock.AccountBlock.SendBlockList) != 0 || isRetry || err != nil ||
		sendCancelRegisterBlock.AccountBlock.Quota != contracts.CancelRegisterGas ||
		!bytes.Equal(sendCancelRegisterBlock.AccountBlock.Data, block15Data) ||
		db.balanceMap[addr1][ledger.ViteTokenId].Cmp(balance1) != 0 {
		t.Fatalf("send cancel register transaction error")
	}
	db.accountBlockMap[addr1][hash15] = sendCancelRegisterBlock.AccountBlock

	hash23 := types.DataHash([]byte{2, 3})
	block23 := &ledger.AccountBlock{
		Height:         3,
		AccountAddress: addr2,
		BlockType:      ledger.BlockTypeReceive,
		PrevHash:       hash21,
		FromBlockHash:  hash15,
	}
	vm = NewVM()
	//vm.Debug = true
	db.addr = addr2
	receiveCancelRegisterBlock, isRetry, err := vm.RunV2(db, block23, sendCancelRegisterBlock.AccountBlock, NewTestGlobalStatus(0, snapshot5))
	registrationData, _ = abi.ABIConsensusGroup.PackVariable(abi.VariableNameRegistration, nodeName, addr6, addr1, helper.Big0, uint64(0), int64(-1), snapshot5.Timestamp.Unix(), hisAddrList)
	if receiveCancelRegisterBlock == nil ||
		len(receiveCancelRegisterBlock.AccountBlock.SendBlockList) != 1 || isRetry || err != nil ||
		db.balanceMap[addr2][ledger.ViteTokenId].Cmp(helper.Big0) != 0 ||
		db.balanceMap[addr1][ledger.ViteTokenId].Cmp(balance1) != 0 ||
		!bytes.Equal(db.storageMap[addr2][ToKey(locHashRegister)], registrationData) ||
		len(receiveCancelRegisterBlock.AccountBlock.Data) != 33 ||
		receiveCancelRegisterBlock.AccountBlock.Data[32] != byte(0) ||
		receiveCancelRegisterBlock.AccountBlock.Quota != 0 ||
		receiveCancelRegisterBlock.AccountBlock.SendBlockList[0].AccountAddress != addr2 ||
		receiveCancelRegisterBlock.AccountBlock.SendBlockList[0].ToAddress != addr1 ||
		receiveCancelRegisterBlock.AccountBlock.SendBlockList[0].BlockType != ledger.BlockTypeSendCall {
		t.Fatalf("receive cancel register transaction error")
	}
	db.accountBlockMap[addr2][hash23] = receiveCancelRegisterBlock.AccountBlock
	hash24 := types.DataHash([]byte{2, 4})
	receiveCancelRegisterBlock.AccountBlock.SendBlockList[0].Hash = hash24
	receiveCancelRegisterBlock.AccountBlock.SendBlockList[0].PrevHash = hash23
	db.accountBlockMap[addr2][hash24] = receiveCancelRegisterBlock.AccountBlock.SendBlockList[0]

	hash16 := types.DataHash([]byte{1, 6})
	block16 := &ledger.AccountBlock{
		Height:         6,
		AccountAddress: addr1,
		BlockType:      ledger.BlockTypeReceive,
		PrevHash:       hash16,
		FromBlockHash:  hash23,
		Hash:           hash16,
	}
	vm = NewVM()
	//vm.Debug = true
	db.addr = addr1
	balance1.Add(balance1, block13.Amount)
	receiveCancelRegisterRefundBlock, isRetry, err := vm.RunV2(db, block16, receiveCancelRegisterBlock.AccountBlock.SendBlockList[0], nil)
	if receiveCancelRegisterRefundBlock == nil ||
		len(receiveCancelRegisterRefundBlock.AccountBlock.SendBlockList) != 0 || isRetry || err != nil ||
		db.balanceMap[addr2][ledger.ViteTokenId].Cmp(helper.Big0) != 0 ||
		db.balanceMap[addr1][ledger.ViteTokenId].Cmp(balance1) != 0 ||
		receiveCancelRegisterRefundBlock.AccountBlock.Quota != 21000 {
		t.Fatalf("receive cancel register refund transaction error")
	}
	db.accountBlockMap[addr1][hash16] = receiveCancelRegisterRefundBlock.AccountBlock

	// Reward
	hash17 := types.DataHash([]byte{1, 7})
	block17Data, _ := abi.ABIConsensusGroup.PackMethod(abi.MethodNameReward, types.SNAPSHOT_GID, nodeName, addr1)
	block17 := &ledger.AccountBlock{
		Height:         7,
		ToAddress:      addr2,
		AccountAddress: addr1,
		Amount:         big.NewInt(0),
		TokenId:        ledger.ViteTokenId,
		BlockType:      ledger.BlockTypeSendCall,
		Fee:            big.NewInt(0),
		PrevHash:       hash16,
		Data:           block17Data,
		Hash:           hash17,
	}
	vm = NewVM()
	//vm.Debug = true
	db.addr = addr1
	sendRewardBlock, isRetry, err := vm.RunV2(db, block17, nil, nil)
	if sendRewardBlock == nil ||
		len(sendRewardBlock.AccountBlock.SendBlockList) != 0 || isRetry || err != nil ||
		sendRewardBlock.AccountBlock.Quota != contracts.RewardGas ||
		!bytes.Equal(sendRewardBlock.AccountBlock.Data, block17Data) ||
		db.balanceMap[addr1][ledger.ViteTokenId].Cmp(balance1) != 0 {
		t.Fatalf("send cancel register transaction error")
	}
	db.accountBlockMap[addr1][hash17] = sendRewardBlock.AccountBlock

	hash25 := types.DataHash([]byte{2, 5})
	block25 := &ledger.AccountBlock{
		Height:         5,
		AccountAddress: addr2,
		BlockType:      ledger.BlockTypeReceive,
		PrevHash:       hash24,
		FromBlockHash:  hash17,
	}
	vm = NewVM()
	//vm.Debug = true
	db.addr = addr2
	receiveRewardBlock, isRetry, err := vm.RunV2(db, block25, sendRewardBlock.AccountBlock, NewTestGlobalStatus(0, snapshot5))
	registrationData, _ = abi.ABIConsensusGroup.PackVariable(abi.VariableNameRegistration, nodeName, addr6, addr1, helper.Big0, uint64(0), int64(-1), snapshot5.Timestamp.Unix(), hisAddrList)
	if receiveRewardBlock == nil ||
		len(receiveRewardBlock.AccountBlock.SendBlockList) != 0 || isRetry || err != util.ErrInvalidMethodParam ||
		db.balanceMap[addr2][ledger.ViteTokenId].Cmp(helper.Big0) != 0 ||
		db.balanceMap[addr1][ledger.ViteTokenId].Cmp(balance1) != 0 ||
		!bytes.Equal(db.storageMap[addr2][ToKey(locHashRegister)], registrationData) ||
		len(receiveRewardBlock.AccountBlock.Data) != 33 ||
		receiveRewardBlock.AccountBlock.Data[32] != byte(1) ||
		receiveRewardBlock.AccountBlock.Quota != 0 {
		t.Fatalf("receive reward transaction error")
	}
	db.accountBlockMap[addr2][hash25] = receiveRewardBlock.AccountBlock
}

func TestContractsVote(t *testing.T) {
	// prepare db
	viteTotalSupply := new(big.Int).Mul(big.NewInt(2e6), big.NewInt(1e18))
	db, addr1, _, hash12, snapshot2, _ := prepareDb(viteTotalSupply)
	// vote
	addr3 := types.AddressConsensusGroup
	nodeName := "s1"
	block13Data, _ := abi.ABIConsensusGroup.PackMethod(abi.MethodNameVote, types.SNAPSHOT_GID, nodeName)
	hash13 := types.DataHash([]byte{1, 3})
	block13 := &ledger.AccountBlock{
		Height:         3,
		ToAddress:      addr3,
		AccountAddress: addr1,
		PrevHash:       hash12,
		Amount:         big.NewInt(0),
		TokenId:        ledger.ViteTokenId,
		BlockType:      ledger.BlockTypeSendCall,
		Fee:            big.NewInt(0),
		Data:           block13Data,
		Hash:           hash13,
	}
	vm := NewVM()
	//vm.Debug = true
	db.addr = addr1
	sendVoteBlock, isRetry, err := vm.RunV2(db, block13, nil, nil)
	if sendVoteBlock == nil ||
		len(sendVoteBlock.AccountBlock.SendBlockList) != 0 || isRetry || err != nil ||
		!bytes.Equal(sendVoteBlock.AccountBlock.Data, block13Data) ||
		sendVoteBlock.AccountBlock.Quota != contracts.VoteGas {
		t.Fatalf("send vote transaction error")
	}
	db.accountBlockMap[addr1][hash13] = sendVoteBlock.AccountBlock

	hash31 := types.DataHash([]byte{3, 1})
	block31 := &ledger.AccountBlock{
		Height:         1,
		AccountAddress: addr3,
		BlockType:      ledger.BlockTypeReceive,
		FromBlockHash:  hash13,
		Hash:           hash31,
	}
	vm = NewVM()
	//vm.Debug = true
	db.addr = addr3
	receiveVoteBlock, isRetry, err := vm.RunV2(db, block31, sendVoteBlock.AccountBlock, NewTestGlobalStatus(0, snapshot2))
	voteKey := abi.GetVoteKey(addr1, types.SNAPSHOT_GID)
	voteData, _ := abi.ABIConsensusGroup.PackVariable(abi.VariableNameVoteStatus, nodeName)
	if receiveVoteBlock == nil ||
		len(receiveVoteBlock.AccountBlock.SendBlockList) != 0 || isRetry || err != nil ||
		!bytes.Equal(db.storageMap[addr3][ToKey(voteKey)], voteData) ||
		len(receiveVoteBlock.AccountBlock.Data) != 33 ||
		receiveVoteBlock.AccountBlock.Data[32] != byte(0) ||
		receiveVoteBlock.AccountBlock.Quota != 0 {
		t.Fatalf("receive vote transaction error")
	}
	db.accountBlockMap[addr3] = make(map[types.Hash]*ledger.AccountBlock)
	db.accountBlockMap[addr3][hash31] = receiveVoteBlock.AccountBlock

	addr4, _ := types.BytesToAddress(helper.HexToBytes("e5bf58cacfb74cf8c49a1d5e59d3919c9a4cb9ed"))
	db.accountBlockMap[addr4] = make(map[types.Hash]*ledger.AccountBlock)
	nodeName2 := "s2"
	block14Data, _ := abi.ABIConsensusGroup.PackMethod(abi.MethodNameVote, types.SNAPSHOT_GID, nodeName2)
	hash14 := types.DataHash([]byte{1, 4})
	block14 := &ledger.AccountBlock{
		Height:         4,
		ToAddress:      addr3,
		AccountAddress: addr1,
		Amount:         big.NewInt(0),
		TokenId:        ledger.ViteTokenId,
		BlockType:      ledger.BlockTypeSendCall,
		Fee:            big.NewInt(0),
		PrevHash:       hash13,
		Data:           block14Data,
		Hash:           hash14,
	}
	vm = NewVM()
	//vm.Debug = true
	db.addr = addr1
	sendVoteBlock2, isRetry, err := vm.RunV2(db, block14, nil, nil)
	if sendVoteBlock2 == nil ||
		len(sendVoteBlock2.AccountBlock.SendBlockList) != 0 || isRetry || err != nil ||
		!bytes.Equal(sendVoteBlock2.AccountBlock.Data, block14Data) ||
		sendVoteBlock2.AccountBlock.Quota != contracts.VoteGas {
		t.Fatalf("send vote transaction 2 error")
	}
	db.accountBlockMap[addr1][hash14] = sendVoteBlock2.AccountBlock

	hash32 := types.DataHash([]byte{3, 2})
	block32 := &ledger.AccountBlock{
		Height:         2,
		AccountAddress: addr3,
		BlockType:      ledger.BlockTypeReceive,
		PrevHash:       hash31,
		FromBlockHash:  hash14,
		Hash:           hash32,
	}
	vm = NewVM()
	//vm.Debug = true
	db.addr = addr3
	receiveVoteBlock2, isRetry, err := vm.RunV2(db, block32, sendVoteBlock2.AccountBlock, NewTestGlobalStatus(0, snapshot2))
	voteData, _ = abi.ABIConsensusGroup.PackVariable(abi.VariableNameVoteStatus, nodeName2)
	if receiveVoteBlock2 == nil ||
		len(receiveVoteBlock2.AccountBlock.SendBlockList) != 0 || isRetry || err != nil ||
		!bytes.Equal(db.storageMap[addr3][ToKey(voteKey)], voteData) ||
		len(receiveVoteBlock2.AccountBlock.Data) != 33 ||
		receiveVoteBlock2.AccountBlock.Data[32] != byte(0) ||
		receiveVoteBlock2.AccountBlock.Quota != 0 {
		t.Fatalf("receive vote transaction 2 error")
	}
	db.accountBlockMap[addr3][hash32] = receiveVoteBlock2.AccountBlock

	// get contracts data
	db.addr = types.AddressConsensusGroup
	if voteList, _ := abi.GetVoteList(db, types.SNAPSHOT_GID); len(voteList) != 1 || voteList[0].NodeName != nodeName2 {
		t.Fatalf("get vote list failed")
	}

	// cancel vote
	block15Data, _ := abi.ABIConsensusGroup.PackMethod(abi.MethodNameCancelVote, types.SNAPSHOT_GID)
	hash15 := types.DataHash([]byte{1, 5})
	block15 := &ledger.AccountBlock{
		Height:         5,
		ToAddress:      addr3,
		AccountAddress: addr1,
		Amount:         big.NewInt(0),
		TokenId:        ledger.ViteTokenId,
		BlockType:      ledger.BlockTypeSendCall,
		Fee:            big.NewInt(0),
		PrevHash:       hash14,
		Data:           block15Data,
		Hash:           hash15,
	}
	vm = NewVM()
	//vm.Debug = true
	db.addr = addr1
	sendCancelVoteBlock, isRetry, err := vm.RunV2(db, block15, nil, nil)
	if sendCancelVoteBlock == nil ||
		len(sendCancelVoteBlock.AccountBlock.SendBlockList) != 0 || isRetry || err != nil ||
		!bytes.Equal(sendCancelVoteBlock.AccountBlock.Data, block15Data) ||
		sendCancelVoteBlock.AccountBlock.Quota != contracts.CancelVoteGas {
		t.Fatalf("send cancel vote transaction error")
	}
	db.accountBlockMap[addr1][hash15] = sendCancelVoteBlock.AccountBlock

	hash33 := types.DataHash([]byte{3, 3})
	block33 := &ledger.AccountBlock{
		Height:         3,
		AccountAddress: addr3,
		BlockType:      ledger.BlockTypeReceive,
		PrevHash:       hash32,
		FromBlockHash:  hash15,
		Hash:           hash33,
	}
	vm = NewVM()
	//vm.Debug = true
	db.addr = addr3
	receiveCancelVoteBlock, isRetry, err := vm.RunV2(db, block33, sendCancelVoteBlock.AccountBlock, NewTestGlobalStatus(0, snapshot2))
	if receiveCancelVoteBlock == nil ||
		len(receiveCancelVoteBlock.AccountBlock.SendBlockList) != 0 || isRetry || err != nil ||
		len(db.storageMap[addr3][ToKey(voteKey)]) != 0 ||
		len(receiveCancelVoteBlock.AccountBlock.Data) != 33 ||
		receiveCancelVoteBlock.AccountBlock.Data[32] != byte(0) ||
		receiveCancelVoteBlock.AccountBlock.Quota != 0 {
		t.Fatalf("receive cancel vote transaction error")
	}
	db.accountBlockMap[addr3][hash33] = receiveCancelVoteBlock.AccountBlock
}

func TestContractsPledge(t *testing.T) {
	// prepare db
	viteTotalSupply := new(big.Int).Mul(big.NewInt(2e6), big.NewInt(1e18))
	db, addr1, _, hash12, snapshot2, timestamp := prepareDb(viteTotalSupply)
	// pledge
	balance1 := new(big.Int).Set(viteTotalSupply)
	addr4, _, _ := types.CreateAddress()
	db.accountBlockMap[addr4] = make(map[types.Hash]*ledger.AccountBlock)
	addr5 := types.AddressPledge
	pledgeAmount := new(big.Int).Set(new(big.Int).Mul(big.NewInt(1000), util.AttovPerVite))
	block13Data, err := abi.ABIPledge.PackMethod(abi.MethodNamePledge, addr4)
	hash13 := types.DataHash([]byte{1, 3})
	block13 := &ledger.AccountBlock{
		Height:         3,
		ToAddress:      addr5,
		AccountAddress: addr1,
		Amount:         pledgeAmount,
		TokenId:        ledger.ViteTokenId,
		BlockType:      ledger.BlockTypeSendCall,
		Fee:            big.NewInt(0),
		PrevHash:       hash12,
		Data:           block13Data,
		Hash:           hash13,
	}
	vm := NewVM()
	//vm.Debug = true
	db.addr = addr1
	sendPledgeBlock, isRetry, err := vm.RunV2(db, block13, nil, nil)
	balance1.Sub(balance1, pledgeAmount)
	if sendPledgeBlock == nil ||
		len(sendPledgeBlock.AccountBlock.SendBlockList) != 0 || isRetry || err != nil ||
		db.balanceMap[addr1][ledger.ViteTokenId].Cmp(balance1) != 0 ||
		!bytes.Equal(sendPledgeBlock.AccountBlock.Data, block13Data) ||
		sendPledgeBlock.AccountBlock.Quota != contracts.PledgeGas {
		t.Fatalf("send pledge transaction error")
	}
	db.accountBlockMap[addr1][hash13] = sendPledgeBlock.AccountBlock

	hash51 := types.DataHash([]byte{5, 1})
	block51 := &ledger.AccountBlock{
		Height:         1,
		AccountAddress: addr5,
		BlockType:      ledger.BlockTypeReceive,
		FromBlockHash:  hash13,
		Hash:           hash51,
	}
	vm = NewVM()
	//vm.Debug = true
	db.addr = addr5
	receivePledgeBlock, isRetry, err := vm.RunV2(db, block51, sendPledgeBlock.AccountBlock, NewTestGlobalStatus(0, snapshot2))
	beneficialKey := abi.GetPledgeBeneficialKey(addr4)
	pledgeKey := abi.GetPledgeKey(addr1, addr4)
	withdrawHeight := snapshot2.Height + 3600*24*3
	if receivePledgeBlock == nil ||
		len(receivePledgeBlock.AccountBlock.SendBlockList) != 0 || isRetry || err != nil ||
		!bytes.Equal(db.storageMap[addr5][ToKey(pledgeKey)], helper.JoinBytes(helper.LeftPadBytes(pledgeAmount.Bytes(), helper.WordSize), helper.LeftPadBytes(new(big.Int).SetUint64(withdrawHeight).Bytes(), helper.WordSize), helper.LeftPadBytes(addr4.Bytes(), helper.WordSize))) ||
		!bytes.Equal(db.storageMap[addr5][ToKey(beneficialKey)], helper.LeftPadBytes(pledgeAmount.Bytes(), helper.WordSize)) ||
		db.balanceMap[addr5][ledger.ViteTokenId].Cmp(pledgeAmount) != 0 ||
		len(receivePledgeBlock.AccountBlock.Data) != 33 ||
		receivePledgeBlock.AccountBlock.Data[32] != byte(0) ||
		receivePledgeBlock.AccountBlock.Quota != 0 {
		t.Fatalf("receive pledge transaction error")
	}
	db.accountBlockMap[addr5] = make(map[types.Hash]*ledger.AccountBlock)
	db.accountBlockMap[addr5][hash51] = receivePledgeBlock.AccountBlock

	block14Data, _ := abi.ABIPledge.PackMethod(abi.MethodNamePledge, addr4)
	hash14 := types.DataHash([]byte{1, 4})
	block14 := &ledger.AccountBlock{
		Height:         4,
		ToAddress:      addr5,
		AccountAddress: addr1,
		Amount:         pledgeAmount,
		TokenId:        ledger.ViteTokenId,
		BlockType:      ledger.BlockTypeSendCall,
		Fee:            big.NewInt(0),
		PrevHash:       hash13,
		Data:           block14Data,
		Hash:           hash14,
	}
	vm = NewVM()
	//vm.Debug = true
	db.addr = addr1
	sendPledgeBlock2, isRetry, err := vm.RunV2(db, block14, nil, nil)
	balance1.Sub(balance1, pledgeAmount)
	if sendPledgeBlock2 == nil ||
		len(sendPledgeBlock2.AccountBlock.SendBlockList) != 0 || isRetry || err != nil ||
		!bytes.Equal(sendPledgeBlock2.AccountBlock.Data, block14Data) ||
		db.balanceMap[addr1][ledger.ViteTokenId].Cmp(balance1) != 0 ||
		sendPledgeBlock2.AccountBlock.Quota != contracts.PledgeGas {
		t.Fatalf("send pledge transaction 2 error")
	}
	db.accountBlockMap[addr1][hash14] = sendPledgeBlock2.AccountBlock

	hash52 := types.DataHash([]byte{5, 2})
	block52 := &ledger.AccountBlock{
		Height:         2,
		AccountAddress: addr5,
		BlockType:      ledger.BlockTypeReceive,
		PrevHash:       hash51,
		FromBlockHash:  hash14,
		Hash:           hash52,
	}
	vm = NewVM()
	//vm.Debug = true
	db.addr = addr5
	receivePledgeBlock2, isRetry, err := vm.RunV2(db, block52, sendPledgeBlock2.AccountBlock, NewTestGlobalStatus(0, snapshot2))
	newPledgeAmount := new(big.Int).Add(pledgeAmount, pledgeAmount)
	if receivePledgeBlock2 == nil ||
		len(receivePledgeBlock2.AccountBlock.SendBlockList) != 0 || isRetry || err != nil ||
		!bytes.Equal(db.storageMap[addr5][ToKey(pledgeKey)], helper.JoinBytes(helper.LeftPadBytes(newPledgeAmount.Bytes(), helper.WordSize), helper.LeftPadBytes(new(big.Int).SetUint64(withdrawHeight).Bytes(), helper.WordSize), helper.LeftPadBytes(addr4.Bytes(), helper.WordSize))) ||
		!bytes.Equal(db.storageMap[addr5][ToKey(beneficialKey)], helper.LeftPadBytes(newPledgeAmount.Bytes(), helper.WordSize)) ||
		db.balanceMap[addr5][ledger.ViteTokenId].Cmp(newPledgeAmount) != 0 ||
		len(receivePledgeBlock2.AccountBlock.Data) != 33 ||
		receivePledgeBlock2.AccountBlock.Data[32] != byte(0) ||
		receivePledgeBlock2.AccountBlock.Quota != 0 {
		t.Fatalf("receive pledge transaction 2 error")
	}
	db.accountBlockMap[addr5][hash52] = receivePledgeBlock2.AccountBlock

	// get contracts data
	db.addr = types.AddressPledge
	if pledgeAmount, _ := db.GetPledgeBeneficialAmount(&addr4); pledgeAmount.Cmp(newPledgeAmount) != 0 {
		t.Fatalf("get pledge beneficial amount failed")
	}

	if pledgeAmount, _ := db.GetPledgeBeneficialAmount(&addr4); pledgeAmount.Cmp(newPledgeAmount) != 0 {
		t.Fatalf("get pledge beneficial amount failed")
	}

	if pledgeInfoList, _, _ := abi.GetPledgeInfoList(db, addr1); len(pledgeInfoList) != 1 || pledgeInfoList[0].BeneficialAddr != addr4 {
		t.Fatalf("get pledge amount failed")
	}

	// cancel pledge
	for i := uint64(1); i <= uint64(3600*24*3); i++ {
		timei := time.Unix(timestamp+100+int64(i), 0)
		snapshoti := &ledger.SnapshotBlock{Height: 2 + i, Timestamp: &timei, Hash: types.DataHash([]byte{10, byte(2 + i)})}
		db.snapshotBlockList = append(db.snapshotBlockList, snapshoti)
	}
	currentSnapshot := db.snapshotBlockList[len(db.snapshotBlockList)-1]

	block15Data, _ := abi.ABIPledge.PackMethod(abi.MethodNameCancelPledge, addr4, pledgeAmount)
	hash15 := types.DataHash([]byte{1, 5})
	block15 := &ledger.AccountBlock{
		Height:         5,
		ToAddress:      addr5,
		AccountAddress: addr1,
		Amount:         helper.Big0,
		TokenId:        ledger.ViteTokenId,
		BlockType:      ledger.BlockTypeSendCall,
		Fee:            big.NewInt(0),
		PrevHash:       hash14,
		Data:           block15Data,
		Hash:           hash15,
	}
	vm = NewVM()
	//vm.Debug = true
	db.addr = addr1
	sendCancelPledgeBlock, isRetry, err := vm.RunV2(db, block15, nil, nil)
	if sendCancelPledgeBlock == nil ||
		len(sendCancelPledgeBlock.AccountBlock.SendBlockList) != 0 || isRetry || err != nil ||
		!bytes.Equal(sendCancelPledgeBlock.AccountBlock.Data, block15Data) ||
		sendCancelPledgeBlock.AccountBlock.Quota != contracts.CancelPledgeGas {
		t.Fatalf("send cancel pledge transaction error")
	}
	db.accountBlockMap[addr1][hash15] = sendCancelPledgeBlock.AccountBlock

	hash53 := types.DataHash([]byte{5, 3})
	block53 := &ledger.AccountBlock{
		Height:         3,
		AccountAddress: addr5,
		BlockType:      ledger.BlockTypeReceive,
		PrevHash:       hash52,
		FromBlockHash:  hash15,
		Hash:           hash53,
	}
	vm = NewVM()
	//vm.Debug = true
	db.addr = addr5
	receiveCancelPledgeBlock, isRetry, err := vm.RunV2(db, block53, sendCancelPledgeBlock.AccountBlock, NewTestGlobalStatus(0, currentSnapshot))
	if receiveCancelPledgeBlock == nil ||
		len(receiveCancelPledgeBlock.AccountBlock.SendBlockList) != 1 || isRetry || err != nil ||
		!bytes.Equal(db.storageMap[addr5][ToKey(pledgeKey)], helper.JoinBytes(helper.LeftPadBytes(pledgeAmount.Bytes(), helper.WordSize), helper.LeftPadBytes(new(big.Int).SetUint64(withdrawHeight).Bytes(), helper.WordSize), helper.LeftPadBytes(addr4.Bytes(), helper.WordSize))) ||
		!bytes.Equal(db.storageMap[addr5][ToKey(beneficialKey)], helper.LeftPadBytes(pledgeAmount.Bytes(), helper.WordSize)) ||
		db.balanceMap[addr5][ledger.ViteTokenId].Cmp(pledgeAmount) != 0 ||
		len(receiveCancelPledgeBlock.AccountBlock.Data) != 33 ||
		receiveCancelPledgeBlock.AccountBlock.Data[32] != byte(0) ||
		receiveCancelPledgeBlock.AccountBlock.Quota != 0 ||
		receiveCancelPledgeBlock.AccountBlock.SendBlockList[0].ToAddress != addr1 ||
		receiveCancelPledgeBlock.AccountBlock.SendBlockList[0].Amount.Cmp(pledgeAmount) != 0 ||
		receiveCancelPledgeBlock.AccountBlock.SendBlockList[0].Fee.Sign() != 0 ||
		len(receiveCancelPledgeBlock.AccountBlock.SendBlockList[0].Data) != 0 {
		t.Fatalf("receive cancel pledge transaction error")
	}
	db.accountBlockMap[addr5][hash53] = receiveCancelPledgeBlock.AccountBlock
	hash54 := types.DataHash([]byte{5, 4})
	receiveCancelPledgeBlock.AccountBlock.SendBlockList[0].Hash = hash54
	receiveCancelPledgeBlock.AccountBlock.SendBlockList[0].PrevHash = hash53
	db.accountBlockMap[addr5][hash54] = receiveCancelPledgeBlock.AccountBlock.SendBlockList[0]

	hash16 := types.DataHash([]byte{1, 6})
	block16 := &ledger.AccountBlock{
		Height:         6,
		AccountAddress: addr1,
		BlockType:      ledger.BlockTypeReceive,
		PrevHash:       hash15,
		FromBlockHash:  hash54,
		Hash:           hash16,
	}
	vm = NewVM()
	//vm.Debug = true
	db.addr = addr1
	receiveCancelPledgeRefundBlock, isRetry, err := vm.RunV2(db, block16, receiveCancelPledgeBlock.AccountBlock.SendBlockList[0], NewTestGlobalStatus(0, currentSnapshot))
	balance1.Add(balance1, pledgeAmount)
	if receiveCancelPledgeRefundBlock == nil ||
		len(receiveCancelPledgeRefundBlock.AccountBlock.SendBlockList) != 0 || isRetry || err != nil ||
		db.balanceMap[addr1][ledger.ViteTokenId].Cmp(balance1) != 0 ||
		receiveCancelPledgeRefundBlock.AccountBlock.Quota != 21000 {
		t.Fatalf("receive cancel pledge refund transaction error")
	}
	db.accountBlockMap[addr1][hash16] = receiveCancelPledgeRefundBlock.AccountBlock

	block17Data, _ := abi.ABIPledge.PackMethod(abi.MethodNameCancelPledge, addr4, pledgeAmount)
	hash17 := types.DataHash([]byte{1, 7})
	block17 := &ledger.AccountBlock{
		Height:         17,
		ToAddress:      addr5,
		AccountAddress: addr1,
		Amount:         helper.Big0,
		TokenId:        ledger.ViteTokenId,
		BlockType:      ledger.BlockTypeSendCall,
		Fee:            big.NewInt(0),
		PrevHash:       hash16,
		Data:           block17Data,
		Hash:           hash17,
	}
	vm = NewVM()
	//vm.Debug = true
	db.addr = addr1
	sendCancelPledgeBlock2, isRetry, err := vm.RunV2(db, block17, nil, nil)
	if sendCancelPledgeBlock2 == nil ||
		len(sendCancelPledgeBlock2.AccountBlock.SendBlockList) != 0 || isRetry || err != nil ||
		!bytes.Equal(sendCancelPledgeBlock2.AccountBlock.Data, block17Data) ||
		sendCancelPledgeBlock2.AccountBlock.Quota != contracts.CancelPledgeGas {
		t.Fatalf("send cancel pledge transaction 2 error")
	}
	db.accountBlockMap[addr1][hash17] = sendCancelPledgeBlock2.AccountBlock

	hash55 := types.DataHash([]byte{5, 5})
	block55 := &ledger.AccountBlock{
		Height:         5,
		AccountAddress: addr5,
		BlockType:      ledger.BlockTypeReceive,
		PrevHash:       hash54,
		FromBlockHash:  hash17,
		Hash:           hash55,
	}
	vm = NewVM()
	//vm.Debug = true
	db.addr = addr5
	receiveCancelPledgeBlock2, isRetry, err := vm.RunV2(db, block55, sendCancelPledgeBlock2.AccountBlock, NewTestGlobalStatus(0, currentSnapshot))
	if receiveCancelPledgeBlock2 == nil ||
		len(receiveCancelPledgeBlock2.AccountBlock.SendBlockList) != 1 || isRetry || err != nil ||
		len(db.storageMap[addr5][ToKey(pledgeKey)]) != 0 ||
		len(db.storageMap[addr5][ToKey(beneficialKey)]) != 0 ||
		db.balanceMap[addr5][ledger.ViteTokenId].Cmp(helper.Big0) != 0 ||
		len(receiveCancelPledgeBlock2.AccountBlock.Data) != 33 ||
		receiveCancelPledgeBlock2.AccountBlock.Data[32] != byte(0) ||
		receiveCancelPledgeBlock2.AccountBlock.Quota != 0 ||
		receiveCancelPledgeBlock.AccountBlock.SendBlockList[0].ToAddress != addr1 ||
		receiveCancelPledgeBlock.AccountBlock.SendBlockList[0].Amount.Cmp(pledgeAmount) != 0 ||
		receiveCancelPledgeBlock.AccountBlock.SendBlockList[0].Fee.Sign() != 0 ||
		len(receiveCancelPledgeBlock.AccountBlock.SendBlockList[0].Data) != 0 {
		t.Fatalf("receive cancel pledge transaction 2 error")
	}
	db.accountBlockMap[addr5][hash55] = receiveCancelPledgeBlock2.AccountBlock
	hash56 := types.DataHash([]byte{5, 6})
	receiveCancelPledgeBlock2.AccountBlock.SendBlockList[0].Hash = hash56
	receiveCancelPledgeBlock2.AccountBlock.SendBlockList[0].PrevHash = hash55
	db.accountBlockMap[addr5][hash56] = receiveCancelPledgeBlock2.AccountBlock.SendBlockList[0]

	hash18 := types.DataHash([]byte{1, 8})
	block18 := &ledger.AccountBlock{
		Height:         8,
		AccountAddress: addr1,
		BlockType:      ledger.BlockTypeReceive,
		PrevHash:       hash18,
		FromBlockHash:  hash56,
		Hash:           hash18,
	}
	vm = NewVM()
	//vm.Debug = true
	db.addr = addr1
	balance1.Add(balance1, pledgeAmount)
	receiveCancelPledgeRefundBlock2, isRetry, err := vm.RunV2(db, block18, receiveCancelPledgeBlock2.AccountBlock.SendBlockList[0], NewTestGlobalStatus(0, currentSnapshot))
	if receiveCancelPledgeRefundBlock2 == nil ||
		len(receiveCancelPledgeRefundBlock2.AccountBlock.SendBlockList) != 0 || isRetry || err != nil ||
		db.balanceMap[addr1][ledger.ViteTokenId].Cmp(balance1) != 0 ||
		receiveCancelPledgeRefundBlock2.AccountBlock.Quota != 21000 {
		t.Fatalf("receive cancel pledge refund transaction 2 error")
	}
	db.accountBlockMap[addr1][hash18] = receiveCancelPledgeRefundBlock2.AccountBlock
}

func TestContractsMintageV2(t *testing.T) {
	// prepare db
	viteTotalSupply := new(big.Int).Mul(big.NewInt(1e9), big.NewInt(1e18))
	db, addr1, _, hash12, snapshot2, _ := prepareDb(viteTotalSupply)
	// mint
	balance1 := new(big.Int).Set(viteTotalSupply)
	addr2 := types.AddressMintage
	isReIssuable := true
	tokenName := "test token"
	tokenSymbol := "t"
	totalSupply := big.NewInt(1e10)
	maxSupply := new(big.Int).Mul(big.NewInt(2), totalSupply)
	decimals := uint8(3)
	ownerBurnOnly := true
	tokenId := abi.NewTokenId(addr1, 3, hash12)
	//fee := new(big.Int).Mul(big.NewInt(1e3), util.AttovPerVite)
	//pledgeAmount := big.NewInt(0)
	//withdrawHeight := uint64(0)
	fee := big.NewInt(0)
	pledgeAmount := new(big.Int).Mul(big.NewInt(1e5), util.AttovPerVite)
	withdrawHeight := snapshot2.Height + 3600*24*30*3
	balance1.Sub(balance1, fee)
	balance1.Sub(balance1, pledgeAmount)
	block13Data, err := abi.ABIMintage.PackMethod(abi.MethodNameMint, isReIssuable, tokenId, tokenName, tokenSymbol, totalSupply, decimals, maxSupply, ownerBurnOnly)
	hash13 := types.DataHash([]byte{1, 3})
	block13 := &ledger.AccountBlock{
		Height:         3,
		ToAddress:      addr2,
		AccountAddress: addr1,
		Amount:         pledgeAmount,
		TokenId:        ledger.ViteTokenId,
		BlockType:      ledger.BlockTypeSendCall,
		Fee:            fee,
		PrevHash:       hash12,
		Data:           block13Data,
		Hash:           hash13,
	}
	vm := NewVM()
	db.addr = addr1
	sendMintageBlock, isRetry, err := vm.RunV2(db, block13, nil, nil)
	if sendMintageBlock == nil ||
		len(sendMintageBlock.AccountBlock.SendBlockList) != 0 || isRetry || err != nil ||
		db.balanceMap[addr1][ledger.ViteTokenId].Cmp(balance1) != 0 ||
		!bytes.Equal(sendMintageBlock.AccountBlock.Data, block13Data) ||
		sendMintageBlock.AccountBlock.Amount.Cmp(pledgeAmount) != 0 ||
		sendMintageBlock.AccountBlock.Fee.Cmp(fee) != 0 ||
		sendMintageBlock.AccountBlock.Quota != contracts.MintGas {
		t.Fatalf("send mint transaction error")
	}
	db.accountBlockMap[addr1][hash13] = sendMintageBlock.AccountBlock

	hash21 := types.DataHash([]byte{2, 1})
	block21 := &ledger.AccountBlock{
		Height:         1,
		AccountAddress: addr2,
		BlockType:      ledger.BlockTypeReceive,
		FromBlockHash:  hash13,
		Hash:           hash21,
	}
	vm = NewVM()
	db.addr = addr2
	receiveMintageBlock, isRetry, err := vm.RunV2(db, block21, sendMintageBlock.AccountBlock, NewTestGlobalStatus(0, snapshot2))
	key := abi.GetMintageKey(tokenId)
	tokenInfoData, _ := abi.ABIMintage.PackVariable(abi.VariableNameTokenInfo, tokenName, tokenSymbol, totalSupply, decimals, addr1, pledgeAmount, withdrawHeight, addr1, isReIssuable, maxSupply, ownerBurnOnly)
	if receiveMintageBlock == nil ||
		len(receiveMintageBlock.AccountBlock.SendBlockList) != 1 || isRetry || err != nil ||
		!bytes.Equal(db.storageMap[addr2][ToKey(key)], tokenInfoData) ||
		db.balanceMap[addr2][ledger.ViteTokenId].Cmp(pledgeAmount) != 0 ||
		len(receiveMintageBlock.AccountBlock.Data) != 33 ||
		receiveMintageBlock.AccountBlock.Data[32] != byte(0) ||
		receiveMintageBlock.AccountBlock.Quota != 0 ||
		receiveMintageBlock.AccountBlock.SendBlockList[0].Amount.Cmp(totalSupply) != 0 ||
		receiveMintageBlock.AccountBlock.SendBlockList[0].ToAddress != addr1 ||
		receiveMintageBlock.AccountBlock.SendBlockList[0].BlockType != ledger.BlockTypeSendReward ||
		receiveMintageBlock.AccountBlock.SendBlockList[0].TokenId != tokenId ||
		len(db.logList) != 1 ||
		db.logList[0].Topics[0] != abi.ABIMintage.Events[abi.EventNameMint].Id() ||
		!bytes.Equal(db.logList[0].Topics[1].Bytes(), helper.LeftPadBytes(tokenId.Bytes(), 32)) {
		t.Fatalf("receive mint transaction error")
	}
	db.accountBlockMap[addr2] = make(map[types.Hash]*ledger.AccountBlock)
	db.accountBlockMap[addr2][hash21] = receiveMintageBlock.AccountBlock
	hash22 := types.DataHash([]byte{2, 2})
	receiveMintageBlock.AccountBlock.SendBlockList[0].Hash = hash22
	receiveMintageBlock.AccountBlock.SendBlockList[0].PrevHash = hash21
	db.accountBlockMap[addr2][hash22] = receiveMintageBlock.AccountBlock.SendBlockList[0]

	hash14 := types.DataHash([]byte{1, 4})
	block14 := &ledger.AccountBlock{
		Height:         4,
		AccountAddress: addr1,
		BlockType:      ledger.BlockTypeReceive,
		FromBlockHash:  hash22,
		PrevHash:       hash13,
		Hash:           hash14,
	}
	vm = NewVM()
	db.addr = addr1
	tokenBalance := new(big.Int).Set(totalSupply)
	receiveMintageRewardBlock, isRetry, err := vm.RunV2(db, block14, receiveMintageBlock.AccountBlock.SendBlockList[0], nil)
	if receiveMintageRewardBlock == nil ||
		len(receiveMintageRewardBlock.AccountBlock.SendBlockList) != 0 || isRetry || err != nil ||
		db.balanceMap[addr1][tokenId].Cmp(tokenBalance) != 0 ||
		receiveMintageRewardBlock.AccountBlock.Quota != 21000 {
		t.Fatalf("receive mintage reward transaction error")
	}
	db.accountBlockMap[addr1][hash14] = receiveMintageRewardBlock.AccountBlock

	// get contracts data
	db.addr = types.AddressMintage
	if tokenInfo, _ := abi.GetTokenById(db, tokenId); tokenInfo == nil || tokenInfo.TokenName != tokenName {
		t.Fatalf("get token by id failed")
	}
	if tokenMap, _ := abi.GetTokenMap(db); len(tokenMap) != 2 || tokenMap[tokenId].TokenName != tokenName {
		t.Fatalf("get token map failed")
	}

	if tokenMap, _ := abi.GetTokenMapByOwner(db, addr1); len(tokenMap) != 1 {
		t.Fatalf("get token map by owner failed")
	}

	// issue
	addr3, _, _ := types.CreateAddress()
	db.storageMap[types.AddressPledge][ToKey(abi.GetPledgeBeneficialKey(addr3))], _ = abi.ABIPledge.PackVariable(abi.VariableNamePledgeBeneficial, new(big.Int).Mul(big.NewInt(1e9), big.NewInt(1e18)))
	reIssueAmount := big.NewInt(1000)
	block15Data, err := abi.ABIMintage.PackMethod(abi.MethodNameIssue, tokenId, reIssueAmount, addr3)
	hash15 := types.DataHash([]byte{1, 5})
	block15 := &ledger.AccountBlock{
		Height:         5,
		ToAddress:      addr2,
		AccountAddress: addr1,
		Amount:         big.NewInt(0),
		TokenId:        ledger.ViteTokenId,
		BlockType:      ledger.BlockTypeSendCall,
		Fee:            big.NewInt(0),
		PrevHash:       hash14,
		Data:           block15Data,
		Hash:           hash15,
	}
	vm = NewVM()
	db.addr = addr1
	sendIssueBlock, isRetry, err := vm.RunV2(db, block15, nil, nil)
	if sendIssueBlock == nil ||
		len(sendIssueBlock.AccountBlock.SendBlockList) != 0 || isRetry || err != nil ||
		db.balanceMap[addr1][ledger.ViteTokenId].Cmp(balance1) != 0 ||
		!bytes.Equal(sendIssueBlock.AccountBlock.Data, block15Data) ||
		sendIssueBlock.AccountBlock.Amount.Cmp(big.NewInt(0)) != 0 ||
		sendIssueBlock.AccountBlock.Quota != contracts.IssueGas {
		t.Fatalf("send issue transaction error")
	}
	db.accountBlockMap[addr1][hash15] = sendIssueBlock.AccountBlock

	hash23 := types.DataHash([]byte{2, 3})
	block23 := &ledger.AccountBlock{
		Height:         3,
		AccountAddress: addr2,
		BlockType:      ledger.BlockTypeReceive,
		FromBlockHash:  hash15,
		Hash:           hash21,
		PrevHash:       hash22,
	}
	vm = NewVM()
	db.addr = addr2
	receiveIssueBlock, isRetry, err := vm.RunV2(db, block23, sendIssueBlock.AccountBlock, NewTestGlobalStatus(0, snapshot2))
	totalSupply = totalSupply.Add(totalSupply, reIssueAmount)
	tokenInfoData, _ = abi.ABIMintage.PackVariable(abi.VariableNameTokenInfo, tokenName, tokenSymbol, totalSupply, decimals, addr1, pledgeAmount, withdrawHeight, addr1, isReIssuable, maxSupply, ownerBurnOnly)
	if receiveIssueBlock == nil ||
		len(receiveIssueBlock.AccountBlock.SendBlockList) != 1 || isRetry || err != nil ||
		!bytes.Equal(db.storageMap[addr2][ToKey(key)], tokenInfoData) ||
		db.balanceMap[addr2][ledger.ViteTokenId].Cmp(pledgeAmount) != 0 ||
		len(receiveIssueBlock.AccountBlock.Data) != 33 ||
		receiveIssueBlock.AccountBlock.Data[32] != byte(0) ||
		receiveIssueBlock.AccountBlock.Quota != 0 ||
		receiveIssueBlock.AccountBlock.SendBlockList[0].Amount.Cmp(reIssueAmount) != 0 ||
		receiveIssueBlock.AccountBlock.SendBlockList[0].ToAddress != addr3 ||
		receiveIssueBlock.AccountBlock.SendBlockList[0].BlockType != ledger.BlockTypeSendReward ||
		receiveIssueBlock.AccountBlock.SendBlockList[0].TokenId != tokenId ||
		len(db.logList) != 2 ||
		db.logList[1].Topics[0] != abi.ABIMintage.Events[abi.EventNameIssue].Id() ||
		!bytes.Equal(db.logList[1].Topics[1].Bytes(), helper.LeftPadBytes(tokenId.Bytes(), 32)) {
		t.Fatalf("receive issue transaction error")
	}
	db.accountBlockMap[addr2] = make(map[types.Hash]*ledger.AccountBlock)
	db.accountBlockMap[addr2][hash23] = receiveIssueBlock.AccountBlock
	hash24 := types.DataHash([]byte{2, 4})
	receiveIssueBlock.AccountBlock.SendBlockList[0].Hash = hash24
	receiveIssueBlock.AccountBlock.SendBlockList[0].PrevHash = hash23
	db.accountBlockMap[addr2][hash24] = receiveIssueBlock.AccountBlock.SendBlockList[0]

	hash31 := types.DataHash([]byte{3, 1})
	block31 := &ledger.AccountBlock{
		Height:         1,
		AccountAddress: addr3,
		BlockType:      ledger.BlockTypeReceive,
		FromBlockHash:  hash24,
		Hash:           hash31,
	}
	vm = NewVM()
	db.addr = addr3
	receiveIssueRewardBlock, isRetry, err := vm.RunV2(db, block31, receiveIssueBlock.AccountBlock.SendBlockList[0], nil)
	if receiveIssueRewardBlock == nil ||
		len(receiveIssueRewardBlock.AccountBlock.SendBlockList) != 0 || isRetry || err != nil ||
		db.balanceMap[addr3][tokenId].Cmp(reIssueAmount) != 0 ||
		receiveIssueRewardBlock.AccountBlock.Quota != 21000 {
		t.Fatalf("receive issue reward transaction error")
	}
	db.accountBlockMap[addr3] = make(map[types.Hash]*ledger.AccountBlock)
	db.accountBlockMap[addr3][hash31] = receiveIssueRewardBlock.AccountBlock

	// burn
	block16Data, err := abi.ABIMintage.PackMethod(abi.MethodNameBurn)
	hash16 := types.DataHash([]byte{1, 6})
	burnAmount := big.NewInt(1000)
	block16 := &ledger.AccountBlock{
		Height:         6,
		ToAddress:      addr2,
		AccountAddress: addr1,
		Amount:         burnAmount,
		TokenId:        tokenId,
		BlockType:      ledger.BlockTypeSendCall,
		Fee:            big.NewInt(0),
		PrevHash:       hash15,
		Data:           block16Data,
		Hash:           hash16,
	}
	vm = NewVM()
	db.addr = addr1
	sendBurnBlock, isRetry, err := vm.RunV2(db, block16, nil, nil)
	totalSupply = totalSupply.Sub(totalSupply, burnAmount)
	tokenBalance = tokenBalance.Sub(tokenBalance, burnAmount)
	if sendBurnBlock == nil ||
		len(sendBurnBlock.AccountBlock.SendBlockList) != 0 || isRetry || err != nil ||
		db.balanceMap[addr1][ledger.ViteTokenId].Cmp(balance1) != 0 ||
		db.balanceMap[addr1][tokenId].Cmp(tokenBalance) != 0 ||
		!bytes.Equal(sendBurnBlock.AccountBlock.Data, block16Data) ||
		sendBurnBlock.AccountBlock.Amount.Cmp(burnAmount) != 0 ||
		sendBurnBlock.AccountBlock.Quota != contracts.BurnGas {
		t.Fatalf("send burn transaction error")
	}
	db.accountBlockMap[addr1][hash16] = sendBurnBlock.AccountBlock

	hash25 := types.DataHash([]byte{2, 5})
	block25 := &ledger.AccountBlock{
		Height:         5,
		AccountAddress: addr2,
		BlockType:      ledger.BlockTypeReceive,
		FromBlockHash:  hash16,
		Hash:           hash25,
		PrevHash:       hash24,
	}
	vm = NewVM()
	db.addr = addr2
	receiveBurnBlock, isRetry, err := vm.RunV2(db, block25, sendBurnBlock.AccountBlock, NewTestGlobalStatus(0, snapshot2))
	tokenInfoData, _ = abi.ABIMintage.PackVariable(abi.VariableNameTokenInfo, tokenName, tokenSymbol, totalSupply, decimals, addr1, pledgeAmount, withdrawHeight, addr1, isReIssuable, maxSupply, ownerBurnOnly)
	if receiveBurnBlock == nil ||
		len(receiveBurnBlock.AccountBlock.SendBlockList) != 0 || isRetry || err != nil ||
		!bytes.Equal(db.storageMap[addr2][ToKey(key)], tokenInfoData) ||
		db.balanceMap[addr2][ledger.ViteTokenId].Cmp(pledgeAmount) != 0 ||
		db.balanceMap[addr2][tokenId].Cmp(helper.Big0) != 0 ||
		len(receiveBurnBlock.AccountBlock.Data) != 33 ||
		receiveBurnBlock.AccountBlock.Data[32] != byte(0) ||
		receiveBurnBlock.AccountBlock.Quota != 0 ||
		len(db.logList) != 3 ||
		db.logList[2].Topics[0] != abi.ABIMintage.Events[abi.EventNameBurn].Id() ||
		!bytes.Equal(db.logList[2].Topics[1].Bytes(), helper.LeftPadBytes(tokenId.Bytes(), 32)) ||
		!bytes.Equal(db.logList[2].Data, append(helper.LeftPadBytes(addr1.Bytes(), 32), helper.LeftPadBytes(burnAmount.Bytes(), 32)...)) {
		t.Fatalf("receive burn transaction error")
	}
	db.accountBlockMap[addr2] = make(map[types.Hash]*ledger.AccountBlock)
	db.accountBlockMap[addr2][hash25] = receiveBurnBlock.AccountBlock

	// transfer owner
	block17Data, err := abi.ABIMintage.PackMethod(abi.MethodNameTransferOwner, tokenId, addr3)
	hash17 := types.DataHash([]byte{1, 7})
	block17 := &ledger.AccountBlock{
		Height:         7,
		ToAddress:      addr2,
		AccountAddress: addr1,
		Amount:         big.NewInt(0),
		TokenId:        ledger.ViteTokenId,
		BlockType:      ledger.BlockTypeSendCall,
		Fee:            big.NewInt(0),
		PrevHash:       hash16,
		Data:           block17Data,
		Hash:           hash17,
	}
	vm = NewVM()
	db.addr = addr1
	sendTransferOwnerBlock, isRetry, err := vm.RunV2(db, block17, nil, nil)
	if sendTransferOwnerBlock == nil ||
		len(sendTransferOwnerBlock.AccountBlock.SendBlockList) != 0 || isRetry || err != nil ||
		db.balanceMap[addr1][ledger.ViteTokenId].Cmp(balance1) != 0 ||
		db.balanceMap[addr1][tokenId].Cmp(tokenBalance) != 0 ||
		!bytes.Equal(sendTransferOwnerBlock.AccountBlock.Data, block17Data) ||
		sendTransferOwnerBlock.AccountBlock.Amount.Cmp(helper.Big0) != 0 ||
		sendTransferOwnerBlock.AccountBlock.Quota != contracts.TransferOwnerGas {
		t.Fatalf("send transfer owner transaction error")
	}
	db.accountBlockMap[addr1][hash17] = sendTransferOwnerBlock.AccountBlock

	hash26 := types.DataHash([]byte{2, 6})
	block26 := &ledger.AccountBlock{
		Height:         6,
		AccountAddress: addr2,
		BlockType:      ledger.BlockTypeReceive,
		FromBlockHash:  hash17,
		Hash:           hash26,
		PrevHash:       hash25,
	}
	vm = NewVM()
	db.addr = addr2
	receiveTransferOwnerBlock, isRetry, err := vm.RunV2(db, block26, sendTransferOwnerBlock.AccountBlock, NewTestGlobalStatus(0, snapshot2))
	tokenInfoData, _ = abi.ABIMintage.PackVariable(abi.VariableNameTokenInfo, tokenName, tokenSymbol, totalSupply, decimals, addr3, pledgeAmount, withdrawHeight, addr1, isReIssuable, maxSupply, ownerBurnOnly)
	if receiveTransferOwnerBlock == nil ||
		len(receiveTransferOwnerBlock.AccountBlock.SendBlockList) != 0 || isRetry || err != nil ||
		!bytes.Equal(db.storageMap[addr2][ToKey(key)], tokenInfoData) ||
		db.balanceMap[addr2][ledger.ViteTokenId].Cmp(pledgeAmount) != 0 ||
		db.balanceMap[addr2][tokenId].Cmp(helper.Big0) != 0 ||
		len(receiveTransferOwnerBlock.AccountBlock.Data) != 33 ||
		receiveTransferOwnerBlock.AccountBlock.Data[32] != byte(0) ||
		receiveTransferOwnerBlock.AccountBlock.Quota != 0 ||
		len(db.logList) != 4 ||
		db.logList[3].Topics[0] != abi.ABIMintage.Events[abi.EventNameTransferOwner].Id() ||
		!bytes.Equal(db.logList[3].Topics[1].Bytes(), helper.LeftPadBytes(tokenId.Bytes(), 32)) ||
		!bytes.Equal(db.logList[3].Data, helper.LeftPadBytes(addr3.Bytes(), 32)) {
		t.Fatalf("receive transfer owner transaction error")
	}
	db.accountBlockMap[addr2] = make(map[types.Hash]*ledger.AccountBlock)
	db.accountBlockMap[addr2][hash26] = receiveTransferOwnerBlock.AccountBlock

	db.addr = types.AddressMintage
	if tokenMap, _ := abi.GetTokenMapByOwner(db, addr1); len(tokenMap) != 0 {
		t.Fatalf("get token map by owner failed")
	}
	if tokenMap, _ := abi.GetTokenMapByOwner(db, addr3); len(tokenMap) != 1 {
		t.Fatalf("get token map by owner failed")
	}

	// change token type
	block32Data, err := abi.ABIMintage.PackMethod(abi.MethodNameChangeTokenType, tokenId)
	hash32 := types.DataHash([]byte{3, 2})
	block32 := &ledger.AccountBlock{
		Height:         2,
		ToAddress:      addr2,
		AccountAddress: addr3,
		Amount:         big.NewInt(0),
		TokenId:        ledger.ViteTokenId,
		BlockType:      ledger.BlockTypeSendCall,
		Fee:            big.NewInt(0),
		Data:           block32Data,
		Hash:           hash32,
		PrevHash:       hash31,
	}
	vm = NewVM()
	db.addr = addr3
	sendChangeTokenTypeBlock, isRetry, err := vm.RunV2(db, block32, nil, nil)
	if sendChangeTokenTypeBlock == nil ||
		len(sendChangeTokenTypeBlock.AccountBlock.SendBlockList) != 0 || isRetry || err != nil ||
		!bytes.Equal(sendChangeTokenTypeBlock.AccountBlock.Data, block32Data) ||
		sendChangeTokenTypeBlock.AccountBlock.Amount.Cmp(helper.Big0) != 0 ||
		sendChangeTokenTypeBlock.AccountBlock.Quota != contracts.ChangeTokenTypeGas {
		t.Fatalf("send change token type transaction error")
	}
	db.accountBlockMap[addr3][hash32] = sendChangeTokenTypeBlock.AccountBlock

	hash27 := types.DataHash([]byte{2, 7})
	block27 := &ledger.AccountBlock{
		Height:         7,
		AccountAddress: addr2,
		BlockType:      ledger.BlockTypeReceive,
		FromBlockHash:  hash31,
		Hash:           hash27,
		PrevHash:       hash26,
	}
	vm = NewVM()
	db.addr = addr2
	receiveChangeTokenTypeBlock, isRetry, err := vm.RunV2(db, block27, sendChangeTokenTypeBlock.AccountBlock, NewTestGlobalStatus(0, snapshot2))
	tokenInfoData, _ = abi.ABIMintage.PackVariable(abi.VariableNameTokenInfo, tokenName, tokenSymbol, totalSupply, decimals, addr3, pledgeAmount, withdrawHeight, addr1, false, big.NewInt(0), false)
	if receiveChangeTokenTypeBlock == nil ||
		len(receiveChangeTokenTypeBlock.AccountBlock.SendBlockList) != 0 || isRetry || err != nil ||
		!bytes.Equal(db.storageMap[addr2][ToKey(key)], tokenInfoData) ||
		db.balanceMap[addr2][ledger.ViteTokenId].Cmp(pledgeAmount) != 0 ||
		db.balanceMap[addr2][tokenId].Cmp(helper.Big0) != 0 ||
		len(receiveChangeTokenTypeBlock.AccountBlock.Data) != 33 ||
		receiveChangeTokenTypeBlock.AccountBlock.Data[32] != byte(0) ||
		receiveChangeTokenTypeBlock.AccountBlock.Quota != 0 ||
		len(db.logList) != 5 ||
		db.logList[4].Topics[0] != abi.ABIMintage.Events[abi.EventNameChangeTokenType].Id() ||
		!bytes.Equal(db.logList[4].Topics[1].Bytes(), helper.LeftPadBytes(tokenId.Bytes(), 32)) {
		t.Fatalf("receive change token type transaction error")
	}
	db.accountBlockMap[addr2] = make(map[types.Hash]*ledger.AccountBlock)
	db.accountBlockMap[addr2][hash27] = receiveChangeTokenTypeBlock.AccountBlock

	db.addr = types.AddressMintage
	if tokenMap, _ := abi.GetTokenMapByOwner(db, addr3); len(tokenMap) != 1 {
		t.Fatalf("get token map by owner failed")
	}
	if tokenMap, _ := abi.GetTokenMapByOwner(db, addr1); len(tokenMap) != 0 {
		t.Fatalf("get token map by owner failed")
	}

	sbtime := time.Now()
	var latestSb *ledger.SnapshotBlock
	for i := uint64(snapshot2.Height + 1); i <= withdrawHeight; i++ {
		latestSb = &ledger.SnapshotBlock{Height: i, Timestamp: &sbtime, Hash: types.DataHash([]byte{10, byte(i)})}
		db.snapshotBlockList = append(db.snapshotBlockList, latestSb)
	}

	// cancel pledge
	block18Data, err := abi.ABIMintage.PackMethod(abi.MethodNameCancelMintPledge, tokenId)
	hash18 := types.DataHash([]byte{1, 8})
	block18 := &ledger.AccountBlock{
		Height:         8,
		ToAddress:      addr2,
		AccountAddress: addr1,
		Amount:         big.NewInt(0),
		TokenId:        ledger.ViteTokenId,
		BlockType:      ledger.BlockTypeSendCall,
		Fee:            big.NewInt(0),
		Data:           block18Data,
		Hash:           hash18,
		PrevHash:       hash17,
	}
	vm = NewVM()
	db.addr = addr1
	cancelPledgeBlock, isRetry, err := vm.RunV2(db, block18, nil, nil)
	if cancelPledgeBlock == nil ||
		len(cancelPledgeBlock.AccountBlock.SendBlockList) != 0 || isRetry || err != nil ||
		!bytes.Equal(cancelPledgeBlock.AccountBlock.Data, block18Data) ||
		cancelPledgeBlock.AccountBlock.Amount.Cmp(helper.Big0) != 0 ||
		cancelPledgeBlock.AccountBlock.Quota != contracts.MintageCancelPledgeGas {
		t.Fatalf("send cancel pledge transaction error")
	}
	db.accountBlockMap[addr3][hash18] = cancelPledgeBlock.AccountBlock

	hash28 := types.DataHash([]byte{2, 8})
	block28 := &ledger.AccountBlock{
		Height:         8,
		AccountAddress: addr2,
		BlockType:      ledger.BlockTypeReceive,
		FromBlockHash:  hash18,
		Hash:           hash28,
		PrevHash:       hash27,
	}
	vm = NewVM()
	db.addr = addr2
	receiveCancelPledgeBlock, isRetry, err := vm.RunV2(db, block28, cancelPledgeBlock.AccountBlock, NewTestGlobalStatus(0, latestSb))
	tokenInfoData, _ = abi.ABIMintage.PackVariable(abi.VariableNameTokenInfo, tokenName, tokenSymbol, totalSupply, decimals, addr3, helper.Big0, uint64(0), addr1, false, big.NewInt(0), false)
	if receiveCancelPledgeBlock == nil ||
		len(receiveCancelPledgeBlock.AccountBlock.SendBlockList) != 1 || isRetry || err != nil ||
		!bytes.Equal(db.storageMap[addr2][ToKey(key)], tokenInfoData) ||
		db.balanceMap[addr2][ledger.ViteTokenId].Cmp(helper.Big0) != 0 ||
		db.balanceMap[addr2][tokenId].Cmp(helper.Big0) != 0 ||
		len(receiveCancelPledgeBlock.AccountBlock.Data) != 33 ||
		receiveCancelPledgeBlock.AccountBlock.Data[32] != byte(0) ||
		receiveCancelPledgeBlock.AccountBlock.Quota != 0 ||
		len(db.logList) != 5 ||
		receiveCancelPledgeBlock.AccountBlock.SendBlockList[0].Amount.Cmp(pledgeAmount) != 0 ||
		receiveCancelPledgeBlock.AccountBlock.SendBlockList[0].ToAddress != addr1 ||
		receiveCancelPledgeBlock.AccountBlock.SendBlockList[0].BlockType != ledger.BlockTypeSendCall {
		t.Fatalf("receive cancel pledge transaction error")
	}
	db.accountBlockMap[addr2] = make(map[types.Hash]*ledger.AccountBlock)
	db.accountBlockMap[addr2][hash28] = receiveCancelPledgeBlock.AccountBlock
}

func TestCheckTokenName(t *testing.T) {
	tests := []struct {
		data string
		exp  bool
	}{
		{"", false},
		{" ", false},
		{"a", true},
		{"ab", true},
		{"ab ", false},
		{"a b", true},
		{"a  b", false},
		{"a _b", true},
		{"_a", true},
		{"_a b c", true},
		{"_a bb c", true},
		{"_a bb cc", true},
		{"_a bb  cc", false},
	}
	for _, test := range tests {
		if ok, _ := regexp.MatchString("^([0-9a-zA-Z_]+[ ]?)*[0-9a-zA-Z_]$", test.data); ok != test.exp {
			t.Fatalf("match string error, [%v] expected %v, got %v", test.data, test.exp, ok)
		}
	}
}

func TestGenesisBlockData(t *testing.T) {
	tokenName := "ViteToken"
	tokenSymbol := "ViteToken"
	decimals := uint8(18)
	totalSupply := new(big.Int).Mul(big.NewInt(1e18), big.NewInt(1e9))
	viteAddress, _, _ := types.CreateAddress()
	mintageData, err := abi.ABIMintage.PackVariable(abi.VariableNameTokenInfo, tokenName, tokenSymbol, totalSupply, decimals, viteAddress, big.NewInt(0), uint64(0), viteAddress, true, helper.Tt256m1, false)
	if err != nil {
		t.Fatalf("pack mintage variable error, %v", err)
	}
	fmt.Println("-------------mintage genesis block-------------")
	fmt.Printf("address: %v\n", hex.EncodeToString(types.AddressMintage.Bytes()))
	fmt.Printf("AccountBlock{\n\tBlockType: %v\n\tAccountAddress: %v,\n\tHeight: %v,\n\tAmount: %v,\n\tTokenId:ledger.ViteTokenId,\n\tQuota:0,\n\tFee:%v\n}\n",
		ledger.BlockTypeReceive, hex.EncodeToString(types.AddressMintage.Bytes()), 1, big.NewInt(0), big.NewInt(0))
	fmt.Printf("Storage:{\n\t%v:%v\n}\n", hex.EncodeToString(abi.GetMintageKey(ledger.ViteTokenId)), hex.EncodeToString(mintageData))

	fmt.Println("-------------vite owner genesis block-------------")
	fmt.Println("address: viteAddress")
	fmt.Printf("AccountBlock{\n\tBlockType: %v,\n\tAccountAddress: viteAddress,\n\tHeight: %v,\n\tAmount: %v,\n\tTokenId:ledger.ViteTokenId,\n\tQuota:0,\n\tFee:%v,\n\tData:%v,\n}\n",
		ledger.BlockTypeReceive, 1, totalSupply, big.NewInt(0), []byte{})
	fmt.Printf("Storage:{\n\t$balance:ledger.ViteTokenId:%v\n}\n", totalSupply)

	conditionRegisterData, err := abi.ABIConsensusGroup.PackVariable(abi.VariableNameConditionRegisterOfPledge, new(big.Int).Mul(big.NewInt(1e5), util.AttovPerVite), ledger.ViteTokenId, uint64(3600*24*90))
	if err != nil {
		t.Fatalf("pack register condition variable error, %v", err)
	}
	snapshotConsensusGroupData, err := abi.ABIConsensusGroup.PackVariable(abi.VariableNameConsensusGroupInfo,
		uint8(25),
		int64(1),
		int64(3),
		uint8(2),
		uint8(50),
		uint16(1),
		uint8(0),
		ledger.ViteTokenId,
		uint8(1),
		conditionRegisterData,
		uint8(1),
		[]byte{},
		viteAddress,
		big.NewInt(0),
		uint64(1))
	if err != nil {
		t.Fatalf("pack consensus group data variable error, %v", err)
	}
	commonConsensusGroupData, err := abi.ABIConsensusGroup.PackVariable(abi.VariableNameConsensusGroupInfo,
		uint8(25),
		int64(3),
		int64(1),
		uint8(2),
		uint8(50),
		uint16(48),
		uint8(1),
		ledger.ViteTokenId,
		uint8(1),
		conditionRegisterData,
		uint8(1),
		[]byte{},
		viteAddress,
		big.NewInt(0),
		uint64(1))
	if err != nil {
		t.Fatalf("pack consensus group data variable error, %v", err)
	}
	fmt.Println("-------------snapshot consensus group and common consensus group genesis block-------------")
	fmt.Printf("address:%v\n", hex.EncodeToString(types.AddressConsensusGroup.Bytes()))
	fmt.Printf("AccountBlock{\n\tBlockType: %v,\n\tAccountAddress: %v,\n\tHeight: %v,\n\tAmount: %v,\n\tTokenId:ledger.ViteTokenId,\n\tQuota:0,\n\tFee:%v,\n\tData:%v,\n}\n",
		ledger.BlockTypeReceive, hex.EncodeToString(types.AddressConsensusGroup.Bytes()), 1, big.NewInt(0), big.NewInt(0), []byte{})
	fmt.Printf("Storage:{\n\t%v:%v,\n\t%v:%v}\n", hex.EncodeToString(abi.GetConsensusGroupKey(types.SNAPSHOT_GID)), hex.EncodeToString(snapshotConsensusGroupData), hex.EncodeToString(abi.GetConsensusGroupKey(types.DELEGATE_GID)), hex.EncodeToString(commonConsensusGroupData))

	fmt.Println("-------------snapshot consensus group and common consensus group register genesis block-------------")
	fmt.Printf("address:%v\n", hex.EncodeToString(types.AddressConsensusGroup.Bytes()))
	fmt.Printf("AccountBlock{\n\tBlockType: %v,\n\tAccountAddress: %v,\n\tHeight: %v,\n\tAmount: %v,\n\tTokenId:ledger.ViteTokenId,\n\tQuota:0,\n\tFee:%v,\n\tData:%v,\n}\n",
		ledger.BlockTypeReceive, hex.EncodeToString(types.AddressConsensusGroup.Bytes()), 1, big.NewInt(0), big.NewInt(0), []byte{})
	fmt.Printf("Storage:{\n")
	for i := 1; i <= 25; i++ {
		addr, _, _ := types.CreateAddress()
		registerData, err := abi.ABIConsensusGroup.PackVariable(abi.VariableNameRegistration, "node"+strconv.Itoa(i), addr, addr, helper.Big0, uint64(1), int64(1), int64(0), []types.Address{addr})
		if err != nil {
			t.Fatalf("pack registration variable error, %v", err)
		}
		snapshotKey := abi.GetRegisterKey("snapshotNode1", types.SNAPSHOT_GID)
		fmt.Printf("\t%v: %v\n", hex.EncodeToString(snapshotKey), hex.EncodeToString(registerData))
	}
	fmt.Println("}")
}
