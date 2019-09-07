package vm

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/common/fork"
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/contracts/abi"
	"github.com/vitelabs/go-vite/vm/util"
	"io/ioutil"
	"math/big"
	"os"
	"strconv"
	"testing"
	"time"
)

func init() {
	InitVMConfig(false, false, false, false, common.HomeDir())
	initFork()
}

func initFork() {
	fork.SetForkPoints(&config.ForkPoints{
		SeedFork: &config.ForkPoint{Height: 100, Version: 1},
		DexFork:  &config.ForkPoint{Height: 200, Version: 1},
		StemFork: &config.ForkPoint{Height: 300, Version: 1}})
}

func TestVmRun(t *testing.T) {
	// prepare db
	viteTotalSupply := new(big.Int).Mul(big.NewInt(1e9), util.AttovPerVite)
	db, addr1, _, hash12, snapshot, _ := prepareDb(viteTotalSupply)

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
	data13, _ := hex.DecodeString("0000000000000000000201010b608060405260858060116000396000f300608060405260043610603e5763ffffffff7c0100000000000000000000000000000000000000000000000000000000600035041663f021ab8f81146043575b600080fd5b604c600435604e565b005b6000805490910190555600a165627a7a72305820b8d8d60a46c6ac6569047b17b012aa1ea458271f9bc8078ef0cff9208999d0900029")
	hash13 := types.DataHash([]byte{1, 3})
	block13 := &ledger.AccountBlock{
		Height:         3,
		AccountAddress: addr1,
		BlockType:      ledger.BlockTypeSendCreate,
		PrevHash:       hash12,
		Amount:         big.NewInt(1e18),
		Fee:            big.NewInt(0),
		TokenId:        ledger.ViteTokenId,
		Data:           data13,
		Hash:           hash13,
	}
	vm := NewVM(nil)
	//vm.Debug = true
	db.addr = addr1
	sendCreateBlock, isRetry, err := vm.RunV2(db, block13, nil, nil)
	balance1.Sub(balance1, block13.Amount)
	balance1.Sub(balance1, createContractFee)
	if sendCreateBlock == nil || isRetry ||
		err != nil ||
		sendCreateBlock.AccountBlock.Quota != 32084 ||
		sendCreateBlock.AccountBlock.Quota != sendCreateBlock.AccountBlock.QuotaUsed ||
		sendCreateBlock.AccountBlock.Fee.Cmp(createContractFee) != 0 ||
		db.balanceMap[addr1][ledger.ViteTokenId].Cmp(balance1) != 0 {
		t.Fatalf("send create transaction error")
	}
	db.accountBlockMap[addr1][hash13] = sendCreateBlock.AccountBlock

	// receive create
	addr2 := sendCreateBlock.AccountBlock.ToAddress
	db.storageMap[types.AddressPledge][ToKey(abi.GetPledgeBeneficialKey(addr2))], _ = abi.ABIPledge.PackVariable(abi.VariableNamePledgeBeneficial, new(big.Int).Mul(big.NewInt(1e9), big.NewInt(1e18)))
	balance2 := big.NewInt(0)

	hash21 := types.DataHash([]byte{2, 1})
	block21 := &ledger.AccountBlock{
		Height:         1,
		AccountAddress: addr2,
		FromBlockHash:  hash13,
		BlockType:      ledger.BlockTypeReceive,
		Hash:           hash21,
	}
	vm = NewVM(nil)
	//vm.Debug = true
	db.addr = addr2
	receiveCreateBlockList, isRetry, err := vm.RunV2(db, block21, sendCreateBlock.AccountBlock, nil)
	balance2.Add(balance2, block13.Amount)
	if receiveCreateBlockList == nil ||
		len(receiveCreateBlockList.AccountBlock.SendBlockList) != 0 || isRetry || err != nil ||
		receiveCreateBlockList.AccountBlock.Quota != 0 ||
		receiveCreateBlockList.AccountBlock.Quota != receiveCreateBlockList.AccountBlock.QuotaUsed ||
		db.contractMetaMap[addr2] == nil ||
		db.contractMetaMap[addr2].Gid != types.DELEGATE_GID ||
		db.balanceMap[addr2][ledger.ViteTokenId].Cmp(balance2) != 0 {
		t.Fatalf("receive create transaction error")
	}
	db.accountBlockMap[addr2] = make(map[types.Hash]*ledger.AccountBlock)
	db.accountBlockMap[addr2][hash21] = receiveCreateBlockList.AccountBlock

	// send call
	data14, _ := hex.DecodeString("f021ab8f0000000000000000000000000000000000000000000000000000000000000005")
	hash14 := types.DataHash([]byte{1, 4})
	block14 := &ledger.AccountBlock{
		Height:         4,
		AccountAddress: addr1,
		ToAddress:      addr2,
		BlockType:      ledger.BlockTypeSendCall,
		Fee:            big.NewInt(0),
		PrevHash:       hash13,
		Amount:         big.NewInt(1e18),
		TokenId:        ledger.ViteTokenId,
		Data:           data14,
		Hash:           hash14,
	}
	vm = NewVM(nil)
	//vm.Debug = true
	db.addr = addr1
	sendCallBlock, isRetry, err := vm.RunV2(db, block14, nil, nil)
	balance1.Sub(balance1, block14.Amount)
	if sendCallBlock == nil ||
		len(sendCallBlock.AccountBlock.SendBlockList) != 0 || isRetry || err != nil ||
		sendCallBlock.AccountBlock.Quota != 25792 ||
		sendCallBlock.AccountBlock.Quota != sendCallBlock.AccountBlock.QuotaUsed ||
		db.balanceMap[addr1][ledger.ViteTokenId].Cmp(balance1) != 0 {
		t.Fatalf("send call transaction error")
	}
	db.accountBlockMap[addr1][hash14] = sendCallBlock.AccountBlock

	snapshot = &ledger.SnapshotBlock{Height: 3, Timestamp: snapshot.Timestamp, Hash: types.DataHash([]byte{10, 3})}
	db.snapshotBlockList = append(db.snapshotBlockList, snapshot)

	// receive call
	hash22 := types.DataHash([]byte{2, 2})
	block22 := &ledger.AccountBlock{
		Height:         2,
		AccountAddress: addr2,
		FromBlockHash:  hash14,
		PrevHash:       hash21,
		BlockType:      ledger.BlockTypeReceive,
		Hash:           hash22,
	}
	vm = NewVM(nil)
	//vm.Debug = true
	db.addr = addr2
	receiveCallBlock, isRetry, err := vm.RunV2(db, block22, sendCallBlock.AccountBlock, nil)
	balance2.Add(balance2, block14.Amount)
	if receiveCallBlock == nil ||
		len(receiveCallBlock.AccountBlock.SendBlockList) != 0 || isRetry || err != nil ||
		receiveCallBlock.AccountBlock.Quota != 41530 ||
		receiveCallBlock.AccountBlock.Quota != receiveCallBlock.AccountBlock.QuotaUsed ||
		db.balanceMap[addr2][ledger.ViteTokenId].Cmp(big.NewInt(2e18)) != 0 {
		t.Fatalf("receive call transaction error")
	}
	db.accountBlockMap[addr2][hash22] = receiveCallBlock.AccountBlock

	// send call error, insufficient balance
	data15, _ := hex.DecodeString("f021ab8f0000000000000000000000000000000000000000000000000000000000000005")
	hash15 := types.DataHash([]byte{1, 5})
	block15 := &ledger.AccountBlock{
		Height:         5,
		AccountAddress: addr1,
		ToAddress:      addr2,
		BlockType:      ledger.BlockTypeSendCall,
		Fee:            big.NewInt(0),
		PrevHash:       hash14,
		Amount:         viteTotalSupply,
		TokenId:        ledger.ViteTokenId,
		Data:           data15,
		Hash:           hash15,
	}
	vm = NewVM(nil)
	//vm.Debug = true
	db.addr = addr1
	sendCallBlock2, isRetry, err := vm.RunV2(db, block15, nil, nil)
	if sendCallBlock2 != nil || err != util.ErrInsufficientBalance {
		t.Fatalf("send call transaction 2 error")
	}
	// receive call error, execution revert
	data15, _ = hex.DecodeString("")
	block15 = &ledger.AccountBlock{
		Height:         5,
		AccountAddress: addr1,
		ToAddress:      addr2,
		BlockType:      ledger.BlockTypeSendCall,
		Fee:            big.NewInt(0),
		PrevHash:       hash14,
		Amount:         big.NewInt(50),
		TokenId:        ledger.ViteTokenId,
		Data:           data15,
		Hash:           hash15,
	}
	vm = NewVM(nil)
	//vm.Debug = true
	db.addr = addr1
	sendCallBlock2, isRetry, err = vm.RunV2(db, block15, nil, nil)
	db.accountBlockMap[addr1][hash15] = sendCallBlock2.AccountBlock
	// receive call
	hash23 := types.DataHash([]byte{2, 3})
	block23 := &ledger.AccountBlock{
		Height:         3,
		AccountAddress: addr2,
		FromBlockHash:  hash15,
		PrevHash:       hash22,
		BlockType:      ledger.BlockTypeReceive,
	}
	vm = NewVM(nil)
	//vm.Debug = true
	db.addr = addr2
	receiveCallBlock2, isRetry, err := vm.RunV2(db, block23, sendCallBlock2.AccountBlock, nil)
	if receiveCallBlock2 == nil ||
		len(receiveCallBlock2.AccountBlock.SendBlockList) != 1 || isRetry || err != util.ErrExecutionReverted ||
		receiveCallBlock2.AccountBlock.Quota != 21246 ||
		receiveCallBlock2.AccountBlock.Quota != receiveCallBlock2.AccountBlock.QuotaUsed ||
		len(receiveCallBlock2.AccountBlock.Data) != 33 ||
		receiveCallBlock2.AccountBlock.Data[32] != 1 ||
		receiveCallBlock2.AccountBlock.SendBlockList[0].BlockType != ledger.BlockTypeSendRefund ||
		receiveCallBlock2.AccountBlock.SendBlockList[0].AccountAddress != addr2 ||
		receiveCallBlock2.AccountBlock.SendBlockList[0].ToAddress != addr1 ||
		receiveCallBlock2.AccountBlock.SendBlockList[0].Amount.Cmp(block15.Amount) != 0 ||
		receiveCallBlock2.AccountBlock.SendBlockList[0].TokenId != ledger.ViteTokenId ||
		receiveCallBlock2.AccountBlock.SendBlockList[0].Fee.Sign() != 0 ||
		len(receiveCallBlock2.AccountBlock.SendBlockList[0].Data) != 0 {
		t.Fatalf("receive call transaction error")
	}
	db.accountBlockMap[addr2][hash23] = receiveCallBlock2.AccountBlock
}

