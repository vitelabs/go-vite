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
	"github.com/vitelabs/go-vite/vm/util"
	"math/big"
	"regexp"
	"strconv"
	"testing"
	"time"
)

func TestContractsRegisterRun(t *testing.T) {
	// prepare db
	viteTotalSupply := new(big.Int).Mul(big.NewInt(1e9), big.NewInt(1e18))
	db, addr1, _, hash12, snapshot2, timestamp := prepareDb(viteTotalSupply)
	blockTime := time.Now()
	// register
	balance1 := new(big.Int).Set(viteTotalSupply)
	addr6, privateKey6, _ := types.CreateAddress()
	addr7, privateKey7, _ := types.CreateAddress()
	publicKey6 := ed25519.PublicKey(privateKey6.PubByte())
	publicKey7 := ed25519.PublicKey(privateKey7.PubByte())
	db.accountBlockMap[addr6] = make(map[types.Hash]*ledger.AccountBlock)
	db.accountBlockMap[addr7] = make(map[types.Hash]*ledger.AccountBlock)
	addr2 := contracts.AddressRegister
	nodeName := "super1"
	sign := ed25519.Sign(privateKey7, contracts.GetRegisterMessageForSignature(addr1, types.SNAPSHOT_GID))
	block13Data, err := contracts.ABIRegister.PackMethod(contracts.MethodNameRegister, types.SNAPSHOT_GID, nodeName, addr7, []byte(publicKey7), sign)
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
		SnapshotHash:   snapshot2.Hash,
		Timestamp:      &blockTime,
	}
	vm := NewVM()
	vm.Debug = true
	db.addr = addr1
	block13DataGas, _ := util.DataGasCost(block13Data)
	sendRegisterBlockList, isRetry, err := vm.Run(db, block13, nil)
	balance1.Sub(balance1, block13.Amount)
	if len(sendRegisterBlockList) != 1 || isRetry || err != nil ||
		sendRegisterBlockList[0].AccountBlock.Quota != block13DataGas+62200 ||
		db.balanceMap[addr1][ledger.ViteTokenId].Cmp(balance1) != 0 {
		t.Fatalf("send register transaction error")
	}
	db.accountBlockMap[addr1][hash13] = sendRegisterBlockList[0].AccountBlock

	hash21 := types.DataHash([]byte{2, 1})
	block21 := &ledger.AccountBlock{
		Height:         1,
		AccountAddress: addr2,
		BlockType:      ledger.BlockTypeReceive,
		FromBlockHash:  hash13,
		SnapshotHash:   snapshot2.Hash,
		Timestamp:      &blockTime,
	}
	vm = NewVM()
	vm.Debug = true
	locHashRegister, _ := types.BytesToHash(contracts.GetRegisterKey(nodeName, types.SNAPSHOT_GID))
	registrationData, _ := contracts.ABIRegister.PackVariable(contracts.VariableNameRegistration, nodeName, addr7, addr1, block13.Amount, snapshot2.Height, snapshot2.Height, uint64(0))
	db.addr = addr2
	receiveRegisterBlockList, isRetry, err := vm.Run(db, block21, sendRegisterBlockList[0].AccountBlock)
	if len(receiveRegisterBlockList) != 1 || isRetry || err != nil ||
		db.balanceMap[addr1][ledger.ViteTokenId].Cmp(balance1) != 0 ||
		!bytes.Equal(db.storageMap[addr2][string(locHashRegister.Bytes())], registrationData) ||
		receiveRegisterBlockList[0].AccountBlock.Quota != 0 {
		t.Fatalf("receive register transaction error")
	}
	db.accountBlockMap[addr2] = make(map[types.Hash]*ledger.AccountBlock)
	db.accountBlockMap[addr2][hash21] = receiveRegisterBlockList[0].AccountBlock

	// update registration
	sign = ed25519.Sign(privateKey6, contracts.GetRegisterMessageForSignature(addr1, types.SNAPSHOT_GID))
	block14Data, err := contracts.ABIRegister.PackMethod(contracts.MethodNameUpdateRegistration, types.SNAPSHOT_GID, nodeName, addr6, []byte(publicKey6), sign)
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
		SnapshotHash:   snapshot2.Hash,
		Timestamp:      &blockTime,
	}
	vm = NewVM()
	vm.Debug = true
	db.addr = addr1
	block14DataGas, _ := util.DataGasCost(block14Data)
	sendRegisterBlockList2, isRetry, err := vm.Run(db, block14, nil)
	if len(sendRegisterBlockList2) != 1 || isRetry || err != nil ||
		sendRegisterBlockList2[0].AccountBlock.Quota != block14DataGas+62200 ||
		db.balanceMap[addr1][ledger.ViteTokenId].Cmp(balance1) != 0 {
		t.Fatalf("send update registration transaction error")
	}
	db.accountBlockMap[addr1][hash14] = sendRegisterBlockList2[0].AccountBlock

	hash22 := types.DataHash([]byte{2, 2})
	block22 := &ledger.AccountBlock{
		Height:         2,
		AccountAddress: addr2,
		BlockType:      ledger.BlockTypeReceive,
		FromBlockHash:  hash14,
		PrevHash:       hash21,
		SnapshotHash:   snapshot2.Hash,
		Timestamp:      &blockTime,
	}
	vm = NewVM()
	vm.Debug = true
	registrationData, _ = contracts.ABIRegister.PackVariable(contracts.VariableNameRegistration, nodeName, addr6, addr1, block13.Amount, snapshot2.Height, snapshot2.Height, uint64(0))
	db.addr = addr2
	receiveRegisterBlockList2, isRetry, err := vm.Run(db, block22, sendRegisterBlockList2[0].AccountBlock)
	if len(receiveRegisterBlockList2) != 1 || isRetry || err != nil ||
		db.balanceMap[addr1][ledger.ViteTokenId].Cmp(balance1) != 0 ||
		!bytes.Equal(db.storageMap[addr2][string(locHashRegister.Bytes())], registrationData) ||
		receiveRegisterBlockList2[0].AccountBlock.Quota != 0 {
		t.Fatalf("receive update registration transaction error")
	}
	db.accountBlockMap[addr2][hash22] = receiveRegisterBlockList2[0].AccountBlock

	// get contracts data
	db.addr = contracts.AddressRegister
	if registerList := contracts.GetRegisterList(db, types.SNAPSHOT_GID); len(registerList) != 1 || registerList[0].Name != nodeName {
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
	block15Data, _ := contracts.ABIRegister.PackMethod(contracts.MethodNameCancelRegister, types.SNAPSHOT_GID, nodeName)
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
		SnapshotHash:   snapshot5.Hash,
		Timestamp:      &blockTime,
	}
	vm = NewVM()
	vm.Debug = true
	db.addr = addr1
	block15DataGas, _ := util.DataGasCost(block15Data)
	sendCancelRegisterBlockList, isRetry, err := vm.Run(db, block15, nil)
	if len(sendCancelRegisterBlockList) != 1 || isRetry || err != nil ||
		sendCancelRegisterBlockList[0].AccountBlock.Quota != block15DataGas+83200 ||
		db.balanceMap[addr1][ledger.ViteTokenId].Cmp(balance1) != 0 {
		t.Fatalf("send cancel register transaction error")
	}
	db.accountBlockMap[addr1][hash15] = sendCancelRegisterBlockList[0].AccountBlock

	hash23 := types.DataHash([]byte{2, 3})
	block23 := &ledger.AccountBlock{
		Height:         3,
		AccountAddress: addr2,
		BlockType:      ledger.BlockTypeReceive,
		PrevHash:       hash21,
		FromBlockHash:  hash15,
		SnapshotHash:   snapshot5.Hash,
		Timestamp:      &blockTime,
	}
	vm = NewVM()
	vm.Debug = true
	db.addr = addr2
	receiveCancelRegisterBlockList, isRetry, err := vm.Run(db, block23, sendCancelRegisterBlockList[0].AccountBlock)
	registrationData, _ = contracts.ABIRegister.PackVariable(contracts.VariableNameRegistration, nodeName, addr6, addr1, helper.Big0, uint64(0), snapshot2.Height, snapshot5.Height)
	if len(receiveCancelRegisterBlockList) != 2 || isRetry || err != nil ||
		db.balanceMap[addr2][ledger.ViteTokenId].Cmp(helper.Big0) != 0 ||
		db.balanceMap[addr1][ledger.ViteTokenId].Cmp(balance1) != 0 ||
		!bytes.Equal(db.storageMap[addr2][string(locHashRegister.Bytes())], registrationData) ||
		receiveCancelRegisterBlockList[0].AccountBlock.Quota != 0 ||
		receiveCancelRegisterBlockList[1].AccountBlock.Quota != 0 ||
		receiveCancelRegisterBlockList[1].AccountBlock.Height != 4 ||
		receiveCancelRegisterBlockList[1].AccountBlock.AccountAddress != addr2 ||
		receiveCancelRegisterBlockList[1].AccountBlock.ToAddress != addr1 ||
		receiveCancelRegisterBlockList[1].AccountBlock.BlockType != ledger.BlockTypeSendCall {
		t.Fatalf("receive cancel register transaction error")
	}
	db.accountBlockMap[addr2][hash23] = receiveCancelRegisterBlockList[0].AccountBlock
	hash24 := types.DataHash([]byte{2, 4})
	db.accountBlockMap[addr2][hash24] = receiveCancelRegisterBlockList[1].AccountBlock

	hash16 := types.DataHash([]byte{1, 6})
	block16 := &ledger.AccountBlock{
		Height:         6,
		AccountAddress: addr1,
		BlockType:      ledger.BlockTypeReceive,
		PrevHash:       hash16,
		FromBlockHash:  hash23,
		SnapshotHash:   snapshot5.Hash,
		Timestamp:      &blockTime,
	}
	vm = NewVM()
	vm.Debug = true
	db.addr = addr1
	balance1.Add(balance1, block13.Amount)
	receiveCancelRegisterRefundBlockList, isRetry, err := vm.Run(db, block16, receiveCancelRegisterBlockList[1].AccountBlock)
	if len(receiveCancelRegisterRefundBlockList) != 1 || isRetry || err != nil ||
		db.balanceMap[addr2][ledger.ViteTokenId].Cmp(helper.Big0) != 0 ||
		db.balanceMap[addr1][ledger.ViteTokenId].Cmp(balance1) != 0 ||
		receiveCancelRegisterRefundBlockList[0].AccountBlock.Quota != 21000 {
		t.Fatalf("receive cancel register refund transaction error")
	}
	db.accountBlockMap[addr1][hash16] = receiveCancelRegisterRefundBlockList[0].AccountBlock

	// reward
	time6 := time.Unix(timestamp+4, 0)
	snapshot6 := &ledger.SnapshotBlock{Height: snapshot5.Height + 256, Timestamp: &time6, Hash: types.DataHash([]byte{10, byte(6)})}
	db.snapshotBlockList = append(db.snapshotBlockList, snapshot6)
	db.storageMap[contracts.AddressPledge][string(types.DataHash(addr1.Bytes()).Bytes())], _ = contracts.ABIPledge.PackVariable(contracts.VariableNamePledgeBeneficial, new(big.Int).Mul(big.NewInt(1e6), big.NewInt(1e18)))
	db.storageMap[contracts.AddressPledge][string(contracts.GetPledgeBeneficialKey(addr7))], _ = contracts.ABIPledge.PackVariable(contracts.VariableNamePledgeBeneficial, new(big.Int).Mul(big.NewInt(1e6), big.NewInt(1e18)))
	block17Data, _ := contracts.ABIRegister.PackMethod(contracts.MethodNameReward, types.SNAPSHOT_GID, nodeName, addr7, uint64(0), uint64(0), helper.Big0)
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
		SnapshotHash:   snapshot6.Hash,
		Timestamp:      &blockTime,
	}
	vm = NewVM()
	vm.Debug = true
	db.addr = addr1
	sendRewardBlockList, isRetry, err := vm.Run(db, block17, nil)
	block17DataGas, _ := util.DataGasCost(sendRewardBlockList[0].AccountBlock.Data)
	reward := new(big.Int).Mul(big.NewInt(2), new(big.Int).Div(viteTotalSupply, big.NewInt(1051200000)))
	block17DataExpected, _ := contracts.ABIRegister.PackMethod(contracts.MethodNameReward, types.SNAPSHOT_GID, nodeName, addr7, snapshot6.Height-60*30, snapshot2.Height, reward)
	if len(sendRewardBlockList) != 1 || isRetry || err != nil ||
		sendRewardBlockList[0].AccountBlock.Quota != block17DataGas+83200+200*778 ||
		!bytes.Equal(sendRewardBlockList[0].AccountBlock.Data, block17DataExpected) {
		t.Fatalf("send reward transaction error")
	}
	db.accountBlockMap[addr1][hash17] = sendRewardBlockList[0].AccountBlock

	hash25 := types.DataHash([]byte{2, 5})
	block25 := &ledger.AccountBlock{
		Height:         5,
		AccountAddress: addr2,
		BlockType:      ledger.BlockTypeReceive,
		PrevHash:       hash24,
		FromBlockHash:  hash17,
		SnapshotHash:   snapshot6.Hash,
		Timestamp:      &blockTime,
	}
	vm = NewVM()
	vm.Debug = true
	db.addr = addr2
	receiveRewardBlockList, isRetry, err := vm.Run(db, block25, sendRewardBlockList[0].AccountBlock)
	registrationData, _ = contracts.ABIRegister.PackVariable(contracts.VariableNameRegistration, nodeName, addr6, addr1, helper.Big0, uint64(0), snapshot6.Height-60*30, snapshot5.Height)
	if len(receiveRewardBlockList) != 2 || isRetry || err != nil ||
		db.balanceMap[addr2][ledger.ViteTokenId].Cmp(helper.Big0) != 0 ||
		db.balanceMap[addr1][ledger.ViteTokenId].Cmp(viteTotalSupply) != 0 ||
		!bytes.Equal(db.storageMap[addr2][string(locHashRegister.Bytes())], registrationData) ||
		receiveRewardBlockList[0].AccountBlock.Quota != 0 ||
		receiveRewardBlockList[1].AccountBlock.Quota != 0 ||
		receiveRewardBlockList[1].AccountBlock.Height != 6 ||
		receiveRewardBlockList[1].AccountBlock.AccountAddress != addr2 ||
		receiveRewardBlockList[1].AccountBlock.ToAddress != addr7 ||
		receiveRewardBlockList[1].AccountBlock.BlockType != ledger.BlockTypeSendReward {
		t.Fatalf("receive reward transaction error")
	}
	db.accountBlockMap[addr2][hash25] = receiveRewardBlockList[0].AccountBlock
	hash26 := types.DataHash([]byte{2, 6})
	db.accountBlockMap[addr2][hash26] = receiveRewardBlockList[1].AccountBlock

	hash71 := types.DataHash([]byte{7, 1})
	block71 := &ledger.AccountBlock{
		Height:         1,
		AccountAddress: addr7,
		BlockType:      ledger.BlockTypeReceive,
		FromBlockHash:  hash25,
		SnapshotHash:   snapshot6.Hash,
		Timestamp:      &blockTime,
		Nonce:          []byte{1},
	}
	vm = NewVM()
	vm.Debug = true
	db.addr = addr7
	receiveRewardRefundBlockList, isRetry, err := vm.Run(db, block71, receiveRewardBlockList[1].AccountBlock)
	if len(receiveRewardRefundBlockList) != 1 || isRetry || err != nil ||
		db.balanceMap[addr2][ledger.ViteTokenId].Cmp(helper.Big0) != 0 ||
		db.balanceMap[addr7][ledger.ViteTokenId].Cmp(reward) != 0 ||
		receiveRewardRefundBlockList[0].AccountBlock.Quota != 21000 {
		t.Fatalf("receive reward refund transaction error")
	}
	db.accountBlockMap[addr7][hash71] = receiveRewardRefundBlockList[0].AccountBlock
}

