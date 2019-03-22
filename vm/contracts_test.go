package vm

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/fork"
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/config"
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

/*func TestContractsRefundWithVmContext(t *testing.T) {
	// init chain
	chn := chain.NewChain(&config.Config{
		DataDir: filepath.Join(common.HomeDir(), "Library/GVite/devdata"),
		Chain:   &config.Chain{GenesisFile: filepath.Join(common.HomeDir(), "genesis.json")},
	})
	chn.Init()
	chn.Start()

	// init sendBlock and receive block
	snapshotBlock := chn.GetLatestSnapshotBlock()
	addr, _ := types.HexToAddress("vite_ab24ef68b84e642c0ddca06beec81c9acb1977bbd7da27a87a")
	contractAddr := types.AddressRegister
	contractBalance, _ := chn.GetAccountBalanceByTokenId(&contractAddr, &ledger.ViteTokenId)
	vm := NewVM()
	prevAccountBlock, err := chn.GetLatestAccountBlock(&contractAddr)
	if err != nil || prevAccountBlock == nil {
		t.Fatalf("prev contract account block is nil")
	}
	db, err := vm_context.NewVmContext(chn, &snapshotBlock.Hash, &prevAccountBlock.Hash, &contractAddr)
	if err != nil {
		t.Fatalf("new vm context failed %v", err)
	}
	nodeName := "s1"
	registerData, _ := abi.ABIRegister.PackMethod(abi.MethodNameRegister, types.SNAPSHOT_GID, nodeName, addr)
	timestamp := time.Now()
	sendHash := types.DataHash([]byte{1, 1})
	sendBlock := ledger.AccountBlock{
		BlockType:      ledger.BlockTypeSendCall,
		AccountAddress: addr,
		ToAddress:      contractAddr,
		Amount:         new(big.Int).Mul(big.NewInt(5e5), big.NewInt(1e18)),
		TokenId:        ledger.ViteTokenId,
		Fee:            big.NewInt(10),
		Data:           registerData,
		Timestamp:      &timestamp,
		Hash:           sendHash,
		Height:         1,
	}
	chn.InsertAccountBlocks([]*vm_context.VmAccountBlock{{&sendBlock, db}})
	receiveBlock := ledger.AccountBlock{
		BlockType:      ledger.BlockTypeReceive,
		Height:         prevAccountBlock.Height + 1,
		AccountAddress: contractAddr,
		Timestamp:      &timestamp,
		FromBlockHash:  sendHash,
	}
	refundAmount := new(big.Int).Add(sendBlock.Amount, sendBlock.Fee)

	// duplicate register, refund
	receiveBlockList, isRetry, err := vm.Run(db, &receiveBlock, &sendBlock)
	if len(receiveBlockList) != 2 || isRetry || err == nil ||
		receiveBlockList[0].AccountBlock.BlockType != ledger.BlockTypeReceive ||
		!bytes.Equal(receiveBlockList[0].AccountBlock.Data, append(receiveBlockList[0].AccountBlock.StateHash.Bytes(), byte(1))) ||
		receiveBlockList[0].AccountBlock.Quota != 0 ||
		len(receiveBlockList[0].AccountBlock.Data) != 33 ||
		receiveBlockList[0].AccountBlock.Data[32] != byte(1) ||
		!bytes.Equal(receiveBlockList[1].AccountBlock.Data, []byte{1}) ||
		receiveBlockList[1].AccountBlock.BlockType != ledger.BlockTypeSendCall ||
		receiveBlockList[1].AccountBlock.Height != receiveBlockList[0].AccountBlock.Height+1 ||
		receiveBlockList[1].AccountBlock.TokenId != ledger.ViteTokenId ||
		receiveBlockList[1].AccountBlock.Amount.Cmp(refundAmount) != 0 ||
		receiveBlockList[1].AccountBlock.Fee.Sign() != 0 ||
		receiveBlockList[1].AccountBlock.AccountAddress != contractAddr ||
		receiveBlockList[1].AccountBlock.ToAddress != addr ||
		receiveBlockList[1].AccountBlock.Quota != 0 ||
		receiveBlockList[0].VmContext.GetBalance(&contractAddr, &ledger.ViteTokenId).Cmp(new(big.Int).Add(contractBalance, refundAmount)) != 0 ||
		receiveBlockList[1].VmContext.GetBalance(&contractAddr, &ledger.ViteTokenId).Cmp(contractBalance) != 0 {
		t.Fatalf("refund error")
	}
}*/

func TestContractsRefund(t *testing.T) {
	// prepare db
	viteTotalSupply := new(big.Int).Mul(big.NewInt(1e9), big.NewInt(1e18))
	db, addr1, _, hash12, snapshot2, _ := prepareDb(viteTotalSupply)
	blockTime := time.Now()

	addr2 := types.AddressRegister
	nodeName := "s1"
	locHashRegister, _ := types.BytesToHash(abi.GetRegisterKey(nodeName, types.SNAPSHOT_GID))
	registrationDataOld := db.storageMap[addr2][string(locHashRegister.Bytes())]
	contractBalance := db.GetBalance(&addr2, &ledger.ViteTokenId)
	// register with an existed super node name, get refund
	balance1 := new(big.Int).Set(viteTotalSupply)
	addr6, _, _ := types.CreateAddress()
	db.accountBlockMap[addr6] = make(map[types.Hash]*ledger.AccountBlock)
	block13Data, err := abi.ABIRegister.PackMethod(abi.MethodNameRegister, types.SNAPSHOT_GID, nodeName, addr6)
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
		Hash:           hash13,
	}
	vm := NewVM()
	//vm.Debug = true
	db.addr = addr1
	sendRegisterBlockList, isRetry, err := vm.Run(db, block13, nil)
	balance1.Sub(balance1, block13.Amount)
	if len(sendRegisterBlockList) != 1 || isRetry || err != nil ||
		sendRegisterBlockList[0].AccountBlock.Quota != contracts.RegisterGas ||
		!bytes.Equal(sendRegisterBlockList[0].AccountBlock.Data, block13Data) ||
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
		Hash:           hash21,
	}
	vm = NewVM()
	//vm.Debug = true
	db.addr = addr2
	receiveRegisterBlockList, isRetry, err := vm.Run(db, block21, sendRegisterBlockList[0].AccountBlock)
	contractBalance.Add(contractBalance, block13.Amount)
	if len(receiveRegisterBlockList) != 2 || isRetry || err == nil ||
		db.balanceMap[addr1][ledger.ViteTokenId].Cmp(balance1) != 0 ||
		!bytes.Equal(db.storageMap[addr2][string(locHashRegister.Bytes())], registrationDataOld) ||
		receiveRegisterBlockList[0].AccountBlock.Quota != 0 ||
		len(receiveRegisterBlockList[0].AccountBlock.Data) != 33 ||
		receiveRegisterBlockList[0].AccountBlock.Data[32] != byte(1) ||
		receiveRegisterBlockList[1].AccountBlock.Height != 2 ||
		receiveRegisterBlockList[1].AccountBlock.Quota != 0 ||
		receiveRegisterBlockList[1].AccountBlock.TokenId != block13.TokenId ||
		receiveRegisterBlockList[1].AccountBlock.Amount.Cmp(block13.Amount) != 0 ||
		receiveRegisterBlockList[1].AccountBlock.BlockType != ledger.BlockTypeSendCall ||
		receiveRegisterBlockList[1].AccountBlock.AccountAddress != block13.ToAddress ||
		receiveRegisterBlockList[1].AccountBlock.ToAddress != block13.AccountAddress ||
		db.GetBalance(&addr2, &ledger.ViteTokenId).Cmp(contractBalance) != 0 ||
		!bytes.Equal(receiveRegisterBlockList[1].AccountBlock.Data, []byte{1}) {
		t.Fatalf("receive register transaction error")
	}
	db.accountBlockMap[addr2] = make(map[types.Hash]*ledger.AccountBlock)
	db.accountBlockMap[addr2][hash21] = receiveRegisterBlockList[0].AccountBlock
	hash22 := types.DataHash([]byte{2, 2})
	receiveRegisterBlockList[1].AccountBlock.Hash = hash22
	receiveRegisterBlockList[1].AccountBlock.PrevHash = hash21
	db.accountBlockMap[addr2][hash22] = receiveRegisterBlockList[1].AccountBlock

	hash14 := types.DataHash([]byte{1, 4})
	block14 := &ledger.AccountBlock{
		Height:         4,
		AccountAddress: addr1,
		BlockType:      ledger.BlockTypeReceive,
		FromBlockHash:  hash22,
		SnapshotHash:   snapshot2.Hash,
		Timestamp:      &blockTime,
		Hash:           hash14,
	}
	vm = NewVM()
	//vm.Debug = true
	db.addr = addr1
	receiveRegisterRefuncBlockList, isRetry, err := vm.Run(db, block14, receiveRegisterBlockList[1].AccountBlock)
	balance1.Add(balance1, block13.Amount)
	if len(receiveRegisterRefuncBlockList) != 1 || isRetry || err != nil ||
		db.balanceMap[addr1][ledger.ViteTokenId].Cmp(balance1) != 0 ||
		receiveRegisterRefuncBlockList[0].AccountBlock.Quota != 21000 {
		t.Fatalf("receive register refund transaction error")
	}
	db.accountBlockMap[addr1][hash14] = receiveRegisterRefuncBlockList[0].AccountBlock
}

