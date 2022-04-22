package vm

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math/big"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/vitelabs/go-vite/v2/common/helper"
	"github.com/vitelabs/go-vite/v2/common/types"
	"github.com/vitelabs/go-vite/v2/crypto/ed25519"
	ledger "github.com/vitelabs/go-vite/v2/interfaces/core"
	"github.com/vitelabs/go-vite/v2/ledger/consensus/core"
	"github.com/vitelabs/go-vite/v2/vm/contracts/abi"
	"github.com/vitelabs/go-vite/v2/vm/util"
)

func TestContractsRefund(t *testing.T) {
	initEmptyFork(t) //
	// prepare db
	viteTotalSupply := new(big.Int).Mul(big.NewInt(1e9), big.NewInt(1e18))
	db, addr1, _, hash12, snapshot2, _ := PrepareDb(viteTotalSupply)

	addr2 := types.AddressGovernance
	sbpName := "s1"
	locHashRegister, _ := types.BytesToHash(abi.GetRegistrationInfoKey(sbpName, types.SNAPSHOT_GID))
	registrationDataOld := db.StorageMap[addr2][ToKey(locHashRegister.Bytes())]
	db.Addr = addr2
	contractBalance, _ := db.GetBalance(&ledger.ViteTokenId)
	// register with an existed super node name, get refund
	balance1 := new(big.Int).Set(viteTotalSupply)
	addr6, _, _ := types.CreateAddress()
	db.AccountBlockMap[addr6] = make(map[types.Hash]*ledger.AccountBlock)
	block13Data, err := abi.ABIGovernance.PackMethod(abi.MethodNameRegister, types.SNAPSHOT_GID, sbpName, addr6)
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
	vm := NewVM(nil, nil)
	//vm.Debug = true
	db.Addr = addr1
	sendRegisterBlock, isRetry, err := vm.RunV2(db, block13, nil, nil)
	balance1.Sub(balance1, block13.Amount)
	if sendRegisterBlock == nil ||
		len(sendRegisterBlock.AccountBlock.SendBlockList) != 0 || isRetry || err != nil ||
		sendRegisterBlock.AccountBlock.Quota != vm.gasTable.RegisterQuota ||
		sendRegisterBlock.AccountBlock.Quota != sendRegisterBlock.AccountBlock.QuotaUsed ||
		!bytes.Equal(sendRegisterBlock.AccountBlock.Data, block13Data) ||
		db.BalanceMap[addr1][ledger.ViteTokenId].Cmp(balance1) != 0 {
		t.Fatalf("send register transaction error")
	}
	db.AccountBlockMap[addr1][hash13] = sendRegisterBlock.AccountBlock

	hash21 := types.DataHash([]byte{2, 1})
	block21 := &ledger.AccountBlock{
		Height:         1,
		AccountAddress: addr2,
		BlockType:      ledger.BlockTypeReceive,
		FromBlockHash:  hash13,
		Hash:           hash21,
	}
	vm = NewVM(nil, nil)
	//vm.Debug = true
	db.Addr = addr2
	receiveRegisterBlock, isRetry, err := vm.RunV2(db, block21, sendRegisterBlock.AccountBlock, NewTestGlobalStatus(0, snapshot2))
	contractBalance.Add(contractBalance, block13.Amount)
	newBalance, _ := db.GetBalance(&ledger.ViteTokenId)
	if receiveRegisterBlock == nil ||
		len(receiveRegisterBlock.AccountBlock.SendBlockList) != 1 || isRetry || err == nil ||
		db.BalanceMap[addr1][ledger.ViteTokenId].Cmp(balance1) != 0 ||
		!bytes.Equal(db.StorageMap[addr2][ToKey(locHashRegister.Bytes())], registrationDataOld) ||
		receiveRegisterBlock.AccountBlock.Quota != 0 ||
		receiveRegisterBlock.AccountBlock.Quota != receiveRegisterBlock.AccountBlock.QuotaUsed ||
		len(receiveRegisterBlock.AccountBlock.Data) != 33 ||
		receiveRegisterBlock.AccountBlock.Data[32] != byte(1) ||
		receiveRegisterBlock.AccountBlock.SendBlockList[0].TokenId != block13.TokenId ||
		receiveRegisterBlock.AccountBlock.SendBlockList[0].Amount.Cmp(block13.Amount) != 0 ||
		receiveRegisterBlock.AccountBlock.SendBlockList[0].BlockType != ledger.BlockTypeSendCall ||
		receiveRegisterBlock.AccountBlock.SendBlockList[0].AccountAddress != block13.ToAddress ||
		receiveRegisterBlock.AccountBlock.SendBlockList[0].ToAddress != block13.AccountAddress ||
		newBalance.Cmp(contractBalance) != 0 ||
		!bytes.Equal(receiveRegisterBlock.AccountBlock.SendBlockList[0].Data, []byte{}) {
		t.Fatalf("receive register transaction error")
	}
	db.AccountBlockMap[addr2] = make(map[types.Hash]*ledger.AccountBlock)
	db.AccountBlockMap[addr2][hash21] = receiveRegisterBlock.AccountBlock
	hash22 := types.DataHash([]byte{2, 2})
	receiveRegisterBlock.AccountBlock.SendBlockList[0].Hash = hash22
	receiveRegisterBlock.AccountBlock.SendBlockList[0].PrevHash = hash21
	db.AccountBlockMap[addr2][hash22] = receiveRegisterBlock.AccountBlock.SendBlockList[0]

	hash14 := types.DataHash([]byte{1, 4})
	block14 := &ledger.AccountBlock{
		Height:         4,
		AccountAddress: addr1,
		BlockType:      ledger.BlockTypeReceive,
		FromBlockHash:  hash22,
		Hash:           hash14,
	}
	vm = NewVM(nil, nil)
	//vm.Debug = true
	db.Addr = addr1
	receiveRegisterRefuncBlock, isRetry, err := vm.RunV2(db, block14, receiveRegisterBlock.AccountBlock.SendBlockList[0], nil)
	balance1.Add(balance1, block13.Amount)
	if receiveRegisterRefuncBlock == nil ||
		len(receiveRegisterRefuncBlock.AccountBlock.SendBlockList) != 0 || isRetry || err != nil ||
		db.BalanceMap[addr1][ledger.ViteTokenId].Cmp(balance1) != 0 ||
		receiveRegisterRefuncBlock.AccountBlock.Quota != 21000 ||
		receiveRegisterRefuncBlock.AccountBlock.Quota != receiveRegisterRefuncBlock.AccountBlock.QuotaUsed {
		t.Fatalf("receive register refund transaction error")
	}
	db.AccountBlockMap[addr1][hash14] = receiveRegisterRefuncBlock.AccountBlock
}