/*func TestDelegateCall(t *testing.T) {
	// prepare db, add account1, add account2 with code, add account3 with code
	db := newNoDatabase()
	// code1 return 1+2
	addr1, _, _ := types.CreateAddress()
	code1 := []byte{1, byte(PUSH1), 1, byte(PUSH1), 2, byte(ADD), byte(PUSH1), 32, byte(DUP1), byte(SWAP2), byte(SWAP1), byte(MSTORE), byte(PUSH1), 32, byte(SWAP1), byte(RETURN)}
	db.codeMap = make(map[types.Address][]byte)
	db.codeMap[addr1] = code1

	addr2, _, _ := types.CreateAddress()
	code2 := helper.JoinBytes([]byte{1, byte(PUSH1), 32, byte(PUSH1), 0, byte(PUSH1), 0, byte(PUSH1), 0, byte(PUSH20)}, addr1.Bytes(), []byte{byte(DELEGATECALL), byte(PUSH1), 32, byte(PUSH1), 0, byte(RETURN)})
	db.codeMap[addr2] = code2

	vm := NewVM(nil)
	vm.globalStatus = &util.GlobalStatus{0, &ledger.SnapshotBlock{}}
	vm.i = newInterpreter(1, false)
	//vm.Debug = true
	sendCallBlock := ledger.AccountBlock{
		AccountAddress: addr1,
		ToAddress:      addr2,
		BlockType:      ledger.BlockTypeSendCall,
		Amount:         big.NewInt(10),
		Fee:            big.NewInt(0),
		TokenId:        ledger.ViteTokenId,
	}
	receiveCallBlock := &ledger.AccountBlock{
		AccountAddress: addr2,
		BlockType:      ledger.BlockTypeReceive,
	}
	c := newContract(
		receiveCallBlock,
		db,
		&sendCallBlock,
		nil,
		1000000,
		0)
	c.setCallCode(addr2, code2[1:])
	ret, err := c.run(vm)
	if err != nil || !bytes.Equal(ret, helper.LeftPadBytes([]byte{3}, 32)) {
		t.Fatalf("delegate call error")
	}
}*/