func TestContractsRegister(t *testing.T) {
	// prepare db
	viteTotalSupply := new(big.Int).Mul(big.NewInt(1e9), big.NewInt(1e18))
	db, addr1, _, hash12, snapshot2, timestamp := prepareDb(viteTotalSupply)
	blockTime := time.Now()
	// register
	balance1 := new(big.Int).Set(viteTotalSupply)
	addr6, privateKey6, _ := types.CreateAddress()
	addr7, _, _ := types.CreateAddress()
	publicKey6 := ed25519.PublicKey(privateKey6.PubByte())
	db.accountBlockMap[addr6] = make(map[types.Hash]*ledger.AccountBlock)
	db.accountBlockMap[addr7] = make(map[types.Hash]*ledger.AccountBlock)
	addr2 := types.AddressRegister
	nodeName := "super1"
	block13Data, err := abi.ABIRegister.PackMethod(abi.MethodNameRegister, types.SNAPSHOT_GID, nodeName, addr7)
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
		Hash:           hash13,
	}
	vm := NewVM()
	//vm.Debug = true
	db.addr = addr1
	sendRegisterBlockList, isRetry, err := vm.Run(db, block13, nil)
	balance1.Sub(balance1, block13.Amount)
	if len(sendRegisterBlockList) != 1 || isRetry || err != nil ||
		sendRegisterBlockList[0].AccountBlock.Quota != contracts.RegisterGas ||
		!bytes.Equal(sendRegisterBlockList[0].AccountBlock.Data, block13Data) ||
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
		Hash:           hash21,
	}
	vm = NewVM()
	//vm.Debug = true
	locHashRegister, _ := types.BytesToHash(abi.GetRegisterKey(nodeName, types.SNAPSHOT_GID))
	hisAddrList := []types.Address{addr7}
	withdrawHeight := snapshot2.Height + 3600*24*90
	registrationData, _ := abi.ABIRegister.PackVariable(abi.VariableNameRegistration, nodeName, addr7, addr1, block13.Amount, withdrawHeight, uint64(0), uint64(0), hisAddrList)
	db.addr = addr2
	receiveRegisterBlockList, isRetry, err := vm.Run(db, block21, sendRegisterBlockList[0].AccountBlock)
	if len(receiveRegisterBlockList) != 1 || isRetry || err != nil ||
		db.balanceMap[addr1][ledger.ViteTokenId].Cmp(balance1) != 0 ||
		!bytes.Equal(db.storageMap[addr2][string(locHashRegister.Bytes())], registrationData) ||
		len(receiveRegisterBlockList[0].AccountBlock.Data) != 33 ||
		receiveRegisterBlockList[0].AccountBlock.Data[32] != byte(0) ||
		receiveRegisterBlockList[0].AccountBlock.Quota != 0 {
		t.Fatalf("receive register transaction error")
	}
	db.accountBlockMap[addr2] = make(map[types.Hash]*ledger.AccountBlock)
	db.accountBlockMap[addr2][hash21] = receiveRegisterBlockList[0].AccountBlock

	// update registration
	block14Data, err := abi.ABIRegister.PackMethod(abi.MethodNameUpdateRegistration, types.SNAPSHOT_GID, nodeName, addr6)
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
		Hash:           hash14,
	}
	vm = NewVM()
	//vm.Debug = true
	db.addr = addr1
	sendRegisterBlockList2, isRetry, err := vm.Run(db, block14, nil)
	if len(sendRegisterBlockList2) != 1 || isRetry || err != nil ||
		sendRegisterBlockList2[0].AccountBlock.Quota != contracts.UpdateRegistrationGas ||
		!bytes.Equal(sendRegisterBlockList2[0].AccountBlock.Data, block14Data) ||
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
		Hash:           hash22,
	}
	vm = NewVM()
	//vm.Debug = true
	hisAddrList = append(hisAddrList, addr6)
	registrationData, _ = abi.ABIRegister.PackVariable(abi.VariableNameRegistration, nodeName, addr6, addr1, block13.Amount, withdrawHeight, uint64(0), uint64(0), hisAddrList)
	db.addr = addr2
	receiveRegisterBlockList2, isRetry, err := vm.Run(db, block22, sendRegisterBlockList2[0].AccountBlock)
	if len(receiveRegisterBlockList2) != 1 || isRetry || err != nil ||
		db.balanceMap[addr1][ledger.ViteTokenId].Cmp(balance1) != 0 ||
		!bytes.Equal(db.storageMap[addr2][string(locHashRegister.Bytes())], registrationData) ||
		len(receiveRegisterBlockList2[0].AccountBlock.Data) != 33 ||
		receiveRegisterBlockList2[0].AccountBlock.Data[32] != byte(0) ||
		receiveRegisterBlockList2[0].AccountBlock.Quota != 0 {
		t.Fatalf("receive update registration transaction error")
	}
	db.accountBlockMap[addr2][hash22] = receiveRegisterBlockList2[0].AccountBlock

	// get contracts data
	db.addr = types.AddressRegister
	if registerList := abi.GetCandidateList(db, types.SNAPSHOT_GID, nil); len(registerList) != 3 || len(registerList[0].Name) == 0 {
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
	block15Data, _ := abi.ABIRegister.PackMethod(abi.MethodNameCancelRegister, types.SNAPSHOT_GID, nodeName)
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
		Hash:           hash15,
	}
	vm = NewVM()
	//vm.Debug = true
	db.addr = addr1
	sendCancelRegisterBlockList, isRetry, err := vm.Run(db, block15, nil)
	if len(sendCancelRegisterBlockList) != 1 || isRetry || err != nil ||
		sendCancelRegisterBlockList[0].AccountBlock.Quota != contracts.CancelRegisterGas ||
		!bytes.Equal(sendCancelRegisterBlockList[0].AccountBlock.Data, block15Data) ||
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
	//vm.Debug = true
	db.addr = addr2
	receiveCancelRegisterBlockList, isRetry, err := vm.Run(db, block23, sendCancelRegisterBlockList[0].AccountBlock)
	registrationData, _ = abi.ABIRegister.PackVariable(abi.VariableNameRegistration, nodeName, addr6, addr1, helper.Big0, uint64(0), uint64(0), snapshot5.Height, hisAddrList)
	if len(receiveCancelRegisterBlockList) != 2 || isRetry || err != nil ||
		db.balanceMap[addr2][ledger.ViteTokenId].Cmp(helper.Big0) != 0 ||
		db.balanceMap[addr1][ledger.ViteTokenId].Cmp(balance1) != 0 ||
		!bytes.Equal(db.storageMap[addr2][string(locHashRegister.Bytes())], registrationData) ||
		len(receiveCancelRegisterBlockList[0].AccountBlock.Data) != 33 ||
		receiveCancelRegisterBlockList[0].AccountBlock.Data[32] != byte(0) ||
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
	receiveCancelRegisterBlockList[1].AccountBlock.Hash = hash24
	receiveCancelRegisterBlockList[1].AccountBlock.PrevHash = hash23
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
		Hash:           hash16,
	}
	vm = NewVM()
	//vm.Debug = true
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

	/*// reward
	time6 := time.Unix(timestamp+4, 0)
	snapshot6 := &ledger.SnapshotBlock{Height: snapshot5.Height + 256, Timestamp: &time6, Hash: types.DataHash([]byte{10, byte(6)})}
	db.snapshotBlockList = append(db.snapshotBlockList, snapshot6)
	db.storageMap[abi.AddressPledge][string(types.DataHash(addr1.Bytes()).Bytes())], _ = abi.ABIPledge.PackVariable(abi.VariableNamePledgeBeneficial, new(big.Int).Mul(big.NewInt(1e6), big.NewInt(1e18)))
	db.storageMap[abi.AddressPledge][string(abi.GetPledgeBeneficialKey(addr7))], _ = abi.ABIPledge.PackVariable(abi.VariableNamePledgeBeneficial, new(big.Int).Mul(big.NewInt(1e6), big.NewInt(1e18)))
	block17Data, _ := abi.ABIRegister.PackMethod(abi.MethodNameReward, types.SNAPSHOT_GID, nodeName, addr7)
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
	reward := new(big.Int).Mul(big.NewInt(2), new(big.Int).Div(viteTotalSupply, big.NewInt(1051200000)))
	block17DataExpected, _ := abi.ABIRegister.PackMethod(abi.MethodNameReward, types.SNAPSHOT_GID, nodeName, addr7)
	if len(sendRewardBlockList) != 1 || isRetry || err != nil ||
		sendRewardBlockList[0].AccountBlock.Quota != contracts.RewardGas ||
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
	registrationData, _ = abi.ABIRegister.PackVariable(abi.VariableNameRegistration, nodeName, addr6, addr1, helper.Big0, uint64(0), snapshot6.Height-60*30, snapshot5.Height, hisAddrList)
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
	db.accountBlockMap[addr7][hash71] = receiveRewardRefundBlockList[0].AccountBlock*/
}