func TestContractsRegister(t *testing.T) {
	initCustomFork(t) //
	// prepare db
	viteTotalSupply := new(big.Int).Mul(big.NewInt(1e9), big.NewInt(1e18))
	db, addr1, _, hash12, snapshot2, timestamp := PrepareDb(viteTotalSupply)

	reader := util.NewVMConsensusReader(newConsensusReaderTest(db.GetGenesisSnapshotBlock().Timestamp.Unix(), 24*3600, nil))
	// register
	balance1 := new(big.Int).Set(viteTotalSupply)
	addr6, privateKey6, _ := types.CreateAddress()
	addr7, _, _ := types.CreateAddress()
	publicKey6 := ed25519.PublicKey(privateKey6.PubByte())
	db.AccountBlockMap[addr6] = make(map[types.Hash]*ledger.AccountBlock)
	db.AccountBlockMap[addr7] = make(map[types.Hash]*ledger.AccountBlock)
	addr2 := types.AddressGovernance
	sbpName := "super1"
	block13Data, err := abi.ABIGovernance.PackMethod(abi.MethodNameRegister, types.SNAPSHOT_GID, sbpName, addr7)
	hash13 := types.DataHash([]byte{1, 3})
	block13 := &ledger.AccountBlock{
		Height:         3,
		ToAddress:      addr2,
		AccountAddress: addr1,
		BlockType:      ledger.BlockTypeSendCall,
		PrevHash:       hash12,
		Amount:         new(big.Int).Mul(big.NewInt(5e5), big.NewInt(1e18)),
		Fee:            big.NewInt(0),
		Data:           block13Data,
		TokenId:        ledger.ViteTokenId,
		Hash:           hash13,
	}
	vm := NewVM(reader, nil)
	//vm.Debug = true
	db.Addr = addr1
	sendRegisterBlock, isRetry, err := vm.RunV2(db, block13, nil, nil)
	balance1.Sub(balance1, block13.Amount)
	if sendRegisterBlock == nil ||
		len(sendRegisterBlock.AccountBlock.SendBlockList) != 0 || isRetry || err != nil ||
		sendRegisterBlock.AccountBlock.Quota != vm.gasTable.RegisterQuota ||
		sendRegisterBlock.AccountBlock.Quota != sendRegisterBlock.AccountBlock.QuotaUsed ||
		!bytes.Equal(sendRegisterBlock.AccountBlock.Data, block13Data) ||
		db.BalanceMap[addr1][ledger.ViteTokenId].Cmp(balance1) != 0 {
		t.Fatalf("send register transaction error")
	}
	db.AccountBlockMap[addr1][hash13] = sendRegisterBlock.AccountBlock

	hash21 := types.DataHash([]byte{2, 1})
	block21 := &ledger.AccountBlock{
		Height:         1,
		AccountAddress: addr2,
		BlockType:      ledger.BlockTypeReceive,
		FromBlockHash:  hash13,
		Hash:           hash21,
	}
	vm = NewVM(reader, nil)
	//vm.Debug = true
	locHashRegister := abi.GetRegistrationInfoKey(sbpName, types.SNAPSHOT_GID)
	hisAddrList := []types.Address{addr7}
	expirationHeight := snapshot2.Height + 3600*24*90
	registrationData, _ := abi.ABIGovernance.PackVariable(abi.VariableNameRegistrationInfo, sbpName, addr7, addr1, block13.Amount, expirationHeight, snapshot2.Timestamp.Unix(), int64(0), hisAddrList)
	db.Addr = addr2
	receiveRegisterBlock, isRetry, err := vm.RunV2(db, block21, sendRegisterBlock.AccountBlock, NewTestGlobalStatus(0, snapshot2))
	if receiveRegisterBlock == nil ||
		len(receiveRegisterBlock.AccountBlock.SendBlockList) != 0 || isRetry || err != nil ||
		db.BalanceMap[addr1][ledger.ViteTokenId].Cmp(balance1) != 0 ||
		!bytes.Equal(db.StorageMap[addr2][ToKey(locHashRegister)], registrationData) ||
		len(receiveRegisterBlock.AccountBlock.Data) != 33 ||
		receiveRegisterBlock.AccountBlock.Data[32] != byte(0) ||
		receiveRegisterBlock.AccountBlock.Quota != 0 ||
		receiveRegisterBlock.AccountBlock.Quota != receiveRegisterBlock.AccountBlock.QuotaUsed {
		t.Fatalf("receive register transaction error")
	}
	db.AccountBlockMap[addr2] = make(map[types.Hash]*ledger.AccountBlock)
	db.AccountBlockMap[addr2][hash21] = receiveRegisterBlock.AccountBlock

	// update registration
	block14Data, err := abi.ABIGovernance.PackMethod(abi.MethodNameUpdateBlockProducingAddress, types.SNAPSHOT_GID, sbpName, addr6)
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
	vm = NewVM(reader, nil)
	//vm.Debug = true
	db.Addr = addr1
	sendRegisterBlock2, isRetry, err := vm.RunV2(db, block14, nil, nil)
	if sendRegisterBlock2 == nil ||
		len(sendRegisterBlock2.AccountBlock.SendBlockList) != 0 || isRetry || err != nil ||
		sendRegisterBlock2.AccountBlock.Quota != vm.gasTable.UpdateBlockProducingAddressQuota ||
		sendRegisterBlock2.AccountBlock.Quota != sendRegisterBlock2.AccountBlock.QuotaUsed ||
		!bytes.Equal(sendRegisterBlock2.AccountBlock.Data, block14Data) ||
		db.BalanceMap[addr1][ledger.ViteTokenId].Cmp(balance1) != 0 {
		t.Fatalf("send update registration transaction error")
	}
	db.AccountBlockMap[addr1][hash14] = sendRegisterBlock2.AccountBlock

	hash22 := types.DataHash([]byte{2, 2})
	block22 := &ledger.AccountBlock{
		Height:         2,
		AccountAddress: addr2,
		BlockType:      ledger.BlockTypeReceive,
		FromBlockHash:  hash14,
		PrevHash:       hash21,
		Hash:           hash22,
	}
	vm = NewVM(reader, nil)
	//vm.Debug = true
	hisAddrList = append(hisAddrList, addr6)
	registrationData, _ = abi.ABIGovernance.PackVariable(abi.VariableNameRegistrationInfo, sbpName, addr6, addr1, block13.Amount, expirationHeight, snapshot2.Timestamp.Unix(), int64(0), hisAddrList)
	db.Addr = addr2
	receiveRegisterBlock2, isRetry, err := vm.RunV2(db, block22, sendRegisterBlock2.AccountBlock, NewTestGlobalStatus(0, snapshot2))
	if receiveRegisterBlock2 == nil ||
		len(receiveRegisterBlock2.AccountBlock.SendBlockList) != 0 || isRetry || err != nil ||
		db.BalanceMap[addr1][ledger.ViteTokenId].Cmp(balance1) != 0 ||
		!bytes.Equal(db.StorageMap[addr2][ToKey(locHashRegister)], registrationData) ||
		len(receiveRegisterBlock2.AccountBlock.Data) != 33 ||
		receiveRegisterBlock2.AccountBlock.Data[32] != byte(0) ||
		receiveRegisterBlock2.AccountBlock.Quota != 0 ||
		receiveRegisterBlock2.AccountBlock.Quota != receiveRegisterBlock2.AccountBlock.QuotaUsed {
		t.Fatalf("receive update registration transaction error")
	}
	db.AccountBlockMap[addr2][hash22] = receiveRegisterBlock2.AccountBlock

	// get contracts data
	db.Addr = types.AddressGovernance
	if registerList, _ := abi.GetCandidateList(db, types.SNAPSHOT_GID); len(registerList) != 3 || len(registerList[0].Name) == 0 {
		t.Fatalf("get register list failed")
	}

	// cancel register
	time3 := time.Unix(timestamp+1, 0)
	snapshot3 := &ledger.SnapshotBlock{Height: 3, Timestamp: &time3, Hash: types.DataHash([]byte{10, 3}), PublicKey: publicKey6}
	db.SnapshotBlockList = append(db.SnapshotBlockList, snapshot3)
	time4 := time.Unix(timestamp+2, 0)
	snapshot4 := &ledger.SnapshotBlock{Height: 4, Timestamp: &time4, Hash: types.DataHash([]byte{10, 4}), PublicKey: publicKey6}
	db.SnapshotBlockList = append(db.SnapshotBlockList, snapshot4)
	time5 := time.Unix(timestamp+1+3600*24*90, 0)
	snapshot5 := &ledger.SnapshotBlock{Height: 3 + 3600*24*90, Timestamp: &time5, Hash: types.DataHash([]byte{10, 5})}
	db.SnapshotBlockList = append(db.SnapshotBlockList, snapshot5)

	hash15 := types.DataHash([]byte{1, 5})
	block15Data, _ := abi.ABIGovernance.PackMethod(abi.MethodNameRevoke, types.SNAPSHOT_GID, sbpName)
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
	vm = NewVM(reader, nil)
	//vm.Debug = true
	db.Addr = addr1
	sendCancelRegisterBlock, isRetry, err := vm.RunV2(db, block15, nil, nil)
	if sendCancelRegisterBlock == nil ||
		len(sendCancelRegisterBlock.AccountBlock.SendBlockList) != 0 || isRetry || err != nil ||
		sendCancelRegisterBlock.AccountBlock.Quota != vm.gasTable.RevokeQuota ||
		sendCancelRegisterBlock.AccountBlock.Quota != sendCancelRegisterBlock.AccountBlock.QuotaUsed ||
		!bytes.Equal(sendCancelRegisterBlock.AccountBlock.Data, block15Data) ||
		db.BalanceMap[addr1][ledger.ViteTokenId].Cmp(balance1) != 0 {
		t.Fatalf("send cancel register transaction error")
	}
	db.AccountBlockMap[addr1][hash15] = sendCancelRegisterBlock.AccountBlock

	hash23 := types.DataHash([]byte{2, 3})
	block23 := &ledger.AccountBlock{
		Height:         3,
		AccountAddress: addr2,
		BlockType:      ledger.BlockTypeReceive,
		PrevHash:       hash21,
		FromBlockHash:  hash15,
	}
	vm = NewVM(reader, nil)
	//vm.Debug = true
	db.Addr = addr2
	receiveCancelRegisterBlock, isRetry, err := vm.RunV2(db, block23, sendCancelRegisterBlock.AccountBlock, NewTestGlobalStatus(0, snapshot5))
	registrationData, _ = abi.ABIGovernance.PackVariable(abi.VariableNameRegistrationInfoV2, sbpName, addr6, addr1, addr1, helper.Big0, uint64(0), snapshot2.Timestamp.Unix(), snapshot5.Timestamp.Unix(), hisAddrList)
	if receiveCancelRegisterBlock == nil ||
		len(receiveCancelRegisterBlock.AccountBlock.SendBlockList) != 1 || isRetry || err != nil ||
		db.BalanceMap[addr2][ledger.ViteTokenId].Cmp(helper.Big0) != 0 ||
		db.BalanceMap[addr1][ledger.ViteTokenId].Cmp(balance1) != 0 ||
		!bytes.Equal(db.StorageMap[addr2][ToKey(locHashRegister)], registrationData) ||
		len(receiveCancelRegisterBlock.AccountBlock.Data) != 33 ||
		receiveCancelRegisterBlock.AccountBlock.Data[32] != byte(0) ||
		receiveCancelRegisterBlock.AccountBlock.Quota != 0 ||
		receiveCancelRegisterBlock.AccountBlock.Quota != receiveCancelRegisterBlock.AccountBlock.QuotaUsed ||
		receiveCancelRegisterBlock.AccountBlock.SendBlockList[0].AccountAddress != addr2 ||
		receiveCancelRegisterBlock.AccountBlock.SendBlockList[0].ToAddress != addr1 ||
		receiveCancelRegisterBlock.AccountBlock.SendBlockList[0].BlockType != ledger.BlockTypeSendCall {
		t.Fatalf("receive cancel register transaction error")
	}
	db.AccountBlockMap[addr2][hash23] = receiveCancelRegisterBlock.AccountBlock
	hash24 := types.DataHash([]byte{2, 4})
	receiveCancelRegisterBlock.AccountBlock.SendBlockList[0].Hash = hash24
	receiveCancelRegisterBlock.AccountBlock.SendBlockList[0].PrevHash = hash23
	db.AccountBlockMap[addr2][hash24] = receiveCancelRegisterBlock.AccountBlock.SendBlockList[0]

	hash16 := types.DataHash([]byte{1, 6})
	block16 := &ledger.AccountBlock{
		Height:         6,
		AccountAddress: addr1,
		BlockType:      ledger.BlockTypeReceive,
		PrevHash:       hash16,
		FromBlockHash:  hash23,
		Hash:           hash16,
	}
	vm = NewVM(reader, nil)
	//vm.Debug = true
	db.Addr = addr1
	balance1.Add(balance1, block13.Amount)
	receiveCancelRegisterRefundBlock, isRetry, err := vm.RunV2(db, block16, receiveCancelRegisterBlock.AccountBlock.SendBlockList[0], nil)
	if receiveCancelRegisterRefundBlock == nil ||
		len(receiveCancelRegisterRefundBlock.AccountBlock.SendBlockList) != 0 || isRetry || err != nil ||
		db.BalanceMap[addr2][ledger.ViteTokenId].Cmp(helper.Big0) != 0 ||
		db.BalanceMap[addr1][ledger.ViteTokenId].Cmp(balance1) != 0 ||
		receiveCancelRegisterRefundBlock.AccountBlock.Quota != 21000 ||
		receiveCancelRegisterRefundBlock.AccountBlock.Quota != receiveCancelRegisterRefundBlock.AccountBlock.QuotaUsed {
		t.Fatalf("receive cancel register refund transaction error")
	}
	db.AccountBlockMap[addr1][hash16] = receiveCancelRegisterRefundBlock.AccountBlock

	// TODO reward
	// Reward
	hash17 := types.DataHash([]byte{1, 7})
	block17Data, _ := abi.ABIGovernance.PackMethod(abi.MethodNameWithdrawReward, types.SNAPSHOT_GID, sbpName, addr1)
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
	vm = NewVM(nil, nil)
	//vm.Debug = true
	db.Addr = addr1
	sendRewardBlock, isRetry, err := vm.RunV2(db, block17, nil, nil)
	if sendRewardBlock == nil ||
		len(sendRewardBlock.AccountBlock.SendBlockList) != 0 || isRetry || err != nil ||
		sendRewardBlock.AccountBlock.Quota != vm.gasTable.WithdrawRewardQuota ||
		sendRewardBlock.AccountBlock.Quota != sendRewardBlock.AccountBlock.QuotaUsed ||
		!bytes.Equal(sendRewardBlock.AccountBlock.Data, block17Data) ||
		db.BalanceMap[addr1][ledger.ViteTokenId].Cmp(balance1) != 0 {
		t.Fatalf("send cancel register transaction error")
	}
	db.AccountBlockMap[addr1][hash17] = sendRewardBlock.AccountBlock

	hash25 := types.DataHash([]byte{2, 5})
	block25 := &ledger.AccountBlock{
		Height:         5,
		AccountAddress: addr2,
		BlockType:      ledger.BlockTypeReceive,
		PrevHash:       hash24,
		FromBlockHash:  hash17,
	}
	vm = NewVM(reader, nil)
	//vm.Debug = true
	db.Addr = addr2
	receiveRewardBlock, isRetry, err := vm.RunV2(db, block25, sendRewardBlock.AccountBlock, NewTestGlobalStatus(0, snapshot5))
	registrationData, _ = abi.ABIGovernance.PackVariable(abi.VariableNameRegistrationInfoV2, sbpName, addr6, addr1, addr1, helper.Big0, uint64(0), int64(snapshot5.Timestamp.Unix()-2-24*3600), snapshot5.Timestamp.Unix(), hisAddrList)
	if receiveRewardBlock == nil ||
		len(receiveRewardBlock.AccountBlock.SendBlockList) != 0 || isRetry || err != nil ||
		db.BalanceMap[addr2][ledger.ViteTokenId].Cmp(helper.Big0) != 0 ||
		db.BalanceMap[addr1][ledger.ViteTokenId].Cmp(balance1) != 0 ||
		!bytes.Equal(db.StorageMap[addr2][ToKey(locHashRegister)], registrationData) ||
		len(receiveRewardBlock.AccountBlock.Data) != 33 ||
		receiveRewardBlock.AccountBlock.Data[32] != byte(0) ||
		receiveRewardBlock.AccountBlock.Quota != 0 ||
		receiveRewardBlock.AccountBlock.Quota != receiveRewardBlock.AccountBlock.QuotaUsed {
		t.Fatalf("receive reward transaction error")
	}
	db.AccountBlockMap[addr2][hash25] = receiveRewardBlock.AccountBlock
}