func TestCall(t *testing.T) {
	// prepare db, add account1, add account2 with code, add account3 with code
	viteTotalSupply := new(big.Int).Mul(big.NewInt(1e9), util.AttovPerVite)
	db, addr1, _, hash12, _, _ := prepareDb(viteTotalSupply)

	// code2 calls addr1 with data=100 and amount=10
	addr2 := types.Address{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1}
	code2 := []byte{
		1,
		byte(PUSH1), 32, byte(PUSH1), 100, byte(PUSH1), 0, byte(DUP1), byte(SWAP2), byte(SWAP1), byte(MSTORE),
		byte(PUSH1), 10, byte(PUSH10), 'V', 'I', 'T', 'E', ' ', 'T', 'O', 'K', 'E', 'N', byte(PUSH21), 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 2, 1, byte(CALL)}
	db.codeMap[addr2] = code2
	db.contractMetaMap[addr2] = &ledger.ContractMeta{Gid: types.DELEGATE_GID, SendConfirmedTimes: 1, QuotaRatio: 10}

	// code3 return amount+data
	addr3 := types.Address{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 2, 1}
	code3 := []byte{1, byte(CALLVALUE), byte(PUSH1), 0, byte(CALLDATALOAD), byte(ADD), byte(PUSH1), 32, byte(DUP1), byte(SWAP2), byte(SWAP1), byte(MSTORE), byte(PUSH1), 32, byte(SWAP1), byte(RETURN)}
	db.codeMap[addr3] = code3
	db.contractMetaMap[addr3] = &ledger.ContractMeta{Gid: types.DELEGATE_GID, SendConfirmedTimes: 2, QuotaRatio: 10}

	db.accountBlockMap[addr2] = make(map[types.Hash]*ledger.AccountBlock)
	db.storageMap[types.AddressPledge][ToKey(abi.GetPledgeBeneficialKey(addr2))], _ = abi.ABIPledge.PackVariable(abi.VariableNamePledgeBeneficial, new(big.Int).Mul(big.NewInt(1e9), big.NewInt(1e18)))

	db.accountBlockMap[addr3] = make(map[types.Hash]*ledger.AccountBlock)
	db.storageMap[types.AddressPledge][ToKey(abi.GetPledgeBeneficialKey(addr3))], _ = abi.ABIPledge.PackVariable(abi.VariableNamePledgeBeneficial, new(big.Int).Mul(big.NewInt(1e9), big.NewInt(1e18)))

	vm := NewVM(nil)
	//vm.Debug = true
	// call contract
	balance1 := db.balanceMap[addr1][ledger.ViteTokenId]
	hash13 := types.DataHash([]byte{1, 3})
	block13 := &ledger.AccountBlock{
		Height:         3,
		AccountAddress: addr1,
		BlockType:      ledger.BlockTypeSendCall,
		PrevHash:       hash12,
		Amount:         big.NewInt(10),
		Fee:            big.NewInt(0),
		TokenId:        ledger.ViteTokenId,
		Hash:           hash13,
		ToAddress:      addr2,
		Difficulty:     big.NewInt(67108863),
		Nonce:          []byte{1},
		Data:           []byte{1},
	}
	db.addr = addr1
	sendCallBlock, isRetry, err := vm.RunV2(db, block13, nil, nil)
	balance1.Sub(balance1, block13.Amount)
	if sendCallBlock == nil ||
		len(sendCallBlock.AccountBlock.SendBlockList) != 0 ||
		isRetry ||
		err != nil ||
		sendCallBlock.AccountBlock.Quota != 68 ||
		sendCallBlock.AccountBlock.QuotaUsed != 21068 ||
		db.balanceMap[addr1][ledger.ViteTokenId].Cmp(balance1) != 0 {
		t.Fatalf("send call transaction error")
	}
	db.accountBlockMap[addr1][hash13] = sendCallBlock.AccountBlock

	// contract2 receive call
	balance2 := big.NewInt(0)
	hash21 := types.DataHash([]byte{2, 1})
	block21 := &ledger.AccountBlock{
		Height:         1,
		AccountAddress: addr2,
		FromBlockHash:  hash13,
		BlockType:      ledger.BlockTypeReceive,
		Hash:           hash21,
	}
	vm = NewVM(nil)
	//vm.Debug = true
	db.addr = addr2
	receiveCallBlock, isRetry, err := vm.RunV2(db, block21, sendCallBlock.AccountBlock, nil)
	if receiveCallBlock == nil ||
		len(receiveCallBlock.AccountBlock.SendBlockList) != 1 || isRetry || err != nil ||
		receiveCallBlock.AccountBlock.Quota != 34409 ||
		receiveCallBlock.AccountBlock.Quota != receiveCallBlock.AccountBlock.QuotaUsed ||
		len(receiveCallBlock.AccountBlock.Data) != 33 ||
		receiveCallBlock.AccountBlock.Data[32] != 0 ||
		receiveCallBlock.AccountBlock.SendBlockList[0].BlockType != ledger.BlockTypeSendCall ||
		receiveCallBlock.AccountBlock.SendBlockList[0].AccountAddress != addr2 ||
		receiveCallBlock.AccountBlock.SendBlockList[0].ToAddress != addr3 ||
		receiveCallBlock.AccountBlock.SendBlockList[0].Amount.Cmp(big.NewInt(10)) != 0 ||
		receiveCallBlock.AccountBlock.SendBlockList[0].Fee.Sign() != 0 ||
		!bytes.Equal(receiveCallBlock.AccountBlock.SendBlockList[0].Data, helper.LeftPadBytes([]byte{100}, 32)) ||
		db.balanceMap[addr2][ledger.ViteTokenId].Cmp(balance2) != 0 {
		t.Fatalf("contract receive call transaction error")
	}
	db.accountBlockMap[addr2][hash21] = receiveCallBlock.AccountBlock
	hash22 := types.DataHash([]byte{2, 2})
	receiveCallBlock.AccountBlock.SendBlockList[0].PrevHash = hash21
	receiveCallBlock.AccountBlock.SendBlockList[0].Hash = hash22
	db.accountBlockMap[addr2][hash22] = receiveCallBlock.AccountBlock.SendBlockList[0]

	// contract3 receive call
	balance3 := new(big.Int).Set(block13.Amount)
	hash31 := types.DataHash([]byte{3, 1})
	block31 := &ledger.AccountBlock{
		Height:         1,
		AccountAddress: addr3,
		FromBlockHash:  hash22,
		BlockType:      ledger.BlockTypeReceive,
		Hash:           hash31,
	}
	vm = NewVM(nil)
	//vm.Debug = true
	db.addr = addr3
	receiveCallBlock2, isRetry, err := vm.RunV2(db, block31, receiveCallBlock.AccountBlock.SendBlockList[0], nil)
	if receiveCallBlock2 == nil ||
		len(receiveCallBlock2.AccountBlock.SendBlockList) != 0 || isRetry || err != nil ||
		receiveCallBlock2.AccountBlock.Quota != 21438 ||
		receiveCallBlock2.AccountBlock.Quota != receiveCallBlock2.AccountBlock.QuotaUsed ||
		len(receiveCallBlock2.AccountBlock.Data) != 33 ||
		receiveCallBlock2.AccountBlock.Data[32] != 0 ||
		db.balanceMap[addr3][ledger.ViteTokenId].Cmp(balance3) != 0 {
		t.Fatalf("contract receive call transaction error")
	}
	db.accountBlockMap[addr3][hash31] = receiveCallBlock2.AccountBlock
}