func TestContractsVote(t *testing.T) {
	// prepare db
	viteTotalSupply := new(big.Int).Mul(big.NewInt(2e6), big.NewInt(1e18))
	db, addr1, _, hash12, snapshot2, _ := prepareDb(viteTotalSupply)
	blockTime := time.Now()
	// vote
	addr3 := types.AddressVote
	nodeName := "s1"
	block13Data, _ := abi.ABIVote.PackMethod(abi.MethodNameVote, types.SNAPSHOT_GID, nodeName)
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
		Hash:           hash13,
	}
	vm := NewVM()
	//vm.Debug = true
	db.addr = addr1
	sendVoteBlockList, isRetry, err := vm.Run(db, block13, nil)
	if len(sendVoteBlockList) != 1 || isRetry || err != nil ||
		!bytes.Equal(sendVoteBlockList[0].AccountBlock.Data, block13Data) ||
		sendVoteBlockList[0].AccountBlock.Quota != contracts.VoteGas {
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
		Hash:           hash31,
	}
	vm = NewVM()
	//vm.Debug = true
	db.addr = addr3
	receiveVoteBlockList, isRetry, err := vm.Run(db, block31, sendVoteBlockList[0].AccountBlock)
	voteKey := abi.GetVoteKey(addr1, types.SNAPSHOT_GID)
	voteData, _ := abi.ABIVote.PackVariable(abi.VariableNameVoteStatus, nodeName)
	if len(receiveVoteBlockList) != 1 || isRetry || err != nil ||
		!bytes.Equal(db.storageMap[addr3][string(voteKey)], voteData) ||
		len(receiveVoteBlockList[0].AccountBlock.Data) != 33 ||
		receiveVoteBlockList[0].AccountBlock.Data[32] != byte(0) ||
		receiveVoteBlockList[0].AccountBlock.Quota != 0 {
		t.Fatalf("receive vote transaction error")
	}
	db.accountBlockMap[addr3] = make(map[types.Hash]*ledger.AccountBlock)
	db.accountBlockMap[addr3][hash31] = receiveVoteBlockList[0].AccountBlock

	addr4, _ := types.BytesToAddress(helper.HexToBytes("e5bf58cacfb74cf8c49a1d5e59d3919c9a4cb9ed"))
	db.accountBlockMap[addr4] = make(map[types.Hash]*ledger.AccountBlock)
	nodeName2 := "s2"
	block14Data, _ := abi.ABIVote.PackMethod(abi.MethodNameVote, types.SNAPSHOT_GID, nodeName2)
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
		Hash:           hash14,
	}
	vm = NewVM()
	//vm.Debug = true
	db.addr = addr1
	sendVoteBlockList2, isRetry, err := vm.Run(db, block14, nil)
	if len(sendVoteBlockList2) != 1 || isRetry || err != nil ||
		!bytes.Equal(sendVoteBlockList2[0].AccountBlock.Data, block14Data) ||
		sendVoteBlockList2[0].AccountBlock.Quota != contracts.VoteGas {
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
		Hash:           hash32,
	}
	vm = NewVM()
	//vm.Debug = true
	db.addr = addr3
	receiveVoteBlockList2, isRetry, err := vm.Run(db, block32, sendVoteBlockList2[0].AccountBlock)
	voteData, _ = abi.ABIVote.PackVariable(abi.VariableNameVoteStatus, nodeName2)
	if len(receiveVoteBlockList2) != 1 || isRetry || err != nil ||
		!bytes.Equal(db.storageMap[addr3][string(voteKey)], voteData) ||
		len(receiveVoteBlockList2[0].AccountBlock.Data) != 33 ||
		receiveVoteBlockList2[0].AccountBlock.Data[32] != byte(0) ||
		receiveVoteBlockList2[0].AccountBlock.Quota != 0 {
		t.Fatalf("receive vote transaction 2 error")
	}
	db.accountBlockMap[addr3][hash32] = receiveVoteBlockList2[0].AccountBlock

	// get contracts data
	db.addr = types.AddressVote
	if voteList := abi.GetVoteList(db, types.SNAPSHOT_GID, nil); len(voteList) != 1 || voteList[0].NodeName != nodeName2 {
		t.Fatalf("get vote list failed")
	}

	// cancel vote
	block15Data, _ := abi.ABIVote.PackMethod(abi.MethodNameCancelVote, types.SNAPSHOT_GID)
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
		Hash:           hash15,
	}
	vm = NewVM()
	//vm.Debug = true
	db.addr = addr1
	sendCancelVoteBlockList, isRetry, err := vm.Run(db, block15, nil)
	if len(sendCancelVoteBlockList) != 1 || isRetry || err != nil ||
		!bytes.Equal(sendCancelVoteBlockList[0].AccountBlock.Data, block15Data) ||
		sendCancelVoteBlockList[0].AccountBlock.Quota != contracts.CancelVoteGas {
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
		Hash:           hash33,
	}
	vm = NewVM()
	//vm.Debug = true
	db.addr = addr3
	receiveCancelVoteBlockList, isRetry, err := vm.Run(db, block33, sendCancelVoteBlockList[0].AccountBlock)
	if len(receiveCancelVoteBlockList) != 1 || isRetry || err != nil ||
		len(db.storageMap[addr3][string(voteKey)]) != 0 ||
		len(receiveCancelVoteBlockList[0].AccountBlock.Data) != 33 ||
		receiveCancelVoteBlockList[0].AccountBlock.Data[32] != byte(0) ||
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
		SnapshotHash:   snapshot2.Hash,
		Timestamp:      &blockTime,
		Hash:           hash13,
	}
	vm := NewVM()
	//vm.Debug = true
	db.addr = addr1
	sendPledgeBlockList, isRetry, err := vm.Run(db, block13, nil)
	balance1.Sub(balance1, pledgeAmount)
	if len(sendPledgeBlockList) != 1 || isRetry || err != nil ||
		db.balanceMap[addr1][ledger.ViteTokenId].Cmp(balance1) != 0 ||
		!bytes.Equal(sendPledgeBlockList[0].AccountBlock.Data, block13Data) ||
		sendPledgeBlockList[0].AccountBlock.Quota != contracts.PledgeGas {
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
		Hash:           hash51,
	}
	vm = NewVM()
	//vm.Debug = true
	db.addr = addr5
	receivePledgeBlockList, isRetry, err := vm.Run(db, block51, sendPledgeBlockList[0].AccountBlock)
	beneficialKey := abi.GetPledgeBeneficialKey(addr4)
	pledgeKey := abi.GetPledgeKey(addr1, beneficialKey)
	withdrawHeight := snapshot2.Height + 3600*24*3
	if len(receivePledgeBlockList) != 1 || isRetry || err != nil ||
		!bytes.Equal(db.storageMap[addr5][string(pledgeKey)], helper.JoinBytes(helper.LeftPadBytes(pledgeAmount.Bytes(), helper.WordSize), helper.LeftPadBytes(new(big.Int).SetUint64(withdrawHeight).Bytes(), helper.WordSize))) ||
		!bytes.Equal(db.storageMap[addr5][string(beneficialKey)], helper.LeftPadBytes(pledgeAmount.Bytes(), helper.WordSize)) ||
		db.balanceMap[addr5][ledger.ViteTokenId].Cmp(pledgeAmount) != 0 ||
		len(receivePledgeBlockList[0].AccountBlock.Data) != 33 ||
		receivePledgeBlockList[0].AccountBlock.Data[32] != byte(0) ||
		receivePledgeBlockList[0].AccountBlock.Quota != 0 {
		t.Fatalf("receive pledge transaction error")
	}
	db.accountBlockMap[addr5] = make(map[types.Hash]*ledger.AccountBlock)
	db.accountBlockMap[addr5][hash51] = receivePledgeBlockList[0].AccountBlock

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
		SnapshotHash:   snapshot2.Hash,
		Timestamp:      &blockTime,
		Hash:           hash14,
	}
	vm = NewVM()
	//vm.Debug = true
	db.addr = addr1
	sendPledgeBlockList2, isRetry, err := vm.Run(db, block14, nil)
	balance1.Sub(balance1, pledgeAmount)
	if len(sendPledgeBlockList2) != 1 || isRetry || err != nil ||
		!bytes.Equal(sendPledgeBlockList2[0].AccountBlock.Data, block14Data) ||
		db.balanceMap[addr1][ledger.ViteTokenId].Cmp(balance1) != 0 ||
		sendPledgeBlockList2[0].AccountBlock.Quota != contracts.PledgeGas {
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
		Hash:           hash52,
	}
	vm = NewVM()
	//vm.Debug = true
	db.addr = addr5
	receivePledgeBlockList2, isRetry, err := vm.Run(db, block52, sendPledgeBlockList2[0].AccountBlock)
	newPledgeAmount := new(big.Int).Add(pledgeAmount, pledgeAmount)
	if len(receivePledgeBlockList2) != 1 || isRetry || err != nil ||
		!bytes.Equal(db.storageMap[addr5][string(pledgeKey)], helper.JoinBytes(helper.LeftPadBytes(newPledgeAmount.Bytes(), helper.WordSize), helper.LeftPadBytes(new(big.Int).SetUint64(withdrawHeight).Bytes(), helper.WordSize))) ||
		!bytes.Equal(db.storageMap[addr5][string(beneficialKey)], helper.LeftPadBytes(newPledgeAmount.Bytes(), helper.WordSize)) ||
		db.balanceMap[addr5][ledger.ViteTokenId].Cmp(newPledgeAmount) != 0 ||
		len(receivePledgeBlockList2[0].AccountBlock.Data) != 33 ||
		receivePledgeBlockList2[0].AccountBlock.Data[32] != byte(0) ||
		receivePledgeBlockList2[0].AccountBlock.Quota != 0 {
		t.Fatalf("receive pledge transaction 2 error")
	}
	db.accountBlockMap[addr5][hash52] = receivePledgeBlockList2[0].AccountBlock

	// get contracts data
	db.addr = types.AddressPledge
	if pledgeAmount := abi.GetPledgeBeneficialAmount(db, addr4); pledgeAmount.Cmp(newPledgeAmount) != 0 {
		t.Fatalf("get pledge beneficial amount failed")
	}

	if pledgeInfoList, _ := abi.GetPledgeInfoList(db, addr1); len(pledgeInfoList) != 1 || pledgeInfoList[0].BeneficialAddr != addr4 {
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
		SnapshotHash:   currentSnapshot.Hash,
		Timestamp:      &blockTime,
		Hash:           hash15,
	}
	vm = NewVM()
	//vm.Debug = true
	db.addr = addr1
	sendCancelPledgeBlockList, isRetry, err := vm.Run(db, block15, nil)
	if len(sendCancelPledgeBlockList) != 1 || isRetry || err != nil ||
		!bytes.Equal(sendCancelPledgeBlockList[0].AccountBlock.Data, block15Data) ||
		sendCancelPledgeBlockList[0].AccountBlock.Quota != contracts.CancelPledgeGas {
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
		Hash:           hash53,
	}
	vm = NewVM()
	//vm.Debug = true
	db.addr = addr5
	receiveCancelPledgeBlockList, isRetry, err := vm.Run(db, block53, sendCancelPledgeBlockList[0].AccountBlock)
	if len(receiveCancelPledgeBlockList) != 2 || isRetry || err != nil ||
		receiveCancelPledgeBlockList[1].AccountBlock.Height != 4 ||
		!bytes.Equal(db.storageMap[addr5][string(pledgeKey)], helper.JoinBytes(helper.LeftPadBytes(pledgeAmount.Bytes(), helper.WordSize), helper.LeftPadBytes(new(big.Int).SetUint64(withdrawHeight).Bytes(), helper.WordSize))) ||
		!bytes.Equal(db.storageMap[addr5][string(beneficialKey)], helper.LeftPadBytes(pledgeAmount.Bytes(), helper.WordSize)) ||
		db.balanceMap[addr5][ledger.ViteTokenId].Cmp(pledgeAmount) != 0 ||
		len(receiveCancelPledgeBlockList[0].AccountBlock.Data) != 33 ||
		receiveCancelPledgeBlockList[0].AccountBlock.Data[32] != byte(0) ||
		receiveCancelPledgeBlockList[0].AccountBlock.Quota != 0 ||
		receiveCancelPledgeBlockList[1].AccountBlock.Quota != 0 {
		t.Fatalf("receive cancel pledge transaction error")
	}
	db.accountBlockMap[addr5][hash53] = receiveCancelPledgeBlockList[0].AccountBlock
	hash54 := types.DataHash([]byte{5, 4})
	receiveCancelPledgeBlockList[1].AccountBlock.Hash = hash54
	receiveCancelPledgeBlockList[1].AccountBlock.PrevHash = hash53
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
		Hash:           hash16,
	}
	vm = NewVM()
	//vm.Debug = true
	db.addr = addr1
	receiveCancelPledgeRefundBlockList, isRetry, err := vm.Run(db, block16, receiveCancelPledgeBlockList[1].AccountBlock)
	balance1.Add(balance1, pledgeAmount)
	if len(receiveCancelPledgeRefundBlockList) != 1 || isRetry || err != nil ||
		db.balanceMap[addr1][ledger.ViteTokenId].Cmp(balance1) != 0 ||
		receiveCancelPledgeRefundBlockList[0].AccountBlock.Quota != 21000 {
		t.Fatalf("receive cancel pledge refund transaction error")
	}
	db.accountBlockMap[addr1][hash16] = receiveCancelPledgeRefundBlockList[0].AccountBlock

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
		SnapshotHash:   currentSnapshot.Hash,
		Timestamp:      &blockTime,
		Hash:           hash17,
	}
	vm = NewVM()
	//vm.Debug = true
	db.addr = addr1
	sendCancelPledgeBlockList2, isRetry, err := vm.Run(db, block17, nil)
	if len(sendCancelPledgeBlockList2) != 1 || isRetry || err != nil ||
		!bytes.Equal(sendCancelPledgeBlockList2[0].AccountBlock.Data, block17Data) ||
		sendCancelPledgeBlockList2[0].AccountBlock.Quota != contracts.CancelPledgeGas {
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
		Hash:           hash55,
	}
	vm = NewVM()
	//vm.Debug = true
	db.addr = addr5
	receiveCancelPledgeBlockList2, isRetry, err := vm.Run(db, block55, sendCancelPledgeBlockList2[0].AccountBlock)
	if len(receiveCancelPledgeBlockList2) != 2 || isRetry || err != nil ||
		receiveCancelPledgeBlockList2[1].AccountBlock.Height != 6 ||
		len(db.storageMap[addr5][string(pledgeKey)]) != 0 ||
		len(db.storageMap[addr5][string(beneficialKey)]) != 0 ||
		db.balanceMap[addr5][ledger.ViteTokenId].Cmp(helper.Big0) != 0 ||
		len(receiveCancelPledgeBlockList2[0].AccountBlock.Data) != 33 ||
		receiveCancelPledgeBlockList2[0].AccountBlock.Data[32] != byte(0) ||
		receiveCancelPledgeBlockList2[0].AccountBlock.Quota != 0 ||
		receiveCancelPledgeBlockList2[1].AccountBlock.Quota != 0 {
		t.Fatalf("receive cancel pledge transaction 2 error")
	}
	db.accountBlockMap[addr5][hash55] = receiveCancelPledgeBlockList2[0].AccountBlock
	hash56 := types.DataHash([]byte{5, 6})
	receiveCancelPledgeBlockList2[1].AccountBlock.Hash = hash56
	receiveCancelPledgeBlockList2[1].AccountBlock.PrevHash = hash55
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
		Hash:           hash18,
	}
	vm = NewVM()
	//vm.Debug = true
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

