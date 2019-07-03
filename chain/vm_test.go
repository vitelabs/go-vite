package chain

import (
	rand2 "crypto/rand"
	"encoding/hex"
	"fmt"
	"github.com/vitelabs/go-vite/chain/test_tools"
	"github.com/vitelabs/go-vite/chain/utils"
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/consensus/core"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/generator"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vite/net/message"
	"github.com/vitelabs/go-vite/vm"
	"github.com/vitelabs/go-vite/vm/contracts/abi"
	"github.com/vitelabs/go-vite/vm/util"
	"github.com/vitelabs/go-vite/vm_db"
	"math/big"
	"math/rand"
	"testing"
	"time"
)

func initVMEnvironment() {
	vm.InitVMConfig(false, true, false, "")
	chainInstance, err := NewChainInstance("unit_test/devdata", false)
	if err != nil {
		panic(err)
	}
	TearDown(chainInstance)
	Clear(chainInstance)
}

var (
	testAddr, _    = types.HexToAddress("vite_ab24ef68b84e642c0ddca06beec81c9acb1977bbd7da27a87a")
	testTokenId, _ = types.HexToTokenTypeId("tti_045e6ca837c143cd477b32f3")

	pledgeAmount         = new(big.Int).Mul(big.NewInt(134), big1e18)
	registerPledgeAmount = new(big.Int).Mul(big.NewInt(5e5), big1e18)
	mintFee              = new(big.Int).Mul(big.NewInt(1e3), big1e18)
	createContractFee    = new(big.Int).Mul(big.NewInt(10), big1e18)

	big1e18 = big.NewInt(1e18)
	big0    = big.NewInt(0)

	mockConsensus = &test_tools.MockConsensus{
		Cr: &test_tools.MockConsensusReader{
			DayTimeIndex: test_tools.MockTimeIndex{
				GenesisTime: time.Unix(1558411200, 0),
				Interval:    3 * time.Second,
			},
			DayStatsMap: map[uint64]*core.DayStats{
				0: {
					Index: 0,
					Stats: map[string]*core.SbpStats{
						"vite super snapshot producer for test 01": {
							Index:            0,
							BlockNum:         10,
							ExceptedBlockNum: 20,
							VoteCnt:          &core.BigInt{big.NewInt(100)},
							Name:             "vite super snapshot producer for test 01",
						},
					},
					VoteSum:    &core.BigInt{big.NewInt(100)},
					BlockTotal: 10,
				},
			},
		},
	}
)

func BenchmarkVMCallSend(b *testing.B) {
	sendBlock := makeSendBlock(testAddr, testAddr, nil, big.NewInt(1), big0)
	benchmarkSend(b, sendBlock)
}

func BenchmarkVMCallReceive(b *testing.B) {
	sendBlock := makeSendBlock(testAddr, testAddr, nil, big.NewInt(1), big0)
	receiveBlock := makeReceiveBlock(testAddr)
	benchmarkReceive(b, sendBlock, receiveBlock)
}

func BenchmarkVMCreateSend(b *testing.B) {
	benchmarkSend(b, makeCreateSendBlock())
}

func BenchmarkVMCreateReceive(b *testing.B) {
	sendBlock := makeCreateSendBlock()
	receiveBlock := makeReceiveBlock(sendBlock.ToAddress)
	benchmarkReceive(b, sendBlock, receiveBlock)
}
func makeCreateSendBlock() *ledger.AccountBlock {
	createContractData, _ := hex.DecodeString("000000000000000000020105010a")
	sendBlock := makeSendBlock(testAddr, testAddr, createContractData, big.NewInt(1), createContractFee)
	sendBlock.BlockType = ledger.BlockTypeSendCreate
	sendBlock.ToAddress = util.NewContractAddress(sendBlock.AccountAddress, sendBlock.Height, sendBlock.PrevHash)
	return sendBlock
}

// write storage to disk not involved in built-in contract benchmarks
func BenchmarkVMPledgeSend(b *testing.B) {
	sendBlock := makePledgeSendBlock(testAddr)
	benchmarkSend(b, sendBlock)
}
func BenchmarkVMPledgeReceive(b *testing.B) {
	sendBlock := makePledgeSendBlock(testAddr)
	receiveBlock := makeReceiveBlock(types.AddressPledge)
	benchmarkReceive(b, sendBlock, receiveBlock)
}
func makePledgeSendBlock(addr types.Address) *ledger.AccountBlock {
	data, err := abi.ABIPledge.PackMethod(abi.MethodNamePledge, addr)
	if err != nil {
		panic(err)
	}
	return makeSendBlock(addr, types.AddressPledge, data, pledgeAmount, big0)
}