func BenchmarkVMTransfer(b *testing.B) {
	viteTotalSupply := new(big.Int).Mul(big.NewInt(1e9), util.AttovPerVite)
	db, addr1, _, hash12, _, _ := prepareDb(viteTotalSupply)

	// send call
	b.ResetTimer()
	prevHash := hash12
	addr2, _, _ := types.CreateAddress()
	amount := big.NewInt(1)
	db.addr = addr1
	for i := 3; i < b.N+3; i++ {
		hashi := types.DataHash([]byte(strconv.Itoa(i)))
		blocki := &ledger.AccountBlock{
			Height:         uint64(i),
			AccountAddress: addr1,
			ToAddress:      addr2,
			BlockType:      ledger.BlockTypeSendCall,
			PrevHash:       prevHash,
			Amount:         amount,
			TokenId:        ledger.ViteTokenId,
			Hash:           hashi,
		}
		vm := NewVM(nil)
		sendCallBlock, _, err := vm.RunV2(db, blocki, nil, nil)
		if err != nil {
			b.Fatal(err)
		}
		db.accountBlockMap[addr1][hashi] = sendCallBlock.AccountBlock
		prevHash = hashi
	}
}

func TestVmForTest(t *testing.T) {
	InitVMConfig(true, true, true, false, "")
	db, _, _, _, _, _ := prepareDb(big.NewInt(0))

	addr1, _, _ := types.CreateAddress()
	block11 := &ledger.AccountBlock{
		Height:         1,
		AccountAddress: addr1,
		BlockType:      ledger.BlockTypeSendCall,
		Amount:         big.NewInt(0),
		Fee:            big.NewInt(0),
		TokenId:        ledger.ViteTokenId,
	}
	vm := NewVM(nil)
	//vm.Debug = true
	db.addr = addr1
	sendCallBlock, isRetry, err := vm.RunV2(db, block11, nil, nil)
	if sendCallBlock == nil ||
		len(sendCallBlock.AccountBlock.SendBlockList) != 0 || isRetry || err != nil {
		t.Fatalf("init test vm config failed")
	}
}

