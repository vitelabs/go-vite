package vm

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"math/big"
	"regexp"
	"testing"
	"time"
)

func TestContractsRegisterRun(t *testing.T) {
	// prepare db
	viteTotalSupply := new(big.Int).Mul(big.NewInt(2e6), big.NewInt(1e18))
	db, addr1, hash12, snapshot2, timestamp := prepareDb(viteTotalSupply)
	// register
	balance1 := new(big.Int).Set(viteTotalSupply)
	addr6, _, _ := types.CreateAddress()
	addr7, _, _ := types.CreateAddress()
	db.accountBlockMap[addr6] = make(map[types.Hash]*ledger.AccountBlock)
	db.accountBlockMap[addr7] = make(map[types.Hash]*ledger.AccountBlock)
	addr2 := AddressRegister
	nodeName := "super1"
	block13Data, err := ABI_register.PackMethod(MethodNameRegister, *ledger.CommonGid(), nodeName, addr7, addr6)
	hash13 := types.DataHash([]byte{1, 3})
	block13 := &ledger.AccountBlock{
		Height:         3,
		ToAddress:      addr2,
		AccountAddress: addr1,
		BlockType:      ledger.BlockTypeSendCall,
		PrevHash:       hash12,
		Amount:         new(big.Int).Mul(big.NewInt(1e6), big.NewInt(1e18)),
		Data:           block13Data,
		TokenId:        *ledger.ViteTokenId(),
		SnapshotHash:   snapshot2.Hash,
	}
	vm := NewVM(db)
	vm.Debug = true
	db.addr = addr1
	block13DataGas, _ := dataGasCost(block13Data)
	sendRegisterBlockList, isRetry, err := vm.Run(block13, nil)
	balance1.Sub(balance1, block13.Amount)
	if len(sendRegisterBlockList) != 1 || isRetry || err != nil ||
		sendRegisterBlockList[0].Quota != block13DataGas+registerGas ||
		db.balanceMap[addr1][*ledger.ViteTokenId()].Cmp(balance1) != 0 {
		t.Fatalf("send register transaction error")
	}
	db.accountBlockMap[addr1][hash13] = sendRegisterBlockList[0]

	hash21 := types.DataHash([]byte{2, 1})
	block21 := &ledger.AccountBlock{
		Height:         1,
		AccountAddress: addr2,
		BlockType:      ledger.BlockTypeReceive,
		FromBlockHash:  hash13,
		SnapshotHash:   snapshot2.Hash,
	}
	vm = NewVM(db)
	vm.Debug = true
	locHashRegister, _ := types.BytesToHash(getRegisterKey(nodeName, *ledger.CommonGid()))
	registrationData, _ := ABI_register.PackVariable(VariableNameRegistration, nodeName, addr7, addr1, addr6, block13.Amount, snapshot2.Timestamp.Unix(), snapshot2.Height, uint64(0))
	db.addr = addr2
	updateReveiceBlockBySendBlock(block21, sendRegisterBlockList[0])
	receiveRegisterBlockList, isRetry, err := vm.Run(block21, sendRegisterBlockList[0])
	if len(receiveRegisterBlockList) != 1 || isRetry || err != nil ||
		db.balanceMap[addr1][*ledger.ViteTokenId()].Cmp(balance1) != 0 ||
		!bytes.Equal(db.storageMap[addr2][locHashRegister], registrationData) ||
		receiveRegisterBlockList[0].Quota != 0 {
		t.Fatalf("receive register transaction error")
	}
	db.accountBlockMap[addr2] = make(map[types.Hash]*ledger.AccountBlock)
	db.accountBlockMap[addr2][hash21] = receiveRegisterBlockList[0]

	// update registration
	block14Data, err := ABI_register.PackMethod(MethodNameUpdateRegistration, *ledger.CommonGid(), nodeName, addr6, addr7)
	hash14 := types.DataHash([]byte{1, 4})
	block14 := &ledger.AccountBlock{
		Height:         4,
		ToAddress:      addr2,
		AccountAddress: addr1,
		BlockType:      ledger.BlockTypeSendCall,
		PrevHash:       hash13,
		Data:           block14Data,
		Amount:         big.NewInt(0),
		TokenId:        *ledger.ViteTokenId(),
		SnapshotHash:   snapshot2.Hash,
	}
	vm = NewVM(db)
	vm.Debug = true
	db.addr = addr1
	block14DataGas, _ := dataGasCost(block14Data)
	sendRegisterBlockList2, isRetry, err := vm.Run(block14, nil)
	if len(sendRegisterBlockList2) != 1 || isRetry || err != nil ||
		sendRegisterBlockList2[0].Quota != block14DataGas+updateRegistrationGas ||
		db.balanceMap[addr1][*ledger.ViteTokenId()].Cmp(balance1) != 0 {
		t.Fatalf("send update registration transaction error")
	}
	db.accountBlockMap[addr1][hash14] = sendRegisterBlockList2[0]

	hash22 := types.DataHash([]byte{2, 2})
	block22 := &ledger.AccountBlock{
		Height:         2,
		AccountAddress: addr2,
		BlockType:      ledger.BlockTypeReceive,
		FromBlockHash:  hash14,
		PrevHash:       hash21,
		SnapshotHash:   snapshot2.Hash,
	}
	vm = NewVM(db)
	vm.Debug = true
	registrationData, _ = ABI_register.PackVariable(VariableNameRegistration, nodeName, addr6, addr1, addr7, block13.Amount, snapshot2.Timestamp.Unix(), snapshot2.Height, uint64(0))
	db.addr = addr2
	updateReveiceBlockBySendBlock(block22, sendRegisterBlockList2[0])
	receiveRegisterBlockList2, isRetry, err := vm.Run(block22, sendRegisterBlockList2[0])
	if len(receiveRegisterBlockList2) != 1 || isRetry || err != nil ||
		db.balanceMap[addr1][*ledger.ViteTokenId()].Cmp(balance1) != 0 ||
		!bytes.Equal(db.storageMap[addr2][locHashRegister], registrationData) ||
		receiveRegisterBlockList2[0].Quota != 0 {
		t.Fatalf("receive update registration transaction error")
	}
	db.accountBlockMap[addr2][hash22] = receiveRegisterBlockList2[0]

	// cancel register
	time3 := time.Unix(timestamp+1, 0)
	snapshot3 := &ledger.SnapshotBlock{Height: 3, Timestamp: &time3, Hash: types.DataHash([]byte{10, 3}), Producer: addr7}
	db.snapshotBlockList = append(db.snapshotBlockList, snapshot3)
	time4 := time.Unix(timestamp+2, 0)
	snapshot4 := &ledger.SnapshotBlock{Height: 4, Timestamp: &time4, Hash: types.DataHash([]byte{10, 4}), Producer: addr7}
	db.snapshotBlockList = append(db.snapshotBlockList, snapshot4)

	hash15 := types.DataHash([]byte{1, 5})
	block15Data, _ := ABI_register.PackMethod(MethodNameCancelRegister, *ledger.CommonGid(), nodeName)
	block15 := &ledger.AccountBlock{
		Height:         5,
		ToAddress:      addr2,
		AccountAddress: addr1,
		Amount:         big.NewInt(0),
		TokenId:        *ledger.ViteTokenId(),
		BlockType:      ledger.BlockTypeSendCall,
		PrevHash:       hash13,
		Data:           block15Data,
		SnapshotHash:   snapshot4.Hash,
	}
	vm = NewVM(db)
	vm.Debug = true
	db.addr = addr1
	block15DataGas, _ := dataGasCost(block15Data)
	sendCancelRegisterBlockList, isRetry, err := vm.Run(block15, nil)
	if len(sendCancelRegisterBlockList) != 1 || isRetry || err != nil ||
		sendCancelRegisterBlockList[0].Quota != block15DataGas+cancelRegisterGas ||
		db.balanceMap[addr1][*ledger.ViteTokenId()].Cmp(balance1) != 0 {
		t.Fatalf("send cancel register transaction error")
	}
	db.accountBlockMap[addr1][hash15] = sendCancelRegisterBlockList[0]

	hash23 := types.DataHash([]byte{2, 3})
	block23 := &ledger.AccountBlock{
		Height:         3,
		AccountAddress: addr2,
		BlockType:      ledger.BlockTypeReceive,
		PrevHash:       hash21,
		FromBlockHash:  hash15,
		SnapshotHash:   snapshot4.Hash,
	}
	vm = NewVM(db)
	vm.Debug = true
	db.addr = addr2
	updateReveiceBlockBySendBlock(block23, block15)
	receiveCancelRegisterBlockList, isRetry, err := vm.Run(block23, block15)
	registrationData, _ = ABI_register.PackVariable(VariableNameRegistration, nodeName, addr6, addr1, addr7, helper.Big0, int64(0), snapshot2.Height, snapshot4.Height)
	if len(receiveCancelRegisterBlockList) != 2 || isRetry || err != nil ||
		db.balanceMap[addr2][*ledger.ViteTokenId()].Cmp(helper.Big0) != 0 ||
		db.balanceMap[addr1][*ledger.ViteTokenId()].Cmp(balance1) != 0 ||
		!bytes.Equal(db.storageMap[addr2][locHashRegister], registrationData) ||
		receiveCancelRegisterBlockList[0].Quota != 0 ||
		receiveCancelRegisterBlockList[1].Quota != 0 ||
		receiveCancelRegisterBlockList[1].Height != 4 ||
		!bytes.Equal(receiveCancelRegisterBlockList[1].AccountAddress.Bytes(), addr2.Bytes()) ||
		!bytes.Equal(receiveCancelRegisterBlockList[1].ToAddress.Bytes(), addr1.Bytes()) ||
		receiveCancelRegisterBlockList[1].BlockType != ledger.BlockTypeSendCall {
		t.Fatalf("receive cancel register transaction error")
	}
	db.accountBlockMap[addr2][hash23] = receiveCancelRegisterBlockList[0]
	hash24 := types.DataHash([]byte{2, 4})
	db.accountBlockMap[addr2][hash24] = receiveCancelRegisterBlockList[1]

	hash16 := types.DataHash([]byte{1, 5})
	block16 := &ledger.AccountBlock{
		Height:         6,
		AccountAddress: addr1,
		BlockType:      ledger.BlockTypeReceive,
		PrevHash:       hash16,
		FromBlockHash:  hash23,
		SnapshotHash:   snapshot4.Hash,
	}
	vm = NewVM(db)
	vm.Debug = true
	db.addr = addr1
	balance1.Add(balance1, block13.Amount)
	updateReveiceBlockBySendBlock(block16, receiveCancelRegisterBlockList[1])
	receiveCancelRegisterRefundBlockList, isRetry, err := vm.Run(block16, receiveCancelRegisterBlockList[1])
	if len(receiveCancelRegisterRefundBlockList) != 1 || isRetry || err != nil ||
		db.balanceMap[addr2][*ledger.ViteTokenId()].Cmp(helper.Big0) != 0 ||
		db.balanceMap[addr1][*ledger.ViteTokenId()].Cmp(balance1) != 0 ||
		receiveCancelRegisterRefundBlockList[0].Quota != 21000 {
		t.Fatalf("receive cancel register refund transaction error")
	}
	db.accountBlockMap[addr1][hash16] = receiveCancelRegisterRefundBlockList[0]

	// reward
	for i := uint64(1); i <= 50; i++ {
		timei := time.Unix(timestamp+2+int64(i), 0)
		snapshoti := &ledger.SnapshotBlock{Height: 4 + i, Timestamp: &timei, Hash: types.DataHash([]byte{10, byte(4 + i)}), Producer: addr1}
		db.snapshotBlockList = append(db.snapshotBlockList, snapshoti)
	}
	snapshot54 := db.snapshotBlockList[53]
	db.storageMap[AddressPledge][types.DataHash(addr7.Bytes())], _ = ABI_pledge.PackVariable(VariableNamePledgeBeneficial, big.NewInt(1e18))
	block71Data, _ := ABI_register.PackMethod(MethodNameReward, *ledger.CommonGid(), nodeName, uint64(0), uint64(0), common.Big0)
	hash71 := types.DataHash([]byte{7, 1})
	block71 := &ledger.AccountBlock{
		Height:         1,
		ToAddress:      addr2,
		AccountAddress: addr7,
		Amount:         big.NewInt(0),
		TokenId:        *ledger.ViteTokenId(),
		BlockType:      ledger.BlockTypeSendCall,
		PrevHash:       hash16,
		Data:           block71Data,
		SnapshotHash:   snapshot54.Hash,
	}
	vm = NewVM(db)
	vm.Debug = true
	db.addr = addr7
	sendRewardBlockList, isRetry, err := vm.Run(block71, nil)
	block71DataGas, _ := dataGasCost(sendRewardBlockList[0].Data)
	reward := new(big.Int).Mul(big.NewInt(2), rewardPerBlock)
	block71DataExpected, _ := ABI_register.PackMethod(MethodNameReward, *ledger.CommonGid(), nodeName, snapshot4.Height, snapshot2.Height, reward)
	if len(sendRewardBlockList) != 1 || isRetry || err != nil ||
		sendRewardBlockList[0].Quota != block71DataGas+rewardGas+calcRewardGasPerPage ||
		!bytes.Equal(sendRewardBlockList[0].Data, block71DataExpected) {
		t.Fatalf("send reward transaction error")
	}
	db.accountBlockMap[addr7][hash71] = sendRewardBlockList[0]

	hash25 := types.DataHash([]byte{2, 5})
	block25 := &ledger.AccountBlock{
		Height:         5,
		AccountAddress: addr2,
		BlockType:      ledger.BlockTypeReceive,
		PrevHash:       hash23,
		FromBlockHash:  hash71,
		SnapshotHash:   snapshot54.Hash,
	}
	vm = NewVM(db)
	vm.Debug = true
	db.addr = addr2
	updateReveiceBlockBySendBlock(block25, block71)
	receiveRewardBlockList, isRetry, err := vm.Run(block25, block71)
	if len(receiveRewardBlockList) != 2 || isRetry || err != nil ||
		db.balanceMap[addr2][*ledger.ViteTokenId()].Cmp(helper.Big0) != 0 ||
		db.balanceMap[addr1][*ledger.ViteTokenId()].Cmp(viteTotalSupply) != 0 ||
		len(db.storageMap[addr2][locHashRegister]) != 0 ||
		receiveRewardBlockList[0].Quota != 0 ||
		receiveRewardBlockList[1].Quota != 0 ||
		receiveRewardBlockList[1].Height != 6 ||
		!bytes.Equal(receiveRewardBlockList[1].AccountAddress.Bytes(), addr2.Bytes()) ||
		!bytes.Equal(receiveRewardBlockList[1].ToAddress.Bytes(), addr7.Bytes()) ||
		receiveRewardBlockList[1].BlockType != ledger.BlockTypeSendReward {
		t.Fatalf("receive reward transaction error")
	}
	db.accountBlockMap[addr2][hash25] = receiveRewardBlockList[0]
	hash26 := types.DataHash([]byte{2, 6})
	db.accountBlockMap[addr2][hash26] = receiveRewardBlockList[1]

	hash72 := types.DataHash([]byte{7, 2})
	block72 := &ledger.AccountBlock{
		Height:         2,
		AccountAddress: addr7,
		BlockType:      ledger.BlockTypeReceive,
		PrevHash:       hash71,
		FromBlockHash:  hash25,
		SnapshotHash:   snapshot54.Hash,
	}
	vm = NewVM(db)
	vm.Debug = true
	db.addr = addr7
	updateReveiceBlockBySendBlock(block72, receiveRewardBlockList[1])
	receiveRewardRefundBlockList, isRetry, err := vm.Run(block72, receiveRewardBlockList[1])
	if len(receiveRewardRefundBlockList) != 1 || isRetry || err != nil ||
		db.balanceMap[addr2][*ledger.ViteTokenId()].Cmp(helper.Big0) != 0 ||
		db.balanceMap[addr7][*ledger.ViteTokenId()].Cmp(reward) != 0 ||
		receiveRewardRefundBlockList[0].Quota != 21000 {
		t.Fatalf("receive reward refund transaction error")
	}
	db.accountBlockMap[addr7][hash72] = receiveRewardRefundBlockList[0]
}