func TestContractsVote(t *testing.T) {
	// prepare db
	viteTotalSupply := new(big.Int).Mul(big.NewInt(2e6), big.NewInt(1e18))
	db, addr1, _, hash12, snapshot2, _ := prepareDb(viteTotalSupply)
	blockTime := time.Now()
	// vote
	addr3 := contracts.AddressVote
	nodeName := "super1"
	block13Data, _ := contracts.ABIVote.PackMethod(contracts.MethodNameVote, types.SNAPSHOT_GID, nodeName)
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
		SnapshotHash:   snapshot2.Hash,
		Timestamp:      &blockTime,
	}
	vm := NewVM()
	vm.Debug = true
	db.addr = addr1
	block13DataGas, _ := util.DataGasCost(block13.Data)
	sendVoteBlockList, isRetry, err := vm.Run(db, block13, nil)
	if len(sendVoteBlockList) != 1 || isRetry || err != nil ||
		sendVoteBlockList[0].AccountBlock.Quota != block13DataGas+62000 {
		t.Fatalf("send vote transaction error")
	}
	db.accountBlockMap[addr1][hash13] = sendVoteBlockList[0].AccountBlock

	hash31 := types.DataHash([]byte{3, 1})
	block31 := &ledger.AccountBlock{
		Height:         1,
		AccountAddress: addr3,
		BlockType:      ledger.BlockTypeReceive,
		FromBlockHash:  hash13,
		SnapshotHash:   snapshot2.Hash,
		Timestamp:      &blockTime,
	}
	vm = NewVM()
	vm.Debug = true
	db.addr = addr3
	receiveVoteBlockList, isRetry, err := vm.Run(db, block31, sendVoteBlockList[0].AccountBlock)
	locHashVote, _ := types.BytesToHash(contracts.GetVoteKey(addr1, types.SNAPSHOT_GID))
	voteData, _ := contracts.ABIVote.PackVariable(contracts.VariableNameVoteStatus, nodeName)
	if len(receiveVoteBlockList) != 1 || isRetry || err != nil ||
		!bytes.Equal(db.storageMap[addr3][string(locHashVote.Bytes())], voteData) ||
		receiveVoteBlockList[0].AccountBlock.Quota != 0 {
		t.Fatalf("receive vote transaction error")
	}
	db.accountBlockMap[addr3] = make(map[types.Hash]*ledger.AccountBlock)
	db.accountBlockMap[addr3][hash31] = receiveVoteBlockList[0].AccountBlock

	addr4, _ := types.BytesToAddress(helper.HexToBytes("e5bf58cacfb74cf8c49a1d5e59d3919c9a4cb9ed"))
	db.accountBlockMap[addr4] = make(map[types.Hash]*ledger.AccountBlock)
	nodeName2 := "super2"
	block14Data, _ := contracts.ABIVote.PackMethod(contracts.MethodNameVote, types.SNAPSHOT_GID, nodeName2)
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
		SnapshotHash:   snapshot2.Hash,
		Timestamp:      &blockTime,
	}
	vm = NewVM()
	vm.Debug = true
	db.addr = addr1
	sendVoteBlockList2, isRetry, err := vm.Run(db, block14, nil)
	block14DataGas, _ := util.DataGasCost(block14.Data)
	if len(sendVoteBlockList2) != 1 || isRetry || err != nil ||
		sendVoteBlockList2[0].AccountBlock.Quota != block14DataGas+62000 {
		t.Fatalf("send vote transaction 2 error")
	}
	db.accountBlockMap[addr1][hash14] = sendVoteBlockList2[0].AccountBlock

	hash32 := types.DataHash([]byte{3, 2})
	block32 := &ledger.AccountBlock{
		Height:         2,
		AccountAddress: addr3,
		BlockType:      ledger.BlockTypeReceive,
		PrevHash:       hash31,
		FromBlockHash:  hash14,
		SnapshotHash:   snapshot2.Hash,
		Timestamp:      &blockTime,
	}
	vm = NewVM()
	vm.Debug = true
	db.addr = addr3
	receiveVoteBlockList2, isRetry, err := vm.Run(db, block32, sendVoteBlockList2[0].AccountBlock)
	voteData, _ = contracts.ABIVote.PackVariable(contracts.VariableNameVoteStatus, nodeName2)
	if len(receiveVoteBlockList2) != 1 || isRetry || err != nil ||
		!bytes.Equal(db.storageMap[addr3][string(locHashVote.Bytes())], voteData) ||
		receiveVoteBlockList2[0].AccountBlock.Quota != 0 {
		t.Fatalf("receive vote transaction 2 error")
	}
	db.accountBlockMap[addr3][hash32] = receiveVoteBlockList2[0].AccountBlock

	// get contracts data
	db.addr = contracts.AddressVote
	if voteList := contracts.GetVoteList(db, types.SNAPSHOT_GID); len(voteList) != 1 || voteList[0].NodeName != nodeName2 {
		t.Fatalf("get vote list failed")
	}

	// cancel vote
	block15Data, _ := contracts.ABIVote.PackMethod(contracts.MethodNameCancelVote, types.SNAPSHOT_GID)
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
		SnapshotHash:   snapshot2.Hash,
		Timestamp:      &blockTime,
	}
	vm = NewVM()
	vm.Debug = true
	db.addr = addr1
	sendCancelVoteBlockList, isRetry, err := vm.Run(db, block15, nil)
	if len(sendCancelVoteBlockList) != 1 || isRetry || err != nil ||
		sendCancelVoteBlockList[0].AccountBlock.Quota != 62464 {
		t.Fatalf("send cancel vote transaction error")
	}
	db.accountBlockMap[addr1][hash15] = sendCancelVoteBlockList[0].AccountBlock

	hash33 := types.DataHash([]byte{3, 3})
	block33 := &ledger.AccountBlock{
		Height:         3,
		AccountAddress: addr3,
		BlockType:      ledger.BlockTypeReceive,
		PrevHash:       hash32,
		FromBlockHash:  hash15,
		SnapshotHash:   snapshot2.Hash,
		Timestamp:      &blockTime,
	}
	vm = NewVM()
	vm.Debug = true
	db.addr = addr3
	receiveCancelVoteBlockList, isRetry, err := vm.Run(db, block33, sendCancelVoteBlockList[0].AccountBlock)
	if len(receiveCancelVoteBlockList) != 1 || isRetry || err != nil ||
		len(db.storageMap[addr3][string(locHashVote.Bytes())]) != 0 ||
		receiveCancelVoteBlockList[0].AccountBlock.Quota != 0 {
		t.Fatalf("receive cancel vote transaction error")
	}
	db.accountBlockMap[addr3][hash33] = receiveCancelVoteBlockList[0].AccountBlock
}

