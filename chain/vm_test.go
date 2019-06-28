package chain

import (
	rand2 "crypto/rand"
	"fmt"
	"github.com/vitelabs/go-vite/chain/test_tools"
	"github.com/vitelabs/go-vite/chain/utils"
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/generator"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm"
	"github.com/vitelabs/go-vite/vm/contracts/abi"
	"github.com/vitelabs/go-vite/vm/util"
	"github.com/vitelabs/go-vite/vm_db"
	"math/big"
	"math/rand"
	"testing"
	"time"
)

func initBuiltinContractEnvironment() {
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

	big1e18 = big.NewInt(1e18)
	big0    = big.NewInt(0)
)

func BenchmarkPledgeSend(b *testing.B) {
	sendBlock := makePledgeSendBlock(testAddr)
	benchmarkSend(b, sendBlock)
}
func BenchmarkPledgeReceive(b *testing.B) {
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

func BenchmarkCancelPledgeSend(b *testing.B) {
	sendBlock := makeCancelPledgeSendBlock(testAddr)
	benchmarkSend(b, sendBlock)
}
func BenchmarkCancelPledgeReceive(b *testing.B) {
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

func BenchmarkRegisterSend(b *testing.B) {
	sendBlock := makeRegisterSendBlock(testAddr)
	benchmarkSend(b, sendBlock)
}
func BenchmarkRegisterReceive(b *testing.B) {
	sendBlock := makeRegisterSendBlock(testAddr)
	receiveBlock := makeReceiveBlock(types.AddressConsensusGroup)
	benchmarkReceive(b, sendBlock, receiveBlock)
}
func makeRegisterSendBlock(addr types.Address) *ledger.AccountBlock {
	data, err := abi.ABIConsensusGroup.PackMethod(abi.MethodNameRegister, types.SNAPSHOT_GID, "testnode", addr)
	if err != nil {
		panic(err)
	}
	return makeSendBlock(addr, types.AddressConsensusGroup, data, registerPledgeAmount, big0)
}

func BenchmarkCancelRegisterSend(b *testing.B) {
	sendBlock := makeCancelRegisterSendBlock(testAddr)
	benchmarkSend(b, sendBlock)
}
func BenchmarkCancelRegisterReceive(b *testing.B) {
	sendBlock := makeCancelRegisterSendBlock(testAddr)
	receiveBlock := makeReceiveBlock(types.AddressConsensusGroup)
	benchmarkReceive(b, sendBlock, receiveBlock)
}
func makeCancelRegisterSendBlock(addr types.Address) *ledger.AccountBlock {
	data, err := abi.ABIConsensusGroup.PackMethod(abi.MethodNameCancelRegister, types.SNAPSHOT_GID, "s1")
	if err != nil {
		panic(err)
	}
	return makeSendBlock(addr, types.AddressConsensusGroup, data, big0, big0)
}

func BenchmarkRewardSend(b *testing.B) {
	sendBlock := makeRewardSendBlock(testAddr)
	benchmarkSend(b, sendBlock)
}
func BenchmarkRewardReceive(b *testing.B) {
	sendBlock := makeRewardSendBlock(testAddr)
	receiveBlock := makeReceiveBlock(types.AddressConsensusGroup)
	benchmarkReceive(b, sendBlock, receiveBlock)
}
func makeRewardSendBlock(addr types.Address) *ledger.AccountBlock {
	data, err := abi.ABIConsensusGroup.PackMethod(abi.MethodNameReward, types.SNAPSHOT_GID, "s1", addr)
	if err != nil {
		panic(err)
	}
	return makeSendBlock(addr, types.AddressConsensusGroup, data, big0, big0)
}

func BenchmarkUpdateRegistrationSend(b *testing.B) {
	sendBlock := makeUpdateRegistrationBlock(testAddr)
	benchmarkSend(b, sendBlock)
}
func BenchmarkUpdateRegistrationReceive(b *testing.B) {
	sendBlock := makeUpdateRegistrationBlock(testAddr)
	receiveBlock := makeReceiveBlock(types.AddressConsensusGroup)
	benchmarkReceive(b, sendBlock, receiveBlock)
}
func makeUpdateRegistrationBlock(addr types.Address) *ledger.AccountBlock {
	nodeAddr, _, err := types.CreateAddress()
	if err != nil {
		panic(err)
	}
	data, err := abi.ABIConsensusGroup.PackMethod(abi.MethodNameUpdateRegistration, types.SNAPSHOT_GID, "s1", nodeAddr)
	if err != nil {
		panic(err)
	}
	return makeSendBlock(addr, types.AddressConsensusGroup, data, big0, big0)
}

func BenchmarkVoteSend(b *testing.B) {
	sendBlock := makeVoteBlock(testAddr)
	benchmarkSend(b, sendBlock)
}
func BenchmarkVoteReceive(b *testing.B) {
	sendBlock := makeVoteBlock(testAddr)
	receiveBlock := makeReceiveBlock(types.AddressConsensusGroup)
	benchmarkReceive(b, sendBlock, receiveBlock)
}
func makeVoteBlock(addr types.Address) *ledger.AccountBlock {
	data, err := abi.ABIConsensusGroup.PackMethod(abi.MethodNameVote, types.SNAPSHOT_GID, "s1")
	if err != nil {
		panic(err)
	}
	return makeSendBlock(addr, types.AddressConsensusGroup, data, big0, big0)
}

func BenchmarkCancelVoteSend(b *testing.B) {
	sendBlock := makeCancelVoteBlock(testAddr)
	benchmarkSend(b, sendBlock)
}
func BenchmarkCancelVoteReceive(b *testing.B) {
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

func BenchmarkMintSend(b *testing.B) {
	sendBlock := makeMintBlock(testAddr)
	benchmarkSend(b, sendBlock)
}
func BenchmarkMintReceive(b *testing.B) {
	sendBlock := makeMintBlock(testAddr)
	receiveBlock := makeReceiveBlock(types.AddressMintage)
	benchmarkReceive(b, sendBlock, receiveBlock)
}
func makeMintBlock(addr types.Address) *ledger.AccountBlock {
	data, err := abi.ABIMintage.PackMethod(abi.MethodNameMint, true, "new test token", "NTT", big.NewInt(1e4), uint8(2), big.NewInt(1e5), false)
	if err != nil {
		panic(err)
	}
	return makeSendBlock(addr, types.AddressMintage, data, big0, mintFee)
}

func BenchmarkIssueSend(b *testing.B) {
	sendBlock := makeIssueBlock(testAddr)
	benchmarkSend(b, sendBlock)
}
func BenchmarkIssueReceive(b *testing.B) {
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

func BenchmarkBurnSend(b *testing.B) {
	sendBlock := makeBurnBlock(testAddr)
	benchmarkSend(b, sendBlock)
}
func BenchmarkBurnReceive(b *testing.B) {
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

func BenchmarkTransferOwnerSend(b *testing.B) {
	sendBlock := makeTransferOwnerBlock(testAddr)
	benchmarkSend(b, sendBlock)
}
func BenchmarkTransferOwnerReceive(b *testing.B) {
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

func BenchmarkChangeTokenTypeSend(b *testing.B) {
	sendBlock := makeChangeTokenTypeBlock(testAddr)
	benchmarkSend(b, sendBlock)
}
func BenchmarkChangeTokenTypeReceive(b *testing.B) {
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
	initBuiltinContractEnvironment()
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
	initBuiltinContractEnvironment()
	chainInstance, _, _ := SetUp(0, 0, 0)
	defer TearDown(chainInstance)
	b.ResetTimer()
	prevBlock, err := chainInstance.GetLatestAccountBlock(receiveBlock.AccountAddress)
	if err != nil {
		panic(err)
	}
	if prevBlock != nil {
		receiveBlock.PrevHash = prevBlock.Hash
	}
	globalStatus := generator.NewVMGlobalStatus(chainInstance, chainInstance.GetLatestSnapshotBlock(), types.Hash{})
	c := &test_tools.MockConsensus{
		Cr: &test_tools.MockConsensusReader{
			DayTimeIndex: test_tools.MockTimeIndex{
				GenesisTime: *chainInstance.GetGenesisSnapshotBlock().Timestamp,
				Interval:    3,
			},
		},
	}
	cr := util.NewVmConsensusReader(c.SBPReader())
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
	return &ledger.AccountBlock{
		BlockType:      ledger.BlockTypeSendCall,
		AccountAddress: fromAddr,
		ToAddress:      toAddr,
		PrevHash:       types.Hash{},
		Amount:         amount,
		Fee:            fee,
		TokenId:        ledger.ViteTokenId,
		Data:           data,
	}
}

func makeReceiveBlock(addr types.Address) *ledger.AccountBlock {
	return &ledger.AccountBlock{
		BlockType:      ledger.BlockTypeReceive,
		AccountAddress: addr,
	}
}

func BenchmarkSetValue(b *testing.B) {
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

func BenchmarkGetValue(b *testing.B) {
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
func BenchmarkGetBalance(b *testing.B) {
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

func BenchmarkGetSeed(b *testing.B) {
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

func BenchmarkCheckConfirmTime(b *testing.B) {
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

func TestCalcStorageSize(t *testing.T) {
	addr, _ := types.HexToAddress("vite_0000000000000000000000000000000000000003f6af7459b9")
	key := types.DataHash(big.NewInt(1).Bytes())
	skey := chain_utils.CreateStorageValueKey(&addr, key.Bytes())
	balanceSkey := chain_utils.CreateBalanceKey(addr, ledger.ViteTokenId)
	fmt.Printf("storage key size(byte): %v, value size(byte): 32, total size(byte):%v\n", len(skey), len(skey)+32)
	fmt.Printf("balance storage key size(byte): %v, value size(byte): 32, total size(byte):%v\n", len(balanceSkey), len(balanceSkey)+32)
}

func TestTest(t *testing.T) {
	tid := types.CreateTokenTypeId([]byte{1, 2, 3})
	fmt.Println(tid.String())
}