func BenchmarkVMCancelPledgeSend(b *testing.B) {
	sendBlock := makeCancelPledgeSendBlock(testAddr)
	benchmarkSend(b, sendBlock)
}
func BenchmarkVMCancelPledgeReceive(b *testing.B) {
	sendBlock := makeCancelPledgeSendBlock(testAddr)
	receiveBlock := makeReceiveBlock(types.AddressPledge)
	benchmarkReceive(b, sendBlock, receiveBlock)
}
func makeCancelPledgeSendBlock(addr types.Address) *ledger.AccountBlock {
	data, err := abi.ABIPledge.PackMethod(abi.MethodNameCancelPledge, addr, pledgeAmount)
	if err != nil {
		panic(err)
	}
	return makeSendBlock(addr, types.AddressPledge, data, big0, big0)
}

func BenchmarkVMRegisterSend(b *testing.B) {
	sendBlock := makeRegisterSendBlock(testAddr)
	benchmarkSend(b, sendBlock)
}
func BenchmarkVMRegisterReceive(b *testing.B) {
	sendBlock := makeRegisterSendBlock(testAddr)
	receiveBlock := makeReceiveBlock(types.AddressConsensusGroup)
	benchmarkReceive(b, sendBlock, receiveBlock)
}
func makeRegisterSendBlock(addr types.Address) *ledger.AccountBlock {
	data, err := abi.ABIConsensusGroup.PackMethod(abi.MethodNameRegister, types.SNAPSHOT_GID, "vite super snapshot producer for test 02", addr)
	if err != nil {
		panic(err)
	}
	return makeSendBlock(addr, types.AddressConsensusGroup, data, registerPledgeAmount, big0)
}

func BenchmarkVMCancelRegisterSend(b *testing.B) {
	sendBlock := makeCancelRegisterSendBlock(testAddr)
	benchmarkSend(b, sendBlock)
}
func BenchmarkVMCancelRegisterReceive(b *testing.B) {
	sendBlock := makeCancelRegisterSendBlock(testAddr)
	receiveBlock := makeReceiveBlock(types.AddressConsensusGroup)
	benchmarkReceive(b, sendBlock, receiveBlock)
}
func makeCancelRegisterSendBlock(addr types.Address) *ledger.AccountBlock {
	data, err := abi.ABIConsensusGroup.PackMethod(abi.MethodNameCancelRegister, types.SNAPSHOT_GID, "vite super snapshot producer for test 01")
	if err != nil {
		panic(err)
	}
	return makeSendBlock(addr, types.AddressConsensusGroup, data, big0, big0)
}

func BenchmarkVMRewardSend(b *testing.B) {
	sendBlock := makeRewardSendBlock(testAddr)
	benchmarkSend(b, sendBlock)
}
func BenchmarkVMRewardReceive(b *testing.B) {
	sendBlock := makeRewardSendBlock(testAddr)
	receiveBlock := makeReceiveBlock(types.AddressConsensusGroup)
	benchmarkReceive(b, sendBlock, receiveBlock)
}
func makeRewardSendBlock(addr types.Address) *ledger.AccountBlock {
	data, err := abi.ABIConsensusGroup.PackMethod(abi.MethodNameReward, types.SNAPSHOT_GID, "vite super snapshot producer for test 01", addr)
	if err != nil {
		panic(err)
	}
	return makeSendBlock(addr, types.AddressConsensusGroup, data, big0, big0)
}

func BenchmarkVMUpdateRegistrationSend(b *testing.B) {
	sendBlock := makeUpdateRegistrationBlock(testAddr)
	benchmarkSend(b, sendBlock)
}
func BenchmarkVMUpdateRegistrationReceive(b *testing.B) {
	sendBlock := makeUpdateRegistrationBlock(testAddr)
	receiveBlock := makeReceiveBlock(types.AddressConsensusGroup)
	benchmarkReceive(b, sendBlock, receiveBlock)
}
func makeUpdateRegistrationBlock(addr types.Address) *ledger.AccountBlock {
	nodeAddr, _, err := types.CreateAddress()
	if err != nil {
		panic(err)
	}
	data, err := abi.ABIConsensusGroup.PackMethod(abi.MethodNameUpdateRegistration, types.SNAPSHOT_GID, "vite super snapshot producer for test 01", nodeAddr)
	if err != nil {
		panic(err)
	}
	return makeSendBlock(addr, types.AddressConsensusGroup, data, big0, big0)
}