func TestContractsPledge(t *testing.T) {
	// prepare db
	viteTotalSupply := new(big.Int).Mul(big.NewInt(2e6), big.NewInt(1e18))
	db, addr1, _, hash12, snapshot2, timestamp := prepareDb(viteTotalSupply)
	blockTime := time.Now()
	// pledge
	balance1 := new(big.Int).Set(viteTotalSupply)
	addr4, _, _ := types.CreateAddress()
	db.accountBlockMap[addr4] = make(map[types.Hash]*ledger.AccountBlock)
	addr5 := contracts.AddressPledge
	pledgeAmount := new(big.Int).Set(new(big.Int).Mul(big.NewInt(10), util.AttovPerVite))
	block13Data, err := contracts.ABIPledge.PackMethod(contracts.MethodNamePledge, addr4)
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
		SnapshotHash:   snapshot2.Hash,
		Timestamp:      &blockTime,
	}
	vm := NewVM()
	vm.Debug = true
	db.addr = addr1
	sendPledgeBlockList, isRetry, err := vm.Run(db, block13, nil)
	balance1.Sub(balance1, pledgeAmount)
	if len(sendPledgeBlockList) != 1 || isRetry || err != nil ||
		db.balanceMap[addr1][ledger.ViteTokenId].Cmp(balance1) != 0 ||
		sendPledgeBlockList[0].AccountBlock.Quota != 21000 {
		t.Fatalf("send pledge transaction error")
	}
	db.accountBlockMap[addr1][hash13] = sendPledgeBlockList[0].AccountBlock

	hash51 := types.DataHash([]byte{5, 1})
	block51 := &ledger.AccountBlock{
		Height:         1,
		AccountAddress: addr5,
		BlockType:      ledger.BlockTypeReceive,
		FromBlockHash:  hash13,
		SnapshotHash:   snapshot2.Hash,
		Timestamp:      &blockTime,
	}
	vm = NewVM()
	vm.Debug = true
	db.addr = addr5
	receivePledgeBlockList, isRetry, err := vm.Run(db, block51, sendPledgeBlockList[0].AccountBlock)
	beneficialKey := contracts.GetPledgeBeneficialKey(addr4)
	pledgeKey := contracts.GetPledgeKey(addr1, beneficialKey)
	withdrawHeight := snapshot2.Height + 3600*24*3
	if len(receivePledgeBlockList) != 1 || isRetry || err != nil ||
		!bytes.Equal(db.storageMap[addr5][string(pledgeKey)], helper.JoinBytes(helper.LeftPadBytes(pledgeAmount.Bytes(), helper.WordSize), helper.LeftPadBytes(new(big.Int).SetUint64(withdrawHeight).Bytes(), helper.WordSize))) ||
		!bytes.Equal(db.storageMap[addr5][string(beneficialKey)], helper.LeftPadBytes(pledgeAmount.Bytes(), helper.WordSize)) ||
		db.balanceMap[addr5][ledger.ViteTokenId].Cmp(pledgeAmount) != 0 ||
		receivePledgeBlockList[0].AccountBlock.Quota != 0 {
		t.Fatalf("receive pledge transaction error")
	}
	db.accountBlockMap[addr5] = make(map[types.Hash]*ledger.AccountBlock)
	db.accountBlockMap[addr5][hash51] = receivePledgeBlockList[0].AccountBlock

	block14Data, _ := contracts.ABIPledge.PackMethod(contracts.MethodNamePledge, addr4)
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
		SnapshotHash:   snapshot2.Hash,
		Timestamp:      &blockTime,
	}
	vm = NewVM()
	vm.Debug = true
	db.addr = addr1
	sendPledgeBlockList2, isRetry, err := vm.Run(db, block14, nil)
	balance1.Sub(balance1, pledgeAmount)
	if len(sendPledgeBlockList2) != 1 || isRetry || err != nil ||
		db.balanceMap[addr1][ledger.ViteTokenId].Cmp(balance1) != 0 ||
		sendPledgeBlockList2[0].AccountBlock.Quota != 21000 {
		t.Fatalf("send pledge transaction 2 error")
	}
	db.accountBlockMap[addr1][hash14] = sendPledgeBlockList2[0].AccountBlock

	hash52 := types.DataHash([]byte{5, 2})
	block52 := &ledger.AccountBlock{
		Height:         2,
		AccountAddress: addr5,
		BlockType:      ledger.BlockTypeReceive,
		PrevHash:       hash51,
		FromBlockHash:  hash14,
		SnapshotHash:   snapshot2.Hash,
		Timestamp:      &blockTime,
	}
	vm = NewVM()
	vm.Debug = true
	db.addr = addr5
	receivePledgeBlockList2, isRetry, err := vm.Run(db, block52, sendPledgeBlockList2[0].AccountBlock)
	newPledgeAmount := new(big.Int).Add(pledgeAmount, pledgeAmount)
	if len(receivePledgeBlockList2) != 1 || isRetry || err != nil ||
		!bytes.Equal(db.storageMap[addr5][string(pledgeKey)], helper.JoinBytes(helper.LeftPadBytes(newPledgeAmount.Bytes(), helper.WordSize), helper.LeftPadBytes(new(big.Int).SetUint64(withdrawHeight).Bytes(), helper.WordSize))) ||
		!bytes.Equal(db.storageMap[addr5][string(beneficialKey)], helper.LeftPadBytes(newPledgeAmount.Bytes(), helper.WordSize)) ||
		db.balanceMap[addr5][ledger.ViteTokenId].Cmp(newPledgeAmount) != 0 ||
		receivePledgeBlockList2[0].AccountBlock.Quota != 0 {
		t.Fatalf("receive pledge transaction 2 error")
	}
	db.accountBlockMap[addr5][hash52] = receivePledgeBlockList2[0].AccountBlock

	// get contracts data
	db.addr = contracts.AddressPledge
	if pledgeAmount := contracts.GetPledgeBeneficialAmount(db, addr4); pledgeAmount.Cmp(newPledgeAmount) != 0 {
		t.Fatalf("get pledge beneficial amount failed")
	}

	if pledgeInfoList := contracts.GetPledgeInfoList(db, addr1); len(pledgeInfoList) != 1 || pledgeInfoList[0].BeneficialAddr != addr4 {
		t.Fatalf("get pledge amount failed")
	}

	// cancel pledge
	for i := uint64(1); i <= uint64(3600*24*3); i++ {
		timei := time.Unix(timestamp+100+int64(i), 0)
		snapshoti := &ledger.SnapshotBlock{Height: 2 + i, Timestamp: &timei, Hash: types.DataHash([]byte{10, byte(2 + i)})}
		db.snapshotBlockList = append(db.snapshotBlockList, snapshoti)
	}
	currentSnapshot := db.snapshotBlockList[len(db.snapshotBlockList)-1]

	block15Data, _ := contracts.ABIPledge.PackMethod(contracts.MethodNameCancelPledge, addr4, pledgeAmount)
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
		SnapshotHash:   currentSnapshot.Hash,
		Timestamp:      &blockTime,
	}
	vm = NewVM()
	vm.Debug = true
	db.addr = addr1
	sendCancelPledgeBlockList, isRetry, err := vm.Run(db, block15, nil)
	block15DataGasCost, _ := util.DataGasCost(sendCancelPledgeBlockList[0].AccountBlock.Data)
	if len(sendCancelPledgeBlockList) != 1 || isRetry || err != nil ||
		sendCancelPledgeBlockList[0].AccountBlock.Quota != block15DataGasCost+103400 {
		t.Fatalf("send cancel pledge transaction error")
	}
	db.accountBlockMap[addr1][hash15] = sendCancelPledgeBlockList[0].AccountBlock

	hash53 := types.DataHash([]byte{5, 3})
	block53 := &ledger.AccountBlock{
		Height:         3,
		AccountAddress: addr5,
		BlockType:      ledger.BlockTypeReceive,
		PrevHash:       hash52,
		FromBlockHash:  hash15,
		SnapshotHash:   currentSnapshot.Hash,
		Timestamp:      &blockTime,
	}
	vm = NewVM()
	vm.Debug = true
	db.addr = addr5
	receiveCancelPledgeBlockList, isRetry, err := vm.Run(db, block53, sendCancelPledgeBlockList[0].AccountBlock)
	if len(receiveCancelPledgeBlockList) != 2 || isRetry || err != nil ||
		receiveCancelPledgeBlockList[1].AccountBlock.Height != 4 ||
		!bytes.Equal(db.storageMap[addr5][string(pledgeKey)], helper.JoinBytes(helper.LeftPadBytes(pledgeAmount.Bytes(), helper.WordSize), helper.LeftPadBytes(new(big.Int).SetUint64(withdrawHeight).Bytes(), helper.WordSize))) ||
		!bytes.Equal(db.storageMap[addr5][string(beneficialKey)], helper.LeftPadBytes(pledgeAmount.Bytes(), helper.WordSize)) ||
		db.balanceMap[addr5][ledger.ViteTokenId].Cmp(pledgeAmount) != 0 ||
		receiveCancelPledgeBlockList[0].AccountBlock.Quota != 0 ||
		receiveCancelPledgeBlockList[1].AccountBlock.Quota != 0 {
		t.Fatalf("receive cancel pledge transaction error")
	}
	db.accountBlockMap[addr5][hash53] = receiveCancelPledgeBlockList[0].AccountBlock
	hash54 := types.DataHash([]byte{5, 4})
	db.accountBlockMap[addr5][hash54] = receiveCancelPledgeBlockList[1].AccountBlock

	hash16 := types.DataHash([]byte{1, 6})
	block16 := &ledger.AccountBlock{
		Height:         6,
		AccountAddress: addr1,
		BlockType:      ledger.BlockTypeReceive,
		PrevHash:       hash15,
		FromBlockHash:  hash54,
		SnapshotHash:   currentSnapshot.Hash,
		Timestamp:      &blockTime,
	}
	vm = NewVM()
	vm.Debug = true
	db.addr = addr1
	receiveCancelPledgeRefundBlockList, isRetry, err := vm.Run(db, block16, receiveCancelPledgeBlockList[1].AccountBlock)
	balance1.Add(balance1, pledgeAmount)
	if len(receiveCancelPledgeRefundBlockList) != 1 || isRetry || err != nil ||
		db.balanceMap[addr1][ledger.ViteTokenId].Cmp(balance1) != 0 ||
		receiveCancelPledgeRefundBlockList[0].AccountBlock.Quota != 21000 {
		t.Fatalf("receive cancel pledge refund transaction error")
	}
	db.accountBlockMap[addr1][hash16] = receiveCancelPledgeRefundBlockList[0].AccountBlock

	block17Data, _ := contracts.ABIPledge.PackMethod(contracts.MethodNameCancelPledge, addr4, pledgeAmount)
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
		SnapshotHash:   currentSnapshot.Hash,
		Timestamp:      &blockTime,
	}
	vm = NewVM()
	vm.Debug = true
	db.addr = addr1
	sendCancelPledgeBlockList2, isRetry, err := vm.Run(db, block17, nil)
	block17DataGas, _ := util.DataGasCost(sendCancelPledgeBlockList2[0].AccountBlock.Data)
	if len(sendCancelPledgeBlockList2) != 1 || isRetry || err != nil ||
		sendCancelPledgeBlockList2[0].AccountBlock.Quota != block17DataGas+103400 {
		t.Fatalf("send cancel pledge transaction 2 error")
	}
	db.accountBlockMap[addr1][hash17] = sendCancelPledgeBlockList2[0].AccountBlock

	hash55 := types.DataHash([]byte{5, 5})
	block55 := &ledger.AccountBlock{
		Height:         5,
		AccountAddress: addr5,
		BlockType:      ledger.BlockTypeReceive,
		PrevHash:       hash54,
		FromBlockHash:  hash17,
		SnapshotHash:   currentSnapshot.Hash,
		Timestamp:      &blockTime,
	}
	vm = NewVM()
	vm.Debug = true
	db.addr = addr5
	receiveCancelPledgeBlockList2, isRetry, err := vm.Run(db, block55, sendCancelPledgeBlockList2[0].AccountBlock)
	if len(receiveCancelPledgeBlockList2) != 2 || isRetry || err != nil ||
		receiveCancelPledgeBlockList2[1].AccountBlock.Height != 6 ||
		len(db.storageMap[addr5][string(pledgeKey)]) != 0 ||
		len(db.storageMap[addr5][string(beneficialKey)]) != 0 ||
		db.balanceMap[addr5][ledger.ViteTokenId].Cmp(helper.Big0) != 0 ||
		receiveCancelPledgeBlockList2[0].AccountBlock.Quota != 0 ||
		receiveCancelPledgeBlockList2[1].AccountBlock.Quota != 0 {
		t.Fatalf("receive cancel pledge transaction 2 error")
	}
	db.accountBlockMap[addr5][hash55] = receiveCancelPledgeBlockList2[0].AccountBlock
	hash56 := types.DataHash([]byte{5, 6})
	db.accountBlockMap[addr5][hash56] = receiveCancelPledgeBlockList2[1].AccountBlock

	hash18 := types.DataHash([]byte{1, 8})
	block18 := &ledger.AccountBlock{
		Height:         8,
		AccountAddress: addr1,
		BlockType:      ledger.BlockTypeReceive,
		PrevHash:       hash18,
		FromBlockHash:  hash56,
		SnapshotHash:   currentSnapshot.Hash,
		Timestamp:      &blockTime,
	}
	vm = NewVM()
	vm.Debug = true
	db.addr = addr1
	balance1.Add(balance1, pledgeAmount)
	receiveCancelPledgeRefundBlockList2, isRetry, err := vm.Run(db, block18, receiveCancelPledgeBlockList2[1].AccountBlock)
	if len(receiveCancelPledgeRefundBlockList2) != 1 || isRetry || err != nil ||
		db.balanceMap[addr1][ledger.ViteTokenId].Cmp(balance1) != 0 ||
		receiveCancelPledgeRefundBlockList2[0].AccountBlock.Quota != 21000 {
		t.Fatalf("receive cancel pledge refund transaction 2 error")
	}
	db.accountBlockMap[addr1][hash18] = receiveCancelPledgeRefundBlockList2[0].AccountBlock
}