type TestCaseMap map[string]TestCase

type TestCaseSendBlock struct {
	ToAddress types.Address
	Amount    string
	TokenID   types.TokenTypeId
	Data      string
}
type TestLog struct {
	Data   string
	Topics []string
}
type TestCase struct {
	SBHeight      uint64
	SBTime        int64
	FromAddress   types.Address
	ToAddress     types.Address
	InputData     string
	Amount        string
	TokenID       types.TokenTypeId
	Code          string
	ReturnData    string
	QuotaTotal    uint64
	QuotaLeft     uint64
	Err           string
	Storage       map[string]string
	PreStorage    map[string]string
	LogHash       string
	LogList       []TestLog
	SendBlockList []*TestCaseSendBlock
	Seed          uint64
}

func TestVm(t *testing.T) {
	testDir := "./test/"
	testFiles, ok := ioutil.ReadDir(testDir)
	if ok != nil {
		t.Fatalf("read dir failed, %v", ok)
	}
	for _, testFile := range testFiles {
		if testFile.IsDir() {
			continue
		}
		/*if testFile.Name() != "contract.json" {
			continue
		}*/
		file, ok := os.Open(testDir + testFile.Name())
		if ok != nil {
			t.Fatalf("open test file failed, %v", ok)
		}
		testCaseMap := new(TestCaseMap)
		if ok := json.NewDecoder(file).Decode(testCaseMap); ok != nil {
			t.Fatalf("decode test file failed, %v", ok)
		}

		for k, testCase := range *testCaseMap {
			var sbTime time.Time
			if testCase.SBTime > 0 {
				sbTime = time.Unix(testCase.SBTime, 0)
			} else {
				sbTime = time.Now()
			}
			sb := ledger.SnapshotBlock{
				Height:    testCase.SBHeight,
				Timestamp: &sbTime,
				Hash:      types.DataHash([]byte{1, 1}),
			}
			vm := NewVM(nil)
			vm.i = newInterpreter(1, false)
			vm.gasTable = util.GasTableByHeight(1)
			vm.globalStatus = NewTestGlobalStatus(testCase.Seed, &sb)
			//fmt.Printf("testcase %v: %v\n", testFile.Name(), k)
			inputData, _ := hex.DecodeString(testCase.InputData)
			amount, _ := hex.DecodeString(testCase.Amount)
			sendCallBlock := ledger.AccountBlock{
				AccountAddress: testCase.FromAddress,
				ToAddress:      testCase.ToAddress,
				BlockType:      ledger.BlockTypeSendCall,
				Data:           inputData,
				Amount:         new(big.Int).SetBytes(amount),
				Fee:            big.NewInt(0),
				TokenId:        testCase.TokenID,
			}
			receiveCallBlock := &ledger.AccountBlock{
				AccountAddress: testCase.ToAddress,
				BlockType:      ledger.BlockTypeReceive,
			}
			db := newMemoryDatabase(testCase.ToAddress, &sb)
			if len(testCase.PreStorage) > 0 {
				for k, v := range testCase.PreStorage {
					vByte, _ := hex.DecodeString(v)
					db.storage[k] = vByte
					db.originalStorage[k] = vByte
				}
			}
			c := newContract(
				receiveCallBlock,
				db,
				&sendCallBlock,
				sendCallBlock.Data,
				testCase.QuotaTotal)
			code, _ := hex.DecodeString(testCase.Code)
			c.setCallCode(testCase.ToAddress, code)
			util.AddBalance(db, &sendCallBlock.TokenId, sendCallBlock.Amount)
			ret, err := c.run(vm)
			returnData, _ := hex.DecodeString(testCase.ReturnData)
			if (err == nil && testCase.Err != "") || (err != nil && testCase.Err != err.Error()) {
				t.Fatalf("%v: %v failed, err not match, expected %v, got %v", testFile.Name(), k, testCase.Err, err)
			}
			if err == nil || err.Error() == "execution reverted" {
				if bytes.Compare(returnData, ret) != 0 {
					t.Fatalf("%v: %v failed, return Data error, expected %v, got %v", testFile.Name(), k, returnData, ret)
				} else if c.quotaLeft != testCase.QuotaLeft {
					t.Fatalf("%v: %v failed, quota left error, expected %v, got %v", testFile.Name(), k, testCase.QuotaLeft, c.quotaLeft)
				} else if checkStorageResult := checkStorage(db, testCase.Storage); checkStorageResult != "" {
					t.Fatalf("%v: %v failed, storage error, %v", testFile.Name(), k, checkStorageResult)
				} else if len(testCase.LogHash) > 0 {
					if logHash := db.GetLogListHash(); (logHash == nil && len(testCase.LogHash) != 0) || (logHash != nil && logHash.String() != testCase.LogHash) {
						t.Fatalf("%v: %v failed, log hash error, expected\n%v,\ngot\n%v", testFile.Name(), k, testCase.LogHash, logHash)
					}
				} else if len(testCase.LogList) > 0 {
					if checkLogListResult := checkLogList(testCase.LogList, db); checkLogListResult != "" {
						t.Fatalf("%v: %v failed, log list error, %v", testFile.Name(), k, checkLogListResult)
					}
				} else if checkSendBlockListResult := checkSendBlockList(testCase.SendBlockList, vm.sendBlockList); checkSendBlockListResult != "" {
					t.Fatalf("%v: %v failed, send block list error, %v", testFile.Name(), k, checkSendBlockListResult)
				}
			}
		}
	}
}
func checkLogList(expected []TestLog, got *memoryDatabase) string {
	if len(expected) != len(got.logList) {
		return "expected len " + strconv.Itoa(len(expected)) + ", got len" + strconv.Itoa(len(got.logList))
	}
	for index, lGot := range got.logList {
		lexpected := expected[index]
		if len(lexpected.Topics) != len(lGot.Topics) {
			return strconv.Itoa(index) + "th log topic len not match, expected " + strconv.Itoa(len(lexpected.Topics)) + ", got " + strconv.Itoa(len(lGot.Topics))
		}
		if dataStr := hex.EncodeToString(lGot.Data); dataStr != lexpected.Data {
			return "expected " + strconv.Itoa(index) + "th log data: " + lexpected.Data + ", got: " + dataStr
		}
		for topicIndex, t := range lGot.Topics {
			if topicStr := t.String(); topicStr != lexpected.Topics[topicIndex] {
				return "expected " + strconv.Itoa(index) + ":" + strconv.Itoa(topicIndex) + "th topic: " + lexpected.Topics[topicIndex] + ", got: " + topicStr
			}
		}
	}
	return ""
}