/*func TestContractsConsensusGroup(t *testing.T) {
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
	if len(sendCreateConsensusGroupBlockList) != 1 || isRetry || err != nil ||
		sendCreateConsensusGroupBlockList[0].AccountBlock.Quota != contracts.CreateConsensusGroupGas ||
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
	if len(sendCancelConsensusGroupBlockList) != 1 || isRetry || err != nil ||
		sendCancelConsensusGroupBlockList[0].AccountBlock.Quota != contracts.CancelConsensusGroupGas ||
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
	balance1.Sub(balance1, pledgeAmount)
	if len(sendRecreateConsensusGroupBlockList) != 1 || isRetry || err != nil ||
		sendRecreateConsensusGroupBlockList[0].AccountBlock.Quota != contracts.ReCreateConsensusGroupGas ||
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
}*/

func TestContractsMintage(t *testing.T) {
	// prepare db
	viteTotalSupply := new(big.Int).Mul(big.NewInt(2e6), big.NewInt(1e18))
	db, addr1, _, hash12, snapshot2, _ := prepareDb(viteTotalSupply)
	blockTime := time.Now()
	// mintage
	balance1 := new(big.Int).Set(viteTotalSupply)
	addr2 := types.AddressMintage
	tokenName := "test token"
	tokenSymbol := "t"
	totalSupply := big.NewInt(1e10)
	decimals := uint8(3)
	newtokenId := abi.NewTokenId(addr1, 3, hash12, snapshot2.Hash)
	block13Data, err := abi.ABIMintage.PackMethod(abi.MethodNameMintage, newtokenId, tokenName, tokenSymbol, totalSupply, decimals)
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
		Hash:           hash13,
	}
	vm := NewVM()
	//vm.Debug = true
	db.addr = addr1
	sendMintageBlockList, isRetry, err := vm.Run(db, block13, nil)
	balance1.Sub(balance1, new(big.Int).Mul(big.NewInt(1e3), util.AttovPerVite))
	if len(sendMintageBlockList) != 1 || isRetry || err != nil ||
		db.balanceMap[addr1][ledger.ViteTokenId].Cmp(balance1) != 0 ||
		sendMintageBlockList[0].AccountBlock.Fee.Cmp(new(big.Int).Mul(big.NewInt(1e3), util.AttovPerVite)) != 0 ||
		!bytes.Equal(sendMintageBlockList[0].AccountBlock.Data, block13Data) ||
		sendMintageBlockList[0].AccountBlock.Amount.Cmp(big.NewInt(0)) != 0 ||
		sendMintageBlockList[0].AccountBlock.Quota != contracts.MintageGas {
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
		Hash:           hash21,
	}
	vm = NewVM()
	//vm.Debug = true
	db.addr = addr2
	receiveMintageBlockList, isRetry, err := vm.Run(db, block21, sendMintageBlockList[0].AccountBlock)
	tokenId, _ := types.BytesToTokenTypeId(sendMintageBlockList[0].AccountBlock.Data[26:36])
	key, _ := types.BytesToHash(sendMintageBlockList[0].AccountBlock.Data[4:36])
	tokenInfoData, _ := abi.ABIMintage.PackVariable(abi.VariableNameMintage, tokenName, tokenSymbol, totalSupply, decimals, addr1, big.NewInt(0), uint64(0))
	if len(receiveMintageBlockList) != 2 || isRetry || err != nil ||
		!bytes.Equal(db.storageMap[addr2][string(key.Bytes())], tokenInfoData) ||
		db.balanceMap[addr2][ledger.ViteTokenId].Cmp(helper.Big0) != 0 ||
		len(receiveMintageBlockList[0].AccountBlock.Data) != 33 ||
		receiveMintageBlockList[0].AccountBlock.Data[32] != byte(0) ||
		receiveMintageBlockList[0].AccountBlock.Quota != 0 {
		t.Fatalf("receive mintage transaction error")
	}
	db.accountBlockMap[addr2] = make(map[types.Hash]*ledger.AccountBlock)
	db.accountBlockMap[addr2][hash21] = receiveMintageBlockList[0].AccountBlock
	hash22 := types.DataHash([]byte{2, 2})
	receiveMintageBlockList[1].AccountBlock.Hash = hash22
	receiveMintageBlockList[1].AccountBlock.PrevHash = hash21
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
		Hash:           hash14,
	}
	vm = NewVM()
	//vm.Debug = true
	db.addr = addr1
	receiveMintageRewardBlockList, isRetry, err := vm.Run(db, block14, receiveMintageBlockList[1].AccountBlock)
	if len(receiveMintageRewardBlockList) != 1 || isRetry || err != nil ||
		db.balanceMap[addr1][tokenId].Cmp(totalSupply) != 0 ||
		receiveMintageRewardBlockList[0].AccountBlock.Quota != 21000 {
		t.Fatalf("receive mintage reward transaction error")
	}
	db.accountBlockMap[addr1][hash14] = receiveMintageRewardBlockList[0].AccountBlock

	// get contracts data
	db.addr = types.AddressMintage
	if tokenInfo := abi.GetTokenById(db, tokenId); tokenInfo == nil || tokenInfo.TokenName != tokenName {
		t.Fatalf("get token by id failed")
	}
	if tokenMap := abi.GetTokenMap(db); len(tokenMap) != 2 || tokenMap[tokenId].TokenName != tokenName {
		t.Fatalf("get token map failed")
	}
}

