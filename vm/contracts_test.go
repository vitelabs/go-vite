package vm

import (
	"bytes"
	"encoding/hex"
	"github.com/ethereum/go-ethereum/common"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/vm/util"
	"math/big"
	"testing"
	"time"
)

func TestContractsRun(t *testing.T) {
	// prepare db
	viteTotalSupply := new(big.Int).Mul(big.NewInt(2e6), big.NewInt(1e18))
	db, addr1, hash12, snapshot2, timestamp := prepareDb(viteTotalSupply)
	// register
	addr2 := AddressRegister
	block13Data, err := ABI_register.PackMethod(MethodNameRegister, util.SnapshotGid)
	hash13 := types.DataHash([]byte{1, 3})
	block13 := &NoAccountBlock{
		height:         big.NewInt(3),
		toAddress:      addr2,
		accountAddress: addr1,
		blockType:      BlockTypeSendCall,
		prevHash:       hash12,
		amount:         new(big.Int).Mul(big.NewInt(1e6), big.NewInt(1e18)),
		data:           block13Data,
		tokenId:        util.ViteTokenTypeId,
		snapshotHash:   snapshot2.Hash(),
		depth:          1,
	}
	vm := NewVM(db, CreateNoAccountBlock)
	vm.Debug = true
	sendRegisterBlockList, isRetry, err := vm.Run(block13)
	if len(sendRegisterBlockList) != 1 || isRetry || err != nil ||
		sendRegisterBlockList[0].Quota() != 62664 ||
		db.balanceMap[addr1][util.ViteTokenTypeId].Cmp(new(big.Int).Mul(big.NewInt(1e6), big.NewInt(1e18))) != 0 {
		t.Fatalf("send register transaction error")
	}
	db.accountBlockMap[addr1][hash13] = sendRegisterBlockList[0]

	hash21 := types.DataHash([]byte{2, 1})
	block21 := &NoAccountBlock{
		height:         big.NewInt(1),
		toAddress:      addr2,
		accountAddress: addr1,
		blockType:      BlockTypeReceive,
		fromBlockHash:  hash13,
		snapshotHash:   snapshot2.Hash(),
		depth:          1,
	}
	vm = NewVM(db, CreateNoAccountBlock)
	vm.Debug = true
	locHashRegister := getKey(addr1, util.SnapshotGid)
	receiveRegisterBlockList, isRetry, err := vm.Run(block21)
	if len(receiveRegisterBlockList) != 1 || isRetry || err != nil ||
		db.balanceMap[addr1][util.ViteTokenTypeId].Cmp(new(big.Int).Mul(big.NewInt(1e6), big.NewInt(1e18))) != 0 ||
		!bytes.Equal(db.storageMap[addr2][locHashRegister], util.JoinBytes(util.LeftPadBytes(block13.Amount().Bytes(), 32), util.LeftPadBytes(new(big.Int).SetInt64(snapshot2.timestamp).Bytes(), 32), util.LeftPadBytes(snapshot2.height.Bytes(), 32), util.LeftPadBytes(util.Big0.Bytes(), 32))) ||
		receiveRegisterBlockList[0].Quota() != 0 {
		t.Fatalf("receive register transaction error")
	}
	db.accountBlockMap[addr2] = make(map[types.Hash]VmAccountBlock)
	db.accountBlockMap[addr2][hash21] = receiveRegisterBlockList[0]

	// cancel register
	snapshot3 := &NoSnapshotBlock{height: big.NewInt(3), timestamp: timestamp + 1, hash: types.DataHash([]byte{10, 3}), producer: addr1}
	db.snapshotBlockList = append(db.snapshotBlockList, snapshot3)
	snapshot4 := &NoSnapshotBlock{height: big.NewInt(4), timestamp: timestamp + 2, hash: types.DataHash([]byte{10, 4}), producer: addr1}
	db.snapshotBlockList = append(db.snapshotBlockList, snapshot4)

	hash14 := types.DataHash([]byte{1, 4})
	block14Data, _ := ABI_register.PackMethod(MethodNameCancelRegister, util.SnapshotGid)
	block14 := &NoAccountBlock{
		height:         big.NewInt(4),
		toAddress:      addr2,
		accountAddress: addr1,
		amount:         big.NewInt(0),
		tokenId:        util.ViteTokenTypeId,
		blockType:      BlockTypeSendCall,
		prevHash:       hash13,
		data:           block14Data,
		snapshotHash:   snapshot4.Hash(),
		depth:          1,
	}
	vm = NewVM(db, CreateNoAccountBlock)
	vm.Debug = true
	sendCancelRegisterBlockList, isRetry, err := vm.Run(block14)
	if len(sendCancelRegisterBlockList) != 1 || isRetry || err != nil ||
		sendCancelRegisterBlockList[0].Quota() != 83664 ||
		db.balanceMap[addr1][util.ViteTokenTypeId].Cmp(new(big.Int).Mul(big.NewInt(1e6), big.NewInt(1e18))) != 0 {
		t.Fatalf("send cancel register transaction error")
	}
	db.accountBlockMap[addr1][hash14] = sendCancelRegisterBlockList[0]

	hash22 := types.DataHash([]byte{2, 2})
	block22 := &NoAccountBlock{
		height:         big.NewInt(2),
		toAddress:      addr2,
		accountAddress: addr1,
		blockType:      BlockTypeReceive,
		prevHash:       hash21,
		fromBlockHash:  hash14,
		snapshotHash:   snapshot4.Hash(),
		depth:          1,
	}
	vm = NewVM(db, CreateNoAccountBlock)
	vm.Debug = true
	receiveCancelRegisterBlockList, isRetry, err := vm.Run(block22)
	if len(receiveCancelRegisterBlockList) != 2 || isRetry || err != nil ||
		db.balanceMap[addr2][util.ViteTokenTypeId].Cmp(util.Big0) != 0 ||
		db.balanceMap[addr1][util.ViteTokenTypeId].Cmp(new(big.Int).Mul(big.NewInt(1e6), big.NewInt(1e18))) != 0 ||
		!bytes.Equal(db.storageMap[addr2][locHashRegister], util.JoinBytes(util.EmptyWord, util.EmptyWord, util.LeftPadBytes(snapshot2.height.Bytes(), 32), util.LeftPadBytes(snapshot4.height.Bytes(), 32))) ||
		receiveCancelRegisterBlockList[0].Quota() != 0 ||
		receiveCancelRegisterBlockList[1].Quota() != 0 ||
		receiveCancelRegisterBlockList[1].Height().Cmp(big.NewInt(3)) != 0 ||
		receiveCancelRegisterBlockList[1].Depth() != 2 ||
		!bytes.Equal(receiveCancelRegisterBlockList[1].AccountAddress().Bytes(), addr2.Bytes()) ||
		!bytes.Equal(receiveCancelRegisterBlockList[1].ToAddress().Bytes(), addr1.Bytes()) ||
		receiveCancelRegisterBlockList[1].BlockType() != BlockTypeSendCall {
		t.Fatalf("receive cancel register transaction error")
	}
	db.accountBlockMap[addr2][hash22] = receiveCancelRegisterBlockList[0]
	hash23 := types.DataHash([]byte{2, 3})
	db.accountBlockMap[addr2][hash23] = receiveCancelRegisterBlockList[1]

	hash15 := types.DataHash([]byte{1, 5})
	block15 := &NoAccountBlock{
		height:         big.NewInt(5),
		toAddress:      addr1,
		accountAddress: addr2,
		blockType:      BlockTypeReceive,
		prevHash:       hash14,
		fromBlockHash:  hash23,
		snapshotHash:   snapshot4.Hash(),
		depth:          2,
	}
	vm = NewVM(db, CreateNoAccountBlock)
	vm.Debug = true
	receiveCancelRegisterRefundBlockList, isRetry, err := vm.Run(block15)
	if len(receiveCancelRegisterRefundBlockList) != 1 || isRetry || err != nil ||
		db.balanceMap[addr2][util.ViteTokenTypeId].Cmp(util.Big0) != 0 ||
		db.balanceMap[addr1][util.ViteTokenTypeId].Cmp(viteTotalSupply) != 0 ||
		receiveCancelRegisterRefundBlockList[0].Quota() != 21000 {
		t.Fatalf("receive cancel register refund transaction error")
	}
	db.accountBlockMap[addr1][hash15] = receiveCancelRegisterRefundBlockList[0]

	// reward
	for i := int64(1); i <= 50; i++ {
		snapshoti := &NoSnapshotBlock{height: big.NewInt(4 + i), timestamp: timestamp + 2 + i, hash: types.DataHash([]byte{10, byte(4 + i)}), producer: addr1}
		db.snapshotBlockList = append(db.snapshotBlockList, snapshoti)
	}
	snapshot54 := db.snapshotBlockList[53]
	block16Data, _ := ABI_register.PackMethod(MethodNameReward, util.SnapshotGid, common.Big0, common.Big0, common.Big0)
	hash16 := types.DataHash([]byte{1, 6})
	block16 := &NoAccountBlock{
		height:         big.NewInt(6),
		toAddress:      addr2,
		accountAddress: addr1,
		amount:         big.NewInt(0),
		tokenId:        util.ViteTokenTypeId,
		blockType:      BlockTypeSendCall,
		prevHash:       hash15,
		data:           block16Data,
		snapshotHash:   snapshot54.Hash(),
		depth:          1,
	}
	vm = NewVM(db, CreateNoAccountBlock)
	vm.Debug = true
	sendRewardBlockList, isRetry, err := vm.Run(block16)
	reward := new(big.Int).Mul(big.NewInt(2), rewardPerBlock)
	block16DataExpected, _ := ABI_register.PackMethod(MethodNameReward, util.SnapshotGid, snapshot4.Height(), snapshot2.Height(), reward)
	if len(sendRewardBlockList) != 1 || isRetry || err != nil ||
		sendRewardBlockList[0].Quota() != 84760 ||
		!bytes.Equal(sendRewardBlockList[0].Data(), block16DataExpected) {
		t.Fatalf("send reward transaction error")
	}
	db.accountBlockMap[addr1][hash16] = sendRewardBlockList[0]

	hash24 := types.DataHash([]byte{2, 4})
	block24 := &NoAccountBlock{
		height:         big.NewInt(4),
		toAddress:      addr2,
		accountAddress: addr1,
		blockType:      BlockTypeReceive,
		prevHash:       hash23,
		fromBlockHash:  hash16,
		snapshotHash:   snapshot54.Hash(),
		depth:          1,
	}
	vm = NewVM(db, CreateNoAccountBlock)
	vm.Debug = true
	receiveRewardBlockList, isRetry, err := vm.Run(block24)
	if len(receiveRewardBlockList) != 2 || isRetry || err != nil ||
		db.balanceMap[addr2][util.ViteTokenTypeId].Cmp(util.Big0) != 0 ||
		db.balanceMap[addr1][util.ViteTokenTypeId].Cmp(viteTotalSupply) != 0 ||
		len(db.storageMap[addr2][locHashRegister]) != 0 ||
		receiveRewardBlockList[0].Quota() != 0 ||
		receiveRewardBlockList[1].Quota() != 0 ||
		receiveRewardBlockList[1].Height().Cmp(big.NewInt(5)) != 0 ||
		receiveRewardBlockList[1].Depth() != 2 ||
		!bytes.Equal(receiveRewardBlockList[1].AccountAddress().Bytes(), addr2.Bytes()) ||
		!bytes.Equal(receiveRewardBlockList[1].ToAddress().Bytes(), addr1.Bytes()) ||
		receiveRewardBlockList[1].BlockType() != BlockTypeSendReward {
		t.Fatalf("receive reward transaction error")
	}
	db.accountBlockMap[addr2][hash24] = receiveRewardBlockList[0]
	hash25 := types.DataHash([]byte{2, 5})
	db.accountBlockMap[addr2][hash25] = receiveRewardBlockList[1]

	hash17 := types.DataHash([]byte{1, 7})
	block17 := &NoAccountBlock{
		height:         big.NewInt(7),
		toAddress:      addr1,
		accountAddress: addr2,
		blockType:      BlockTypeReceive,
		prevHash:       hash16,
		fromBlockHash:  hash25,
		snapshotHash:   snapshot54.Hash(),
		depth:          2,
	}
	vm = NewVM(db, CreateNoAccountBlock)
	vm.Debug = true
	receiveRewardRefundBlockList, isRetry, err := vm.Run(block17)
	if len(receiveRewardRefundBlockList) != 1 || isRetry || err != nil ||
		db.balanceMap[addr2][util.ViteTokenTypeId].Cmp(util.Big0) != 0 ||
		db.balanceMap[addr1][util.ViteTokenTypeId].Cmp(new(big.Int).Add(viteTotalSupply, reward)) != 0 ||
		receiveRewardRefundBlockList[0].Quota() != 21000 {
		t.Fatalf("receive reward refund transaction error")
	}
	db.accountBlockMap[addr1][hash17] = receiveRewardRefundBlockList[0]

	// vote
	addr3 := AddressVote
	block18Data, _ := ABI_vote.PackMethod(MethodNameVote, util.SnapshotGid, addr1)
	hash18 := types.DataHash([]byte{1, 8})
	block18 := &NoAccountBlock{
		height:         big.NewInt(8),
		toAddress:      addr3,
		accountAddress: addr1,
		amount:         big.NewInt(0),
		tokenId:        util.ViteTokenTypeId,
		blockType:      BlockTypeSendCall,
		prevHash:       hash17,
		data:           block18Data,
		snapshotHash:   snapshot54.Hash(),
		depth:          1,
	}
	vm = NewVM(db, CreateNoAccountBlock)
	vm.Debug = true
	sendVoteBlockList, isRetry, err := vm.Run(block18)
	if len(sendVoteBlockList) != 1 || isRetry || err != nil ||
		sendVoteBlockList[0].Quota() != 63872 {
		t.Fatalf("send vote transaction error")
	}
	db.accountBlockMap[addr1][hash18] = sendVoteBlockList[0]

	hash31 := types.DataHash([]byte{3, 1})
	block31 := &NoAccountBlock{
		height:         big.NewInt(1),
		toAddress:      addr3,
		accountAddress: addr1,
		blockType:      BlockTypeReceive,
		prevHash:       hash25,
		fromBlockHash:  hash18,
		snapshotHash:   snapshot54.Hash(),
		depth:          1,
	}
	vm = NewVM(db, CreateNoAccountBlock)
	vm.Debug = true
	receiveVoteBlockList, isRetry, err := vm.Run(block31)
	locHashVote := getKey(addr1, util.SnapshotGid)
	if len(receiveVoteBlockList) != 1 || isRetry || err != nil ||
		!bytes.Equal(db.storageMap[addr3][locHashVote], util.LeftPadBytes(addr1.Bytes(), 32)) ||
		receiveVoteBlockList[0].Quota() != 0 {
		t.Fatalf("receive vote transaction error")
	}
	db.accountBlockMap[addr3] = make(map[types.Hash]VmAccountBlock)
	db.accountBlockMap[addr3][hash31] = receiveVoteBlockList[0]

	addr4, _, _ := types.CreateAddress()
	db.accountBlockMap[addr4] = make(map[types.Hash]VmAccountBlock)
	block19Data, _ := ABI_vote.PackMethod(MethodNameVote, util.SnapshotGid, addr4)
	hash19 := types.DataHash([]byte{1, 9})
	block19 := &NoAccountBlock{
		height:         big.NewInt(9),
		toAddress:      addr3,
		accountAddress: addr1,
		amount:         big.NewInt(0),
		tokenId:        util.ViteTokenTypeId,
		blockType:      BlockTypeSendCall,
		prevHash:       hash18,
		data:           block19Data,
		snapshotHash:   snapshot54.Hash(),
		depth:          1,
	}
	vm = NewVM(db, CreateNoAccountBlock)
	vm.Debug = true
	sendVoteBlockList2, isRetry, err := vm.Run(block19)
	if len(sendVoteBlockList2) != 1 || isRetry || err != nil ||
		sendVoteBlockList2[0].Quota() != 63872 {
		t.Fatalf("send vote transaction 2 error")
	}
	db.accountBlockMap[addr1][hash19] = sendVoteBlockList2[0]

	hash32 := types.DataHash([]byte{3, 2})
	block32 := &NoAccountBlock{
		height:         big.NewInt(2),
		toAddress:      addr3,
		accountAddress: addr1,
		blockType:      BlockTypeReceive,
		prevHash:       hash31,
		fromBlockHash:  hash19,
		snapshotHash:   snapshot54.Hash(),
		depth:          1,
	}
	vm = NewVM(db, CreateNoAccountBlock)
	vm.Debug = true
	receiveVoteBlockList2, isRetry, err := vm.Run(block32)
	if len(receiveVoteBlockList2) != 1 || isRetry || err != nil ||
		!bytes.Equal(db.storageMap[addr3][locHashVote], util.LeftPadBytes(addr4.Bytes(), 32)) ||
		receiveVoteBlockList2[0].Quota() != 0 {
		t.Fatalf("receive vote transaction 2 error")
	}
	db.accountBlockMap[addr3][hash32] = receiveVoteBlockList2[0]

	// cancel vote
	block1aData, _ := ABI_vote.PackMethod(MethodNameCancelVote, util.SnapshotGid)
	hash1a := types.DataHash([]byte{1, 10})
	block1a := &NoAccountBlock{
		height:         big.NewInt(10),
		toAddress:      addr3,
		accountAddress: addr1,
		amount:         big.NewInt(0),
		tokenId:        util.ViteTokenTypeId,
		blockType:      BlockTypeSendCall,
		prevHash:       hash19,
		data:           block1aData,
		snapshotHash:   snapshot54.Hash(),
		depth:          1,
	}
	vm = NewVM(db, CreateNoAccountBlock)
	vm.Debug = true
	sendCancelVoteBlockList, isRetry, err := vm.Run(block1a)
	if len(sendCancelVoteBlockList) != 1 || isRetry || err != nil ||
		sendCancelVoteBlockList[0].Quota() != 62464 {
		t.Fatalf("send cancel vote transaction error")
	}
	db.accountBlockMap[addr1][hash1a] = sendCancelVoteBlockList[0]

	hash33 := types.DataHash([]byte{3, 3})
	block33 := &NoAccountBlock{
		height:         big.NewInt(3),
		toAddress:      addr3,
		accountAddress: addr1,
		blockType:      BlockTypeReceive,
		prevHash:       hash32,
		fromBlockHash:  hash1a,
		snapshotHash:   snapshot54.Hash(),
		depth:          1,
	}
	vm = NewVM(db, CreateNoAccountBlock)
	vm.Debug = true
	receiveCancelVoteBlockList, isRetry, err := vm.Run(block33)
	if len(receiveCancelVoteBlockList) != 1 || isRetry || err != nil ||
		len(db.storageMap[addr3][locHashVote]) != 0 ||
		receiveCancelVoteBlockList[0].Quota() != 0 {
		t.Fatalf("receive cancel vote transaction error")
	}
	db.accountBlockMap[addr3][hash33] = receiveCancelVoteBlockList[0]

	// pledge
	addr5 := AddressPledge
	pledgeAmount := reward
	withdrawTime := timestamp + 53 + pledgeTime
	block1bData, err := ABI_pledge.PackMethod(MethodNamePledge, addr4, withdrawTime)
	hash1b := types.DataHash([]byte{1, 11})
	block1b := &NoAccountBlock{
		height:         big.NewInt(11),
		toAddress:      addr5,
		accountAddress: addr1,
		amount:         pledgeAmount,
		tokenId:        util.ViteTokenTypeId,
		blockType:      BlockTypeSendCall,
		prevHash:       hash1a,
		data:           block1bData,
		snapshotHash:   snapshot54.Hash(),
		depth:          1,
	}
	vm = NewVM(db, CreateNoAccountBlock)
	vm.Debug = true
	sendPledgeBlockList, isRetry, err := vm.Run(block1b)
	if len(sendPledgeBlockList) != 1 || isRetry || err != nil ||
		db.balanceMap[addr1][util.ViteTokenTypeId].Cmp(viteTotalSupply) != 0 ||
		sendPledgeBlockList[0].Quota() != 84464 {
		t.Fatalf("send pledge transaction error")
	}
	db.accountBlockMap[addr1][hash1b] = sendPledgeBlockList[0]

	hash51 := types.DataHash([]byte{5, 1})
	block51 := &NoAccountBlock{
		height:         big.NewInt(1),
		toAddress:      addr5,
		accountAddress: addr1,
		blockType:      BlockTypeReceive,
		prevHash:       hash33,
		fromBlockHash:  hash1b,
		snapshotHash:   snapshot54.Hash(),
		depth:          1,
	}
	vm = NewVM(db, CreateNoAccountBlock)
	vm.Debug = true
	receivePledgeBlockList, isRetry, err := vm.Run(block51)
	locHashQuota := types.DataHash(addr4.Bytes())
	locHashPledge := types.DataHash(append(addr1.Bytes(), locHashQuota.Bytes()...))
	if len(receivePledgeBlockList) != 1 || isRetry || err != nil ||
		!bytes.Equal(db.storageMap[addr5][locHashPledge], util.JoinBytes(util.LeftPadBytes(pledgeAmount.Bytes(), 32), util.LeftPadBytes(new(big.Int).SetInt64(withdrawTime).Bytes(), 32))) ||
		!bytes.Equal(db.storageMap[addr5][locHashQuota], util.LeftPadBytes(pledgeAmount.Bytes(), 32)) ||
		db.balanceMap[addr5][util.ViteTokenTypeId].Cmp(pledgeAmount) != 0 ||
		receivePledgeBlockList[0].Quota() != 0 {
		t.Fatalf("receive pledge transaction error")
	}
	db.accountBlockMap[addr5] = make(map[types.Hash]VmAccountBlock)
	db.accountBlockMap[addr5][hash51] = receivePledgeBlockList[0]

	withdrawTime = timestamp + 100 + pledgeTime
	block1cData, _ := ABI_pledge.PackMethod(MethodNamePledge, addr4, withdrawTime)
	hash1c := types.DataHash([]byte{1, 12})
	block1c := &NoAccountBlock{
		height:         big.NewInt(12),
		toAddress:      addr5,
		accountAddress: addr1,
		amount:         pledgeAmount,
		tokenId:        util.ViteTokenTypeId,
		blockType:      BlockTypeSendCall,
		prevHash:       hash1b,
		data:           block1cData,
		snapshotHash:   snapshot54.Hash(),
		depth:          1,
	}
	vm = NewVM(db, CreateNoAccountBlock)
	vm.Debug = true
	sendPledgeBlockList2, isRetry, err := vm.Run(block1c)
	if len(sendPledgeBlockList2) != 1 || isRetry || err != nil ||
		db.balanceMap[addr1][util.ViteTokenTypeId].Cmp(new(big.Int).Sub(viteTotalSupply, pledgeAmount)) != 0 ||
		sendPledgeBlockList2[0].Quota() != 84464 {
		t.Fatalf("send pledge transaction 2 error")
	}
	db.accountBlockMap[addr1][hash1c] = sendPledgeBlockList2[0]

	hash52 := types.DataHash([]byte{5, 2})
	block52 := &NoAccountBlock{
		height:         big.NewInt(2),
		toAddress:      addr5,
		accountAddress: addr1,
		blockType:      BlockTypeReceive,
		prevHash:       hash51,
		fromBlockHash:  hash1c,
		snapshotHash:   snapshot54.Hash(),
		depth:          1,
	}
	vm = NewVM(db, CreateNoAccountBlock)
	vm.Debug = true
	receivePledgeBlockList2, isRetry, err := vm.Run(block52)
	newPledgeAmount := new(big.Int).Add(pledgeAmount, pledgeAmount)
	if len(receivePledgeBlockList2) != 1 || isRetry || err != nil ||
		!bytes.Equal(db.storageMap[addr5][locHashPledge], util.JoinBytes(util.LeftPadBytes(newPledgeAmount.Bytes(), 32), util.LeftPadBytes(new(big.Int).SetInt64(withdrawTime).Bytes(), 32))) ||
		!bytes.Equal(db.storageMap[addr5][locHashQuota], util.LeftPadBytes(newPledgeAmount.Bytes(), 32)) ||
		db.balanceMap[addr5][util.ViteTokenTypeId].Cmp(newPledgeAmount) != 0 ||
		receivePledgeBlockList2[0].Quota() != 0 {
		t.Fatalf("receive pledge transaction 2 error")
	}
	db.accountBlockMap[addr5][hash52] = receivePledgeBlockList2[0]

	// cancel pledge
	snapshot55 := &NoSnapshotBlock{height: big.NewInt(55), timestamp: timestamp + 100 + pledgeTime, hash: types.DataHash([]byte{10, 55}), producer: addr1}
	db.snapshotBlockList = append(db.snapshotBlockList, snapshot55)

	block1dData, _ := ABI_pledge.PackMethod(MethodNameCancelPledge, addr4, pledgeAmount)
	hash1d := types.DataHash([]byte{1, 13})
	block1d := &NoAccountBlock{
		height:         big.NewInt(13),
		toAddress:      addr5,
		accountAddress: addr1,
		amount:         util.Big0,
		tokenId:        util.ViteTokenTypeId,
		blockType:      BlockTypeSendCall,
		prevHash:       hash1c,
		data:           block1dData,
		snapshotHash:   snapshot55.Hash(),
		depth:          1,
	}
	vm = NewVM(db, CreateNoAccountBlock)
	vm.Debug = true
	sendCancelPledgeBlockList, isRetry, err := vm.Run(block1d)
	if len(sendCancelPledgeBlockList) != 1 || isRetry || err != nil ||
		sendCancelPledgeBlockList[0].Quota() != 105592 {
		t.Fatalf("send cancel pledge transaction error")
	}
	db.accountBlockMap[addr1][hash1d] = sendCancelPledgeBlockList[0]

	hash53 := types.DataHash([]byte{5, 3})
	block53 := &NoAccountBlock{
		height:         big.NewInt(3),
		toAddress:      addr5,
		accountAddress: addr1,
		blockType:      BlockTypeReceive,
		prevHash:       hash52,
		fromBlockHash:  hash1d,
		snapshotHash:   snapshot55.Hash(),
		depth:          1,
	}
	vm = NewVM(db, CreateNoAccountBlock)
	vm.Debug = true
	receiveCancelPledgeBlockList, isRetry, err := vm.Run(block53)
	if len(receiveCancelPledgeBlockList) != 2 || isRetry || err != nil ||
		receiveCancelPledgeBlockList[1].Height().Cmp(big.NewInt(4)) != 0 ||
		!bytes.Equal(db.storageMap[addr5][locHashPledge], util.JoinBytes(util.LeftPadBytes(pledgeAmount.Bytes(), 32), util.LeftPadBytes(new(big.Int).SetInt64(withdrawTime).Bytes(), 32))) ||
		!bytes.Equal(db.storageMap[addr5][locHashQuota], util.LeftPadBytes(pledgeAmount.Bytes(), 32)) ||
		db.balanceMap[addr5][util.ViteTokenTypeId].Cmp(pledgeAmount) != 0 ||
		receiveCancelPledgeBlockList[0].Quota() != 0 ||
		receiveCancelPledgeBlockList[1].Quota() != 0 {
		t.Fatalf("receive cancel pledge transaction error")
	}
	db.accountBlockMap[addr5][hash53] = receiveCancelPledgeBlockList[0]
	hash54 := types.DataHash([]byte{5, 4})
	db.accountBlockMap[addr5][hash54] = receiveCancelPledgeBlockList[1]

	hash1e := types.DataHash([]byte{1, 14})
	block1e := &NoAccountBlock{
		height:         big.NewInt(14),
		toAddress:      addr1,
		accountAddress: addr5,
		blockType:      BlockTypeReceive,
		prevHash:       hash1d,
		fromBlockHash:  hash54,
		snapshotHash:   snapshot55.Hash(),
		depth:          1,
	}
	vm = NewVM(db, CreateNoAccountBlock)
	vm.Debug = true
	receiveCancelPledgeRefundBlockList, isRetry, err := vm.Run(block1e)
	if len(receiveCancelPledgeRefundBlockList) != 1 || isRetry || err != nil ||
		db.balanceMap[addr1][util.ViteTokenTypeId].Cmp(viteTotalSupply) != 0 ||
		receiveCancelPledgeRefundBlockList[0].Quota() != 21000 {
		t.Fatalf("receive cancel pledge refund transaction error")
	}
	db.accountBlockMap[addr1][hash1e] = receiveCancelPledgeRefundBlockList[0]

	block1fData, _ := ABI_pledge.PackMethod(MethodNameCancelPledge, addr4, pledgeAmount)
	hash1f := types.DataHash([]byte{1, 15})
	block1f := &NoAccountBlock{
		height:         big.NewInt(15),
		toAddress:      addr5,
		accountAddress: addr1,
		amount:         util.Big0,
		tokenId:        util.ViteTokenTypeId,
		blockType:      BlockTypeSendCall,
		prevHash:       hash1e,
		data:           block1fData,
		snapshotHash:   snapshot55.Hash(),
		depth:          1,
	}
	vm = NewVM(db, CreateNoAccountBlock)
	vm.Debug = true
	sendCancelPledgeBlockList2, isRetry, err := vm.Run(block1f)
	if len(sendCancelPledgeBlockList2) != 1 || isRetry || err != nil ||
		sendCancelPledgeBlockList2[0].Quota() != 105592 {
		t.Fatalf("send cancel pledge transaction 2 error")
	}
	db.accountBlockMap[addr1][hash1f] = sendCancelPledgeBlockList2[0]

	hash55 := types.DataHash([]byte{5, 5})
	block55 := &NoAccountBlock{
		height:         big.NewInt(5),
		toAddress:      addr5,
		accountAddress: addr1,
		blockType:      BlockTypeReceive,
		prevHash:       hash54,
		fromBlockHash:  hash1f,
		snapshotHash:   snapshot55.Hash(),
		depth:          1,
	}
	vm = NewVM(db, CreateNoAccountBlock)
	vm.Debug = true
	receiveCancelPledgeBlockList2, isRetry, err := vm.Run(block55)
	if len(receiveCancelPledgeBlockList2) != 2 || isRetry || err != nil ||
		receiveCancelPledgeBlockList2[1].Height().Cmp(big.NewInt(6)) != 0 ||
		len(db.storageMap[addr5][locHashPledge]) != 0 ||
		len(db.storageMap[addr5][locHashQuota]) != 0 ||
		db.balanceMap[addr5][util.ViteTokenTypeId].Cmp(util.Big0) != 0 ||
		receiveCancelPledgeBlockList2[0].Quota() != 0 ||
		receiveCancelPledgeBlockList2[1].Quota() != 0 {
		t.Fatalf("receive cancel pledge transaction 2 error")
	}
	db.accountBlockMap[addr5][hash55] = receiveCancelPledgeBlockList2[0]
	hash56 := types.DataHash([]byte{5, 6})
	db.accountBlockMap[addr5][hash56] = receiveCancelPledgeBlockList2[1]

	hash1g := types.DataHash([]byte{1, 16})
	block1g := &NoAccountBlock{
		height:         big.NewInt(16),
		toAddress:      addr1,
		accountAddress: addr5,
		blockType:      BlockTypeReceive,
		prevHash:       hash1f,
		fromBlockHash:  hash56,
		snapshotHash:   snapshot55.Hash(),
		depth:          1,
	}
	vm = NewVM(db, CreateNoAccountBlock)
	vm.Debug = true
	receiveCancelPledgeRefundBlockList2, isRetry, err := vm.Run(block1g)
	if len(receiveCancelPledgeRefundBlockList2) != 1 || isRetry || err != nil ||
		db.balanceMap[addr1][util.ViteTokenTypeId].Cmp(new(big.Int).Add(viteTotalSupply, pledgeAmount)) != 0 ||
		receiveCancelPledgeRefundBlockList2[0].Quota() != 21000 {
		t.Fatalf("receive cancel pledge refund transaction 2 error")
	}
	db.accountBlockMap[addr1][hash1g] = receiveCancelPledgeRefundBlockList2[0]
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
		util.LeftPadBytes(util.ViteTokenTypeId.Bytes(), 32),
		uint8(1),
		util.JoinBytes(util.LeftPadBytes(big.NewInt(1e18).Bytes(), 32), util.LeftPadBytes(util.ViteTokenTypeId.Bytes(), 32), util.LeftPadBytes(big.NewInt(84600).Bytes(), 32)),
		uint8(1),
		[]byte{})
	hash13 := types.DataHash([]byte{1, 3})
	block13 := &NoAccountBlock{
		height:         big.NewInt(3),
		toAddress:      addr2,
		accountAddress: addr1,
		blockType:      BlockTypeSendCall,
		prevHash:       hash12,
		amount:         big.NewInt(0),
		tokenId:        util.ViteTokenTypeId,
		data:           block13Data,
		snapshotHash:   snapshot2.Hash(),
		depth:          1,
	}
	vm := NewVM(db, CreateNoAccountBlock)
	vm.Debug = true
	sendCreateConsensusGroupBlockList, isRetry, err := vm.Run(block13)
	if len(sendCreateConsensusGroupBlockList) != 1 || isRetry || err != nil ||
		sendCreateConsensusGroupBlockList[0].Quota() != 66568 ||
		!util.AllZero(block13.Data()[4:26]) || util.AllZero(block13.Data()[26:36]) ||
		block13.CreateFee().Cmp(createConsensusGroupFee) != 0 ||
		db.balanceMap[addr1][util.ViteTokenTypeId].Cmp(new(big.Int).Mul(big.NewInt(1e6), big.NewInt(1e18))) != 0 {
		t.Fatalf("send create consensus group transaction error")
	}
	db.accountBlockMap[addr1][hash13] = sendCreateConsensusGroupBlockList[0]

	hash21 := types.DataHash([]byte{2, 1})
	block21 := &NoAccountBlock{
		height:         big.NewInt(1),
		toAddress:      addr2,
		accountAddress: addr1,
		blockType:      BlockTypeReceive,
		fromBlockHash:  hash13,
		snapshotHash:   snapshot2.Hash(),
		depth:          1,
	}
	vm = NewVM(db, CreateNoAccountBlock)
	vm.Debug = true
	locHash := types.DataHash(block13.Data()[26:36])
	receiveCreateConsensusGroupBlockList, isRetry, err := vm.Run(block21)
	groupInfo, _ := ABI_consensusGroup.PackVariable(VariableNameConsensusGroupInfo,
		uint8(25),
		int64(3),
		uint8(1),
		util.LeftPadBytes(util.ViteTokenTypeId.Bytes(), 32),
		uint8(1),
		util.JoinBytes(util.LeftPadBytes(big.NewInt(1e18).Bytes(), 32), util.LeftPadBytes(util.ViteTokenTypeId.Bytes(), 32), util.LeftPadBytes(big.NewInt(84600).Bytes(), 32)),
		uint8(1),
		[]byte{})
	if len(receiveCreateConsensusGroupBlockList) != 1 || isRetry || err != nil ||
		db.balanceMap[addr2][util.ViteTokenTypeId].Sign() != 0 ||
		!bytes.Equal(db.storageMap[addr2][locHash], groupInfo) ||
		receiveCreateConsensusGroupBlockList[0].Quota() != 0 {
		t.Fatalf("receive create consensus group transaction error")
	}
	db.accountBlockMap[addr2] = make(map[types.Hash]VmAccountBlock)
	db.accountBlockMap[addr2][hash21] = receiveCreateConsensusGroupBlockList[0]
}
func TestBytesToGid(t *testing.T) {
	timestamp := time.Now().Unix()
	t.Log(timestamp)
	t.Log(hex.EncodeToString(big.NewInt(timestamp).Bytes()))
}