func checkStorage(got *memoryDatabase, expected map[string]string) string {
	count := 0
	for _, v := range got.storage {
		if len(v) > 0 {
			count = count + 1
		}
	}
	if len(expected) != count {
		return "expected len " + strconv.Itoa(len(expected)) + ", got len" + strconv.Itoa(len(got.storage))
	}
	for k, v := range got.storage {
		if len(v) == 0 {
			continue
		}
		if sv, ok := expected[k]; !ok || sv != hex.EncodeToString(v) {
			return "expect " + k + ": " + sv + ", got " + k + ": " + hex.EncodeToString(v)
		}
	}
	return ""
}

func checkSendBlockList(expected []*TestCaseSendBlock, got []*ledger.AccountBlock) string {
	if len(got) != len(expected) {
		return "expected len " + strconv.Itoa(len(expected)) + ", got len" + strconv.Itoa(len(got))
	}
	for i, expectedSendBlock := range expected {
		gotSendBlock := got[i]
		if gotSendBlock.ToAddress != expectedSendBlock.ToAddress {
			return "expected toAddress " + expectedSendBlock.ToAddress.String() + ", got toAddress " + gotSendBlock.ToAddress.String()
		} else if gotAmount := hex.EncodeToString(gotSendBlock.Amount.Bytes()); gotAmount != expectedSendBlock.Amount {
			return "expected amount " + expectedSendBlock.Amount + ", got amount " + gotAmount
		} else if gotSendBlock.TokenId != expectedSendBlock.TokenID {
			return "expected tokenId " + expectedSendBlock.TokenID.String() + ", got tokenId " + gotSendBlock.TokenId.String()
		} else if gotData := hex.EncodeToString(gotSendBlock.Data); gotData != expectedSendBlock.Data {
			return "expected data " + expectedSendBlock.Data + ", got data " + gotData
		}
	}
	return ""
}