func BenchmarkVMVoteSend(b *testing.B) {
	sendBlock := makeVoteBlock(testAddr)
	benchmarkSend(b, sendBlock)
}
func BenchmarkVMVoteReceive(b *testing.B) {
	sendBlock := makeVoteBlock(testAddr)
	receiveBlock := makeReceiveBlock(types.AddressConsensusGroup)
	benchmarkReceive(b, sendBlock, receiveBlock)
}
func makeVoteBlock(addr types.Address) *ledger.AccountBlock {
	data, err := abi.ABIConsensusGroup.PackMethod(abi.MethodNameVote, types.SNAPSHOT_GID, "vite super snapshot producer for test 01")
	if err != nil {
		panic(err)
	}
	return makeSendBlock(addr, types.AddressConsensusGroup, data, big0, big0)
}

func BenchmarkVMCancelVoteSend(b *testing.B) {
	sendBlock := makeCancelVoteBlock(testAddr)
	benchmarkSend(b, sendBlock)
}
func BenchmarkVMCancelVoteReceive(b *testing.B) {
	sendBlock := makeCancelVoteBlock(testAddr)
	receiveBlock := makeReceiveBlock(types.AddressConsensusGroup)
	benchmarkReceive(b, sendBlock, receiveBlock)
}
func makeCancelVoteBlock(addr types.Address) *ledger.AccountBlock {
	data, err := abi.ABIConsensusGroup.PackMethod(abi.MethodNameCancelVote, types.SNAPSHOT_GID)
	if err != nil {
		panic(err)
	}
	return makeSendBlock(addr, types.AddressConsensusGroup, data, big0, big0)
}

func BenchmarkVMMintSend(b *testing.B) {
	sendBlock := makeMintBlock(testAddr)
	benchmarkSend(b, sendBlock)
}
func BenchmarkVMMintReceive(b *testing.B) {
	sendBlock := makeMintBlock(testAddr)
	receiveBlock := makeReceiveBlock(types.AddressMintage)
	benchmarkReceive(b, sendBlock, receiveBlock)
}
func makeMintBlock(addr types.Address) *ledger.AccountBlock {
	data, err := abi.ABIMintage.PackMethod(abi.MethodNameMint, true, "vite test token minted by user for test1", "VTTMBUFT01", big.NewInt(1e4), uint8(2), big.NewInt(1e5), false)
	if err != nil {
		panic(err)
	}
	return makeSendBlock(addr, types.AddressMintage, data, big0, mintFee)
}

func BenchmarkVMIssueSend(b *testing.B) {
	sendBlock := makeIssueBlock(testAddr)
	benchmarkSend(b, sendBlock)
}
func BenchmarkVMIssueReceive(b *testing.B) {
	sendBlock := makeIssueBlock(testAddr)
	receiveBlock := makeReceiveBlock(types.AddressMintage)
	benchmarkReceive(b, sendBlock, receiveBlock)
}
func makeIssueBlock(addr types.Address) *ledger.AccountBlock {
	data, err := abi.ABIMintage.PackMethod(abi.MethodNameIssue, testTokenId, big.NewInt(1e4), addr)
	if err != nil {
		panic(err)
	}
	return makeSendBlock(addr, types.AddressMintage, data, big0, big0)
}

func BenchmarkVMBurnSend(b *testing.B) {
	sendBlock := makeBurnBlock(testAddr)
	benchmarkSend(b, sendBlock)
}
func BenchmarkVMBurnReceive(b *testing.B) {
	sendBlock := makeBurnBlock(testAddr)
	receiveBlock := makeReceiveBlock(types.AddressMintage)
	benchmarkReceive(b, sendBlock, receiveBlock)
}
func makeBurnBlock(addr types.Address) *ledger.AccountBlock {
	data, err := abi.ABIMintage.PackMethod(abi.MethodNameBurn)
	if err != nil {
		panic(err)
	}
	return makeSendBlock(addr, types.AddressMintage, data, big.NewInt(1e3), big0)
}