func TestContractsVote(t *testing.T) {
	// prepare db
	viteTotalSupply := new(big.Int).Mul(big.NewInt(2e6), big.NewInt(1e18))
	db, addr1, hash12, snapshot2, _ := prepareDb(viteTotalSupply)
	// vote
	addr3 := AddressVote
	nodeName := "super1"
	block13Data, _ := ABI_vote.PackMethod(MethodNameVote, *ledger.CommonGid(), nodeName)
	hash13 := types.DataHash([]byte{1, 6})
	block13 := &ledger.AccountBlock{
		Height:         3,
		ToAddress:      addr3,
		AccountAddress: addr1,
		PrevHash:       hash12,
		Amount:         big.NewInt(0),
		TokenId:        *ledger.ViteTokenId(),
		BlockType:      ledger.BlockTypeSendCall,
		Data:           block13Data,
		SnapshotHash:   snapshot2.Hash,
	}
	vm := NewVM(db)
	vm.Debug = true
	db.addr = addr1
	block13DataGas, _ := dataGasCost(block13.Data)
	sendVoteBlockList, isRetry, err := vm.Run(block13, nil)
	if len(sendVoteBlockList) != 1 || isRetry || err != nil ||
		sendVoteBlockList[0].Quota != block13DataGas+voteGas {
		t.Fatalf("send vote transaction error")
	}
	db.accountBlockMap[addr1][hash13] = sendVoteBlockList[0]

	hash31 := types.DataHash([]byte{3, 1})
	block31 := &ledger.AccountBlock{
		Height:         1,
		AccountAddress: addr3,
		BlockType:      ledger.BlockTypeReceive,
		FromBlockHash:  hash13,
		SnapshotHash:   snapshot2.Hash,
	}
	vm = NewVM(db)
	vm.Debug = true
	db.addr = addr3
	updateReveiceBlockBySendBlock(block31, block13)
	receiveVoteBlockList, isRetry, err := vm.Run(block31, block13)
	locHashVote, _ := types.BytesToHash(getVoteKey(addr1, *ledger.CommonGid()))
	voteData, _ := ABI_vote.PackVariable(VariableNameVoteStatus, nodeName)
	if len(receiveVoteBlockList) != 1 || isRetry || err != nil ||
		!bytes.Equal(db.storageMap[addr3][locHashVote], voteData) ||
		receiveVoteBlockList[0].Quota != 0 {
		t.Fatalf("receive vote transaction error")
	}
	db.accountBlockMap[addr3] = make(map[types.Hash]*ledger.AccountBlock)
	db.accountBlockMap[addr3][hash31] = receiveVoteBlockList[0]

	addr4, _ := types.BytesToAddress(helper.HexToBytes("e5bf58cacfb74cf8c49a1d5e59d3919c9a4cb9ed"))
	db.accountBlockMap[addr4] = make(map[types.Hash]*ledger.AccountBlock)
	nodeName2 := "super2"
	block14Data, _ := ABI_vote.PackMethod(MethodNameVote, *ledger.CommonGid(), nodeName2)
	hash14 := types.DataHash([]byte{1, 9})
	block14 := &ledger.AccountBlock{
		Height:         4,
		ToAddress:      addr3,
		AccountAddress: addr1,
		Amount:         big.NewInt(0),
		TokenId:        *ledger.ViteTokenId(),
		BlockType:      ledger.BlockTypeSendCall,
		PrevHash:       hash13,
		Data:           block14Data,
		SnapshotHash:   snapshot2.Hash,
	}
	vm = NewVM(db)
	vm.Debug = true
	db.addr = addr1
	sendVoteBlockList2, isRetry, err := vm.Run(block14, nil)
	block14DataGas, _ := dataGasCost(block14.Data)
	if len(sendVoteBlockList2) != 1 || isRetry || err != nil ||
		sendVoteBlockList2[0].Quota != block14DataGas+voteGas {
		t.Fatalf("send vote transaction 2 error")
	}
	db.accountBlockMap[addr1][hash14] = sendVoteBlockList2[0]

	hash32 := types.DataHash([]byte{3, 2})
	block32 := &ledger.AccountBlock{
		Height:         2,
		AccountAddress: addr3,
		BlockType:      ledger.BlockTypeReceive,
		PrevHash:       hash31,
		FromBlockHash:  hash14,
		SnapshotHash:   snapshot2.Hash,
	}
	vm = NewVM(db)
	vm.Debug = true
	db.addr = addr3
	updateReveiceBlockBySendBlock(block32, block14)
	receiveVoteBlockList2, isRetry, err := vm.Run(block32, block14)
	voteData, _ = ABI_vote.PackVariable(VariableNameVoteStatus, nodeName2)
	if len(receiveVoteBlockList2) != 1 || isRetry || err != nil ||
		!bytes.Equal(db.storageMap[addr3][locHashVote], voteData) ||
		receiveVoteBlockList2[0].Quota != 0 {
		t.Fatalf("receive vote transaction 2 error")
	}
	db.accountBlockMap[addr3][hash32] = receiveVoteBlockList2[0]
	// cancel vote
	block15Data, _ := ABI_vote.PackMethod(MethodNameCancelVote, *ledger.CommonGid())
	hash15 := types.DataHash([]byte{1, 10})
	block15 := &ledger.AccountBlock{
		Height:         5,
		ToAddress:      addr3,
		AccountAddress: addr1,
		Amount:         big.NewInt(0),
		TokenId:        *ledger.ViteTokenId(),
		BlockType:      ledger.BlockTypeSendCall,
		PrevHash:       hash14,
		Data:           block15Data,
		SnapshotHash:   snapshot2.Hash,
	}
	vm = NewVM(db)
	vm.Debug = true
	db.addr = addr1
	sendCancelVoteBlockList, isRetry, err := vm.Run(block15, nil)
	if len(sendCancelVoteBlockList) != 1 || isRetry || err != nil ||
		sendCancelVoteBlockList[0].Quota != 62464 {
		t.Fatalf("send cancel vote transaction error")
	}
	db.accountBlockMap[addr1][hash15] = sendCancelVoteBlockList[0]

	hash33 := types.DataHash([]byte{3, 3})
	block33 := &ledger.AccountBlock{
		Height:         3,
		AccountAddress: addr3,
		BlockType:      ledger.BlockTypeReceive,
		PrevHash:       hash32,
		FromBlockHash:  hash15,
		SnapshotHash:   snapshot2.Hash,
	}
	vm = NewVM(db)
	vm.Debug = true
	db.addr = addr3
	updateReveiceBlockBySendBlock(block33, block15)
	receiveCancelVoteBlockList, isRetry, err := vm.Run(block33, block15)
	if len(receiveCancelVoteBlockList) != 1 || isRetry || err != nil ||
		len(db.storageMap[addr3][locHashVote]) != 0 ||
		receiveCancelVoteBlockList[0].Quota != 0 {
		t.Fatalf("receive cancel vote transaction error")
	}
	db.accountBlockMap[addr3][hash33] = receiveCancelVoteBlockList[0]
}