type OffchainTestCaseMap map[string]OffchainTestCase
type OffchainTestCase struct {
	SBHeight   uint64
	SBTime     int64
	ToAddress  types.Address
	InputData  string
	Code       string
	ReturnData string
	Err        string
	PreStorage map[string]string
}

func TestOffChainReader(t *testing.T) {
	testCaseMap := new(OffchainTestCaseMap)
	file, ok := os.Open("./test/offchaintest/offchain.json")
	if ok != nil {
		t.Fatalf("open test file failed, %v", ok)
	}
	if ok := json.NewDecoder(file).Decode(testCaseMap); ok != nil {
		t.Fatalf("decode test file failed, %v", ok)
	}

	for k, testCase := range *testCaseMap {
		vm := NewVM(nil)
		vm.i = newInterpreter(1, true)
		var sbTime time.Time
		if testCase.SBTime > 0 {
			sbTime = time.Unix(testCase.SBTime, 0)
		} else {
			sbTime = time.Now()
		}
		sb := ledger.SnapshotBlock{
			Height:    testCase.SBHeight,
			Timestamp: &sbTime,
			Hash:      types.DataHash([]byte{1, 1}),
		}
		db := newMemoryDatabase(testCase.ToAddress, &sb)
		if len(testCase.PreStorage) > 0 {
			for k, v := range testCase.PreStorage {
				vByte, _ := hex.DecodeString(v)
				db.storage[k] = vByte
				db.originalStorage[k] = vByte
			}
		}
		code, _ := hex.DecodeString(testCase.Code)
		inputData, _ := hex.DecodeString(testCase.InputData)
		returndata, err := vm.OffChainReader(db, code, inputData)
		if (err == nil && testCase.Err != "") || (err != nil && testCase.Err != err.Error()) {
			t.Fatalf("%v failed, err not match, expected %v, got %v", k, testCase.Err, err)
		}
		returndataTarget, _ := hex.DecodeString(testCase.ReturnData)
		if !bytes.Equal(returndata, returndataTarget) {
			t.Fatalf("%v return data not match, expected %v, got %v", k, testCase.ReturnData, hex.EncodeToString(returndata))
		}
	}
}