func TestContractsVote(t *testing.T) {
	initEmptyFork(t)
	// prepare db
	viteTotalSupply := new(big.Int).Mul(big.NewInt(2e6), big.NewInt(1e18))
	db, addr1, _, hash12, snapshot2, _ := PrepareDb(viteTotalSupply)
	// vote
	addr3 := types.AddressGovernance
	sbpName := "s1"
	block13Data, _ := abi.ABIGovernance.PackMethod(abi.MethodNameVote, types.SNAPSHOT_GID, sbpName)
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
	vm := NewVM(nil, nil)
	//vm.Debug = true
	db.Addr = addr1
	sendVoteBlock, isRetry, err := vm.RunV2(db, block13, nil, nil)
	if sendVoteBlock == nil ||
		len(sendVoteBlock.AccountBlock.SendBlockList) != 0 || isRetry || err != nil ||
		!bytes.Equal(sendVoteBlock.AccountBlock.Data, block13Data) ||
		sendVoteBlock.AccountBlock.Quota != vm.gasTable.VoteQuota ||
		sendVoteBlock.AccountBlock.Quota != sendVoteBlock.AccountBlock.QuotaUsed {
		t.Fatalf("send vote transaction error")
	}
	db.AccountBlockMap[addr1][hash13] = sendVoteBlock.AccountBlock

	hash31 := types.DataHash([]byte{3, 1})
	block31 := &ledger.AccountBlock{
		Height:         1,
		AccountAddress: addr3,
		BlockType:      ledger.BlockTypeReceive,
		FromBlockHash:  hash13,
		Hash:           hash31,
	}
	vm = NewVM(nil, nil)
	//vm.Debug = true
	db.Addr = addr3
	receiveVoteBlock, isRetry, err := vm.RunV2(db, block31, sendVoteBlock.AccountBlock, NewTestGlobalStatus(0, snapshot2))
	voteKey := abi.GetVoteInfoKey(addr1, types.SNAPSHOT_GID)
	voteData, _ := abi.ABIGovernance.PackVariable(abi.VariableNameVoteInfo, sbpName)
	if receiveVoteBlock == nil ||
		len(receiveVoteBlock.AccountBlock.SendBlockList) != 0 || isRetry || err != nil ||
		!bytes.Equal(db.StorageMap[addr3][ToKey(voteKey)], voteData) ||
		len(receiveVoteBlock.AccountBlock.Data) != 33 ||
		receiveVoteBlock.AccountBlock.Data[32] != byte(0) ||
		receiveVoteBlock.AccountBlock.Quota != 0 ||
		receiveVoteBlock.AccountBlock.Quota != receiveVoteBlock.AccountBlock.QuotaUsed {
		t.Fatalf("receive vote transaction error")
	}
	db.AccountBlockMap[addr3] = make(map[types.Hash]*ledger.AccountBlock)
	db.AccountBlockMap[addr3][hash31] = receiveVoteBlock.AccountBlock

	addr4, _ := types.BytesToAddress(helper.HexToBytes("e5bf58cacfb74cf8c49a1d5e59d3919c9a4cb9ed"))
	db.AccountBlockMap[addr4] = make(map[types.Hash]*ledger.AccountBlock)
	sbpName2 := "s2"
	block14Data, _ := abi.ABIGovernance.PackMethod(abi.MethodNameVote, types.SNAPSHOT_GID, sbpName2)
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
	vm = NewVM(nil, nil)
	//vm.Debug = true
	db.Addr = addr1
	sendVoteBlock2, isRetry, err := vm.RunV2(db, block14, nil, nil)
	if sendVoteBlock2 == nil ||
		len(sendVoteBlock2.AccountBlock.SendBlockList) != 0 || isRetry || err != nil ||
		!bytes.Equal(sendVoteBlock2.AccountBlock.Data, block14Data) ||
		sendVoteBlock2.AccountBlock.Quota != vm.gasTable.VoteQuota ||
		sendVoteBlock2.AccountBlock.Quota != sendVoteBlock2.AccountBlock.QuotaUsed {
		t.Fatalf("send vote transaction 2 error")
	}
	db.AccountBlockMap[addr1][hash14] = sendVoteBlock2.AccountBlock

	hash32 := types.DataHash([]byte{3, 2})
	block32 := &ledger.AccountBlock{
		Height:         2,
		AccountAddress: addr3,
		BlockType:      ledger.BlockTypeReceive,
		PrevHash:       hash31,
		FromBlockHash:  hash14,
		Hash:           hash32,
	}
	vm = NewVM(nil, nil)
	//vm.Debug = true
	db.Addr = addr3
	receiveVoteBlock2, isRetry, err := vm.RunV2(db, block32, sendVoteBlock2.AccountBlock, NewTestGlobalStatus(0, snapshot2))
	voteData, _ = abi.ABIGovernance.PackVariable(abi.VariableNameVoteInfo, sbpName2)
	if receiveVoteBlock2 == nil ||
		len(receiveVoteBlock2.AccountBlock.SendBlockList) != 0 || isRetry || err != nil ||
		!bytes.Equal(db.StorageMap[addr3][ToKey(voteKey)], voteData) ||
		len(receiveVoteBlock2.AccountBlock.Data) != 33 ||
		receiveVoteBlock2.AccountBlock.Data[32] != byte(0) ||
		receiveVoteBlock2.AccountBlock.Quota != 0 ||
		receiveVoteBlock2.AccountBlock.Quota != receiveVoteBlock2.AccountBlock.QuotaUsed {
		t.Fatalf("receive vote transaction 2 error")
	}
	db.AccountBlockMap[addr3][hash32] = receiveVoteBlock2.AccountBlock

	// get contracts data
	db.Addr = types.AddressGovernance
	if voteList, _ := abi.GetVoteList(db, types.SNAPSHOT_GID); len(voteList) != 1 || voteList[0].SbpName != sbpName2 {
		t.Fatalf("get vote list failed")
	}

	// cancel vote
	block15Data, _ := abi.ABIGovernance.PackMethod(abi.MethodNameCancelVote, types.SNAPSHOT_GID)
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
	vm = NewVM(nil, nil)
	//vm.Debug = true
	db.Addr = addr1
	sendCancelVoteBlock, isRetry, err := vm.RunV2(db, block15, nil, nil)
	if sendCancelVoteBlock == nil ||
		len(sendCancelVoteBlock.AccountBlock.SendBlockList) != 0 || isRetry || err != nil ||
		!bytes.Equal(sendCancelVoteBlock.AccountBlock.Data, block15Data) ||
		sendCancelVoteBlock.AccountBlock.Quota != vm.gasTable.CancelVoteQuota ||
		sendCancelVoteBlock.AccountBlock.Quota != sendCancelVoteBlock.AccountBlock.QuotaUsed {
		t.Fatalf("send cancel vote transaction error")
	}
	db.AccountBlockMap[addr1][hash15] = sendCancelVoteBlock.AccountBlock

	hash33 := types.DataHash([]byte{3, 3})
	block33 := &ledger.AccountBlock{
		Height:         3,
		AccountAddress: addr3,
		BlockType:      ledger.BlockTypeReceive,
		PrevHash:       hash32,
		FromBlockHash:  hash15,
		Hash:           hash33,
	}
	vm = NewVM(nil, nil)
	//vm.Debug = true
	db.Addr = addr3
	receiveCancelVoteBlock, isRetry, err := vm.RunV2(db, block33, sendCancelVoteBlock.AccountBlock, NewTestGlobalStatus(0, snapshot2))
	if receiveCancelVoteBlock == nil ||
		len(receiveCancelVoteBlock.AccountBlock.SendBlockList) != 0 || isRetry || err != nil ||
		len(db.StorageMap[addr3][ToKey(voteKey)]) != 0 ||
		len(receiveCancelVoteBlock.AccountBlock.Data) != 33 ||
		receiveCancelVoteBlock.AccountBlock.Data[32] != byte(0) ||
		receiveCancelVoteBlock.AccountBlock.Quota != 0 ||
		receiveCancelVoteBlock.AccountBlock.Quota != receiveCancelVoteBlock.AccountBlock.QuotaUsed {
		t.Fatalf("receive cancel vote transaction error")
	}
	db.AccountBlockMap[addr3][hash33] = receiveCancelVoteBlock.AccountBlock
}