func TestContractsMintageV2(t *testing.T) {
	fork.SetForkPoints(&config.ForkPoints{Smart: &config.ForkPoint{Height: 2}, Mint: &config.ForkPoint{Height: 2}})
	defer initFork()

	// prepare db
	viteTotalSupply := new(big.Int).Mul(big.NewInt(1e9), big.NewInt(1e18))
	db, addr1, _, hash12, snapshot2, _ := prepareDb(viteTotalSupply)
	blockTime := time.Now()
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
	tokenId := abi.NewTokenId(addr1, 3, hash12, snapshot2.Hash)
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
		SnapshotHash:   snapshot2.Hash,
		Timestamp:      &blockTime,
		Hash:           hash13,
	}
	vm := NewVM()
	db.addr = addr1
	sendMintageBlockList, isRetry, err := vm.Run(db, block13, nil)
	if len(sendMintageBlockList) != 1 || isRetry || err != nil ||
		db.balanceMap[addr1][ledger.ViteTokenId].Cmp(balance1) != 0 ||
		!bytes.Equal(sendMintageBlockList[0].AccountBlock.Data, block13Data) ||
		sendMintageBlockList[0].AccountBlock.Amount.Cmp(pledgeAmount) != 0 ||
		sendMintageBlockList[0].AccountBlock.Fee.Cmp(fee) != 0 ||
		sendMintageBlockList[0].AccountBlock.Quota != contracts.MintGas {
		t.Fatalf("send mint transaction error")
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
		Hash:           hash21,
	}
	vm = NewVM()
	db.addr = addr2
	receiveMintageBlockList, isRetry, err := vm.Run(db, block21, sendMintageBlockList[0].AccountBlock)
	key := abi.GetMintageKey(tokenId)
	tokenInfoData, _ := abi.ABIMintage.PackVariable(abi.VariableNameTokenInfo, tokenName, tokenSymbol, totalSupply, decimals, addr1, pledgeAmount, withdrawHeight, addr1, isReIssuable, maxSupply, ownerBurnOnly)
	if len(receiveMintageBlockList) != 2 || isRetry || err != nil ||
		!bytes.Equal(db.storageMap[addr2][string(key)], tokenInfoData) ||
		db.balanceMap[addr2][ledger.ViteTokenId].Cmp(pledgeAmount) != 0 ||
		len(receiveMintageBlockList[0].AccountBlock.Data) != 33 ||
		receiveMintageBlockList[0].AccountBlock.Data[32] != byte(0) ||
		receiveMintageBlockList[0].AccountBlock.Quota != 0 ||
		receiveMintageBlockList[1].AccountBlock.Amount.Cmp(totalSupply) != 0 ||
		receiveMintageBlockList[1].AccountBlock.ToAddress != addr1 ||
		receiveMintageBlockList[1].AccountBlock.BlockType != ledger.BlockTypeSendReward ||
		receiveMintageBlockList[1].AccountBlock.Quota != 0 ||
		receiveMintageBlockList[1].AccountBlock.TokenId != tokenId ||
		receiveMintageBlockList[1].AccountBlock.Height != receiveMintageBlockList[0].AccountBlock.Height+1 ||
		len(db.logList) != 1 ||
		db.logList[0].Topics[0] != abi.ABIMintage.Events[abi.EventNameMint].Id() ||
		!bytes.Equal(db.logList[0].Topics[1].Bytes(), helper.LeftPadBytes(tokenId.Bytes(), 32)) {
		t.Fatalf("receive mint transaction error")
	}
	db.accountBlockMap[addr2] = make(map[types.Hash]*ledger.AccountBlock)
	db.accountBlockMap[addr2][hash21] = receiveMintageBlockList[0].AccountBlock
	hash22 := types.DataHash([]byte{2, 2})
	receiveMintageBlockList[1].AccountBlock.Hash = hash22
	receiveMintageBlockList[1].AccountBlock.PrevHash = hash21
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
		Hash:           hash14,
	}
	vm = NewVM()
	db.addr = addr1
	tokenBalance := new(big.Int).Set(totalSupply)
	receiveMintageRewardBlockList, isRetry, err := vm.Run(db, block14, receiveMintageBlockList[1].AccountBlock)
	if len(receiveMintageRewardBlockList) != 1 || isRetry || err != nil ||
		db.balanceMap[addr1][tokenId].Cmp(tokenBalance) != 0 ||
		receiveMintageRewardBlockList[0].AccountBlock.Quota != 21000 {
		t.Fatalf("receive mintage reward transaction error")
	}
	db.accountBlockMap[addr1][hash14] = receiveMintageRewardBlockList[0].AccountBlock

	// get contracts data
	db.addr = types.AddressMintage
	if tokenInfo := abi.GetTokenById(db, tokenId); tokenInfo == nil || tokenInfo.TokenName != tokenName {
		t.Fatalf("get token by id failed")
	}
	if tokenMap := abi.GetTokenMap(db); len(tokenMap) != 2 || tokenMap[tokenId].TokenName != tokenName {
		t.Fatalf("get token map failed")
	}

	if tokenMap := abi.GetTokenMapByOwner(db, addr1); len(tokenMap) != 1 {
		t.Fatalf("get token map by owner failed")
	}

	// issue
	addr3, _, _ := types.CreateAddress()
	db.storageMap[types.AddressPledge][string(abi.GetPledgeBeneficialKey(addr3))], _ = abi.ABIPledge.PackVariable(abi.VariableNamePledgeBeneficial, new(big.Int).Mul(big.NewInt(1e9), big.NewInt(1e18)))
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
		SnapshotHash:   snapshot2.Hash,
		Timestamp:      &blockTime,
		Hash:           hash15,
	}
	vm = NewVM()
	db.addr = addr1
	sendIssueBlockList, isRetry, err := vm.Run(db, block15, nil)
	if len(sendIssueBlockList) != 1 || isRetry || err != nil ||
		db.balanceMap[addr1][ledger.ViteTokenId].Cmp(balance1) != 0 ||
		!bytes.Equal(sendIssueBlockList[0].AccountBlock.Data, block15Data) ||
		sendIssueBlockList[0].AccountBlock.Amount.Cmp(big.NewInt(0)) != 0 ||
		sendIssueBlockList[0].AccountBlock.Quota != contracts.IssueGas {
		t.Fatalf("send issue transaction error")
	}
	db.accountBlockMap[addr1][hash15] = sendIssueBlockList[0].AccountBlock

	hash23 := types.DataHash([]byte{2, 3})
	block23 := &ledger.AccountBlock{
		Height:         3,
		AccountAddress: addr2,
		BlockType:      ledger.BlockTypeReceive,
		FromBlockHash:  hash15,
		SnapshotHash:   snapshot2.Hash,
		Timestamp:      &blockTime,
		Hash:           hash21,
		PrevHash:       hash22,
	}
	vm = NewVM()
	db.addr = addr2
	receiveIssueBlockList, isRetry, err := vm.Run(db, block23, sendIssueBlockList[0].AccountBlock)
	totalSupply = totalSupply.Add(totalSupply, reIssueAmount)
	tokenInfoData, _ = abi.ABIMintage.PackVariable(abi.VariableNameTokenInfo, tokenName, tokenSymbol, totalSupply, decimals, addr1, pledgeAmount, withdrawHeight, addr1, isReIssuable, maxSupply, ownerBurnOnly)
	if len(receiveIssueBlockList) != 2 || isRetry || err != nil ||
		!bytes.Equal(db.storageMap[addr2][string(key)], tokenInfoData) ||
		db.balanceMap[addr2][ledger.ViteTokenId].Cmp(pledgeAmount) != 0 ||
		len(receiveIssueBlockList[0].AccountBlock.Data) != 33 ||
		receiveIssueBlockList[0].AccountBlock.Data[32] != byte(0) ||
		receiveIssueBlockList[0].AccountBlock.Quota != 0 ||
		receiveIssueBlockList[1].AccountBlock.Amount.Cmp(reIssueAmount) != 0 ||
		receiveIssueBlockList[1].AccountBlock.ToAddress != addr3 ||
		receiveIssueBlockList[1].AccountBlock.BlockType != ledger.BlockTypeSendReward ||
		receiveIssueBlockList[1].AccountBlock.Quota != 0 ||
		receiveIssueBlockList[1].AccountBlock.TokenId != tokenId ||
		receiveIssueBlockList[1].AccountBlock.Height != receiveIssueBlockList[0].AccountBlock.Height+1 ||
		len(db.logList) != 2 ||
		db.logList[1].Topics[0] != abi.ABIMintage.Events[abi.EventNameIssue].Id() ||
		!bytes.Equal(db.logList[1].Topics[1].Bytes(), helper.LeftPadBytes(tokenId.Bytes(), 32)) {
		t.Fatalf("receive issue transaction error")
	}
	db.accountBlockMap[addr2] = make(map[types.Hash]*ledger.AccountBlock)
	db.accountBlockMap[addr2][hash23] = receiveIssueBlockList[0].AccountBlock
	hash24 := types.DataHash([]byte{2, 4})
	receiveIssueBlockList[1].AccountBlock.Hash = hash24
	receiveIssueBlockList[1].AccountBlock.PrevHash = hash23
	db.accountBlockMap[addr2][hash24] = receiveIssueBlockList[1].AccountBlock

	hash31 := types.DataHash([]byte{3, 1})
	block31 := &ledger.AccountBlock{
		Height:         1,
		AccountAddress: addr3,
		BlockType:      ledger.BlockTypeReceive,
		FromBlockHash:  hash24,
		SnapshotHash:   snapshot2.Hash,
		Timestamp:      &blockTime,
		Hash:           hash31,
	}
	vm = NewVM()
	db.addr = addr3
	receiveIssueRewardBlockList, isRetry, err := vm.Run(db, block31, receiveIssueBlockList[1].AccountBlock)
	if len(receiveIssueRewardBlockList) != 1 || isRetry || err != nil ||
		db.balanceMap[addr3][tokenId].Cmp(reIssueAmount) != 0 ||
		receiveIssueRewardBlockList[0].AccountBlock.Quota != 21000 {
		t.Fatalf("receive issue reward transaction error")
	}
	db.accountBlockMap[addr3] = make(map[types.Hash]*ledger.AccountBlock)
	db.accountBlockMap[addr3][hash31] = receiveIssueRewardBlockList[0].AccountBlock

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
		SnapshotHash:   snapshot2.Hash,
		Timestamp:      &blockTime,
		Hash:           hash16,
	}
	vm = NewVM()
	db.addr = addr1
	sendBurnBlockList, isRetry, err := vm.Run(db, block16, nil)
	totalSupply = totalSupply.Sub(totalSupply, burnAmount)
	tokenBalance = tokenBalance.Sub(tokenBalance, burnAmount)
	if len(sendBurnBlockList) != 1 || isRetry || err != nil ||
		db.balanceMap[addr1][ledger.ViteTokenId].Cmp(balance1) != 0 ||
		db.balanceMap[addr1][tokenId].Cmp(tokenBalance) != 0 ||
		!bytes.Equal(sendBurnBlockList[0].AccountBlock.Data, block16Data) ||
		sendBurnBlockList[0].AccountBlock.Amount.Cmp(burnAmount) != 0 ||
		sendBurnBlockList[0].AccountBlock.Quota != contracts.BurnGas {
		t.Fatalf("send burn transaction error")
	}
	db.accountBlockMap[addr1][hash16] = sendBurnBlockList[0].AccountBlock

	hash25 := types.DataHash([]byte{2, 5})
	block25 := &ledger.AccountBlock{
		Height:         5,
		AccountAddress: addr2,
		BlockType:      ledger.BlockTypeReceive,
		FromBlockHash:  hash16,
		SnapshotHash:   snapshot2.Hash,
		Timestamp:      &blockTime,
		Hash:           hash25,
		PrevHash:       hash24,
	}
	vm = NewVM()
	db.addr = addr2
	receiveBurnBlockList, isRetry, err := vm.Run(db, block25, sendBurnBlockList[0].AccountBlock)
	tokenInfoData, _ = abi.ABIMintage.PackVariable(abi.VariableNameTokenInfo, tokenName, tokenSymbol, totalSupply, decimals, addr1, pledgeAmount, withdrawHeight, addr1, isReIssuable, maxSupply, ownerBurnOnly)
	if len(receiveBurnBlockList) != 1 || isRetry || err != nil ||
		!bytes.Equal(db.storageMap[addr2][string(key)], tokenInfoData) ||
		db.balanceMap[addr2][ledger.ViteTokenId].Cmp(pledgeAmount) != 0 ||
		db.balanceMap[addr2][tokenId].Cmp(helper.Big0) != 0 ||
		len(receiveBurnBlockList[0].AccountBlock.Data) != 33 ||
		receiveBurnBlockList[0].AccountBlock.Data[32] != byte(0) ||
		receiveBurnBlockList[0].AccountBlock.Quota != 0 ||
		len(db.logList) != 3 ||
		db.logList[2].Topics[0] != abi.ABIMintage.Events[abi.EventNameBurn].Id() ||
		!bytes.Equal(db.logList[2].Topics[1].Bytes(), helper.LeftPadBytes(tokenId.Bytes(), 32)) ||
		!bytes.Equal(db.logList[2].Data, append(helper.LeftPadBytes(addr1.Bytes(), 32), helper.LeftPadBytes(burnAmount.Bytes(), 32)...)) {
		t.Fatalf("receive burn transaction error")
	}
	db.accountBlockMap[addr2] = make(map[types.Hash]*ledger.AccountBlock)
	db.accountBlockMap[addr2][hash25] = receiveBurnBlockList[0].AccountBlock

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
		SnapshotHash:   snapshot2.Hash,
		Timestamp:      &blockTime,
		Hash:           hash17,
	}
	vm = NewVM()
	db.addr = addr1
	sendTransferOwnerBlockList, isRetry, err := vm.Run(db, block17, nil)
	if len(sendTransferOwnerBlockList) != 1 || isRetry || err != nil ||
		db.balanceMap[addr1][ledger.ViteTokenId].Cmp(balance1) != 0 ||
		db.balanceMap[addr1][tokenId].Cmp(tokenBalance) != 0 ||
		!bytes.Equal(sendTransferOwnerBlockList[0].AccountBlock.Data, block17Data) ||
		sendTransferOwnerBlockList[0].AccountBlock.Amount.Cmp(helper.Big0) != 0 ||
		sendTransferOwnerBlockList[0].AccountBlock.Quota != contracts.TransferOwnerGas {
		t.Fatalf("send transfer owner transaction error")
	}
	db.accountBlockMap[addr1][hash17] = sendTransferOwnerBlockList[0].AccountBlock

	hash26 := types.DataHash([]byte{2, 6})
	block26 := &ledger.AccountBlock{
		Height:         6,
		AccountAddress: addr2,
		BlockType:      ledger.BlockTypeReceive,
		FromBlockHash:  hash17,
		SnapshotHash:   snapshot2.Hash,
		Timestamp:      &blockTime,
		Hash:           hash26,
		PrevHash:       hash25,
	}
	vm = NewVM()
	db.addr = addr2
	receiveTransferOwnerBlockList, isRetry, err := vm.Run(db, block26, sendTransferOwnerBlockList[0].AccountBlock)
	tokenInfoData, _ = abi.ABIMintage.PackVariable(abi.VariableNameTokenInfo, tokenName, tokenSymbol, totalSupply, decimals, addr3, pledgeAmount, withdrawHeight, addr1, isReIssuable, maxSupply, ownerBurnOnly)
	if len(receiveTransferOwnerBlockList) != 1 || isRetry || err != nil ||
		!bytes.Equal(db.storageMap[addr2][string(key)], tokenInfoData) ||
		db.balanceMap[addr2][ledger.ViteTokenId].Cmp(pledgeAmount) != 0 ||
		db.balanceMap[addr2][tokenId].Cmp(helper.Big0) != 0 ||
		len(receiveTransferOwnerBlockList[0].AccountBlock.Data) != 33 ||
		receiveTransferOwnerBlockList[0].AccountBlock.Data[32] != byte(0) ||
		receiveTransferOwnerBlockList[0].AccountBlock.Quota != 0 ||
		len(db.logList) != 4 ||
		db.logList[3].Topics[0] != abi.ABIMintage.Events[abi.EventNameTransferOwner].Id() ||
		!bytes.Equal(db.logList[3].Topics[1].Bytes(), helper.LeftPadBytes(tokenId.Bytes(), 32)) ||
		!bytes.Equal(db.logList[3].Data, helper.LeftPadBytes(addr3.Bytes(), 32)) {
		t.Fatalf("receive transfer owner transaction error")
	}
	db.accountBlockMap[addr2] = make(map[types.Hash]*ledger.AccountBlock)
	db.accountBlockMap[addr2][hash26] = receiveTransferOwnerBlockList[0].AccountBlock

	if tokenMap := abi.GetTokenMapByOwner(db, addr1); len(tokenMap) != 0 {
		t.Fatalf("get token map by owner failed")
	}
	if tokenMap := abi.GetTokenMapByOwner(db, addr3); len(tokenMap) != 1 {
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
		SnapshotHash:   snapshot2.Hash,
		Timestamp:      &blockTime,
		Hash:           hash32,
		PrevHash:       hash31,
	}
	vm = NewVM()
	db.addr = addr3
	sendChangeTokenTypeBlockList, isRetry, err := vm.Run(db, block32, nil)
	if len(sendChangeTokenTypeBlockList) != 1 || isRetry || err != nil ||
		!bytes.Equal(sendChangeTokenTypeBlockList[0].AccountBlock.Data, block32Data) ||
		sendChangeTokenTypeBlockList[0].AccountBlock.Amount.Cmp(helper.Big0) != 0 ||
		sendChangeTokenTypeBlockList[0].AccountBlock.Quota != contracts.ChangeTokenTypeGas {
		t.Fatalf("send change token type transaction error")
	}
	db.accountBlockMap[addr3][hash32] = sendChangeTokenTypeBlockList[0].AccountBlock

	hash27 := types.DataHash([]byte{2, 7})
	block27 := &ledger.AccountBlock{
		Height:         7,
		AccountAddress: addr2,
		BlockType:      ledger.BlockTypeReceive,
		FromBlockHash:  hash31,
		SnapshotHash:   snapshot2.Hash,
		Timestamp:      &blockTime,
		Hash:           hash27,
		PrevHash:       hash26,
	}
	vm = NewVM()
	db.addr = addr2
	receiveChangeTokenTypeBlockList, isRetry, err := vm.Run(db, block27, sendChangeTokenTypeBlockList[0].AccountBlock)
	tokenInfoData, _ = abi.ABIMintage.PackVariable(abi.VariableNameTokenInfo, tokenName, tokenSymbol, totalSupply, decimals, addr3, pledgeAmount, withdrawHeight, addr1, false, big.NewInt(0), false)
	if len(receiveChangeTokenTypeBlockList) != 1 || isRetry || err != nil ||
		!bytes.Equal(db.storageMap[addr2][string(key)], tokenInfoData) ||
		db.balanceMap[addr2][ledger.ViteTokenId].Cmp(pledgeAmount) != 0 ||
		db.balanceMap[addr2][tokenId].Cmp(helper.Big0) != 0 ||
		len(receiveChangeTokenTypeBlockList[0].AccountBlock.Data) != 33 ||
		receiveChangeTokenTypeBlockList[0].AccountBlock.Data[32] != byte(0) ||
		receiveChangeTokenTypeBlockList[0].AccountBlock.Quota != 0 ||
		len(db.logList) != 5 ||
		db.logList[4].Topics[0] != abi.ABIMintage.Events[abi.EventNameChangeTokenType].Id() ||
		!bytes.Equal(db.logList[4].Topics[1].Bytes(), helper.LeftPadBytes(tokenId.Bytes(), 32)) {
		t.Fatalf("receive change token type transaction error")
	}
	db.accountBlockMap[addr2] = make(map[types.Hash]*ledger.AccountBlock)
	db.accountBlockMap[addr2][hash27] = receiveChangeTokenTypeBlockList[0].AccountBlock

	if tokenMap := abi.GetTokenMapByOwner(db, addr3); len(tokenMap) != 1 {
		t.Fatalf("get token map by owner failed")
	}
	if tokenMap := abi.GetTokenMapByOwner(db, addr1); len(tokenMap) != 0 {
		t.Fatalf("get token map by owner failed")
	}

	sbtime := time.Now()
	var latestSb *ledger.SnapshotBlock
	for i := uint64(snapshot2.Height + 1); i <= withdrawHeight; i++ {
		latestSb = &ledger.SnapshotBlock{Height: i, Timestamp: &sbtime, Hash: types.DataHash([]byte{10, byte(i)})}
		db.snapshotBlockList = append(db.snapshotBlockList, latestSb)
	}

	// cancel pledge
	block18Data, err := abi.ABIMintage.PackMethod(abi.MethodNameCancelPledge, tokenId)
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
		SnapshotHash:   latestSb.Hash,
		Timestamp:      &blockTime,
		Hash:           hash18,
		PrevHash:       hash17,
	}
	vm = NewVM()
	db.addr = addr1
	cancelPledgeBlockList, isRetry, err := vm.Run(db, block18, nil)
	if len(cancelPledgeBlockList) != 1 || isRetry || err != nil ||
		!bytes.Equal(cancelPledgeBlockList[0].AccountBlock.Data, block18Data) ||
		cancelPledgeBlockList[0].AccountBlock.Amount.Cmp(helper.Big0) != 0 ||
		cancelPledgeBlockList[0].AccountBlock.Quota != contracts.MintageCancelPledgeGas {
		t.Fatalf("send cancel pledge transaction error")
	}
	db.accountBlockMap[addr3][hash18] = cancelPledgeBlockList[0].AccountBlock

	hash28 := types.DataHash([]byte{2, 8})
	block28 := &ledger.AccountBlock{
		Height:         8,
		AccountAddress: addr2,
		BlockType:      ledger.BlockTypeReceive,
		FromBlockHash:  hash18,
		SnapshotHash:   latestSb.Hash,
		Timestamp:      &blockTime,
		Hash:           hash28,
		PrevHash:       hash27,
	}
	vm = NewVM()
	db.addr = addr2
	receiveCancelPledgeBlockList, isRetry, err := vm.Run(db, block28, cancelPledgeBlockList[0].AccountBlock)
	tokenInfoData, _ = abi.ABIMintage.PackVariable(abi.VariableNameTokenInfo, tokenName, tokenSymbol, totalSupply, decimals, addr3, helper.Big0, uint64(0), addr1, false, big.NewInt(0), false)
	if len(receiveCancelPledgeBlockList) != 2 || isRetry || err != nil ||
		!bytes.Equal(db.storageMap[addr2][string(key)], tokenInfoData) ||
		db.balanceMap[addr2][ledger.ViteTokenId].Cmp(helper.Big0) != 0 ||
		db.balanceMap[addr2][tokenId].Cmp(helper.Big0) != 0 ||
		len(receiveCancelPledgeBlockList[0].AccountBlock.Data) != 33 ||
		receiveCancelPledgeBlockList[0].AccountBlock.Data[32] != byte(0) ||
		receiveCancelPledgeBlockList[0].AccountBlock.Quota != 0 ||
		len(db.logList) != 5 ||
		receiveCancelPledgeBlockList[1].AccountBlock.Height != receiveCancelPledgeBlockList[0].AccountBlock.Height+1 ||
		receiveCancelPledgeBlockList[1].AccountBlock.Amount.Cmp(pledgeAmount) != 0 ||
		receiveCancelPledgeBlockList[1].AccountBlock.ToAddress != addr1 ||
		receiveCancelPledgeBlockList[1].AccountBlock.BlockType != ledger.BlockTypeSendCall {
		t.Fatalf("receive cancel pledge transaction error")
	}
	db.accountBlockMap[addr2] = make(map[types.Hash]*ledger.AccountBlock)
	db.accountBlockMap[addr2][hash28] = receiveCancelPledgeBlockList[0].AccountBlock
}