func TestContractsConsensusGroup(t *testing.T) {
	viteTotalSupply := new(big.Int).Mul(big.NewInt(1e9), util.AttovPerVite)
	db, addr1, _, hash12, snapshot2, timestamp := prepareDb(viteTotalSupply)
	blockTime := time.Now()

	pledgeAmount := new(big.Int).Mul(big.NewInt(1000), util.AttovPerVite)
	pledgeHeight := uint64(3600 * 24 * 3)

	addr2 := contracts.AddressConsensusGroup
	block13Data, _ := contracts.ABIConsensusGroup.PackMethod(contracts.MethodNameCreateConsensusGroup,
		types.Gid{},
		uint8(25),
		int64(3),
		int64(1),
		uint8(2),
		uint8(50),
		ledger.ViteTokenId,
		uint8(1),
		helper.JoinBytes(helper.LeftPadBytes(big.NewInt(1e18).Bytes(), helper.WordSize), helper.LeftPadBytes(ledger.ViteTokenId.Bytes(), helper.WordSize), helper.LeftPadBytes(big.NewInt(259200).Bytes(), helper.WordSize)),
		uint8(1),
		[]byte{})
	hash13 := types.DataHash([]byte{1, 3})
	block13 := &ledger.AccountBlock{
		Height:         3,
		ToAddress:      addr2,
		AccountAddress: addr1,
		BlockType:      ledger.BlockTypeSendCall,
		Fee:            big.NewInt(0),
		PrevHash:       hash12,
		Amount:         new(big.Int).Set(pledgeAmount),
		TokenId:        ledger.ViteTokenId,
		Data:           block13Data,
		SnapshotHash:   snapshot2.Hash,
		Timestamp:      &blockTime,
	}
	vm := NewVM()
	vm.Debug = true
	db.addr = addr1
	sendCreateConsensusGroupBlockList, isRetry, err := vm.Run(db, block13, nil)
	balance1 := new(big.Int).Sub(viteTotalSupply, pledgeAmount)
	quota13, _ := util.DataGasCost(sendCreateConsensusGroupBlockList[0].AccountBlock.Data)
	if len(sendCreateConsensusGroupBlockList) != 1 || isRetry || err != nil ||
		sendCreateConsensusGroupBlockList[0].AccountBlock.Quota != quota13+62200 ||
		!helper.AllZero(sendCreateConsensusGroupBlockList[0].AccountBlock.Data[4:26]) || helper.AllZero(sendCreateConsensusGroupBlockList[0].AccountBlock.Data[26:36]) ||
		block13.Fee.Cmp(helper.Big0) != 0 ||
		db.balanceMap[addr1][ledger.ViteTokenId].Cmp(balance1) != 0 {
		t.Fatalf("send create consensus group transaction error")
	}
	db.accountBlockMap[addr1][hash13] = sendCreateConsensusGroupBlockList[0].AccountBlock

	hash21 := types.DataHash([]byte{2, 1})
	block21 := &ledger.AccountBlock{
		Height:         1,
		AccountAddress: addr2,
		BlockType:      ledger.BlockTypeReceive,
		FromBlockHash:  hash13,
		SnapshotHash:   snapshot2.Hash,
		Timestamp:      &blockTime,
	}
	vm = NewVM()
	vm.Debug = true
	locHash, _ := types.BytesToHash(sendCreateConsensusGroupBlockList[0].AccountBlock.Data[4:36])
	db.addr = addr2
	receiveCreateConsensusGroupBlockList, isRetry, err := vm.Run(db, block21, sendCreateConsensusGroupBlockList[0].AccountBlock)
	groupInfo, _ := contracts.ABIConsensusGroup.PackVariable(contracts.VariableNameConsensusGroupInfo,
		uint8(25),
		int64(3),
		int64(1),
		uint8(2),
		uint8(50),
		ledger.ViteTokenId,
		uint8(1),
		helper.JoinBytes(helper.LeftPadBytes(big.NewInt(1e18).Bytes(), helper.WordSize), helper.LeftPadBytes(ledger.ViteTokenId.Bytes(), helper.WordSize), helper.LeftPadBytes(big.NewInt(259200).Bytes(), helper.WordSize)),
		uint8(1),
		[]byte{},
		addr1,
		pledgeAmount,
		snapshot2.Height+pledgeHeight)
	if len(receiveCreateConsensusGroupBlockList) != 1 || isRetry || err != nil ||
		db.balanceMap[addr2][ledger.ViteTokenId].Cmp(pledgeAmount) != 0 ||
		!bytes.Equal(db.storageMap[addr2][string(locHash.Bytes())], groupInfo) ||
		receiveCreateConsensusGroupBlockList[0].AccountBlock.Quota != 0 {
		t.Fatalf("receive create consensus group transaction error")
	}
	db.accountBlockMap[addr2] = make(map[types.Hash]*ledger.AccountBlock)
	db.accountBlockMap[addr2][hash21] = receiveCreateConsensusGroupBlockList[0].AccountBlock

	// get contracts data
	gid, _ := types.BytesToGid(sendCreateConsensusGroupBlockList[0].AccountBlock.Data[26:36])
	db.addr = contracts.AddressConsensusGroup
	if groupInfo := contracts.GetConsensusGroup(db, gid); groupInfo == nil || groupInfo.NodeCount != 25 {
		t.Fatalf("get group info failed")
	}
	if groupInfoList := contracts.GetActiveConsensusGroupList(db); len(groupInfoList) != 3 || groupInfoList[0].NodeCount != 25 {
		t.Fatalf("get group info list failed")
	}

	// cancel consensus group
	for i := uint64(1); i <= pledgeHeight; i++ {
		timei := time.Unix(timestamp+2+int64(i), 0)
		snapshoti := &ledger.SnapshotBlock{Height: 2 + i, Timestamp: &timei, Hash: types.DataHash([]byte{10, byte(2 + i)})}
		db.snapshotBlockList = append(db.snapshotBlockList, snapshoti)
	}
	newSnapshot := db.snapshotBlockList[len(db.snapshotBlockList)-1]

	block14Data, _ := contracts.ABIConsensusGroup.PackMethod(contracts.MethodNameCancelConsensusGroup, gid)
	hash14 := types.DataHash([]byte{1, 4})
	block14 := &ledger.AccountBlock{
		Height:         4,
		ToAddress:      addr2,
		AccountAddress: addr1,
		BlockType:      ledger.BlockTypeSendCall,
		Fee:            big.NewInt(0),
		PrevHash:       hash13,
		Amount:         big.NewInt(0),
		TokenId:        ledger.ViteTokenId,
		Data:           block14Data,
		SnapshotHash:   newSnapshot.Hash,
		Timestamp:      &blockTime,
	}
	vm = NewVM()
	vm.Debug = true
	db.addr = addr1
	sendCancelConsensusGroupBlockList, isRetry, err := vm.Run(db, block14, nil)
	quota14, _ := util.DataGasCost(block14.Data)
	if len(sendCancelConsensusGroupBlockList) != 1 || isRetry || err != nil ||
		sendCancelConsensusGroupBlockList[0].AccountBlock.Quota != quota14+83200 ||
		db.balanceMap[addr1][ledger.ViteTokenId].Cmp(balance1) != 0 {
		t.Fatalf("send cancel consensus group transaction error")
	}
	db.accountBlockMap[addr1][hash14] = sendCancelConsensusGroupBlockList[0].AccountBlock

	hash22 := types.DataHash([]byte{2, 2})
	block22 := &ledger.AccountBlock{
		Height:         2,
		AccountAddress: addr2,
		BlockType:      ledger.BlockTypeReceive,
		FromBlockHash:  hash14,
		SnapshotHash:   newSnapshot.Hash,
		Timestamp:      &blockTime,
	}
	vm = NewVM()
	vm.Debug = true
	db.addr = addr2
	receiveCancelConsensusGroupBlockList, isRetry, err := vm.Run(db, block22, sendCancelConsensusGroupBlockList[0].AccountBlock)
	groupInfo, _ = contracts.ABIConsensusGroup.PackVariable(contracts.VariableNameConsensusGroupInfo,
		uint8(25),
		int64(3),
		int64(1),
		uint8(2),
		uint8(50),
		ledger.ViteTokenId,
		uint8(1),
		helper.JoinBytes(helper.LeftPadBytes(big.NewInt(1e18).Bytes(), helper.WordSize), helper.LeftPadBytes(ledger.ViteTokenId.Bytes(), helper.WordSize), helper.LeftPadBytes(big.NewInt(259200).Bytes(), helper.WordSize)),
		uint8(1),
		[]byte{},
		addr1,
		helper.Big0,
		uint64(0))
	if len(receiveCancelConsensusGroupBlockList) != 2 || isRetry || err != nil ||
		db.balanceMap[addr2][ledger.ViteTokenId].Sign() != 0 ||
		!bytes.Equal(db.storageMap[addr2][string(locHash.Bytes())], groupInfo) ||
		receiveCancelConsensusGroupBlockList[0].AccountBlock.Quota != 0 ||
		receiveCancelConsensusGroupBlockList[1].AccountBlock.Height != 3 ||
		receiveCancelConsensusGroupBlockList[1].AccountBlock.Quota != 0 ||
		receiveCancelConsensusGroupBlockList[1].AccountBlock.Amount.Cmp(pledgeAmount) != 0 ||
		!util.IsViteToken(receiveCancelConsensusGroupBlockList[1].AccountBlock.TokenId) {
		t.Fatalf("receive cancel consensus group transaction error")
	}
	db.accountBlockMap[addr2][hash22] = receiveCancelConsensusGroupBlockList[0].AccountBlock
	hash23 := types.DataHash([]byte{2, 3})
	db.accountBlockMap[addr2][hash23] = receiveCancelConsensusGroupBlockList[1].AccountBlock

	hash15 := types.DataHash([]byte{1, 5})
	block15 := &ledger.AccountBlock{
		Height:         5,
		AccountAddress: addr1,
		BlockType:      ledger.BlockTypeReceive,
		FromBlockHash:  hash23,
		SnapshotHash:   newSnapshot.Hash,
		Timestamp:      &blockTime,
	}
	vm = NewVM()
	vm.Debug = true
	db.addr = addr1
	receiveCancelConsensusGroupRefundBlockList, isRetry, err := vm.Run(db, block15, receiveCancelConsensusGroupBlockList[1].AccountBlock)
	balance1.Add(balance1, pledgeAmount)
	if len(receiveCancelConsensusGroupRefundBlockList) != 1 || isRetry || err != nil ||
		db.balanceMap[addr1][ledger.ViteTokenId].Cmp(balance1) != 0 ||
		receiveCancelConsensusGroupRefundBlockList[0].AccountBlock.Quota != 21000 {
		t.Fatalf("receive cancel consensus group refund transaction error")
	}
	db.accountBlockMap[addr1][hash15] = receiveCancelConsensusGroupBlockList[0].AccountBlock

	// get contracts data
	db.addr = contracts.AddressConsensusGroup
	if groupInfo := contracts.GetConsensusGroup(db, gid); groupInfo == nil || groupInfo.NodeCount != 25 {
		t.Fatalf("get group info failed")
	}
	if groupInfoList := contracts.GetActiveConsensusGroupList(db); len(groupInfoList) != 2 || groupInfoList[0].NodeCount != 25 {
		t.Fatalf("get active group info list failed")
	}

	// recreate consensus group
	block16Data, _ := contracts.ABIConsensusGroup.PackMethod(contracts.MethodNameReCreateConsensusGroup, gid)
	hash16 := types.DataHash([]byte{1, 6})
	block16 := &ledger.AccountBlock{
		Height:         6,
		ToAddress:      addr2,
		AccountAddress: addr1,
		BlockType:      ledger.BlockTypeSendCall,
		PrevHash:       hash13,
		Amount:         pledgeAmount,
		TokenId:        ledger.ViteTokenId,
		Data:           block16Data,
		SnapshotHash:   newSnapshot.Hash,
		Timestamp:      &blockTime,
	}
	vm = NewVM()
	vm.Debug = true
	db.addr = addr1
	sendRecreateConsensusGroupBlockList, isRetry, err := vm.Run(db, block16, nil)
	quota16, _ := util.DataGasCost(block16.Data)
	balance1.Sub(balance1, pledgeAmount)
	if len(sendRecreateConsensusGroupBlockList) != 1 || isRetry || err != nil ||
		sendRecreateConsensusGroupBlockList[0].AccountBlock.Quota != quota16+62200 ||
		db.balanceMap[addr1][ledger.ViteTokenId].Cmp(balance1) != 0 {
		t.Fatalf("send recreate consensus group transaction error")
	}
	db.accountBlockMap[addr1][hash16] = sendRecreateConsensusGroupBlockList[0].AccountBlock

	hash24 := types.DataHash([]byte{2, 4})
	block24 := &ledger.AccountBlock{
		Height:         4,
		AccountAddress: addr2,
		BlockType:      ledger.BlockTypeReceive,
		FromBlockHash:  hash16,
		SnapshotHash:   newSnapshot.Hash,
		Timestamp:      &blockTime,
	}
	vm = NewVM()
	vm.Debug = true
	db.addr = addr2
	receiveRecreateConsensusGroupBlockList, isRetry, err := vm.Run(db, block24, sendRecreateConsensusGroupBlockList[0].AccountBlock)
	groupInfo, _ = contracts.ABIConsensusGroup.PackVariable(contracts.VariableNameConsensusGroupInfo,
		uint8(25),
		int64(3),
		int64(1),
		uint8(2),
		uint8(50),
		ledger.ViteTokenId,
		uint8(1),
		helper.JoinBytes(helper.LeftPadBytes(big.NewInt(1e18).Bytes(), helper.WordSize), helper.LeftPadBytes(ledger.ViteTokenId.Bytes(), helper.WordSize), helper.LeftPadBytes(big.NewInt(259200).Bytes(), helper.WordSize)),
		uint8(1),
		[]byte{},
		addr1,
		pledgeAmount,
		newSnapshot.Height+pledgeHeight)
	if len(receiveRecreateConsensusGroupBlockList) != 1 || isRetry || err != nil ||
		db.balanceMap[addr2][ledger.ViteTokenId].Cmp(pledgeAmount) != 0 ||
		!bytes.Equal(db.storageMap[addr2][string(locHash.Bytes())], groupInfo) ||
		receiveRecreateConsensusGroupBlockList[0].AccountBlock.Quota != 0 {
		t.Fatalf("receive cancel consensus group transaction error")
	}
	db.accountBlockMap[addr2][hash24] = receiveRecreateConsensusGroupBlockList[0].AccountBlock

	// get contracts data
	db.addr = contracts.AddressConsensusGroup
	if groupInfo := contracts.GetConsensusGroup(db, gid); groupInfo == nil || groupInfo.NodeCount != 25 {
		t.Fatalf("get group info failed")
	}
	if groupInfoList := contracts.GetActiveConsensusGroupList(db); len(groupInfoList) != 3 || groupInfoList[0].NodeCount != 25 {
		t.Fatalf("get active group info list failed")
	}
}