func BenchmarkVMTransferOwnerSend(b *testing.B) {
	sendBlock := makeTransferOwnerBlock(testAddr)
	benchmarkSend(b, sendBlock)
}
func BenchmarkVMTransferOwnerReceive(b *testing.B) {
	sendBlock := makeTransferOwnerBlock(testAddr)
	receiveBlock := makeReceiveBlock(types.AddressMintage)
	benchmarkReceive(b, sendBlock, receiveBlock)
}
func makeTransferOwnerBlock(addr types.Address) *ledger.AccountBlock {
	newOwner, _, err := types.CreateAddress()
	if err != nil {
		panic(err)
	}
	data, err := abi.ABIMintage.PackMethod(abi.MethodNameTransferOwner, testTokenId, newOwner)
	if err != nil {
		panic(err)
	}
	return makeSendBlock(addr, types.AddressMintage, data, big0, big0)
}

func BenchmarkVMChangeTokenTypeSend(b *testing.B) {
	sendBlock := makeChangeTokenTypeBlock(testAddr)
	benchmarkSend(b, sendBlock)
}
func BenchmarkVMChangeTokenTypeReceive(b *testing.B) {
	sendBlock := makeChangeTokenTypeBlock(testAddr)
	receiveBlock := makeReceiveBlock(types.AddressMintage)
	benchmarkReceive(b, sendBlock, receiveBlock)
}
func makeChangeTokenTypeBlock(addr types.Address) *ledger.AccountBlock {
	data, err := abi.ABIMintage.PackMethod(abi.MethodNameChangeTokenType, testTokenId)
	if err != nil {
		panic(err)
	}
	return makeSendBlock(addr, types.AddressMintage, data, big0, big0)
}

func benchmarkSend(b *testing.B, sendBlock *ledger.AccountBlock) {
	initVMEnvironment()
	chainInstance, _, _ := SetUp(0, 0, 0)
	defer TearDown(chainInstance)
	prevBlock, err := chainInstance.GetLatestAccountBlock(sendBlock.AccountAddress)
	if err != nil {
		panic(err)
	}
	if prevBlock != nil {
		sendBlock.PrevHash = prevBlock.Hash
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		db, err := vm_db.NewVmDb(chainInstance, &sendBlock.AccountAddress, &chainInstance.GetLatestSnapshotBlock().Hash, &sendBlock.PrevHash)
		if err != nil {
			panic(err)
		}
		_, _, err = vm.NewVM(nil).RunV2(db, sendBlock, nil, nil)
		if err != nil {
			panic(err)
		}
	}
}

func benchmarkReceive(b *testing.B, sendBlock *ledger.AccountBlock, receiveBlock *ledger.AccountBlock) {
	initVMEnvironment()
	chainInstance, _, _ := SetUp(1, 100, 1)
	defer TearDown(chainInstance)
	if sendBlock.BlockType == ledger.BlockTypeSendCreate {
		prevBlock, err := chainInstance.GetLatestAccountBlock(sendBlock.AccountAddress)
		if err != nil {
			panic(err)
		}
		if prevBlock != nil {
			sendBlock.PrevHash = prevBlock.Hash
		}
		db, err := vm_db.NewVmDb(chainInstance, &sendBlock.AccountAddress, &chainInstance.GetLatestSnapshotBlock().Hash, &sendBlock.PrevHash)
		if err != nil {
			panic(err)
		}
		sendVMBlock, _, err := vm.NewVM(nil).RunV2(db, sendBlock, nil, nil)
		if err != nil {
			panic(err)
		}
		if err := chainInstance.InsertAccountBlock(sendVMBlock); err != nil {
			panic(err)
		}
		receiveBlock.AccountAddress = sendVMBlock.AccountBlock.ToAddress
		pubKey := ed25519.PublicKey(randomByte32())
		_, _, err = InsertSnapshotBlock(chainInstance, false, &pubKey)
		if err != nil {
			panic(err)
		}
	}
	prevBlock, err := chainInstance.GetLatestAccountBlock(receiveBlock.AccountAddress)
	if err != nil {
		panic(err)
	}
	if prevBlock != nil {
		receiveBlock.PrevHash = prevBlock.Hash
	} else {
		receiveBlock.PrevHash = types.Hash{}
	}
	globalStatus := generator.NewVMGlobalStatus(chainInstance, chainInstance.GetLatestSnapshotBlock(), types.Hash{})

	cr := util.NewVmConsensusReader(mockConsensus.SBPReader())
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		db, err := vm_db.NewVmDb(chainInstance, &receiveBlock.AccountAddress, &chainInstance.GetLatestSnapshotBlock().Hash, &receiveBlock.PrevHash)
		if err != nil {
			panic(err)
		}
		_, _, err = vm.NewVM(cr).RunV2(db, receiveBlock, sendBlock, globalStatus)
		if err != nil {
			panic(err)
		}
	}
}