func TestContractsPledge(t *testing.T) {
	// prepare db
	viteTotalSupply := new(big.Int).Mul(big.NewInt(2e6), big.NewInt(1e18))
	db, addr1, hash12, snapshot2, timestamp := prepareDb(viteTotalSupply)
	// pledge
	balance1 := new(big.Int).Set(viteTotalSupply)
	addr4, _, _ := types.CreateAddress()
	db.accountBlockMap[addr4] = make(map[types.Hash]*ledger.AccountBlock)
	addr5 := AddressPledge
	pledgeAmount := big.NewInt(2e18)
	withdrawTime := timestamp + pledgeTime
	block13Data, err := ABI_pledge.PackMethod(MethodNamePledge, addr4, withdrawTime)
	hash13 := types.DataHash([]byte{1, 11})
	block13 := &ledger.AccountBlock{
		Height:         3,
		ToAddress:      addr5,
		AccountAddress: addr1,
		Amount:         pledgeAmount,
		TokenId:        *ledger.ViteTokenId(),
		BlockType:      ledger.BlockTypeSendCall,
		PrevHash:       hash12,
		Data:           block13Data,
		SnapshotHash:   snapshot2.Hash,
	}
	vm := NewVM(db)
	vm.Debug = true
	db.addr = addr1
	sendPledgeBlockList, isRetry, err := vm.Run(block13, nil)
	block13DataGas, _ := dataGasCost(sendPledgeBlockList[0].Data)
	balance1.Sub(balance1, pledgeAmount)
	if len(sendPledgeBlockList) != 1 || isRetry || err != nil ||
		db.balanceMap[addr1][*ledger.ViteTokenId()].Cmp(balance1) != 0 ||
		sendPledgeBlockList[0].Quota != block13DataGas+pledgeGas {
		t.Fatalf("send pledge transaction error")
	}
	db.accountBlockMap[addr1][hash13] = sendPledgeBlockList[0]

	hash51 := types.DataHash([]byte{5, 1})
	block51 := &ledger.AccountBlock{
		Height:         1,
		AccountAddress: addr5,
		BlockType:      ledger.BlockTypeReceive,
		FromBlockHash:  hash13,
		SnapshotHash:   snapshot2.Hash,
	}
	vm = NewVM(db)
	vm.Debug = true
	db.addr = addr5
	updateReveiceBlockBySendBlock(block51, block13)
	receivePledgeBlockList, isRetry, err := vm.Run(block51, block13)
	locHashQuota := types.DataHash(addr4.Bytes())
	locHashPledge := types.DataHash(append(addr1.Bytes(), locHashQuota.Bytes()...))
	if len(receivePledgeBlockList) != 1 || isRetry || err != nil ||
		!bytes.Equal(db.storageMap[addr5][locHashPledge], helper.JoinBytes(helper.LeftPadBytes(pledgeAmount.Bytes(), helper.WordSize), helper.LeftPadBytes(new(big.Int).SetInt64(withdrawTime).Bytes(), helper.WordSize))) ||
		!bytes.Equal(db.storageMap[addr5][locHashQuota], helper.LeftPadBytes(pledgeAmount.Bytes(), helper.WordSize)) ||
		db.balanceMap[addr5][*ledger.ViteTokenId()].Cmp(pledgeAmount) != 0 ||
		receivePledgeBlockList[0].Quota != 0 {
		t.Fatalf("receive pledge transaction error")
	}
	db.accountBlockMap[addr5] = make(map[types.Hash]*ledger.AccountBlock)
	db.accountBlockMap[addr5][hash51] = receivePledgeBlockList[0]

	withdrawTime = timestamp + 100 + pledgeTime
	block14Data, _ := ABI_pledge.PackMethod(MethodNamePledge, addr4, withdrawTime)
	hash14 := types.DataHash([]byte{1, 12})
	block14 := &ledger.AccountBlock{
		Height:         4,
		ToAddress:      addr5,
		AccountAddress: addr1,
		Amount:         pledgeAmount,
		TokenId:        *ledger.ViteTokenId(),
		BlockType:      ledger.BlockTypeSendCall,
		PrevHash:       hash13,
		Data:           block14Data,
		SnapshotHash:   snapshot2.Hash,
	}
	vm = NewVM(db)
	vm.Debug = true
	db.addr = addr1
	sendPledgeBlockList2, isRetry, err := vm.Run(block14, nil)
	balance1.Sub(balance1, pledgeAmount)
	if len(sendPledgeBlockList2) != 1 || isRetry || err != nil ||
		db.balanceMap[addr1][*ledger.ViteTokenId()].Cmp(balance1) != 0 ||
		sendPledgeBlockList2[0].Quota != 84464 {
		t.Fatalf("send pledge transaction 2 error")
	}
	db.accountBlockMap[addr1][hash14] = sendPledgeBlockList2[0]

	hash52 := types.DataHash([]byte{5, 2})
	block52 := &ledger.AccountBlock{
		Height:         2,
		AccountAddress: addr5,
		BlockType:      ledger.BlockTypeReceive,
		PrevHash:       hash51,
		FromBlockHash:  hash14,
		SnapshotHash:   snapshot2.Hash,
	}
	vm = NewVM(db)
	vm.Debug = true
	db.addr = addr5
	updateReveiceBlockBySendBlock(block52, block14)
	receivePledgeBlockList2, isRetry, err := vm.Run(block52, block14)
	newPledgeAmount := new(big.Int).Add(pledgeAmount, pledgeAmount)
	if len(receivePledgeBlockList2) != 1 || isRetry || err != nil ||
		!bytes.Equal(db.storageMap[addr5][locHashPledge], helper.JoinBytes(helper.LeftPadBytes(newPledgeAmount.Bytes(), helper.WordSize), helper.LeftPadBytes(new(big.Int).SetInt64(withdrawTime).Bytes(), helper.WordSize))) ||
		!bytes.Equal(db.storageMap[addr5][locHashQuota], helper.LeftPadBytes(newPledgeAmount.Bytes(), helper.WordSize)) ||
		db.balanceMap[addr5][*ledger.ViteTokenId()].Cmp(newPledgeAmount) != 0 ||
		receivePledgeBlockList2[0].Quota != 0 {
		t.Fatalf("receive pledge transaction 2 error")
	}
	db.accountBlockMap[addr5][hash52] = receivePledgeBlockList2[0]

	// cancel pledge
	time55 := time.Unix(timestamp+100+pledgeTime, 0)
	snapshot55 := &ledger.SnapshotBlock{Height: 55, Timestamp: &time55, Hash: types.DataHash([]byte{10, 55}), Producer: addr1}
	db.snapshotBlockList = append(db.snapshotBlockList, snapshot55)

	block15Data, _ := ABI_pledge.PackMethod(MethodNameCancelPledge, addr4, pledgeAmount)
	hash15 := types.DataHash([]byte{1, 13})
	block15 := &ledger.AccountBlock{
		Height:         5,
		ToAddress:      addr5,
		AccountAddress: addr1,
		Amount:         helper.Big0,
		TokenId:        *ledger.ViteTokenId(),
		BlockType:      ledger.BlockTypeSendCall,
		PrevHash:       hash14,
		Data:           block15Data,
		SnapshotHash:   snapshot55.Hash,
	}
	vm = NewVM(db)
	vm.Debug = true
	db.addr = addr1
	sendCancelPledgeBlockList, isRetry, err := vm.Run(block15, nil)
	if len(sendCancelPledgeBlockList) != 1 || isRetry || err != nil ||
		sendCancelPledgeBlockList[0].Quota != 105592 {
		t.Fatalf("send cancel pledge transaction error")
	}
	db.accountBlockMap[addr1][hash15] = sendCancelPledgeBlockList[0]

	hash53 := types.DataHash([]byte{5, 3})
	block53 := &ledger.AccountBlock{
		Height:         3,
		AccountAddress: addr5,
		BlockType:      ledger.BlockTypeReceive,
		PrevHash:       hash52,
		FromBlockHash:  hash15,
		SnapshotHash:   snapshot55.Hash,
	}
	vm = NewVM(db)
	vm.Debug = true
	db.addr = addr5
	updateReveiceBlockBySendBlock(block53, block15)
	receiveCancelPledgeBlockList, isRetry, err := vm.Run(block53, block15)
	if len(receiveCancelPledgeBlockList) != 2 || isRetry || err != nil ||
		receiveCancelPledgeBlockList[1].Height != 4 ||
		!bytes.Equal(db.storageMap[addr5][locHashPledge], helper.JoinBytes(helper.LeftPadBytes(pledgeAmount.Bytes(), helper.WordSize), helper.LeftPadBytes(new(big.Int).SetInt64(withdrawTime).Bytes(), helper.WordSize))) ||
		!bytes.Equal(db.storageMap[addr5][locHashQuota], helper.LeftPadBytes(pledgeAmount.Bytes(), helper.WordSize)) ||
		db.balanceMap[addr5][*ledger.ViteTokenId()].Cmp(pledgeAmount) != 0 ||
		receiveCancelPledgeBlockList[0].Quota != 0 ||
		receiveCancelPledgeBlockList[1].Quota != 0 {
		t.Fatalf("receive cancel pledge transaction error")
	}
	db.accountBlockMap[addr5][hash53] = receiveCancelPledgeBlockList[0]
	hash54 := types.DataHash([]byte{5, 4})
	db.accountBlockMap[addr5][hash54] = receiveCancelPledgeBlockList[1]

	hash16 := types.DataHash([]byte{1, 14})
	block16 := &ledger.AccountBlock{
		Height:         6,
		AccountAddress: addr1,
		BlockType:      ledger.BlockTypeReceive,
		PrevHash:       hash15,
		FromBlockHash:  hash54,
		SnapshotHash:   snapshot55.Hash,
	}
	vm = NewVM(db)
	vm.Debug = true
	db.addr = addr1
	updateReveiceBlockBySendBlock(block16, receiveCancelPledgeBlockList[1])
	receiveCancelPledgeRefundBlockList, isRetry, err := vm.Run(block16, receiveCancelPledgeBlockList[1])
	balance1.Add(balance1, pledgeAmount)
	if len(receiveCancelPledgeRefundBlockList) != 1 || isRetry || err != nil ||
		db.balanceMap[addr1][*ledger.ViteTokenId()].Cmp(balance1) != 0 ||
		receiveCancelPledgeRefundBlockList[0].Quota != 21000 {
		t.Fatalf("receive cancel pledge refund transaction error")
	}
	db.accountBlockMap[addr1][hash16] = receiveCancelPledgeRefundBlockList[0]

	block1dData, _ := ABI_pledge.PackMethod(MethodNameCancelPledge, addr4, pledgeAmount)
	hash1d := types.DataHash([]byte{1, 15})
	block1d := &ledger.AccountBlock{
		Height:         13,
		ToAddress:      addr5,
		AccountAddress: addr1,
		Amount:         helper.Big0,
		TokenId:        *ledger.ViteTokenId(),
		BlockType:      ledger.BlockTypeSendCall,
		PrevHash:       hash16,
		Data:           block1dData,
		SnapshotHash:   snapshot55.Hash,
	}
	vm = NewVM(db)
	vm.Debug = true
	db.addr = addr1
	sendCancelPledgeBlockList2, isRetry, err := vm.Run(block1d, nil)
	if len(sendCancelPledgeBlockList2) != 1 || isRetry || err != nil ||
		sendCancelPledgeBlockList2[0].Quota != 105592 {
		t.Fatalf("send cancel pledge transaction 2 error")
	}
	db.accountBlockMap[addr1][hash1d] = sendCancelPledgeBlockList2[0]

	hash55 := types.DataHash([]byte{5, 5})
	block55 := &ledger.AccountBlock{
		Height:         5,
		AccountAddress: addr5,
		BlockType:      ledger.BlockTypeReceive,
		PrevHash:       hash54,
		FromBlockHash:  hash1d,
		SnapshotHash:   snapshot55.Hash,
	}
	vm = NewVM(db)
	vm.Debug = true
	db.addr = addr5
	updateReveiceBlockBySendBlock(block55, block1d)
	receiveCancelPledgeBlockList2, isRetry, err := vm.Run(block55, block1d)
	if len(receiveCancelPledgeBlockList2) != 2 || isRetry || err != nil ||
		receiveCancelPledgeBlockList2[1].Height != 6 ||
		len(db.storageMap[addr5][locHashPledge]) != 0 ||
		len(db.storageMap[addr5][locHashQuota]) != 0 ||
		db.balanceMap[addr5][*ledger.ViteTokenId()].Cmp(helper.Big0) != 0 ||
		receiveCancelPledgeBlockList2[0].Quota != 0 ||
		receiveCancelPledgeBlockList2[1].Quota != 0 {
		t.Fatalf("receive cancel pledge transaction 2 error")
	}
	db.accountBlockMap[addr5][hash55] = receiveCancelPledgeBlockList2[0]
	hash56 := types.DataHash([]byte{5, 6})
	db.accountBlockMap[addr5][hash56] = receiveCancelPledgeBlockList2[1]

	hash17 := types.DataHash([]byte{1, 16})
	block17 := &ledger.AccountBlock{
		Height:         7,
		AccountAddress: addr1,
		BlockType:      ledger.BlockTypeReceive,
		PrevHash:       hash1d,
		FromBlockHash:  hash56,
		SnapshotHash:   snapshot55.Hash,
	}
	vm = NewVM(db)
	vm.Debug = true
	db.addr = addr1
	balance1.Add(balance1, pledgeAmount)
	updateReveiceBlockBySendBlock(block17, receiveCancelPledgeBlockList2[1])
	receiveCancelPledgeRefundBlockList2, isRetry, err := vm.Run(block17, receiveCancelPledgeBlockList2[1])
	if len(receiveCancelPledgeRefundBlockList2) != 1 || isRetry || err != nil ||
		db.balanceMap[addr1][*ledger.ViteTokenId()].Cmp(balance1) != 0 ||
		receiveCancelPledgeRefundBlockList2[0].Quota != 21000 {
		t.Fatalf("receive cancel pledge refund transaction 2 error")
	}
	db.accountBlockMap[addr1][hash17] = receiveCancelPledgeRefundBlockList2[0]
}