func TestContractsMintage(t *testing.T) {
	// prepare db
	viteTotalSupply := new(big.Int).Mul(big.NewInt(2e6), big.NewInt(1e18))
	db, addr1, _, hash12, snapshot2, _ := prepareDb(viteTotalSupply)
	blockTime := time.Now()
	// mintage
	balance1 := new(big.Int).Set(viteTotalSupply)
	addr2 := contracts.AddressMintage
	tokenName := "test token"
	tokenSymbol := "t"
	totalSupply := big.NewInt(1e10)
	decimals := uint8(3)
	block13Data, err := contracts.ABIMintage.PackMethod(contracts.MethodNameMintage, types.TokenTypeId{}, tokenName, tokenSymbol, totalSupply, decimals)
	hash13 := types.DataHash([]byte{1, 3})
	block13 := &ledger.AccountBlock{
		Height:         3,
		ToAddress:      addr2,
		AccountAddress: addr1,
		Amount:         big.NewInt(0),
		TokenId:        ledger.ViteTokenId,
		BlockType:      ledger.BlockTypeSendCall,
		Fee:            big.NewInt(0),
		PrevHash:       hash12,
		Data:           block13Data,
		SnapshotHash:   snapshot2.Hash,
		Timestamp:      &blockTime,
	}
	vm := NewVM()
	vm.Debug = true
	db.addr = addr1
	sendMintageBlockList, isRetry, err := vm.Run(db, block13, nil)
	block13DataGas, _ := util.DataGasCost(sendMintageBlockList[0].AccountBlock.Data)
	balance1.Sub(balance1, new(big.Int).Mul(big.NewInt(1e3), util.AttovPerVite))
	if len(sendMintageBlockList) != 1 || isRetry || err != nil ||
		db.balanceMap[addr1][ledger.ViteTokenId].Cmp(balance1) != 0 ||
		sendMintageBlockList[0].AccountBlock.Fee.Cmp(new(big.Int).Mul(big.NewInt(1e3), util.AttovPerVite)) != 0 ||
		sendMintageBlockList[0].AccountBlock.Amount.Cmp(big.NewInt(0)) != 0 ||
		sendMintageBlockList[0].AccountBlock.Quota != block13DataGas+83200 {
		t.Fatalf("send mintage transaction error")
	}
	db.accountBlockMap[addr1][hash13] = sendMintageBlockList[0].AccountBlock

	hash21 := types.DataHash([]byte{2, 1})
	block21 := &ledger.AccountBlock{
		Height:         1,
		AccountAddress: addr2,
		BlockType:      ledger.BlockTypeReceive,
		FromBlockHash:  hash13,
		SnapshotHash:   snapshot2.Hash,
		Timestamp:      &blockTime,
	}
	vm = NewVM()
	vm.Debug = true
	db.addr = addr2
	receiveMintageBlockList, isRetry, err := vm.Run(db, block21, sendMintageBlockList[0].AccountBlock)
	tokenId, _ := types.BytesToTokenTypeId(sendMintageBlockList[0].AccountBlock.Data[26:36])
	key, _ := types.BytesToHash(sendMintageBlockList[0].AccountBlock.Data[4:36])
	tokenInfoData, _ := contracts.ABIMintage.PackVariable(contracts.VariableNameMintage, tokenName, tokenSymbol, totalSupply, decimals, addr1, big.NewInt(0), uint64(0))
	if len(receiveMintageBlockList) != 2 || isRetry || err != nil ||
		!bytes.Equal(db.storageMap[addr2][string(key.Bytes())], tokenInfoData) ||
		db.balanceMap[addr2][ledger.ViteTokenId].Cmp(helper.Big0) != 0 ||
		receiveMintageBlockList[0].AccountBlock.Quota != 0 {
		t.Fatalf("receive mintage transaction error")
	}
	db.accountBlockMap[addr2] = make(map[types.Hash]*ledger.AccountBlock)
	db.accountBlockMap[addr2][hash21] = receiveMintageBlockList[0].AccountBlock
	hash22 := types.DataHash([]byte{2, 2})
	db.accountBlockMap[addr2][hash22] = receiveMintageBlockList[1].AccountBlock

	hash14 := types.DataHash([]byte{1, 4})
	block14 := &ledger.AccountBlock{
		Height:         4,
		AccountAddress: addr1,
		BlockType:      ledger.BlockTypeReceive,
		FromBlockHash:  hash22,
		PrevHash:       hash13,
		SnapshotHash:   snapshot2.Hash,
		Timestamp:      &blockTime,
	}
	vm = NewVM()
	vm.Debug = true
	db.addr = addr1
	receiveMintageRewardBlockList, isRetry, err := vm.Run(db, block14, receiveMintageBlockList[1].AccountBlock)
	if len(receiveMintageRewardBlockList) != 1 || isRetry || err != nil ||
		db.balanceMap[addr1][tokenId].Cmp(totalSupply) != 0 ||
		receiveMintageRewardBlockList[0].AccountBlock.Quota != 21000 {
		t.Fatalf("receive mintage reward transaction error")
	}
	db.accountBlockMap[addr1][hash14] = receiveMintageRewardBlockList[0].AccountBlock

	// get contracts data
	db.addr = contracts.AddressMintage
	if tokenInfo := contracts.GetTokenById(db, tokenId); tokenInfo == nil || tokenInfo.TokenName != tokenName {
		t.Fatalf("get token by id failed")
	}
	if tokenMap := contracts.GetTokenMap(db); len(tokenMap) != 2 || tokenMap[tokenId].TokenName != tokenName {
		t.Fatalf("get token map failed")
	}
}