func makeSendBlock(fromAddr, toAddr types.Address, data []byte, amount, fee *big.Int) *ledger.AccountBlock {
	sendCallBlock := &ledger.AccountBlock{
		BlockType:      ledger.BlockTypeSendCall,
		Height:         1,
		FromBlockHash:  types.ZERO_HASH,
		AccountAddress: fromAddr,
		ToAddress:      toAddr,
		PrevHash:       types.Hash{},
		Amount:         amount,
		Fee:            fee,
		TokenId:        ledger.ViteTokenId,
		Data:           data,
		Quota:          21000,
		QuotaUsed:      21000,
		Difficulty:     big.NewInt(75164738),
	}
	sendCallBlock.Hash, _ = types.HexToHash("8f85502f81fc544cb6700ad9ecc44f3eace065ae8e34d2269d7ff8d7c94ac920")
	sendCallBlock.PublicKey, _ = hex.DecodeString("0e8a21f65c0ac55702512b426b9057ca4a1d39e260f37daae0c956a5d237a0b3")
	sendCallBlock.Nonce, _ = hex.DecodeString("6d2f638c1941473f")
	sendCallBlock.Signature, _ = hex.DecodeString("3a95df1a01fc006261ac653910fea6523e9e452dddfaeeddf521a1cb083985bfdf1c314132d1f8aa608938edaec7b0dbb9f362dbfcf6d16871f96d46e61a840a")
	return sendCallBlock
}

func makeReceiveBlock(addr types.Address) *ledger.AccountBlock {
	receiveCallBlock := &ledger.AccountBlock{
		BlockType:      ledger.BlockTypeReceive,
		AccountAddress: addr,
		Height:         1000215,
		Quota:          37589,
		QuotaUsed:      37589,
	}
	receiveCallBlock.FromBlockHash, _ = types.HexToHash("8f85502f81fc544cb6700ad9ecc44f3eace065ae8e34d2269d7ff8d7c94ac920")
	receiveCallBlock.Hash, _ = types.HexToHash("12fb277325de59489063b40839864aa12df362a6946af6d7da0fcbcd4c8f0c7e")
	receiveCallBlock.PrevHash, _ = types.HexToHash("75291239501f1671c92f6248a79906c60b18b84ae983eea462e20f23b73013dc")
	receiveCallBlock.PublicKey, _ = hex.DecodeString("d5c8311234f52f7c7e98fccab019d5e0348f894f914e663b6a46eb4c2c5c02b1")
	receiveCallBlock.Signature, _ = hex.DecodeString("271af4398f9bfeabb1fc8738e182b50e4bb3a3781aba8ac869dc628a7a39f5c30c545adaff8f54741000ebf9d28b7a5cc0549c0c5f7d8949d66c3b1b6abc8305")
	return receiveCallBlock
}

func BenchmarkVMSetValue(b *testing.B) {
	chainInstance, _, _ := SetUp(0, 0, 0)
	defer TearDown(chainInstance)
	addr, _ := types.HexToAddress("vite_0000000000000000000000000000000000000003f6af7459b9")
	db := vm_db.NewVmDbByAddr(chainInstance, &addr)
	value := helper.Tt256m1.Bytes()
	bigKey := new(big.Int)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := types.DataHash(bigKey.SetInt64(int64(i)).Bytes())
		err := db.SetValue(key.Bytes(), value)
		if err != nil {
			b.Fatalf("db set value failed, err: %v", err)
		}
	}
}