func TestCheckCreateConsensusGroupData(t *testing.T) {
	tests := []struct {
		data string
		err  error
	}{
		{"51891ff2000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000190000000000000000000000000000000000000000000000000000000000000003000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000032000000000000000000000000000000000000000000005649544520544f4b454e00000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000160000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000001e000000000000000000000000000000000000000000000000000000000000000600000000000000000000000000000000000000000000000000de0b6b3a7640000000000000000000000000000000000000000000000005649544520544f4b454e000000000000000000000000000000000000000000000000000000000003f4800000000000000000000000000000000000000000000000000000000000000000", nil},
		{"51891ff2000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000190000000000000000000000000000000000000000000000000000000000000003000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000032000000000000000000000000000000000000000000005649544520544f4b454e00000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000160000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000001e000000000000000000000000000000000000000000000000000000000000000600000000000000000000000000000000000000000000000000de0b6b3a7640000000000000000000000000000000000000000000000005649544520544f4b454e000000000000000000000000000000000000000000000000000000000003f4800000000000000000000000000000000000000000000000000000000000000040000000000000000000000000000000000000000000000000000000000000000a00000000000000000000000000000000000000000000aaaaaaaaaaaaaaaaaaaa", errors.New("")},
		{"51891ff2000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000190000000000000000000000000000000000000000000000000000000000000003000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000032000000000000000000000000000000000000000000005649544520544f4b454e00000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000160000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000001e000000000000000000000000000000000000000000000000000000000000000600000000000000000000000000000000000000000000000000de0b6b3a7640000000000000000000000000000000000000000000000005649544520544f4b454e000000000000000000000000000000000000000000000000000000000003f48000000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000005649544520544f4b454e", errors.New("")},
		{"51891ff2000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000190000000000000000000000000000000000000000000000000000000000000003000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000032000000000000000000000000000000000000000000005649544520544f4b454e00000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000160000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000001e000000000000000000000000000000000000000000000000000000000000000600000000000000000000000000000000000000000000000000de0b6b3a7640000000000000000000000000000000000000000000000005649544520544f4b454e000000000000000000000000000000000000000000000000000000000003f4700000000000000000000000000000000000000000000000000000000000000000", errors.New("")},
		{"51891ff2000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000190000000000000000000000000000000000000000000000000000000000000003000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000032000000000000000000000000000000000000000000005649544520544f4b454e00000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000160000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000001e000000000000000000000000000000000000000000000000000000000000000600000000000000000000000000000000000000000000000000de0b6b3a764000000000000000000000000000000000000000000000000aaaaaaaaaaaaaaaaaaaa000000000000000000000000000000000000000000000000000000000003f4800000000000000000000000000000000000000000000000000000000000000000", errors.New("")},
		{"51891ff2000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000190000000000000000000000000000000000000000000000000000000000000003000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000032000000000000000000000000000000000000000000005649544520544f4b454e00000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000160000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000001e000000000000000000000000000000000000000000000000000000000000000600000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000005649544520544f4b454e000000000000000000000000000000000000000000000000000000000003f4800000000000000000000000000000000000000000000000000000000000000000", errors.New("")},
		{"51891ff2000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000190000000000000000000000000000000000000000000000000000000000000003000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000032000000000000000000000000000000000000000000005649544520544f4b454e00000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000160000000000000000000000000000000000000000000000000000000000000000300000000000000000000000000000000000000000000000000000000000001e000000000000000000000000000000000000000000000000000000000000000600000000000000000000000000000000000000000000000000de0b6b3a7640000000000000000000000000000000000000000000000005649544520544f4b454e000000000000000000000000000000000000000000000000000000000003f4800000000000000000000000000000000000000000000000000000000000000000", errors.New("")},
		{"51891ff2000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000190000000000000000000000000000000000000000000000000000000000000003000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000032000000000000000000000000000000000000000000005649544520544f4b454e00000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000160000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000001e000000000000000000000000000000000000000000000000000000000000000600000000000000000000000000000000000000000000000000de0b6b3a7640000000000000000000000000000000000000000000000005649544520544f4b454e000000000000000000000000000000000000000000000000000000000003f4800000000000000000000000000000000000000000000000000000000000000000", errors.New("")},
		{"51891ff200000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000019000000000000000000000000000000000000000000000000000000000000000300000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000003200000000000000000000000000000000000000000000aaaaaaaaaaaaaaaaaaaa00000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000160000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000001e000000000000000000000000000000000000000000000000000000000000000600000000000000000000000000000000000000000000000000de0b6b3a7640000000000000000000000000000000000000000000000005649544520544f4b454e000000000000000000000000000000000000000000000000000000000003f4800000000000000000000000000000000000000000000000000000000000000000", errors.New("")},
		{"51891ff2000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000190000000000000000000000000000000000000000000000000000000000000003000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000018000000000000000000000000000000000000000000005649544520544f4b454e00000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000160000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000001e000000000000000000000000000000000000000000000000000000000000000600000000000000000000000000000000000000000000000000de0b6b3a7640000000000000000000000000000000000000000000000005649544520544f4b454e000000000000000000000000000000000000000000000000000000000003f4800000000000000000000000000000000000000000000000000000000000000000", errors.New("")},
		{"51891ff20000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001900000000000000000000000000000000000000000000000000000000000000030000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000001a0000000000000000000000000000000000000000000000000000000000000032000000000000000000000000000000000000000000005649544520544f4b454e00000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000160000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000001e000000000000000000000000000000000000000000000000000000000000000600000000000000000000000000000000000000000000000000de0b6b3a7640000000000000000000000000000000000000000000000005649544520544f4b454e000000000000000000000000000000000000000000000000000000000003f4800000000000000000000000000000000000000000000000000000000000000000", errors.New("")},
		{"51891ff200000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000019000000000000000000000000000000000000000000000000000000000000003d000000000000000000000000000000000000000000000000000000000000000a00000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000032000000000000000000000000000000000000000000005649544520544f4b454e00000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000160000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000001e000000000000000000000000000000000000000000000000000000000000000600000000000000000000000000000000000000000000000000de0b6b3a7640000000000000000000000000000000000000000000000005649544520544f4b454e000000000000000000000000000000000000000000000000000000000003f4800000000000000000000000000000000000000000000000000000000000000000", errors.New("")},
		{"51891ff2000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000190000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000025900000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000032000000000000000000000000000000000000000000005649544520544f4b454e00000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000160000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000001e000000000000000000000000000000000000000000000000000000000000000600000000000000000000000000000000000000000000000000de0b6b3a7640000000000000000000000000000000000000000000000005649544520544f4b454e000000000000000000000000000000000000000000000000000000000003f4800000000000000000000000000000000000000000000000000000000000000000", errors.New("")},
		{"51891ff2000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000190000000000000000000000000000000000000000000000000000000000000259000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000032000000000000000000000000000000000000000000005649544520544f4b454e00000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000160000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000001e000000000000000000000000000000000000000000000000000000000000000600000000000000000000000000000000000000000000000000de0b6b3a7640000000000000000000000000000000000000000000000005649544520544f4b454e000000000000000000000000000000000000000000000000000000000003f4800000000000000000000000000000000000000000000000000000000000000000", errors.New("")},
		{"51891ff2000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000670000000000000000000000000000000000000000000000000000000000000003000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000032000000000000000000000000000000000000000000005649544520544f4b454e00000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000160000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000001e000000000000000000000000000000000000000000000000000000000000000600000000000000000000000000000000000000000000000000de0b6b3a7640000000000000000000000000000000000000000000000005649544520544f4b454e000000000000000000000000000000000000000000000000000000000003f4800000000000000000000000000000000000000000000000000000000000000000", errors.New("")},
		{"51891ff2000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000003000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000032000000000000000000000000000000000000000000005649544520544f4b454e00000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000160000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000001e000000000000000000000000000000000000000000000000000000000000000600000000000000000000000000000000000000000000000000de0b6b3a7640000000000000000000000000000000000000000000000005649544520544f4b454e000000000000000000000000000000000000000000000000000000000003f4800000000000000000000000000000000000000000000000000000000000000000", errors.New("")},
	}
	db, _, _, _, _, _ := prepareDb(big.NewInt(1))
	for i, test := range tests {
		inputdata, _ := hex.DecodeString(test.data)
		param := new(types.ConsensusGroupInfo)
		err := abi.ABIConsensusGroup.UnpackMethod(param, abi.MethodNameCreateConsensusGroup, inputdata)
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
		{"00", errors.New(""), false},
		{"46d0ce8b000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000a000000000000000000000000000000000000000000000000000000000000000e00000000000000000000000000000000000000000033b2e3c9fd0803ce80000000000000000000000000000000000000000000000000000000000000000000012000000000000000000000000000000000000000000000000000000000000000956697465546f6b656e0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000956697465546f6b656e0000000000000000000000000000000000000000000000", nil, true},
		{"46d0ce8b000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000a000000000000000000000000000000000000000000000000000000000000000e00000000000000000000000000000000000000000033b2e3c9fd0803ce80000000000000000000000000000000000000000000000000000000000000000000012000000000000000000000000000000000000000000000000000000000000000956697465546f6b656e00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000009", errors.New(""), true},
		{"46d0ce8b000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000a000000000000000000000000000000000000000000000000000000000000000e00000000000000000000000000000000000000000033b2e3c9fd0803ce80000000000000000000000000000000000000000000000000000000000000000000012000000000000000000000000000000000000000000000000000000000000000956697465546f6b656e0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000956697465546f6b651F0000000000000000000000000000000000000000000000", nil, false},
		{"46d0ce8b000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000a000000000000000000000000000000000000000000000000000000000000000e00000000000000000000000000000000000000000033b2e3c9fd0803ce80000000000000000000000000000000000000000000000000000000000000000000012000000000000000000000000000000000000000000000000000000000000000956697465546f6b651F0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000956697465546f6b651F0000000000000000000000000000000000000000000000", nil, false},
		{"46d0ce8b000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000a000000000000000000000000000000000000000000000000000000000000000e00000000000000000000000000000000000000000033b2e3c9fd0803ce80000000000000000000000000000000000000000000000000000000000000000000012000000000000000000000000000000000000000000000000000000000000000956697465546f6b656e0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000a56697465546f6b656e0000000000000000000000000000000000000000000000", nil, false},
		{"46d0ce8b000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000a000000000000000000000000000000000000000000000000000000000000000e00000000000000000000000000000000000000000033b2e3c9fd0803ce8000000000000000000000000000000000000000000000000000000000000000000012e000000000000000000000000000000000000000000000000000000000000000956697465546f6b656e0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000956697465546f6b656e0000000000000000000000000000000000000000000000", nil, false},
	}
	for i, test := range tests {
		inputdata, _ := hex.DecodeString(test.data)
		param := new(abi.ParamMintage)
		err := abi.ABIMintage.UnpackMethod(param, abi.MethodNameMintage, inputdata)
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
		{"chain", true},
		{"ab", true},
		{"ab ", false},
		{"chain b", true},
		{"chain  b", false},
		{"chain _b", true},
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
	mintageData, err := abi.ABIMintage.PackVariable(abi.VariableNameMintage, tokenName, tokenSymbol, totalSupply, decimals, viteAddress, big.NewInt(0), uint64(0))
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

	conditionRegisterData, err := abi.ABIConsensusGroup.PackVariable(abi.VariableNameConditionRegisterOfPledge, new(big.Int).Mul(big.NewInt(1e6), util.AttovPerVite), ledger.ViteTokenId, uint64(3600*24*90))
	if err != nil {
		t.Fatalf("pack register condition variable error, %v", err)
	}
	snapshotConsensusGroupData, err := abi.ABIConsensusGroup.PackVariable(abi.VariableNameConsensusGroupInfo,
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
	commonConsensusGroupData, err := abi.ABIConsensusGroup.PackVariable(abi.VariableNameConsensusGroupInfo,
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
	fmt.Printf("address:%v\n", hex.EncodeToString(types.AddressConsensusGroup.Bytes()))
	fmt.Printf("AccountBlock{\n\tBlockType: %v,\n\tAccountAddress: %v,\n\tHeight: %v,\n\tAmount: %v,\n\tTokenId:ledger.ViteTokenId,\n\tQuota:0,\n\tFee:%v,\n\tData:%v,\n}\n",
		ledger.BlockTypeReceive, hex.EncodeToString(types.AddressConsensusGroup.Bytes()), 1, big.NewInt(0), big.NewInt(0), []byte{})
	fmt.Printf("Storage:{\n\t%v:%v,\n\t%v:%v}\n", hex.EncodeToString(abi.GetConsensusGroupKey(types.SNAPSHOT_GID)), hex.EncodeToString(snapshotConsensusGroupData), hex.EncodeToString(abi.GetConsensusGroupKey(types.DELEGATE_GID)), hex.EncodeToString(commonConsensusGroupData))

	fmt.Println("-------------snapshot consensus group and common consensus group register genesis block-------------")
	fmt.Printf("address:%v\n", hex.EncodeToString(types.AddressRegister.Bytes()))
	fmt.Printf("AccountBlock{\n\tBlockType: %v,\n\tAccountAddress: %v,\n\tHeight: %v,\n\tAmount: %v,\n\tTokenId:ledger.ViteTokenId,\n\tQuota:0,\n\tFee:%v,\n\tData:%v,\n}\n",
		ledger.BlockTypeReceive, hex.EncodeToString(types.AddressRegister.Bytes()), 1, big.NewInt(0), big.NewInt(0), []byte{})
	fmt.Printf("Storage:{\n")
	for i := 1; i <= 25; i++ {
		addr, _, _ := types.CreateAddress()
		registerData, err := abi.ABIRegister.PackVariable(abi.VariableNameRegistration, "node"+strconv.Itoa(i), addr, addr, helper.Big0, uint64(1), uint64(1), uint64(0), []types.Address{addr})
		if err != nil {
			t.Fatalf("pack registration variable error, %v", err)
		}
		snapshotKey := abi.GetRegisterKey("snapshotNode1", types.SNAPSHOT_GID)
		fmt.Printf("\t%v: %v\n", hex.EncodeToString(snapshotKey), hex.EncodeToString(registerData))
	}
	fmt.Println("}")
}