func TestCheckCreateConsensusGroupData(t *testing.T) {
	tests := []struct {
		data string
		err  error
	}{
		{"51891ff2000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000190000000000000000000000000000000000000000000000000000000000000003000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000032000000000000000000000000000000000000000000005649544520544f4b454e00000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000160000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000001e000000000000000000000000000000000000000000000000000000000000000600000000000000000000000000000000000000000000000000de0b6b3a7640000000000000000000000000000000000000000000000005649544520544f4b454e000000000000000000000000000000000000000000000000000000000003f4800000000000000000000000000000000000000000000000000000000000000000", nil},
		{"51891ff2000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000190000000000000000000000000000000000000000000000000000000000000003000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000032000000000000000000000000000000000000000000005649544520544f4b454e00000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000160000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000001e000000000000000000000000000000000000000000000000000000000000000600000000000000000000000000000000000000000000000000de0b6b3a7640000000000000000000000000000000000000000000000005649544520544f4b454e000000000000000000000000000000000000000000000000000000000003f4800000000000000000000000000000000000000000000000000000000000000040000000000000000000000000000000000000000000000000000000000000000a00000000000000000000000000000000000000000000aaaaaaaaaaaaaaaaaaaa", util.ErrInvalidData},
		{"51891ff2000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000190000000000000000000000000000000000000000000000000000000000000003000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000032000000000000000000000000000000000000000000005649544520544f4b454e00000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000160000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000001e000000000000000000000000000000000000000000000000000000000000000600000000000000000000000000000000000000000000000000de0b6b3a7640000000000000000000000000000000000000000000000005649544520544f4b454e000000000000000000000000000000000000000000000000000000000003f48000000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000005649544520544f4b454e", util.ErrInvalidData},
		{"51891ff2000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000190000000000000000000000000000000000000000000000000000000000000003000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000032000000000000000000000000000000000000000000005649544520544f4b454e00000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000160000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000001e000000000000000000000000000000000000000000000000000000000000000600000000000000000000000000000000000000000000000000de0b6b3a7640000000000000000000000000000000000000000000000005649544520544f4b454e000000000000000000000000000000000000000000000000000000000003f4700000000000000000000000000000000000000000000000000000000000000000", util.ErrInvalidData},
		{"51891ff2000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000190000000000000000000000000000000000000000000000000000000000000003000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000032000000000000000000000000000000000000000000005649544520544f4b454e00000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000160000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000001e000000000000000000000000000000000000000000000000000000000000000600000000000000000000000000000000000000000000000000de0b6b3a764000000000000000000000000000000000000000000000000aaaaaaaaaaaaaaaaaaaa000000000000000000000000000000000000000000000000000000000003f4800000000000000000000000000000000000000000000000000000000000000000", util.ErrInvalidData},
		{"51891ff2000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000190000000000000000000000000000000000000000000000000000000000000003000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000032000000000000000000000000000000000000000000005649544520544f4b454e00000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000160000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000001e000000000000000000000000000000000000000000000000000000000000000600000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000005649544520544f4b454e000000000000000000000000000000000000000000000000000000000003f4800000000000000000000000000000000000000000000000000000000000000000", util.ErrInvalidData},
		{"51891ff2000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000190000000000000000000000000000000000000000000000000000000000000003000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000032000000000000000000000000000000000000000000005649544520544f4b454e00000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000160000000000000000000000000000000000000000000000000000000000000000300000000000000000000000000000000000000000000000000000000000001e000000000000000000000000000000000000000000000000000000000000000600000000000000000000000000000000000000000000000000de0b6b3a7640000000000000000000000000000000000000000000000005649544520544f4b454e000000000000000000000000000000000000000000000000000000000003f4800000000000000000000000000000000000000000000000000000000000000000", util.ErrInvalidData},
		{"51891ff2000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000190000000000000000000000000000000000000000000000000000000000000003000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000032000000000000000000000000000000000000000000005649544520544f4b454e00000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000160000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000001e000000000000000000000000000000000000000000000000000000000000000600000000000000000000000000000000000000000000000000de0b6b3a7640000000000000000000000000000000000000000000000005649544520544f4b454e000000000000000000000000000000000000000000000000000000000003f4800000000000000000000000000000000000000000000000000000000000000000", util.ErrInvalidData},
		{"51891ff200000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000019000000000000000000000000000000000000000000000000000000000000000300000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000003200000000000000000000000000000000000000000000aaaaaaaaaaaaaaaaaaaa00000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000160000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000001e000000000000000000000000000000000000000000000000000000000000000600000000000000000000000000000000000000000000000000de0b6b3a7640000000000000000000000000000000000000000000000005649544520544f4b454e000000000000000000000000000000000000000000000000000000000003f4800000000000000000000000000000000000000000000000000000000000000000", util.ErrInvalidData},
		{"51891ff2000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000190000000000000000000000000000000000000000000000000000000000000003000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000018000000000000000000000000000000000000000000005649544520544f4b454e00000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000160000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000001e000000000000000000000000000000000000000000000000000000000000000600000000000000000000000000000000000000000000000000de0b6b3a7640000000000000000000000000000000000000000000000005649544520544f4b454e000000000000000000000000000000000000000000000000000000000003f4800000000000000000000000000000000000000000000000000000000000000000", util.ErrInvalidData},
		{"51891ff20000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001900000000000000000000000000000000000000000000000000000000000000030000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000001a0000000000000000000000000000000000000000000000000000000000000032000000000000000000000000000000000000000000005649544520544f4b454e00000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000160000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000001e000000000000000000000000000000000000000000000000000000000000000600000000000000000000000000000000000000000000000000de0b6b3a7640000000000000000000000000000000000000000000000005649544520544f4b454e000000000000000000000000000000000000000000000000000000000003f4800000000000000000000000000000000000000000000000000000000000000000", util.ErrInvalidData},
		{"51891ff200000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000019000000000000000000000000000000000000000000000000000000000000003d000000000000000000000000000000000000000000000000000000000000000a00000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000032000000000000000000000000000000000000000000005649544520544f4b454e00000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000160000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000001e000000000000000000000000000000000000000000000000000000000000000600000000000000000000000000000000000000000000000000de0b6b3a7640000000000000000000000000000000000000000000000005649544520544f4b454e000000000000000000000000000000000000000000000000000000000003f4800000000000000000000000000000000000000000000000000000000000000000", util.ErrInvalidData},
		{"51891ff2000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000190000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000025900000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000032000000000000000000000000000000000000000000005649544520544f4b454e00000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000160000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000001e000000000000000000000000000000000000000000000000000000000000000600000000000000000000000000000000000000000000000000de0b6b3a7640000000000000000000000000000000000000000000000005649544520544f4b454e000000000000000000000000000000000000000000000000000000000003f4800000000000000000000000000000000000000000000000000000000000000000", util.ErrInvalidData},
		{"51891ff2000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000190000000000000000000000000000000000000000000000000000000000000259000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000032000000000000000000000000000000000000000000005649544520544f4b454e00000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000160000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000001e000000000000000000000000000000000000000000000000000000000000000600000000000000000000000000000000000000000000000000de0b6b3a7640000000000000000000000000000000000000000000000005649544520544f4b454e000000000000000000000000000000000000000000000000000000000003f4800000000000000000000000000000000000000000000000000000000000000000", util.ErrInvalidData},
		{"51891ff2000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000670000000000000000000000000000000000000000000000000000000000000003000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000032000000000000000000000000000000000000000000005649544520544f4b454e00000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000160000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000001e000000000000000000000000000000000000000000000000000000000000000600000000000000000000000000000000000000000000000000de0b6b3a7640000000000000000000000000000000000000000000000005649544520544f4b454e000000000000000000000000000000000000000000000000000000000003f4800000000000000000000000000000000000000000000000000000000000000000", util.ErrInvalidData},
		{"51891ff2000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000003000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000032000000000000000000000000000000000000000000005649544520544f4b454e00000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000160000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000001e000000000000000000000000000000000000000000000000000000000000000600000000000000000000000000000000000000000000000000de0b6b3a7640000000000000000000000000000000000000000000000005649544520544f4b454e000000000000000000000000000000000000000000000000000000000003f4800000000000000000000000000000000000000000000000000000000000000000", util.ErrInvalidData},
	}
	db, _, _, _, _, _ := prepareDb(big.NewInt(1))
	for i, test := range tests {
		inputdata, _ := hex.DecodeString(test.data)
		param := new(contracts.ConsensusGroupInfo)
		err := contracts.ABIConsensusGroup.UnpackMethod(param, contracts.MethodNameCreateConsensusGroup, inputdata)
		if err != nil {
			t.Fatalf("unpack create consensus group param error, data: [%v]", test.data)
		}
		err = contracts.CheckCreateConsensusGroupData(db, param)
		if test.err != nil && err == nil {
			t.Logf("%v th check create consensus group data expected error", i)
		} else if test.err == nil && err != nil {
			t.Logf("%v th check create consensus group data unexpected error", i)
		}
	}
}