// get value from cache
func BenchmarkVMGetValue(b *testing.B) {
	chainInstance, _, _ := SetUp(0, 0, 0)
	defer TearDown(chainInstance)

	pub, pri, err := ed25519.GenerateKey(rand2.Reader)
	if err != nil {
		panic(err)
	}
	addr := types.PubkeyToAddress(pub)
	account := NewAccount(chainInstance, pub, pri)
	toPub, toPri, err := ed25519.GenerateKey(rand2.Reader)
	if err != nil {
		panic(err)
	}
	toAddr := types.PubkeyToAddress(toPub)
	toAccount := NewAccount(chainInstance, toPub, toPri)
	accounts := make(map[types.Address]*Account)
	accounts[addr] = account
	accounts[toAddr] = toAccount
	key := types.DataHash([]byte{1})
	value := new(big.Int).Set(helper.Tt256m1)
	cTxOptions := &CreateTxOptions{
		MockSignature: true, // mock signature
		KeyValue:      map[string][]byte{(string)(key.Bytes()): value.Bytes()},
		Quota:         rand.Uint64() % 100,
	}
	vmBlock, createBlockErr := account.CreateSendBlock(toAccount, cTxOptions)
	if createBlockErr != nil {
		b.Fatalf("create send create failed, err: %v", createBlockErr)
	}
	account.InsertBlock(vmBlock, accounts)
	if err := chainInstance.InsertAccountBlock(vmBlock); err != nil {
		panic(err)
	}
	snapshotBlock, _, err := InsertSnapshotBlock(chainInstance, true, nil)
	if err != nil {
		panic(err)
	}
	// snapshot
	Snapshot(accounts, snapshotBlock)
	chainInstance.flusher.Flush()

	db := vm_db.NewVmDbByAddr(chainInstance, &addr)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := db.GetValue(key.Bytes())
		if err != nil {
			b.Fatalf("db get value failed, err: %v", err)
		}
	}
}

// get value from disk
func BenchmarkVMGetBalance(b *testing.B) {
	chainInstance, _, _ := SetUp(0, 0, 0)
	defer TearDown(chainInstance)
	addr, _ := types.HexToAddress("vite_0000000000000000000000000000000000000003f6af7459b9")
	db := vm_db.NewVmDbByAddr(chainInstance, &addr)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := db.GetBalance(&ledger.ViteTokenId)
		if err != nil {
			b.Fatalf("db get balance failed, err: %v", err)
		}
	}
}

// get seed from cache
func BenchmarkVMGetSeed(b *testing.B) {
	chainInstance, _, _ := SetUp(1, 100, 1)
	defer TearDown(chainInstance)
	sb := chainInstance.GetLatestSnapshotBlock()
	fromHash, _ := types.HexToHash("e94d63b892e7490d0fed33ac4530f515641bb74935a06a0e76fca72577f0fe82")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := chainInstance.GetSeed(sb, fromHash)
		if err != nil {
			b.Fatalf("db get seed failed, err: %v", err)
		}
	}
}

// check confirm time from cache
func BenchmarkVMCheckConfirmTime(b *testing.B) {
	chainInstance, _, _ := SetUp(0, 0, 0)
	defer TearDown(chainInstance)

	pub, pri, err := ed25519.GenerateKey(rand2.Reader)
	if err != nil {
		panic(err)
	}
	addr := types.PubkeyToAddress(pub)
	account := NewAccount(chainInstance, pub, pri)
	accounts := make(map[types.Address]*Account)
	accounts[addr] = account
	latestHeight := account.GetLatestHeight()
	cTxOptions := &CreateTxOptions{
		MockSignature: true,                         // mock signature
		KeyValue:      createKeyValue(latestHeight), // create key value
		VmLogList:     createVmLogList(),            // create vm log list
		//Quota:         rand.Uint64() % 10000,
		Quota: rand.Uint64() % 100,
	}
	cTxOptions.ContractMeta = &ledger.ContractMeta{
		SendConfirmedTimes: 1,
		SeedConfirmedTimes: 1,
		Gid:                types.DataToGid(chain_utils.Uint64ToBytes(uint64(time.Now().UnixNano()))),
	}
	toAddr := types.CreateContractAddress([]byte{1})
	toAccount := &Account{
		Addr:              toAddr,
		chainInstance:     chainInstance,
		OnRoadBlocks:      make(map[types.Hash]*ledger.AccountBlock),
		BlocksMap:         make(map[types.Hash]*ledger.AccountBlock),
		SendBlocksMap:     make(map[types.Hash]*ledger.AccountBlock),
		ReceiveBlocksMap:  make(map[types.Hash]*ledger.AccountBlock),
		ConfirmedBlockMap: make(map[types.Hash]map[types.Hash]struct{}),
		BalanceMap:        make(map[types.Hash]*big.Int),
		LogListMap:        make(map[types.Hash]ledger.VmLogList),
		KvSetMap:          make(map[types.Hash]map[string][]byte),
		UnconfirmedBlocks: make(map[types.Hash]struct{}),
	}
	accounts[toAddr] = toAccount
	vmBlock, createBlockErr := account.CreateSendBlock(toAccount, cTxOptions)
	if createBlockErr != nil {
		b.Fatalf("create send create failed, err: %v", createBlockErr)
	}
	// insert vm block
	account.InsertBlock(vmBlock, accounts)

	// insert vm block to chain
	if err := chainInstance.InsertAccountBlock(vmBlock); err != nil {
		panic(err)
	}
	for i := 0; i < 100; i++ {
		snapshotBlock, _, err := InsertSnapshotBlock(chainInstance, true, nil)
		if err != nil {
			panic(err)
		}

		// snapshot
		Snapshot(accounts, snapshotBlock)
	}
	chainInstance.flusher.Flush()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := chainInstance.GetSnapshotBlockByContractMeta(toAddr, vmBlock.AccountBlock.Hash)
		if err != nil {
			b.Fatalf("db get seed failed, err: %v", err)
		}
	}
}