func TestConsensusGroup(t *testing.T) {
	viteTotalSupply := new(big.Int).Mul(big.NewInt(2e6), big.NewInt(1e18))
	db, addr1, hash12, snapshot2, _ := prepareDb(viteTotalSupply)

	addr2 := AddressConsensusGroup
	block13Data, _ := ABI_consensusGroup.PackMethod(MethodNameCreateConsensusGroup,
		types.Gid{},
		uint8(25),
		int64(3),
		uint8(1),
		helper.LeftPadBytes(ledger.ViteTokenId().Bytes(), helper.WordSize),
		uint8(1),
		helper.JoinBytes(helper.LeftPadBytes(big.NewInt(1e18).Bytes(), helper.WordSize), helper.LeftPadBytes(ledger.ViteTokenId().Bytes(), helper.WordSize), helper.LeftPadBytes(big.NewInt(84600).Bytes(), helper.WordSize)),
		uint8(1),
		[]byte{})
	hash13 := types.DataHash([]byte{1, 3})
	block13 := &ledger.AccountBlock{
		Height:         3,
		ToAddress:      addr2,
		AccountAddress: addr1,
		BlockType:      ledger.BlockTypeSendCall,
		PrevHash:       hash12,
		Amount:         big.NewInt(0),
		TokenId:        *ledger.ViteTokenId(),
		Data:           block13Data,
		SnapshotHash:   snapshot2.Hash,
	}
	vm := NewVM(db)
	vm.Debug = true
	db.addr = addr1
	sendCreateConsensusGroupBlockList, isRetry, err := vm.Run(block13, nil)
	quota13, _ := dataGasCost(block13.Data)
	if len(sendCreateConsensusGroupBlockList) != 1 || isRetry || err != nil ||
		sendCreateConsensusGroupBlockList[0].Quota != quota13+createConsensusGroupGas ||
		!helper.AllZero(block13.Data[4:26]) || helper.AllZero(block13.Data[26:36]) ||
		block13.Fee.Cmp(createConsensusGroupFee) != 0 ||
		db.balanceMap[addr1][*ledger.ViteTokenId()].Cmp(new(big.Int).Mul(big.NewInt(1e6), big.NewInt(1e18))) != 0 {
		t.Fatalf("send create consensus group transaction error")
	}
	db.accountBlockMap[addr1][hash13] = sendCreateConsensusGroupBlockList[0]

	hash21 := types.DataHash([]byte{2, 1})
	block21 := &ledger.AccountBlock{
		Height:         1,
		AccountAddress: addr2,
		BlockType:      ledger.BlockTypeReceive,
		FromBlockHash:  hash13,
		SnapshotHash:   snapshot2.Hash,
	}
	vm = NewVM(db)
	vm.Debug = true
	locHash, _ := types.BytesToHash(block13.Data[4:36])
	db.addr = addr2
	updateReveiceBlockBySendBlock(block21, block13)
	receiveCreateConsensusGroupBlockList, isRetry, err := vm.Run(block21, block13)
	groupInfo, _ := ABI_consensusGroup.PackVariable(VariableNameConsensusGroupInfo,
		uint8(25),
		int64(3),
		uint8(1),
		helper.LeftPadBytes(ledger.ViteTokenId().Bytes(), helper.WordSize),
		uint8(1),
		helper.JoinBytes(helper.LeftPadBytes(big.NewInt(1e18).Bytes(), helper.WordSize), helper.LeftPadBytes(ledger.ViteTokenId().Bytes(), helper.WordSize), helper.LeftPadBytes(big.NewInt(84600).Bytes(), helper.WordSize)),
		uint8(1),
		[]byte{})
	if len(receiveCreateConsensusGroupBlockList) != 1 || isRetry || err != nil ||
		db.balanceMap[addr2][*ledger.ViteTokenId()].Sign() != 0 ||
		!bytes.Equal(db.storageMap[addr2][locHash], groupInfo) ||
		receiveCreateConsensusGroupBlockList[0].Quota != 0 {
		t.Fatalf("receive create consensus group transaction error")
	}
	db.accountBlockMap[addr2] = make(map[types.Hash]*ledger.AccountBlock)
	db.accountBlockMap[addr2][hash21] = receiveCreateConsensusGroupBlockList[0]
}