func TestContractsStake(t *testing.T) {
	initEmptyFork(t)
	// prepare db
	viteTotalSupply := new(big.Int).Mul(big.NewInt(2e6), big.NewInt(1e18))
	db, addr1, _, hash12, snapshot2, timestamp := PrepareDb(viteTotalSupply)
	// stake
	balance1 := new(big.Int).Set(viteTotalSupply)
	addr4, _, _ := types.CreateAddress()
	db.AccountBlockMap[addr4] = make(map[types.Hash]*ledger.AccountBlock)
	addr5 := types.AddressQuota
	stakeAmount := new(big.Int).Set(new(big.Int).Mul(big.NewInt(1000), util.AttovPerVite))
	block13Data, err := abi.ABIQuota.PackMethod(abi.MethodNameStake, addr4)
	hash13 := types.DataHash([]byte{1, 3})
	block13 := &ledger.AccountBlock{
		Height:         3,
		ToAddress:      addr5,
		AccountAddress: addr1,
		Amount:         stakeAmount,
		TokenId:        ledger.ViteTokenId,
		BlockType:      ledger.BlockTypeSendCall,
		Fee:            big.NewInt(0),
		PrevHash:       hash12,
		Data:           block13Data,
		Hash:           hash13,
	}
	vm := NewVM(nil, nil)
	//vm.Debug = true
	db.Addr = addr1
	sendStakeBlock, isRetry, err := vm.RunV2(db, block13, nil, nil)
	balance1.Sub(balance1, stakeAmount)
	if sendStakeBlock == nil ||
		len(sendStakeBlock.AccountBlock.SendBlockList) != 0 || isRetry || err != nil ||
		db.BalanceMap[addr1][ledger.ViteTokenId].Cmp(balance1) != 0 ||
		!bytes.Equal(sendStakeBlock.AccountBlock.Data, block13Data) ||
		sendStakeBlock.AccountBlock.Quota != vm.gasTable.StakeQuota ||
		sendStakeBlock.AccountBlock.Quota != sendStakeBlock.AccountBlock.QuotaUsed {
		t.Fatalf("send stake transaction error")
	}
	db.AccountBlockMap[addr1][hash13] = sendStakeBlock.AccountBlock

	hash51 := types.DataHash([]byte{5, 1})
	block51 := &ledger.AccountBlock{
		Height:         1,
		AccountAddress: addr5,
		BlockType:      ledger.BlockTypeReceive,
		FromBlockHash:  hash13,
		Hash:           hash51,
	}
	vm = NewVM(nil, nil)
	//vm.Debug = true
	db.Addr = addr5
	receiveStakeBlock, isRetry, err := vm.RunV2(db, block51, sendStakeBlock.AccountBlock, NewTestGlobalStatus(0, snapshot2))
	beneficialKey := abi.GetStakeBeneficialKey(addr4)
	stakeInfoKey := abi.GetStakeInfoKey(addr1, 1)
	expirationHeight := snapshot2.Height + 3600*24*3
	stakeInfoBytes, _ := abi.ABIQuota.PackVariable(abi.VariableNameStakeInfo, stakeAmount, expirationHeight, addr4, false, types.Address{}, uint8(0))
	if receiveStakeBlock == nil ||
		len(receiveStakeBlock.AccountBlock.SendBlockList) != 0 || isRetry || err != nil ||
		!bytes.Equal(db.StorageMap[addr5][ToKey(stakeInfoKey)], stakeInfoBytes) ||
		!bytes.Equal(db.StorageMap[addr5][ToKey(beneficialKey)], helper.LeftPadBytes(stakeAmount.Bytes(), helper.WordSize)) ||
		db.BalanceMap[addr5][ledger.ViteTokenId].Cmp(stakeAmount) != 0 ||
		len(receiveStakeBlock.AccountBlock.Data) != 33 ||
		receiveStakeBlock.AccountBlock.Data[32] != byte(0) ||
		receiveStakeBlock.AccountBlock.Quota != 0 ||
		receiveStakeBlock.AccountBlock.Quota != receiveStakeBlock.AccountBlock.QuotaUsed {
		t.Fatalf("receive stake transaction error")
	}
	db.AccountBlockMap[addr5] = make(map[types.Hash]*ledger.AccountBlock)
	db.AccountBlockMap[addr5][hash51] = receiveStakeBlock.AccountBlock

	block14Data, _ := abi.ABIQuota.PackMethod(abi.MethodNameStake, addr4)
	hash14 := types.DataHash([]byte{1, 4})
	block14 := &ledger.AccountBlock{
		Height:         4,
		ToAddress:      addr5,
		AccountAddress: addr1,
		Amount:         stakeAmount,
		TokenId:        ledger.ViteTokenId,
		BlockType:      ledger.BlockTypeSendCall,
		Fee:            big.NewInt(0),
		PrevHash:       hash13,
		Data:           block14Data,
		Hash:           hash14,
	}
	vm = NewVM(nil, nil)
	//vm.Debug = true
	db.Addr = addr1
	sendStakeBlock2, isRetry, err := vm.RunV2(db, block14, nil, nil)
	balance1.Sub(balance1, stakeAmount)
	if sendStakeBlock2 == nil ||
		len(sendStakeBlock2.AccountBlock.SendBlockList) != 0 || isRetry || err != nil ||
		!bytes.Equal(sendStakeBlock2.AccountBlock.Data, block14Data) ||
		db.BalanceMap[addr1][ledger.ViteTokenId].Cmp(balance1) != 0 ||
		sendStakeBlock2.AccountBlock.Quota != vm.gasTable.StakeQuota ||
		sendStakeBlock2.AccountBlock.Quota != sendStakeBlock2.AccountBlock.QuotaUsed {
		t.Fatalf("send stake transaction 2 error")
	}
	db.AccountBlockMap[addr1][hash14] = sendStakeBlock2.AccountBlock

	hash52 := types.DataHash([]byte{5, 2})
	block52 := &ledger.AccountBlock{
		Height:         2,
		AccountAddress: addr5,
		BlockType:      ledger.BlockTypeReceive,
		PrevHash:       hash51,
		FromBlockHash:  hash14,
		Hash:           hash52,
	}
	vm = NewVM(nil, nil)
	//vm.Debug = true
	db.Addr = addr5
	receiveStakeBlock2, isRetry, err := vm.RunV2(db, block52, sendStakeBlock2.AccountBlock, NewTestGlobalStatus(0, snapshot2))
	newStakeAmount := new(big.Int).Add(stakeAmount, stakeAmount)
	stakeInfoBytes, _ = abi.ABIQuota.PackVariable(abi.VariableNameStakeInfo, newStakeAmount, expirationHeight, addr4, false, types.Address{}, uint8(0))
	if receiveStakeBlock2 == nil ||
		len(receiveStakeBlock2.AccountBlock.SendBlockList) != 0 || isRetry || err != nil ||
		!bytes.Equal(db.StorageMap[addr5][ToKey(stakeInfoKey)], stakeInfoBytes) ||
		!bytes.Equal(db.StorageMap[addr5][ToKey(beneficialKey)], helper.LeftPadBytes(newStakeAmount.Bytes(), helper.WordSize)) ||
		db.BalanceMap[addr5][ledger.ViteTokenId].Cmp(newStakeAmount) != 0 ||
		len(receiveStakeBlock2.AccountBlock.Data) != 33 ||
		receiveStakeBlock2.AccountBlock.Data[32] != byte(0) ||
		receiveStakeBlock2.AccountBlock.Quota != 0 ||
		receiveStakeBlock2.AccountBlock.Quota != receiveStakeBlock2.AccountBlock.QuotaUsed {
		t.Fatalf("receive stake transaction 2 error")
	}
	db.AccountBlockMap[addr5][hash52] = receiveStakeBlock2.AccountBlock

	// get contracts data
	db.Addr = types.AddressQuota
	if stakeAmount, _ := db.GetStakeBeneficialAmount(&addr4); stakeAmount.Cmp(newStakeAmount) != 0 {
		t.Fatalf("get stake beneficial amount failed")
	}
	if stakeInfoList, _, _ := abi.GetStakeInfoList(db, addr1); len(stakeInfoList) != 1 ||
		stakeInfoList[0].Beneficiary != addr4 || stakeInfoList[0].Amount.Cmp(newStakeAmount) != 0 {
		t.Fatalf("get stake amount failed")
	}

	// cancel stake
	for i := uint64(1); i <= uint64(3600*24*3); i++ {
		timei := time.Unix(timestamp+100+int64(i), 0)
		snapshoti := &ledger.SnapshotBlock{Height: 2 + i, Timestamp: &timei, Hash: types.DataHash([]byte{10, byte(2 + i)})}
		db.SnapshotBlockList = append(db.SnapshotBlockList, snapshoti)
	}
	currentSnapshot := db.SnapshotBlockList[len(db.SnapshotBlockList)-1]

	block15Data, _ := abi.ABIQuota.PackMethod(abi.MethodNameCancelStake, addr4, big.NewInt(10))
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
	vm = NewVM(nil, nil)
	db.Addr = addr1
	sendCancelStakeBlock, isRetry, err := vm.RunV2(db, block15, nil, nil)
	if sendCancelStakeBlock != nil || isRetry ||
		err == nil || err.Error() != util.ErrInvalidMethodParam.Error() {
		t.Fatalf("send invalid cancel stake transaction error")
	}

	block15Data, _ = abi.ABIQuota.PackMethod(abi.MethodNameCancelStake, addr4, stakeAmount)
	block15 = &ledger.AccountBlock{
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
	vm = NewVM(nil, nil)
	//vm.Debug = true
	db.Addr = addr1
	sendCancelStakeBlock, isRetry, err = vm.RunV2(db, block15, nil, nil)
	if sendCancelStakeBlock == nil ||
		len(sendCancelStakeBlock.AccountBlock.SendBlockList) != 0 || isRetry || err != nil ||
		!bytes.Equal(sendCancelStakeBlock.AccountBlock.Data, block15Data) ||
		sendCancelStakeBlock.AccountBlock.Quota != vm.gasTable.CancelStakeQuota ||
		sendCancelStakeBlock.AccountBlock.Quota != sendCancelStakeBlock.AccountBlock.QuotaUsed {
		t.Fatalf("send cancel stake transaction error")
	}
	db.AccountBlockMap[addr1][hash15] = sendCancelStakeBlock.AccountBlock

	hash53 := types.DataHash([]byte{5, 3})
	block53 := &ledger.AccountBlock{
		Height:         3,
		AccountAddress: addr5,
		BlockType:      ledger.BlockTypeReceive,
		PrevHash:       hash52,
		FromBlockHash:  hash15,
		Hash:           hash53,
	}
	vm = NewVM(nil, nil)
	//vm.Debug = true
	db.Addr = addr5
	receiveCancelStakeBlock, isRetry, err := vm.RunV2(db, block53, sendCancelStakeBlock.AccountBlock, NewTestGlobalStatus(0, currentSnapshot))
	stakeInfoBytes, _ = abi.ABIQuota.PackVariable(abi.VariableNameStakeInfo, stakeAmount, expirationHeight, addr4, false, types.Address{}, uint8(0))
	if receiveCancelStakeBlock == nil ||
		len(receiveCancelStakeBlock.AccountBlock.SendBlockList) != 1 || isRetry || err != nil ||
		!bytes.Equal(db.StorageMap[addr5][ToKey(stakeInfoKey)], stakeInfoBytes) ||
		!bytes.Equal(db.StorageMap[addr5][ToKey(beneficialKey)], helper.LeftPadBytes(stakeAmount.Bytes(), helper.WordSize)) ||
		db.BalanceMap[addr5][ledger.ViteTokenId].Cmp(stakeAmount) != 0 ||
		len(receiveCancelStakeBlock.AccountBlock.Data) != 33 ||
		receiveCancelStakeBlock.AccountBlock.Data[32] != byte(0) ||
		receiveCancelStakeBlock.AccountBlock.Quota != 0 ||
		receiveCancelStakeBlock.AccountBlock.SendBlockList[0].ToAddress != addr1 ||
		receiveCancelStakeBlock.AccountBlock.SendBlockList[0].Amount.Cmp(stakeAmount) != 0 ||
		receiveCancelStakeBlock.AccountBlock.SendBlockList[0].Fee.Sign() != 0 ||
		len(receiveCancelStakeBlock.AccountBlock.SendBlockList[0].Data) != 0 {
		t.Fatalf("receive cancel stake transaction error")
	}
	db.AccountBlockMap[addr5][hash53] = receiveCancelStakeBlock.AccountBlock
	hash54 := types.DataHash([]byte{5, 4})
	receiveCancelStakeBlock.AccountBlock.SendBlockList[0].Hash = hash54
	receiveCancelStakeBlock.AccountBlock.SendBlockList[0].PrevHash = hash53
	db.AccountBlockMap[addr5][hash54] = receiveCancelStakeBlock.AccountBlock.SendBlockList[0]

	hash16 := types.DataHash([]byte{1, 6})
	block16 := &ledger.AccountBlock{
		Height:         6,
		AccountAddress: addr1,
		BlockType:      ledger.BlockTypeReceive,
		PrevHash:       hash15,
		FromBlockHash:  hash54,
		Hash:           hash16,
	}
	vm = NewVM(nil, nil)
	//vm.Debug = true
	db.Addr = addr1
	receiveCancelStakeRefundBlock, isRetry, err := vm.RunV2(db, block16, receiveCancelStakeBlock.AccountBlock.SendBlockList[0], NewTestGlobalStatus(0, currentSnapshot))
	balance1.Add(balance1, stakeAmount)
	if receiveCancelStakeRefundBlock == nil ||
		len(receiveCancelStakeRefundBlock.AccountBlock.SendBlockList) != 0 || isRetry || err != nil ||
		db.BalanceMap[addr1][ledger.ViteTokenId].Cmp(balance1) != 0 ||
		receiveCancelStakeRefundBlock.AccountBlock.Quota != 21000 {
		t.Fatalf("receive cancel stake refund transaction error")
	}
	db.AccountBlockMap[addr1][hash16] = receiveCancelStakeRefundBlock.AccountBlock

	block17Data, _ := abi.ABIQuota.PackMethod(abi.MethodNameCancelStake, addr4, stakeAmount)
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
	vm = NewVM(nil, nil)
	//vm.Debug = true
	db.Addr = addr1
	sendCancelStakeBlock2, isRetry, err := vm.RunV2(db, block17, nil, nil)
	if sendCancelStakeBlock2 == nil ||
		len(sendCancelStakeBlock2.AccountBlock.SendBlockList) != 0 || isRetry || err != nil ||
		!bytes.Equal(sendCancelStakeBlock2.AccountBlock.Data, block17Data) ||
		sendCancelStakeBlock2.AccountBlock.Quota != vm.gasTable.CancelStakeQuota {
		t.Fatalf("send cancel stake transaction 2 error")
	}
	db.AccountBlockMap[addr1][hash17] = sendCancelStakeBlock2.AccountBlock

	hash55 := types.DataHash([]byte{5, 5})
	block55 := &ledger.AccountBlock{
		Height:         5,
		AccountAddress: addr5,
		BlockType:      ledger.BlockTypeReceive,
		PrevHash:       hash54,
		FromBlockHash:  hash17,
		Hash:           hash55,
	}
	vm = NewVM(nil, nil)
	//vm.Debug = true
	db.Addr = addr5
	receiveCancelStakeBlock2, isRetry, err := vm.RunV2(db, block55, sendCancelStakeBlock2.AccountBlock, NewTestGlobalStatus(0, currentSnapshot))
	if receiveCancelStakeBlock2 == nil ||
		len(receiveCancelStakeBlock2.AccountBlock.SendBlockList) != 1 || isRetry || err != nil ||
		len(db.StorageMap[addr5][ToKey(stakeInfoKey)]) != 0 ||
		len(db.StorageMap[addr5][ToKey(beneficialKey)]) != 0 ||
		db.BalanceMap[addr5][ledger.ViteTokenId].Cmp(helper.Big0) != 0 ||
		len(receiveCancelStakeBlock2.AccountBlock.Data) != 33 ||
		receiveCancelStakeBlock2.AccountBlock.Data[32] != byte(0) ||
		receiveCancelStakeBlock2.AccountBlock.Quota != 0 ||
		receiveCancelStakeBlock.AccountBlock.SendBlockList[0].ToAddress != addr1 ||
		receiveCancelStakeBlock.AccountBlock.SendBlockList[0].Amount.Cmp(stakeAmount) != 0 ||
		receiveCancelStakeBlock.AccountBlock.SendBlockList[0].Fee.Sign() != 0 ||
		len(receiveCancelStakeBlock.AccountBlock.SendBlockList[0].Data) != 0 {
		t.Fatalf("receive cancel stake transaction 2 error")
	}
	db.AccountBlockMap[addr5][hash55] = receiveCancelStakeBlock2.AccountBlock
	hash56 := types.DataHash([]byte{5, 6})
	receiveCancelStakeBlock2.AccountBlock.SendBlockList[0].Hash = hash56
	receiveCancelStakeBlock2.AccountBlock.SendBlockList[0].PrevHash = hash55
	db.AccountBlockMap[addr5][hash56] = receiveCancelStakeBlock2.AccountBlock.SendBlockList[0]

	hash18 := types.DataHash([]byte{1, 8})
	block18 := &ledger.AccountBlock{
		Height:         8,
		AccountAddress: addr1,
		BlockType:      ledger.BlockTypeReceive,
		PrevHash:       hash18,
		FromBlockHash:  hash56,
		Hash:           hash18,
	}
	vm = NewVM(nil, nil)
	//vm.Debug = true
	db.Addr = addr1
	balance1.Add(balance1, stakeAmount)
	receiveCancelStakeRefundBlock2, isRetry, err := vm.RunV2(db, block18, receiveCancelStakeBlock2.AccountBlock.SendBlockList[0], NewTestGlobalStatus(0, currentSnapshot))
	if receiveCancelStakeRefundBlock2 == nil ||
		len(receiveCancelStakeRefundBlock2.AccountBlock.SendBlockList) != 0 || isRetry || err != nil ||
		db.BalanceMap[addr1][ledger.ViteTokenId].Cmp(balance1) != 0 ||
		receiveCancelStakeRefundBlock2.AccountBlock.Quota != 21000 {
		t.Fatalf("receive cancel stake refund transaction 2 error")
	}
	db.AccountBlockMap[addr1][hash18] = receiveCancelStakeRefundBlock2.AccountBlock
}

func TestContractsAssetV2(t *testing.T) {
	initEmptyFork(t)
	// prepare db
	viteTotalSupply := new(big.Int).Mul(big.NewInt(1e9), big.NewInt(1e18))
	db, addr1, _, hash12, snapshot2, _ := PrepareDb(viteTotalSupply)
	// issue
	balance1 := new(big.Int).Set(viteTotalSupply)
	addr2 := types.AddressAsset
	isReIssuable := true
	tokenName := "test token"
	tokenSymbol := "T"
	totalSupply := big.NewInt(1e10)
	maxSupply := new(big.Int).Mul(big.NewInt(2), totalSupply)
	decimals := uint8(3)
	ownerBurnOnly := true
	fee := new(big.Int).Mul(big.NewInt(1e3), util.AttovPerVite)
	stakeAmount := big.NewInt(0)
	balance1.Sub(balance1, fee)
	balance1.Sub(balance1, stakeAmount)
	block13Data, err := abi.ABIAsset.PackMethod(abi.MethodNameIssue, isReIssuable, tokenName, tokenSymbol, totalSupply, decimals, maxSupply, ownerBurnOnly)
	hash13 := types.DataHash([]byte{1, 3})
	block13 := &ledger.AccountBlock{
		Height:         3,
		ToAddress:      addr2,
		AccountAddress: addr1,
		Amount:         stakeAmount,
		TokenId:        ledger.ViteTokenId,
		BlockType:      ledger.BlockTypeSendCall,
		Fee:            fee,
		PrevHash:       hash12,
		Data:           block13Data,
		Hash:           hash13,
	}
	vm := NewVM(nil, nil)
	db.Addr = addr1
	sendIssueBlock, isRetry, err := vm.RunV2(db, block13, nil, nil)
	if sendIssueBlock == nil ||
		len(sendIssueBlock.AccountBlock.SendBlockList) != 0 || isRetry || err != nil ||
		db.BalanceMap[addr1][ledger.ViteTokenId].Cmp(balance1) != 0 ||
		!bytes.Equal(sendIssueBlock.AccountBlock.Data, block13Data) ||
		sendIssueBlock.AccountBlock.Amount.Cmp(stakeAmount) != 0 ||
		sendIssueBlock.AccountBlock.Fee.Cmp(fee) != 0 ||
		sendIssueBlock.AccountBlock.Quota != vm.gasTable.IssueQuota {
		t.Fatalf("send issue transaction error")
	}
	db.AccountBlockMap[addr1][hash13] = sendIssueBlock.AccountBlock

	hash21 := types.DataHash([]byte{2, 1})
	block21 := &ledger.AccountBlock{
		Height:         1,
		AccountAddress: addr2,
		BlockType:      ledger.BlockTypeReceive,
		FromBlockHash:  hash13,
		Hash:           hash21,
	}
	vm = NewVM(nil, nil)
	db.Addr = addr2
	receiveIssueBlock, isRetry, err := vm.RunV2(db, block21, sendIssueBlock.AccountBlock, NewTestGlobalStatus(0, snapshot2))
	tokenID := receiveIssueBlock.AccountBlock.SendBlockList[0].TokenId
	key := abi.GetTokenInfoKey(tokenID)
	tokenInfoData, _ := abi.ABIAsset.PackVariable(abi.VariableNameTokenInfo, tokenName, tokenSymbol, totalSupply, decimals, addr1, isReIssuable, maxSupply, ownerBurnOnly, uint16(0))
	if receiveIssueBlock == nil ||
		len(receiveIssueBlock.AccountBlock.SendBlockList) != 1 || isRetry || err != nil ||
		!bytes.Equal(db.StorageMap[addr2][ToKey(key)], tokenInfoData) ||
		db.BalanceMap[addr2][ledger.ViteTokenId].Cmp(stakeAmount) != 0 ||
		len(receiveIssueBlock.AccountBlock.Data) != 33 ||
		receiveIssueBlock.AccountBlock.Data[32] != byte(0) ||
		receiveIssueBlock.AccountBlock.Quota != 0 ||
		receiveIssueBlock.AccountBlock.SendBlockList[0].Amount.Cmp(totalSupply) != 0 ||
		receiveIssueBlock.AccountBlock.SendBlockList[0].ToAddress != addr1 ||
		receiveIssueBlock.AccountBlock.SendBlockList[0].BlockType != ledger.BlockTypeSendReward ||
		receiveIssueBlock.AccountBlock.SendBlockList[0].TokenId != tokenID ||
		len(db.LogList) != 1 ||
		db.LogList[0].Topics[0] != abi.ABIAsset.Events[util.FirstToLower(abi.MethodNameIssue)].Id() ||
		!bytes.Equal(db.LogList[0].Topics[1].Bytes(), helper.LeftPadBytes(tokenID.Bytes(), 32)) {
		t.Fatalf("receive issue transaction error")
	}
	db.AccountBlockMap[addr2] = make(map[types.Hash]*ledger.AccountBlock)
	db.AccountBlockMap[addr2][hash21] = receiveIssueBlock.AccountBlock
	hash22 := types.DataHash([]byte{2, 2})
	receiveIssueBlock.AccountBlock.SendBlockList[0].Hash = hash22
	receiveIssueBlock.AccountBlock.SendBlockList[0].PrevHash = hash21
	db.AccountBlockMap[addr2][hash22] = receiveIssueBlock.AccountBlock.SendBlockList[0]

	hash14 := types.DataHash([]byte{1, 4})
	block14 := &ledger.AccountBlock{
		Height:         4,
		AccountAddress: addr1,
		BlockType:      ledger.BlockTypeReceive,
		FromBlockHash:  hash22,
		PrevHash:       hash13,
		Hash:           hash14,
	}
	vm = NewVM(nil, nil)
	db.Addr = addr1
	tokenBalance := new(big.Int).Set(totalSupply)
	receiveIssueRewardBlock, isRetry, err := vm.RunV2(db, block14, receiveIssueBlock.AccountBlock.SendBlockList[0], nil)
	if receiveIssueRewardBlock == nil ||
		len(receiveIssueRewardBlock.AccountBlock.SendBlockList) != 0 || isRetry || err != nil ||
		db.BalanceMap[addr1][tokenID].Cmp(tokenBalance) != 0 ||
		receiveIssueRewardBlock.AccountBlock.Quota != 21000 {
		t.Fatalf("receive issue reward transaction error")
	}
	db.AccountBlockMap[addr1][hash14] = receiveIssueRewardBlock.AccountBlock

	// get contracts data
	db.Addr = types.AddressAsset
	if tokenInfo, _ := abi.GetTokenByID(db, tokenID); tokenInfo == nil || tokenInfo.TokenName != tokenName {
		t.Fatalf("get token by id failed")
	}
	if tokenMap, _ := abi.GetTokenMap(db); len(tokenMap) != 2 || tokenMap[tokenID].TokenName != tokenName {
		t.Fatalf("get token map failed")
	}

	if tokenMap, _ := abi.GetTokenMapByOwner(db, addr1); len(tokenMap) != 1 {
		t.Fatalf("get token map by owner failed")
	}

	// reIssue
	addr3, _, _ := types.CreateAddress()
	db.StorageMap[types.AddressQuota][ToKey(abi.GetStakeBeneficialKey(addr3))], _ = abi.ABIQuota.PackVariable(abi.VariableNameStakeBeneficial, new(big.Int).Mul(big.NewInt(1e9), big.NewInt(1e18)))
	reIssueAmount := big.NewInt(1000)
	block15Data, err := abi.ABIAsset.PackMethod(abi.MethodNameReIssue, tokenID, reIssueAmount, addr3)
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
	vm = NewVM(nil, nil)
	db.Addr = addr1
	sendReIssueBlock, isRetry, err := vm.RunV2(db, block15, nil, nil)
	if sendReIssueBlock == nil ||
		len(sendReIssueBlock.AccountBlock.SendBlockList) != 0 || isRetry || err != nil ||
		db.BalanceMap[addr1][ledger.ViteTokenId].Cmp(balance1) != 0 ||
		!bytes.Equal(sendReIssueBlock.AccountBlock.Data, block15Data) ||
		sendReIssueBlock.AccountBlock.Amount.Cmp(big.NewInt(0)) != 0 ||
		sendReIssueBlock.AccountBlock.Quota != vm.gasTable.ReIssueQuota {
		t.Fatalf("send reIssue transaction error")
	}
	db.AccountBlockMap[addr1][hash15] = sendReIssueBlock.AccountBlock

	hash23 := types.DataHash([]byte{2, 3})
	block23 := &ledger.AccountBlock{
		Height:         3,
		AccountAddress: addr2,
		BlockType:      ledger.BlockTypeReceive,
		FromBlockHash:  hash15,
		Hash:           hash21,
		PrevHash:       hash22,
	}
	vm = NewVM(nil, nil)
	db.Addr = addr2
	receiveReIssueBlock, isRetry, err := vm.RunV2(db, block23, sendReIssueBlock.AccountBlock, NewTestGlobalStatus(0, snapshot2))
	totalSupply = totalSupply.Add(totalSupply, reIssueAmount)
	tokenInfoData, _ = abi.ABIAsset.PackVariable(abi.VariableNameTokenInfo, tokenName, tokenSymbol, totalSupply, decimals, addr1, isReIssuable, maxSupply, ownerBurnOnly, uint16(0))
	if receiveReIssueBlock == nil ||
		len(receiveReIssueBlock.AccountBlock.SendBlockList) != 1 || isRetry || err != nil ||
		!bytes.Equal(db.StorageMap[addr2][ToKey(key)], tokenInfoData) ||
		db.BalanceMap[addr2][ledger.ViteTokenId].Cmp(stakeAmount) != 0 ||
		len(receiveReIssueBlock.AccountBlock.Data) != 33 ||
		receiveReIssueBlock.AccountBlock.Data[32] != byte(0) ||
		receiveReIssueBlock.AccountBlock.Quota != 0 ||
		receiveReIssueBlock.AccountBlock.SendBlockList[0].Amount.Cmp(reIssueAmount) != 0 ||
		receiveReIssueBlock.AccountBlock.SendBlockList[0].ToAddress != addr3 ||
		receiveReIssueBlock.AccountBlock.SendBlockList[0].BlockType != ledger.BlockTypeSendReward ||
		receiveReIssueBlock.AccountBlock.SendBlockList[0].TokenId != tokenID ||
		len(db.LogList) != 2 ||
		db.LogList[1].Topics[0] != abi.ABIAsset.Events[util.FirstToLower(abi.MethodNameReIssue)].Id() ||
		!bytes.Equal(db.LogList[1].Topics[1].Bytes(), helper.LeftPadBytes(tokenID.Bytes(), 32)) {
		t.Fatalf("receive reIssue transaction error")
	}
	db.AccountBlockMap[addr2] = make(map[types.Hash]*ledger.AccountBlock)
	db.AccountBlockMap[addr2][hash23] = receiveReIssueBlock.AccountBlock
	hash24 := types.DataHash([]byte{2, 4})
	receiveReIssueBlock.AccountBlock.SendBlockList[0].Hash = hash24
	receiveReIssueBlock.AccountBlock.SendBlockList[0].PrevHash = hash23
	db.AccountBlockMap[addr2][hash24] = receiveReIssueBlock.AccountBlock.SendBlockList[0]

	hash31 := types.DataHash([]byte{3, 1})
	block31 := &ledger.AccountBlock{
		Height:         1,
		AccountAddress: addr3,
		BlockType:      ledger.BlockTypeReceive,
		FromBlockHash:  hash24,
		Hash:           hash31,
	}
	vm = NewVM(nil, nil)
	db.Addr = addr3
	receiveReIssueRewardBlock, isRetry, err := vm.RunV2(db, block31, receiveReIssueBlock.AccountBlock.SendBlockList[0], nil)
	if receiveReIssueRewardBlock == nil ||
		len(receiveReIssueRewardBlock.AccountBlock.SendBlockList) != 0 || isRetry || err != nil ||
		db.BalanceMap[addr3][tokenID].Cmp(reIssueAmount) != 0 ||
		receiveReIssueRewardBlock.AccountBlock.Quota != 21000 {
		t.Fatalf("receive reIssue reward transaction error")
	}
	db.AccountBlockMap[addr3] = make(map[types.Hash]*ledger.AccountBlock)
	db.AccountBlockMap[addr3][hash31] = receiveReIssueRewardBlock.AccountBlock

	// burn
	block16Data, err := abi.ABIAsset.PackMethod(abi.MethodNameBurn)
	hash16 := types.DataHash([]byte{1, 6})
	burnAmount := big.NewInt(1000)
	block16 := &ledger.AccountBlock{
		Height:         6,
		ToAddress:      addr2,
		AccountAddress: addr1,
		Amount:         burnAmount,
		TokenId:        tokenID,
		BlockType:      ledger.BlockTypeSendCall,
		Fee:            big.NewInt(0),
		PrevHash:       hash15,
		Data:           block16Data,
		Hash:           hash16,
	}
	vm = NewVM(nil, nil)
	db.Addr = addr1
	sendBurnBlock, isRetry, err := vm.RunV2(db, block16, nil, nil)
	totalSupply = totalSupply.Sub(totalSupply, burnAmount)
	tokenBalance = tokenBalance.Sub(tokenBalance, burnAmount)
	if sendBurnBlock == nil ||
		len(sendBurnBlock.AccountBlock.SendBlockList) != 0 || isRetry || err != nil ||
		db.BalanceMap[addr1][ledger.ViteTokenId].Cmp(balance1) != 0 ||
		db.BalanceMap[addr1][tokenID].Cmp(tokenBalance) != 0 ||
		!bytes.Equal(sendBurnBlock.AccountBlock.Data, block16Data) ||
		sendBurnBlock.AccountBlock.Amount.Cmp(burnAmount) != 0 ||
		sendBurnBlock.AccountBlock.Quota != vm.gasTable.BurnQuota {
		t.Fatalf("send burn transaction error")
	}
	db.AccountBlockMap[addr1][hash16] = sendBurnBlock.AccountBlock

	hash25 := types.DataHash([]byte{2, 5})
	block25 := &ledger.AccountBlock{
		Height:         5,
		AccountAddress: addr2,
		BlockType:      ledger.BlockTypeReceive,
		FromBlockHash:  hash16,
		Hash:           hash25,
		PrevHash:       hash24,
	}
	vm = NewVM(nil, nil)
	db.Addr = addr2
	receiveBurnBlock, isRetry, err := vm.RunV2(db, block25, sendBurnBlock.AccountBlock, NewTestGlobalStatus(0, snapshot2))
	tokenInfoData, _ = abi.ABIAsset.PackVariable(abi.VariableNameTokenInfo, tokenName, tokenSymbol, totalSupply, decimals, addr1, isReIssuable, maxSupply, ownerBurnOnly, uint16(0))
	if receiveBurnBlock == nil ||
		len(receiveBurnBlock.AccountBlock.SendBlockList) != 0 || isRetry || err != nil ||
		!bytes.Equal(db.StorageMap[addr2][ToKey(key)], tokenInfoData) ||
		db.BalanceMap[addr2][ledger.ViteTokenId].Cmp(stakeAmount) != 0 ||
		db.BalanceMap[addr2][tokenID].Cmp(helper.Big0) != 0 ||
		len(receiveBurnBlock.AccountBlock.Data) != 33 ||
		receiveBurnBlock.AccountBlock.Data[32] != byte(0) ||
		receiveBurnBlock.AccountBlock.Quota != 0 ||
		len(db.LogList) != 3 ||
		db.LogList[2].Topics[0] != abi.ABIAsset.Events[util.FirstToLower(abi.MethodNameBurn)].Id() ||
		!bytes.Equal(db.LogList[2].Topics[1].Bytes(), helper.LeftPadBytes(tokenID.Bytes(), 32)) ||
		!bytes.Equal(db.LogList[2].Data, append(helper.LeftPadBytes(addr1.Bytes(), 32), helper.LeftPadBytes(burnAmount.Bytes(), 32)...)) {
		t.Fatalf("receive burn transaction error")
	}
	db.AccountBlockMap[addr2] = make(map[types.Hash]*ledger.AccountBlock)
	db.AccountBlockMap[addr2][hash25] = receiveBurnBlock.AccountBlock

	// transfer owner
	block17Data, err := abi.ABIAsset.PackMethod(abi.MethodNameTransferOwnership, tokenID, addr3)
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
	vm = NewVM(nil, nil)
	db.Addr = addr1
	sendTransferOwnershipBlock, isRetry, err := vm.RunV2(db, block17, nil, nil)
	if sendTransferOwnershipBlock == nil ||
		len(sendTransferOwnershipBlock.AccountBlock.SendBlockList) != 0 || isRetry || err != nil ||
		db.BalanceMap[addr1][ledger.ViteTokenId].Cmp(balance1) != 0 ||
		db.BalanceMap[addr1][tokenID].Cmp(tokenBalance) != 0 ||
		!bytes.Equal(sendTransferOwnershipBlock.AccountBlock.Data, block17Data) ||
		sendTransferOwnershipBlock.AccountBlock.Amount.Cmp(helper.Big0) != 0 ||
		sendTransferOwnershipBlock.AccountBlock.Quota != vm.gasTable.TransferOwnershipQuota {
		t.Fatalf("send transfer owner transaction error")
	}
	db.AccountBlockMap[addr1][hash17] = sendTransferOwnershipBlock.AccountBlock

	hash26 := types.DataHash([]byte{2, 6})
	block26 := &ledger.AccountBlock{
		Height:         6,
		AccountAddress: addr2,
		BlockType:      ledger.BlockTypeReceive,
		FromBlockHash:  hash17,
		Hash:           hash26,
		PrevHash:       hash25,
	}
	vm = NewVM(nil, nil)
	db.Addr = addr2
	receiveTransferOwnershipBlock, isRetry, err := vm.RunV2(db, block26, sendTransferOwnershipBlock.AccountBlock, NewTestGlobalStatus(0, snapshot2))
	tokenInfoData, _ = abi.ABIAsset.PackVariable(abi.VariableNameTokenInfo, tokenName, tokenSymbol, totalSupply, decimals, addr3, isReIssuable, maxSupply, ownerBurnOnly, uint16(0))
	if receiveTransferOwnershipBlock == nil ||
		len(receiveTransferOwnershipBlock.AccountBlock.SendBlockList) != 0 || isRetry || err != nil ||
		!bytes.Equal(db.StorageMap[addr2][ToKey(key)], tokenInfoData) ||
		db.BalanceMap[addr2][ledger.ViteTokenId].Cmp(stakeAmount) != 0 ||
		db.BalanceMap[addr2][tokenID].Cmp(helper.Big0) != 0 ||
		len(receiveTransferOwnershipBlock.AccountBlock.Data) != 33 ||
		receiveTransferOwnershipBlock.AccountBlock.Data[32] != byte(0) ||
		receiveTransferOwnershipBlock.AccountBlock.Quota != 0 ||
		len(db.LogList) != 4 ||
		db.LogList[3].Topics[0] != abi.ABIAsset.Events[util.FirstToLower(abi.MethodNameTransferOwnership)].Id() ||
		!bytes.Equal(db.LogList[3].Topics[1].Bytes(), helper.LeftPadBytes(tokenID.Bytes(), 32)) ||
		!bytes.Equal(db.LogList[3].Data, helper.LeftPadBytes(addr3.Bytes(), 32)) {
		t.Fatalf("receive transfer owner transaction error")
	}
	db.AccountBlockMap[addr2] = make(map[types.Hash]*ledger.AccountBlock)
	db.AccountBlockMap[addr2][hash26] = receiveTransferOwnershipBlock.AccountBlock

	db.Addr = types.AddressAsset
	if tokenMap, _ := abi.GetTokenMapByOwner(db, addr1); len(tokenMap) != 0 {
		t.Fatalf("get token map by owner failed")
	}
	if tokenMap, _ := abi.GetTokenMapByOwner(db, addr3); len(tokenMap) != 1 {
		t.Fatalf("get token map by owner failed")
	}

	// change token type
	block32Data, err := abi.ABIAsset.PackMethod(abi.MethodNameDisableReIssue, tokenID)
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
	vm = NewVM(nil, nil)
	db.Addr = addr3
	sendDisableReIssueBlock, isRetry, err := vm.RunV2(db, block32, nil, nil)
	if sendDisableReIssueBlock == nil ||
		len(sendDisableReIssueBlock.AccountBlock.SendBlockList) != 0 || isRetry || err != nil ||
		!bytes.Equal(sendDisableReIssueBlock.AccountBlock.Data, block32Data) ||
		sendDisableReIssueBlock.AccountBlock.Amount.Cmp(helper.Big0) != 0 ||
		sendDisableReIssueBlock.AccountBlock.Quota != vm.gasTable.DisableReIssueQuota {
		t.Fatalf("send change token type transaction error")
	}
	db.AccountBlockMap[addr3][hash32] = sendDisableReIssueBlock.AccountBlock

	hash27 := types.DataHash([]byte{2, 7})
	block27 := &ledger.AccountBlock{
		Height:         7,
		AccountAddress: addr2,
		BlockType:      ledger.BlockTypeReceive,
		FromBlockHash:  hash31,
		Hash:           hash27,
		PrevHash:       hash26,
	}
	vm = NewVM(nil, nil)
	db.Addr = addr2
	receiveDisableReIssueBlock, isRetry, err := vm.RunV2(db, block27, sendDisableReIssueBlock.AccountBlock, NewTestGlobalStatus(0, snapshot2))
	tokenInfoData, _ = abi.ABIAsset.PackVariable(abi.VariableNameTokenInfo, tokenName, tokenSymbol, totalSupply, decimals, addr3, false, big.NewInt(0), false, uint16(0))
	if receiveDisableReIssueBlock == nil ||
		len(receiveDisableReIssueBlock.AccountBlock.SendBlockList) != 0 || isRetry || err != nil ||
		!bytes.Equal(db.StorageMap[addr2][ToKey(key)], tokenInfoData) ||
		db.BalanceMap[addr2][ledger.ViteTokenId].Cmp(stakeAmount) != 0 ||
		db.BalanceMap[addr2][tokenID].Cmp(helper.Big0) != 0 ||
		len(receiveDisableReIssueBlock.AccountBlock.Data) != 33 ||
		receiveDisableReIssueBlock.AccountBlock.Data[32] != byte(0) ||
		receiveDisableReIssueBlock.AccountBlock.Quota != 0 ||
		len(db.LogList) != 5 ||
		db.LogList[4].Topics[0] != abi.ABIAsset.Events[util.FirstToLower(abi.MethodNameDisableReIssue)].Id() ||
		!bytes.Equal(db.LogList[4].Topics[1].Bytes(), helper.LeftPadBytes(tokenID.Bytes(), 32)) {
		t.Fatalf("receive change token type transaction error")
	}
	db.AccountBlockMap[addr2] = make(map[types.Hash]*ledger.AccountBlock)
	db.AccountBlockMap[addr2][hash27] = receiveDisableReIssueBlock.AccountBlock

	db.Addr = types.AddressAsset
	if tokenMap, _ := abi.GetTokenMapByOwner(db, addr3); len(tokenMap) != 1 {
		t.Fatalf("get token map by owner failed")
	}
	if tokenMap, _ := abi.GetTokenMapByOwner(db, addr1); len(tokenMap) != 0 {
		t.Fatalf("get token map by owner failed")
	}

	// issue again
	balance1.Sub(balance1, fee)
	balance1.Sub(balance1, stakeAmount)
	block18Data, err := abi.ABIAsset.PackMethod(abi.MethodNameIssue, isReIssuable, tokenName, tokenSymbol, totalSupply, decimals, maxSupply, ownerBurnOnly)
	hash18 := types.DataHash([]byte{1, 8})
	block18 := &ledger.AccountBlock{
		Height:         8,
		ToAddress:      addr2,
		AccountAddress: addr1,
		Amount:         stakeAmount,
		TokenId:        ledger.ViteTokenId,
		BlockType:      ledger.BlockTypeSendCall,
		Fee:            fee,
		PrevHash:       hash17,
		Data:           block18Data,
		Hash:           hash18,
	}
	vm = NewVM(nil, nil)
	db.Addr = addr1
	sendIssueBlock2, isRetry, err := vm.RunV2(db, block18, nil, nil)
	if sendIssueBlock2 == nil ||
		len(sendIssueBlock2.AccountBlock.SendBlockList) != 0 || isRetry || err != nil ||
		db.BalanceMap[addr1][ledger.ViteTokenId].Cmp(balance1) != 0 ||
		!bytes.Equal(sendIssueBlock2.AccountBlock.Data, block18Data) ||
		sendIssueBlock2.AccountBlock.Amount.Cmp(stakeAmount) != 0 ||
		sendIssueBlock2.AccountBlock.Fee.Cmp(fee) != 0 ||
		sendIssueBlock2.AccountBlock.Quota != vm.gasTable.IssueQuota {
		t.Fatalf("send issue transaction 2 error")
	}
	db.AccountBlockMap[addr1][hash18] = sendIssueBlock2.AccountBlock

	hash28 := types.DataHash([]byte{2, 8})
	block28 := &ledger.AccountBlock{
		Height:         8,
		AccountAddress: addr2,
		BlockType:      ledger.BlockTypeReceive,
		FromBlockHash:  hash18,
		Hash:           hash28,
		PrevHash:       hash27,
	}
	vm = NewVM(nil, nil)
	db.Addr = addr2
	receiveIssueBlock2, isRetry, err := vm.RunV2(db, block28, sendIssueBlock2.AccountBlock, NewTestGlobalStatus(0, snapshot2))
	newTokenID := receiveIssueBlock2.AccountBlock.SendBlockList[0].TokenId
	newKey := abi.GetTokenInfoKey(newTokenID)
	newTokenInfoData, _ := abi.ABIAsset.PackVariable(abi.VariableNameTokenInfo, tokenName, tokenSymbol, totalSupply, decimals, addr1, isReIssuable, maxSupply, ownerBurnOnly, uint16(1))
	if receiveIssueBlock2 == nil ||
		len(receiveIssueBlock2.AccountBlock.SendBlockList) != 1 || isRetry || err != nil ||
		!bytes.Equal(db.StorageMap[addr2][ToKey(newKey)], newTokenInfoData) ||
		db.BalanceMap[addr2][ledger.ViteTokenId].Cmp(stakeAmount) != 0 ||
		len(receiveIssueBlock2.AccountBlock.Data) != 33 ||
		receiveIssueBlock2.AccountBlock.Data[32] != byte(0) ||
		receiveIssueBlock2.AccountBlock.Quota != 0 ||
		receiveIssueBlock2.AccountBlock.SendBlockList[0].Amount.Cmp(totalSupply) != 0 ||
		receiveIssueBlock2.AccountBlock.SendBlockList[0].ToAddress != addr1 ||
		receiveIssueBlock2.AccountBlock.SendBlockList[0].BlockType != ledger.BlockTypeSendReward ||
		receiveIssueBlock2.AccountBlock.SendBlockList[0].TokenId != newTokenID ||
		len(db.LogList) != 6 ||
		db.LogList[5].Topics[0] != abi.ABIAsset.Events[util.FirstToLower(abi.MethodNameIssue)].Id() ||
		!bytes.Equal(db.LogList[5].Topics[1].Bytes(), helper.LeftPadBytes(newTokenID.Bytes(), 32)) {
		t.Fatalf("receive issue transaction 2 error")
	}
	db.AccountBlockMap[addr2][hash28] = receiveIssueBlock2.AccountBlock
	hash2a := types.DataHash([]byte{2, 10})
	receiveIssueBlock2.AccountBlock.SendBlockList[0].Hash = hash2a
	receiveIssueBlock2.AccountBlock.SendBlockList[0].PrevHash = hash28
	db.AccountBlockMap[addr2][hash2a] = receiveIssueBlock2.AccountBlock.SendBlockList[0]
}

//func TestContractsAssetV3(t *testing.T) {
//	initCustomFork(t)
//	// prepare db
//	viteTotalSupply := new(big.Int).Mul(big.NewInt(1e9), big.NewInt(1e18))
//	db, addr1, _, hash12, snapshot, _ := test.PrepareDb(viteTotalSupply)
//	t2 := time.Unix(1600663514, 0)
//	snapshot20 := &ledger.SnapshotBlock{Height: 2000, Timestamp: &t2, Hash: types.DataHash([]byte{10, 2})}
//	db.SnapshotBlockList = append(db.SnapshotBlockList, snapshot20)
//
//	// issue
//	balance1 := new(big.Int).Set(viteTotalSupply)
//	addr2 := types.AddressAsset
//	isReIssuable := true
//	tokenName := "My Coin"
//	tokenSymbol := "MYC"
//	totalSupply := big.NewInt(1e10)
//	maxSupply := new(big.Int).Mul(big.NewInt(2), totalSupply)
//	decimals := uint8(3)
//
//	fee := new(big.Int).Mul(big.NewInt(1e3), util.AttovPerVite)
//
//	balance1.Sub(balance1, fee)
//	calldata, _ := builtin2.ContractAssetV3.Abi.PackMethod("issue", isReIssuable, tokenName, tokenSymbol, totalSupply, decimals, maxSupply)
//
//	sendBlockIssue := &ledger.AccountBlock{
//		Height:         3,
//		ToAddress:      addr2,
//		AccountAddress: addr1,
//		Amount:         big.NewInt(0),
//		TokenId:        ledger.ViteTokenId,
//		BlockType:      ledger.BlockTypeSendCall,
//		Fee:            fee,
//		PrevHash:       hash12,
//		Data:           calldata,
//		Hash:           types.DataHash([]byte{1, 3}),
//	}
//	receiveBlockIssue := &ledger.AccountBlock{
//		Height:         1,
//		AccountAddress: addr2,
//		BlockType:      ledger.BlockTypeReceive,
//		FromBlockHash:  sendBlockIssue.Hash,
//		Hash:           types.DataHash([]byte{2, 1}),
//	}
//
//	vm := NewVM(nil, nil)
//	db.Addr = addr2
//
//	vmBlock, _, _ := vm.RunV2(db, receiveBlockIssue, sendBlockIssue, NewTestGlobalStatus(0, snapshot))
//
//	triggered := vmBlock.AccountBlock.SendBlockList[0]
//	assert.NotEmpty(t, triggered)
//	assert.Equal(t, ledger.BlockTypeSendReward, triggered.BlockType)
//	assert.Equal(t, totalSupply, triggered.Amount)
//	tokenId := triggered.TokenId
//	tokenInfo, err := abi.GetTokenByID(db, triggered.TokenId)
//	assert.NoError(t, err)
//	assert.Equal(t, tokenName, tokenInfo.TokenName)
//	assert.Equal(t, tokenSymbol, tokenInfo.TokenSymbol)
//	assert.Equal(t, totalSupply, tokenInfo.TotalSupply)
//	assert.Equal(t, maxSupply, tokenInfo.MaxSupply)
//	assert.Equal(t, decimals, tokenInfo.Decimals)
//
//	// name
//	calldata, err = builtin2.ContractAssetV3.Abi.PackMethod("name", tokenId)
//	assert.NoError(t, err)
//	sendBlockName := &ledger.AccountBlock{
//		Height:         4,
//		ToAddress:      addr2,
//		AccountAddress: addr1,
//		Amount:         big.NewInt(0),
//		TokenId:        ledger.ViteTokenId,
//		BlockType:      ledger.BlockTypeSendSyncCall,
//		Fee:            big.NewInt(0),
//		PrevHash:       hash12,
//		Data:           calldata,
//		Hash:           types.DataHash([]byte{1, 4}),
//	}
//	db.SetExecutionContext(&sendBlockName.Hash, &ledger.ExecutionContext{CallbackId: *big.NewInt(1)})
//	receiveBlockName := &ledger.AccountBlock{
//		Height:         2,
//		AccountAddress: addr2,
//		BlockType:      ledger.BlockTypeReceive,
//		FromBlockHash:  sendBlockName.Hash,
//		Hash:           types.DataHash([]byte{2, 2}),
//	}
//
//	vmBlock, _, err = vm.RunV2(db, receiveBlockName, sendBlockName, NewTestGlobalStatus(0, snapshot))
//	assert.NoError(t, err)
//
//	ret := vmBlock.AccountBlock.SendBlockList[0].Data
//	gotName := new(string)
//	builtin2.ContractAssetV3.Abi.Methods["name"].Outputs.Unpack(gotName, ret[4:])
//	assert.NotEmpty(t, gotName)
//	assert.Equal(t, "My Coin", *gotName)
//
//	// mint
//	mintAmount := big.NewInt(1000)
//	calldata, err = builtin2.ContractAssetV3.Abi.PackMethod("mint", tokenId, addr1, mintAmount)
//	assert.NoError(t, err)
//	sendBlockMint := &ledger.AccountBlock{
//		Height:         5,
//		ToAddress:      addr2,
//		AccountAddress: addr1,
//		Amount:         big.NewInt(0),
//		TokenId:        ledger.ViteTokenId,
//		BlockType:      ledger.BlockTypeSendCall,
//		Fee:            big.NewInt(0),
//		PrevHash:       hash12,
//		Data:           calldata,
//		Hash:           types.DataHash([]byte{1, 5}),
//	}
//	receiveBlockMint := &ledger.AccountBlock{
//		Height:         3,
//		AccountAddress: addr2,
//		BlockType:      ledger.BlockTypeReceive,
//		FromBlockHash:  sendBlockMint.Hash,
//		Hash:           types.DataHash([]byte{2, 3}),
//	}
//
//	vmBlock, _, _ = vm.RunV2(db, receiveBlockMint, sendBlockMint, NewTestGlobalStatus(0, snapshot))
//	triggered = vmBlock.AccountBlock.SendBlockList[0]
//	assert.NotEmpty(t, triggered)
//	assert.Equal(t, ledger.BlockTypeSendReward, triggered.BlockType)
//
//	tokenInfo, err = abi.GetTokenByID(db, tokenId)
//	assert.NoError(t, err)
//
//	totalSupply = big.NewInt(0).Add(totalSupply, mintAmount)
//	assert.Equal(t, totalSupply, tokenInfo.TotalSupply)
//
//	assert.Equal(t, tokenId, triggered.TokenId)
//	assert.Equal(t, mintAmount, triggered.Amount)
//
//	// burn
//	burnAmount := big.NewInt(500)
//	calldata, err = builtin2.ContractAssetV3.Abi.PackMethod("burn")
//	assert.NoError(t, err)
//	sendBlockBurn := &ledger.AccountBlock{
//		Height:         6,
//		ToAddress:      addr2,
//		AccountAddress: addr1,
//		Amount:         burnAmount,
//		TokenId:        tokenId,
//		BlockType:      ledger.BlockTypeSendCall,
//		Fee:            big.NewInt(0),
//		PrevHash:       hash12,
//		Data:           calldata,
//		Hash:           types.DataHash([]byte{1, 6}),
//	}
//	receiveBlockBurn := &ledger.AccountBlock{
//		Height:         4,
//		AccountAddress: addr2,
//		BlockType:      ledger.BlockTypeReceive,
//		FromBlockHash:  sendBlockBurn.Hash,
//		Hash:           types.DataHash([]byte{2, 2}),
//	}
//
//	vmBlock, _, _ = vm.RunV2(db, receiveBlockBurn, sendBlockBurn, NewTestGlobalStatus(0, snapshot))
//
//	assert.Empty(t, vmBlock.AccountBlock.SendBlockList)
//	tokenInfo, err = abi.GetTokenByID(db, tokenId)
//	assert.NoError(t, err)
//	totalSupply = big.NewInt(0).Sub(totalSupply, burnAmount)
//	assert.Equal(t, totalSupply, tokenInfo.TotalSupply)
//
//	// issue another token by BEP19
//	calldata, err = builtin2.ContractAssetV3.Abi.PackMethod("issue", true, "My Coin2", "MYC2", big.NewInt(2e10), uint8(2), big.NewInt(2e10))
//	assert.NoError(t, err)
//
//	sendBlockIssue2 := &ledger.AccountBlock{
//		Height:         16,
//		ToAddress:      addr2,
//		AccountAddress: addr1,
//		Amount:         big.NewInt(0),
//		TokenId:        ledger.ViteTokenId,
//		BlockType:      ledger.BlockTypeSendSyncCall,
//		Fee:            fee,
//		PrevHash:       hash12,
//		Data:           calldata,
//		Hash:           types.DataHash([]byte{1, 16}),
//	}
//	db.SetExecutionContext(&sendBlockIssue2.Hash, &ledger.ExecutionContext{CallbackId: *big.NewInt(123)})
//	receiveBlockIssue2 := &ledger.AccountBlock{
//		Height:         14,
//		AccountAddress: addr2,
//		BlockType:      ledger.BlockTypeReceive,
//		FromBlockHash:  sendBlockIssue2.Hash,
//		Hash:           types.DataHash([]byte{2, 14}),
//	}
//
//	vmBlock, _, _ = vm.RunV2(db, receiveBlockIssue2, sendBlockIssue2, NewTestGlobalStatus(0, snapshot))
//
//	assert.NotEmpty(t, vmBlock.AccountBlock.SendBlockList)
//	assert.Equal(t, 2, len(vmBlock.AccountBlock.SendBlockList))
//
//	mintBlock := vmBlock.AccountBlock.SendBlockList[0]
//	callbackBlock := vmBlock.AccountBlock.SendBlockList[1]
//
//	assert.Equal(t, ledger.BlockTypeSendReward, mintBlock.BlockType)
//	assert.Equal(t, ledger.BlockTypeSendCallback, callbackBlock.BlockType)
//
//	ret = callbackBlock.Data
//	assert.Equal(t, big.NewInt(123), big.NewInt(0).SetBytes(ret[:4]))
//	err = builtin2.ContractAssetV3.Abi.Methods["issue"].Outputs.Unpack(&tokenId, ret[4:])
//	assert.NoError(t, err)
//	assert.NotEmpty(t, tokenId)
//
//	tokenInfo, err = abi.GetTokenByID(db, tokenId)
//	assert.NoError(t, err)
//	assert.Equal(t, "My Coin2", tokenInfo.TokenName)
//	assert.Equal(t, "MYC2", tokenInfo.TokenSymbol)
//	assert.Equal(t, big.NewInt(2e10), tokenInfo.TotalSupply)
//	assert.Equal(t, big.NewInt(2e10), tokenInfo.MaxSupply)
//	assert.Equal(t, uint8(2), tokenInfo.Decimals)
//
//	context, err := db.GetExecutionContext(&callbackBlock.Hash)
//	assert.NoError(t, err)
//	assert.NotEmpty(t, context)
//	assert.Equal(t, sendBlockIssue2.Hash, context.ReferrerSendHash)
//}

func TestCheckTokenName(t *testing.T) {
	initEmptyFork(t)
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
		{"abc2", true},
		{"My Coin2", true},
	}
	for _, test := range tests {
		if ok, _ := regexp.MatchString("^([0-9a-zA-Z_]+[ ]?)*[0-9a-zA-Z_]$", test.data); ok != test.exp {
			t.Fatalf("match string error, [%v] expected %v, got %v", test.data, test.exp, ok)
		}
	}
}