// storage size consumed for sstore instruction
func TestCalcStorageSize(t *testing.T) {
	addr, _ := types.HexToAddress("vite_0000000000000000000000000000000000000003f6af7459b9")
	key := types.DataHash(big.NewInt(1).Bytes())
	skey := chain_utils.CreateStorageValueKey(&addr, key.Bytes())
	balanceSkey := chain_utils.CreateBalanceKey(addr, ledger.ViteTokenId)
	fmt.Printf("storage key size(byte): %v, value size(byte): 32, total size(byte):%v\n", len(skey), len(skey)+32)
	fmt.Printf("balance storage key size(byte): %v, value size(byte): 32, total size(byte):%v\n", len(balanceSkey), len(balanceSkey)+32)
}

func TestPrintBlockSize(t *testing.T) {
	printBlockSize("call",
		makeSendBlock(testAddr, testAddr, nil, big1e18, big0),
		makeReceiveBlock(testAddr))

	printBlockSize("pledge",
		makePledgeSendBlock(testAddr),
		makeReceiveBlock(types.AddressPledge))
	printBlockSize("cancelPledge",
		makeCancelPledgeSendBlock(testAddr),
		makeReceiveBlock(types.AddressPledge))

	printBlockSize("register",
		makeRegisterSendBlock(testAddr),
		makeReceiveBlock(types.AddressConsensusGroup))
	printBlockSize("cancelRegister",
		makeCancelRegisterSendBlock(testAddr),
		makeReceiveBlock(types.AddressConsensusGroup))
	printBlockSize("reward",
		makeRewardSendBlock(testAddr),
		makeReceiveBlock(types.AddressConsensusGroup))
	printBlockSize("updateRegistration",
		makeUpdateRegistrationBlock(testAddr),
		makeReceiveBlock(types.AddressConsensusGroup))

	printBlockSize("vote",
		makeVoteBlock(testAddr),
		makeReceiveBlock(types.AddressConsensusGroup))
	printBlockSize("cancelVote",
		makeCancelVoteBlock(testAddr),
		makeReceiveBlock(types.AddressConsensusGroup))

	printBlockSize("mint",
		makeMintBlock(testAddr),
		makeReceiveBlock(types.AddressMintage))
	printBlockSize("issue",
		makeIssueBlock(testAddr),
		makeReceiveBlock(types.AddressMintage))
	printBlockSize("burn",
		makeBurnBlock(testAddr),
		makeReceiveBlock(types.AddressMintage))
	printBlockSize("transferOwner",
		makeTransferOwnerBlock(testAddr),
		makeReceiveBlock(types.AddressMintage))
	printBlockSize("changeTokenType",
		makeChangeTokenTypeBlock(testAddr),
		makeReceiveBlock(types.AddressMintage))
}