type TestGlobalStatus struct {
	seed          uint64
	snapshotBlock *ledger.SnapshotBlock
	randSource    helper.Source64
	setRandSeed   bool
}

func NewTestGlobalStatus(seed uint64, snapshotBlock *ledger.SnapshotBlock) *TestGlobalStatus {
	return &TestGlobalStatus{seed: seed, snapshotBlock: snapshotBlock}
}
func (g *TestGlobalStatus) Seed() (uint64, error) {
	return g.seed, nil
}
func (g *TestGlobalStatus) Random() (uint64, error) {
	if g.setRandSeed {
		return g.randSource.Uint64(), nil
	}
	g.randSource = helper.NewSource64(int64(g.seed))
	g.setRandSeed = true
	return g.randSource.Uint64(), nil
}
func (g *TestGlobalStatus) SnapshotBlock() *ledger.SnapshotBlock {
	return g.snapshotBlock
}

func BenchmarkSendCall(b *testing.B) {
	sbTime := time.Now()
	sb := ledger.SnapshotBlock{
		Height:    1,
		Timestamp: &sbTime,
		Hash:      types.DataHash([]byte{1, 1}),
	}
	sendCallBlock := &ledger.AccountBlock{
		BlockType:  ledger.BlockTypeSendCall,
		Data:       []byte{},
		Amount:     big.NewInt(10),
		Fee:        big.NewInt(0),
		TokenId:    ledger.ViteTokenId,
		Difficulty: big.NewInt(67108863),
	}
	sendCallBlock.AccountAddress, _ = types.HexToAddress("vite_e41be57d38c796984952fad618a9bc91637329b5255cb18906")
	sendCallBlock.ToAddress, _ = types.HexToAddress("vite_098dfae02679a4ca05a4c8bf5dd00a8757f0c622bfccce7d68")
	db := newMemoryDatabase(sendCallBlock.AccountAddress, &sb)
	db.SetBalance(&ledger.ViteTokenId, new(big.Int).Mul(big.NewInt(1e9), big.NewInt(1e18)))
	for i := 0; i < b.N; i++ {
		vm := NewVM(nil)
		_, _, err := vm.RunV2(db, sendCallBlock, nil, nil)
		if err != nil {
			b.Fatalf("vm run failed, err: %v", err)
		}
	}
}

func BenchmarkReceiveCall(b *testing.B) {
	sbTime := time.Now()
	sb := ledger.SnapshotBlock{
		Height:    1,
		Timestamp: &sbTime,
		Hash:      types.DataHash([]byte{1, 1}),
	}
	sendCallBlock := &ledger.AccountBlock{
		BlockType:  ledger.BlockTypeSendCall,
		Data:       []byte{},
		Amount:     big.NewInt(10),
		Fee:        big.NewInt(0),
		TokenId:    ledger.ViteTokenId,
		Difficulty: big.NewInt(67108863),
	}
	sendCallBlock.AccountAddress, _ = types.HexToAddress("vite_e41be57d38c796984952fad618a9bc91637329b5255cb18906")
	sendCallBlock.ToAddress, _ = types.HexToAddress("vite_098dfae02679a4ca05a4c8bf5dd00a8757f0c622bfccce7d68")

	receiveCallBlock := &ledger.AccountBlock{
		BlockType:  ledger.BlockTypeReceive,
		Difficulty: big.NewInt(67108863),
	}

	db := newMemoryDatabase(sendCallBlock.AccountAddress, &sb)
	db.SetBalance(&ledger.ViteTokenId, new(big.Int).Mul(big.NewInt(1e9), big.NewInt(1e18)))
	for i := 0; i < b.N; i++ {
		vm := NewVM(nil)
		_, _, err := vm.RunV2(db, receiveCallBlock, sendCallBlock, nil)
		if err != nil {
			b.Fatalf("vm run failed, err: %v", err)
		}
	}
}