func TestGenesisBlockData(t *testing.T) {
	// vite owner mintage genesis block
	tokenName := "ViteToken"
	tokenSymbol := "ViteToken"
	decimals := uint8(18)
	totalSupply := new(big.Int).Mul(big.NewInt(1e18), big.NewInt(1e9))
	viteAddress, _, _ := types.CreateAddress()
	mintageData, _ := ABI_mintage.PackVariable(VariableNameMintage, tokenName, tokenSymbol, totalSupply, decimals, viteAddress, big.NewInt(0), int64(0))
	fmt.Println("-------------vite owner mintage genesis block-------------")
	fmt.Println("address: viteAddress")
	fmt.Printf("AccountBlock{\n\tBlockType: ledger.BlockTypeReceive,\n\tAccountAddress: viteAddress,\n\tHeight: %v,\n\tAmount: %v,\n\tTokenId:*ledger.ViteTokenId(),\n\tQuota:0,\n\tFee:%v,\n\tData:%v,\n}\n",
		1, totalSupply, big.NewInt(0), hex.EncodeToString(mintageData))
	fmt.Printf("Storage:{\n\t$balance:*ledger.ViteTokenId():%v\n}\n", totalSupply)
	fmt.Printf("SetToken{\n\tTokenId: *ledger.ViteTokenId(),\n\tTokenName: %v,\n\tTotalSupply: %v,\n\tDecimals: %v\n}\n", tokenName, totalSupply, decimals)

	// snapshot consensus group and common consensus group genesis block
	snapshotGid := types.Gid{0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	conditionCountingData, _ := ABI_consensusGroup.PackVariable(VariableNameConditionCounting1, *ledger.ViteTokenId())
	conditionRegisterData, _ := ABI_consensusGroup.PackVariable(VariableNameConditionRegister1, registerAmount, *ledger.ViteTokenId(), registerLockTime)
	consensusGroupData, _ := ABI_consensusGroup.PackVariable(VariableNameConsensusGroupInfo,
		uint8(25),
		int64(3),
		uint8(1),
		conditionCountingData,
		uint8(1),
		conditionRegisterData,
		uint8(1),
		[]byte{})
	fmt.Println("-------------snapshot consensus group and common consensus group genesis block-------------")
	fmt.Printf("address:%v\n", hex.EncodeToString(AddressConsensusGroup.Bytes()))
	fmt.Printf("AccountBlock{\n\tBlockType: ledger.BlockTypeReceive,\n\tAccountAddress: %v,\n\tHeight: %v,\n\tAmount: %v,\n\tTokenId:*ledger.ViteTokenId(),\n\tQuota:0,\n\tFee:%v,\n\tData:%v,\n}\n",
		hex.EncodeToString(AddressConsensusGroup.Bytes()), 1, big.NewInt(0), big.NewInt(0), []byte{})
	fmt.Printf("Storage:{\n\t%v:%v,\n\t%v:%v}\n", hex.EncodeToString(types.DataHash(snapshotGid.Bytes()).Bytes()), consensusGroupData, hex.EncodeToString(types.DataHash(ledger.ViteTokenId().Bytes()).Bytes()), consensusGroupData)

	// snapshot consensus group and common consensus group register genesis block
	fmt.Println("-------------snapshot consensus group and common consensus group register genesis block-------------")
	fmt.Printf("address:%v\n", hex.EncodeToString(AddressRegister.Bytes()))
	fmt.Printf("AccountBlock{\n\tBlockType: ledger.BlockTypeReceive,\n\tAccountAddress: %v,\n\tHeight: %v,\n\tAmount: %v,\n\tTokenId:*ledger.ViteTokenId(),\n\tQuota:0,\n\tFee:%v,\n\tData:%v,\n}\n",
		hex.EncodeToString(AddressRegister.Bytes()), 1, big.NewInt(0), big.NewInt(0), []byte{})
	fmt.Printf("Storage:{\n")
	timestamp := time.Now().Unix() + registerLockTime
	registerData, _ := ABI_register.PackVariable(VariableNameRegistration, common.Big0, timestamp, uint64(1), uint64(0))
	for i := 0; i < 25; i++ {
		snapshotKey := getRegisterKey("snapshotNode1", snapshotGid)
		commonKey := getRegisterKey("commonNode1", *ledger.CommonGid())
		fmt.Printf("\t%v: %v\n\t%v: %v\n", hex.EncodeToString(snapshotKey), hex.EncodeToString(registerData), hex.EncodeToString(commonKey), hex.EncodeToString(registerData))
	}
	fmt.Println("}")
}

func TestCheckTokenInfo(t *testing.T) {
	tests := []struct {
		data   string
		err    error
		result bool
	}{
		{"00", ErrInvalidData, false},
		{"46d0ce8b000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000a000000000000000000000000000000000000000000000000000000000000000e00000000000000000000000000000000000000000033b2e3c9fd0803ce80000000000000000000000000000000000000000000000000000000000000000000012000000000000000000000000000000000000000000000000000000000000000956697465546f6b656e0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000956697465546f6b656e0000000000000000000000000000000000000000000000", nil, true},
		{"46d0ce8b000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000a000000000000000000000000000000000000000000000000000000000000000e00000000000000000000000000000000000000000033b2e3c9fd0803ce80000000000000000000000000000000000000000000000000000000000000000000012000000000000000000000000000000000000000000000000000000000000000956697465546f6b656e00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000009", ErrInvalidData, true},
		{"46d0ce8b000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000a000000000000000000000000000000000000000000000000000000000000000e00000000000000000000000000000000000000000033b2e3c9fd0803ce80000000000000000000000000000000000000000000000000000000000000000000012000000000000000000000000000000000000000000000000000000000000000956697465546f6b656e0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000956697465546f6b651F0000000000000000000000000000000000000000000000", nil, false},
		{"46d0ce8b000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000a000000000000000000000000000000000000000000000000000000000000000e00000000000000000000000000000000000000000033b2e3c9fd0803ce80000000000000000000000000000000000000000000000000000000000000000000012000000000000000000000000000000000000000000000000000000000000000956697465546f6b651F0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000956697465546f6b651F0000000000000000000000000000000000000000000000", nil, false},
		{"46d0ce8b000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000a000000000000000000000000000000000000000000000000000000000000000e00000000000000000000000000000000000000000033b2e3c9fd0803ce80000000000000000000000000000000000000000000000000000000000000000000012000000000000000000000000000000000000000000000000000000000000000956697465546f6b656e0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000a56697465546f6b656e0000000000000000000000000000000000000000000000", nil, false},
		{"46d0ce8b000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000a000000000000000000000000000000000000000000000000000000000000000e00000000000000000000000000000000000000000033b2e3c9fd0803ce80000000000000000000000000000000000000000000000000000000000000000000013000000000000000000000000000000000000000000000000000000000000000956697465546f6b656e0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000956697465546f6b656e0000000000000000000000000000000000000000000000", nil, false},
	}
	for i, test := range tests {
		inputdata, _ := hex.DecodeString(test.data)
		param := new(ParamMintage)
		err := ABI_mintage.UnpackMethod(param, MethodNameMintage, inputdata)
		if test.err != nil && err == nil {
			t.Logf("%v th expected error", i)
		} else if test.err == nil && err != nil {
			t.Logf("%v th unexpected error", i)
		} else if test.err == nil {
			err = checkToken(*param)
			if test.result != (err == nil) {
				t.Fatalf("%v th check token data fail %v %v", i, test, err)
			}
		}
	}
}

func TestCheckTokenName(t *testing.T) {
	tests := []struct {
		data string
		exp  bool
	}{
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