func TestGenesisBlockData(t *testing.T) {
	initEmptyFork(t)
	tokenName := "ViteToken"
	tokenSymbol := "ViteToken"
	decimals := uint8(18)
	totalSupply := new(big.Int).Mul(big.NewInt(1e18), big.NewInt(1e9))
	viteAddress, _, _ := types.CreateAddress()
	issueData, err := abi.ABIAsset.PackVariable(abi.VariableNameTokenInfo, tokenName, tokenSymbol, totalSupply, decimals, viteAddress, true, helper.Tt256m1, false, uint16(0))
	if err != nil {
		t.Fatalf("pack issue data error, %v", err)
	}
	fmt.Println("-------------mintage genesis block-------------")
	fmt.Printf("address: %v\n", hex.EncodeToString(types.AddressAsset.Bytes()))
	fmt.Printf("AccountBlock{\n\tBlockType: %v\n\tAccountAddress: %v,\n\tHeight: %v,\n\tAmount: %v,\n\tTokenId:ledger.ViteTokenId,\n\tQuota:0,\n\tFee:%v\n}\n",
		ledger.BlockTypeReceive, hex.EncodeToString(types.AddressAsset.Bytes()), 1, big.NewInt(0), big.NewInt(0))
	fmt.Printf("Storage:{\n\t%v:%v\n}\n", hex.EncodeToString(abi.GetTokenInfoKey(ledger.ViteTokenId)), hex.EncodeToString(issueData))

	fmt.Println("-------------vite owner genesis block-------------")
	fmt.Println("address: viteAddress")
	fmt.Printf("AccountBlock{\n\tBlockType: %v,\n\tAccountAddress: viteAddress,\n\tHeight: %v,\n\tAmount: %v,\n\tTokenId:ledger.ViteTokenId,\n\tQuota:0,\n\tFee:%v,\n\tData:%v,\n}\n",
		ledger.BlockTypeReceive, 1, totalSupply, big.NewInt(0), []byte{})
	fmt.Printf("Storage:{\n\t$balance:ledger.ViteTokenId:%v\n}\n", totalSupply)

	conditionRegisterData, err := abi.ABIGovernance.PackVariable(abi.VariableNameRegisterStakeParam, new(big.Int).Mul(big.NewInt(1e5), util.AttovPerVite), ledger.ViteTokenId, uint64(3600*24*90))
	if err != nil {
		t.Fatalf("pack register condition variable error, %v", err)
	}
	snapshotConsensusGroupData, err := abi.ABIGovernance.PackVariable(abi.VariableNameConsensusGroupInfo,
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
	commonConsensusGroupData, err := abi.ABIGovernance.PackVariable(abi.VariableNameConsensusGroupInfo,
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
	fmt.Printf("address:%v\n", hex.EncodeToString(types.AddressGovernance.Bytes()))
	fmt.Printf("AccountBlock{\n\tBlockType: %v,\n\tAccountAddress: %v,\n\tHeight: %v,\n\tAmount: %v,\n\tTokenId:ledger.ViteTokenId,\n\tQuota:0,\n\tFee:%v,\n\tData:%v,\n}\n",
		ledger.BlockTypeReceive, hex.EncodeToString(types.AddressGovernance.Bytes()), 1, big.NewInt(0), big.NewInt(0), []byte{})
	fmt.Printf("Storage:{\n\t%v:%v,\n\t%v:%v}\n", hex.EncodeToString(abi.GetConsensusGroupInfoKey(types.SNAPSHOT_GID)), hex.EncodeToString(snapshotConsensusGroupData), hex.EncodeToString(abi.GetConsensusGroupInfoKey(types.DELEGATE_GID)), hex.EncodeToString(commonConsensusGroupData))

	fmt.Println("-------------snapshot consensus group and common consensus group register genesis block-------------")
	fmt.Printf("address:%v\n", hex.EncodeToString(types.AddressGovernance.Bytes()))
	fmt.Printf("AccountBlock{\n\tBlockType: %v,\n\tAccountAddress: %v,\n\tHeight: %v,\n\tAmount: %v,\n\tTokenId:ledger.ViteTokenId,\n\tQuota:0,\n\tFee:%v,\n\tData:%v,\n}\n",
		ledger.BlockTypeReceive, hex.EncodeToString(types.AddressGovernance.Bytes()), 1, big.NewInt(0), big.NewInt(0), []byte{})
	fmt.Printf("Storage:{\n")
	for i := 1; i <= 25; i++ {
		Addr, _, _ := types.CreateAddress()
		registerData, err := abi.ABIGovernance.PackVariable(abi.VariableNameRegistrationInfo, "node"+strconv.Itoa(i), Addr, Addr, helper.Big0, uint64(1), int64(1), int64(0), []types.Address{Addr})
		if err != nil {
			t.Fatalf("pack registration variable error, %v", err)
		}
		snapshotKey := abi.GetRegistrationInfoKey("snapshotNode1", types.SNAPSHOT_GID)
		fmt.Printf("\t%v: %v\n", hex.EncodeToString(snapshotKey), hex.EncodeToString(registerData))
	}
	fmt.Println("}")
}

type emptyConsensusReaderTest struct {
	ti        timeIndex
	detailMap map[uint64]map[string]*ConsensusDetail
}

type ConsensusDetail struct {
	BlockNum         uint64
	ExpectedBlockNum uint64
	VoteCount        *big.Int
}

func newConsensusReaderTest(genesisTime int64, interval int64, detailMap map[uint64]map[string]*ConsensusDetail) *emptyConsensusReaderTest {
	return &emptyConsensusReaderTest{timeIndex{time.Unix(genesisTime, 0), time.Second * time.Duration(interval)}, detailMap}
}

func (r *emptyConsensusReaderTest) DayStats(startIndex uint64, endIndex uint64) ([]*core.DayStats, error) {
	list := make([]*core.DayStats, 0)
	if len(r.detailMap) == 0 {
		return list, nil
	}
	for i := startIndex; i <= endIndex; i++ {
		if i > endIndex {
			break
		}
		m, ok := r.detailMap[i]
		if !ok {
			continue
		}
		blockNum := uint64(0)
		expectedBlockNum := uint64(0)
		voteCount := big.NewInt(0)
		statusMap := make(map[string]*core.SbpStats, len(m))
		for name, detail := range m {
			blockNum = blockNum + detail.BlockNum
			expectedBlockNum = expectedBlockNum + detail.ExpectedBlockNum
			voteCount.Add(voteCount, detail.VoteCount)
			statusMap[name] = &core.SbpStats{i, detail.BlockNum, detail.ExpectedBlockNum, &core.BigInt{detail.VoteCount}, name}
		}
		list = append(list, &core.DayStats{Index: i, Stats: statusMap, VoteSum: &core.BigInt{voteCount}, BlockTotal: blockNum})
	}
	return list, nil
}
func (r *emptyConsensusReaderTest) GetDayTimeIndex() core.TimeIndex {
	return r.ti
}

type timeIndex struct {
	GenesisTime time.Time
	Interval    time.Duration
}

func (ti timeIndex) Index2Time(index uint64) (time.Time, time.Time) {
	sTime := ti.GenesisTime.Add(ti.Interval * time.Duration(index))
	eTime := ti.GenesisTime.Add(ti.Interval * time.Duration(index+1))
	return sTime, eTime
}
func (ti timeIndex) Time2Index(t time.Time) uint64 {
	subSec := int64(t.Sub(ti.GenesisTime).Seconds())
	i := uint64(subSec) / uint64(ti.Interval.Seconds())
	return i
}