func TestCheckTokenInfo(t *testing.T) {
	tests := []struct {
		data   string
		err    error
		result bool
	}{
		{"00", util.ErrInvalidData, false},
		{"46d0ce8b000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000a000000000000000000000000000000000000000000000000000000000000000e00000000000000000000000000000000000000000033b2e3c9fd0803ce80000000000000000000000000000000000000000000000000000000000000000000012000000000000000000000000000000000000000000000000000000000000000956697465546f6b656e0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000956697465546f6b656e0000000000000000000000000000000000000000000000", nil, true},
		{"46d0ce8b000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000a000000000000000000000000000000000000000000000000000000000000000e00000000000000000000000000000000000000000033b2e3c9fd0803ce80000000000000000000000000000000000000000000000000000000000000000000012000000000000000000000000000000000000000000000000000000000000000956697465546f6b656e00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000009", util.ErrInvalidData, true},
		{"46d0ce8b000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000a000000000000000000000000000000000000000000000000000000000000000e00000000000000000000000000000000000000000033b2e3c9fd0803ce80000000000000000000000000000000000000000000000000000000000000000000012000000000000000000000000000000000000000000000000000000000000000956697465546f6b656e0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000956697465546f6b651F0000000000000000000000000000000000000000000000", nil, false},
		{"46d0ce8b000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000a000000000000000000000000000000000000000000000000000000000000000e00000000000000000000000000000000000000000033b2e3c9fd0803ce80000000000000000000000000000000000000000000000000000000000000000000012000000000000000000000000000000000000000000000000000000000000000956697465546f6b651F0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000956697465546f6b651F0000000000000000000000000000000000000000000000", nil, false},
		{"46d0ce8b000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000a000000000000000000000000000000000000000000000000000000000000000e00000000000000000000000000000000000000000033b2e3c9fd0803ce80000000000000000000000000000000000000000000000000000000000000000000012000000000000000000000000000000000000000000000000000000000000000956697465546f6b656e0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000a56697465546f6b656e0000000000000000000000000000000000000000000000", nil, false},
		{"46d0ce8b000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000a000000000000000000000000000000000000000000000000000000000000000e00000000000000000000000000000000000000000033b2e3c9fd0803ce8000000000000000000000000000000000000000000000000000000000000000000012e000000000000000000000000000000000000000000000000000000000000000956697465546f6b656e0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000956697465546f6b656e0000000000000000000000000000000000000000000000", nil, false},
	}
	for i, test := range tests {
		inputdata, _ := hex.DecodeString(test.data)
		param := new(contracts.ParamMintage)
		err := contracts.ABIMintage.UnpackMethod(param, contracts.MethodNameMintage, inputdata)
		if test.err != nil && err == nil {
			t.Logf("%v th expected error", i)
		} else if test.err == nil && err != nil {
			t.Logf("%v th unexpected error", i)
		} else if test.err == nil {
			err = contracts.CheckToken(*param)
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
	mintageData, err := contracts.ABIMintage.PackVariable(contracts.VariableNameMintage, tokenName, tokenSymbol, totalSupply, decimals, viteAddress, big.NewInt(0), uint64(0))
	if err != nil {
		t.Fatalf("pack mintage variable error, %v", err)
	}
	fmt.Println("-------------mintage genesis block-------------")
	fmt.Printf("address: %v\n", hex.EncodeToString(contracts.AddressMintage.Bytes()))
	fmt.Printf("AccountBlock{\n\tBlockType: %v\n\tAccountAddress: %v,\n\tHeight: %v,\n\tAmount: %v,\n\tTokenId:ledger.ViteTokenId,\n\tQuota:0,\n\tFee:%v\n}\n",
		ledger.BlockTypeReceive, hex.EncodeToString(contracts.AddressMintage.Bytes()), 1, big.NewInt(0), big.NewInt(0))
	fmt.Printf("Storage:{\n\t%v:%v\n}\n", hex.EncodeToString(contracts.GetMintageKey(ledger.ViteTokenId)), hex.EncodeToString(mintageData))

	fmt.Println("-------------vite owner genesis block-------------")
	fmt.Println("address: viteAddress")
	fmt.Printf("AccountBlock{\n\tBlockType: %v,\n\tAccountAddress: viteAddress,\n\tHeight: %v,\n\tAmount: %v,\n\tTokenId:ledger.ViteTokenId,\n\tQuota:0,\n\tFee:%v,\n\tData:%v,\n}\n",
		ledger.BlockTypeReceive, 1, totalSupply, big.NewInt(0), []byte{})
	fmt.Printf("Storage:{\n\t$balance:ledger.ViteTokenId:%v\n}\n", totalSupply)

	conditionRegisterData, err := contracts.ABIConsensusGroup.PackVariable(contracts.VariableNameConditionRegisterOfPledge, new(big.Int).Mul(big.NewInt(1e6), util.AttovPerVite), ledger.ViteTokenId, uint64(3600*24*90))
	if err != nil {
		t.Fatalf("pack register condition variable error, %v", err)
	}
	snapshotConsensusGroupData, err := contracts.ABIConsensusGroup.PackVariable(contracts.VariableNameConsensusGroupInfo,
		uint8(25),
		int64(1),
		int64(3),
		uint8(2),
		uint8(50),
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
	commonConsensusGroupData, err := contracts.ABIConsensusGroup.PackVariable(contracts.VariableNameConsensusGroupInfo,
		uint8(25),
		int64(3),
		int64(1),
		uint8(2),
		uint8(50),
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
	fmt.Printf("address:%v\n", hex.EncodeToString(contracts.AddressConsensusGroup.Bytes()))
	fmt.Printf("AccountBlock{\n\tBlockType: %v,\n\tAccountAddress: %v,\n\tHeight: %v,\n\tAmount: %v,\n\tTokenId:ledger.ViteTokenId,\n\tQuota:0,\n\tFee:%v,\n\tData:%v,\n}\n",
		ledger.BlockTypeReceive, hex.EncodeToString(contracts.AddressConsensusGroup.Bytes()), 1, big.NewInt(0), big.NewInt(0), []byte{})
	fmt.Printf("Storage:{\n\t%v:%v,\n\t%v:%v}\n", hex.EncodeToString(contracts.GetConsensusGroupKey(types.SNAPSHOT_GID)), hex.EncodeToString(snapshotConsensusGroupData), hex.EncodeToString(contracts.GetConsensusGroupKey(types.DELEGATE_GID)), hex.EncodeToString(commonConsensusGroupData))

	fmt.Println("-------------snapshot consensus group and common consensus group register genesis block-------------")
	fmt.Printf("address:%v\n", hex.EncodeToString(contracts.AddressRegister.Bytes()))
	fmt.Printf("AccountBlock{\n\tBlockType: %v,\n\tAccountAddress: %v,\n\tHeight: %v,\n\tAmount: %v,\n\tTokenId:ledger.ViteTokenId,\n\tQuota:0,\n\tFee:%v,\n\tData:%v,\n}\n",
		ledger.BlockTypeReceive, hex.EncodeToString(contracts.AddressRegister.Bytes()), 1, big.NewInt(0), big.NewInt(0), []byte{})
	fmt.Printf("Storage:{\n")
	for i := 1; i <= 25; i++ {
		addr, _, _ := types.CreateAddress()
		registerData, err := contracts.ABIRegister.PackVariable(contracts.VariableNameRegistration, "node"+strconv.Itoa(i), addr, addr, helper.Big0, uint64(1), uint64(1), uint64(0))
		if err != nil {
			t.Fatalf("pack registration variable error, %v", err)
		}
		snapshotKey := contracts.GetRegisterKey("snapshotNode1", types.SNAPSHOT_GID)
		fmt.Printf("\t%v: %v\n", hex.EncodeToString(snapshotKey), hex.EncodeToString(registerData))
	}
	fmt.Println("}")
}