func TestPrintCreateContractBlockSize(t *testing.T) {
	sendCreateBlock := makeCreateSendBlock()
	receiveCreateBlock := makeReceiveBlock(sendCreateBlock.ToAddress)
	receiveCreateBlock.Data, _ = hex.DecodeString("000000000000000000000000000000000000000000000000000000000000000000")
	logHash, _ := types.HexToHash("71b6bde5d7b9f1eeb8e38d78e5e7b67fdf3cdbbf38d9c71ce5b79cefd6dfd9cd1cebcf34f1ff5ad9e75cd1a79cdb773c")
	receiveCreateBlock.LogHash = &logHash
	printReceiveBlock("create", receiveCreateBlock)
	meta := &ledger.ContractMeta{
		Gid:                types.DELEGATE_GID,
		SendConfirmedTimes: 12,
		SeedConfirmedTimes: 10,
		CreateBlockHash:    sendCreateBlock.Hash,
		QuotaRatio:         10,
	}
	metaSize := len(chain_utils.CreateContractMetaKey(sendCreateBlock.ToAddress)) + len(meta.Serialize())
	fmt.Printf("meta size: %v\n", metaSize)
}

func printBlockSize(name string, sendBlock, receiveBlock *ledger.AccountBlock) {
	printSendBlock(name, sendBlock)
	initVMEnvironment()
	chainInstance, _, _ := SetUp(1, 100, 1)
	defer TearDown(chainInstance)
	sb := createSnapshotBlock(chainInstance, createSbOption{SnapshotAll: false, Seed: 1, PublicKey: ed25519.PublicKey(randomByte32())})
	ts := time.Now()
	sb.Timestamp = &ts
	sb.Hash = sb.ComputeHash()
	_, err := chainInstance.InsertSnapshotBlock(sb)
	if err != nil {
		panic(err)
	}

	prevBlock, err := chainInstance.GetLatestAccountBlock(receiveBlock.AccountAddress)
	if err != nil {
		panic(err)
	}
	if prevBlock != nil {
		receiveBlock.PrevHash = prevBlock.Hash
	}
	globalStatus := generator.NewVMGlobalStatus(chainInstance, chainInstance.GetLatestSnapshotBlock(), types.Hash{})
	cr := util.NewVmConsensusReader(mockConsensus.SBPReader())
	db, err := vm_db.NewVmDb(chainInstance, &receiveBlock.AccountAddress, &chainInstance.GetLatestSnapshotBlock().Hash, &receiveBlock.PrevHash)
	if err != nil {
		panic(err)
	}
	receiveVMBlock, _, err := vm.NewVM(cr).RunV2(db, receiveBlock, sendBlock, globalStatus)
	if err != nil {
		panic(err)
	}
	printReceiveBlock(name, receiveVMBlock.AccountBlock)
	storageSize := len(receiveVMBlock.VmDb.GetUnsavedBalanceMap()) * 64
	for _, kvmap := range receiveVMBlock.VmDb.GetUnsavedStorage() {
		storageSize = storageSize + len(chain_utils.CreateStorageValueKey(&receiveBlock.AccountAddress, kvmap[0])) + len(kvmap[1])
	}
	fmt.Printf("storage size: receive %v block, %v\n", name, storageSize)
}

func printSendBlock(name string, sendBlock *ledger.AccountBlock) {
	bs, _ := sendBlock.Serialize()
	netB := &message.NewAccountBlock{Block: sendBlock, TTL: 32}
	netBs, _ := netB.Serialize()
	fmt.Printf("blocksize: send %v block, chain block size %v, net block size %v\n", name, len(bs), len(netBs))
}

func printReceiveBlock(name string, receiveBlock *ledger.AccountBlock) {
	if len(receiveBlock.SendBlockList) == 0 {
		bs, _ := receiveBlock.Serialize()
		netB := &message.NewAccountBlock{Block: receiveBlock, TTL: 32}
		netBs, _ := netB.Serialize()
		fmt.Printf("blocksize: receive %v block without send block, chain block size %v, net block size %v\n", name, len(bs), len(netBs))
		return
	}
	for i, _ := range receiveBlock.SendBlockList {
		if receiveBlock.SendBlockList[i].Fee == nil {
			receiveBlock.SendBlockList[i].Fee = big0
		}
		receiveBlock.SendBlockList[i].Hash, _ = types.HexToHash("8f85502f81fc544cb6700ad9ecc44f3eace065ae8e34d2269d7ff8d7c94ac920")
	}
	rsbs, _ := receiveBlock.Serialize()
	netRb := &message.NewAccountBlock{Block: receiveBlock, TTL: 32}
	netRbs, _ := netRb.Serialize()
	fmt.Printf("blocksize: receive %v block with send block, chain block size %v, net block size %v\n", name, len(rsbs), len(netRbs))
}
