package vm

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/util"
	"github.com/vitelabs/go-vite/vm_db"
	"io/ioutil"
	"math/big"
	"os"
	"strconv"
	"testing"
	"time"
)

type VMRunTestCase struct {
	// global status
	SbHeight uint64
	SbTime   int64
	SbHash   string
	CsDetail map[uint64]map[string]*ConsensusDetail
	// block
	BlockType        byte
	SendBlockType    byte
	SendBlockHash    string
	FromAddress      types.Address
	ToAddress        types.Address
	Data             string
	Amount           string
	TokenId          types.TokenTypeId
	Fee              string
	Code             string
	NeedGlobalStatus bool
	BlockHeight      uint64
	// environment
	PledgeBeneficialAmount string
	PreStorage             map[string]string
	PreBalanceMap          map[types.TokenTypeId]string
	PreContractMetaMap     map[types.Address]*ledger.ContractMeta
	ContractMetaMap        map[types.Address]*ledger.ContractMeta
	// result
	Err           string
	IsRetry       bool
	Success       bool
	Quota         uint64
	QuotaUsed     uint64
	BlockData     *string
	SendBlockList []*TestCaseSendBlock
	LogList       []TestLog
	Storage       map[string]string
	BalanceMap    map[types.TokenTypeId]string
}

var (
	quotaInfoList = commonQuotaInfoList()
	prevHash, _   = types.HexToHash("82a8ecfe0df3dea6256651ee3130747386d4d6ab61201ce0050a6fe394a0f595")
	// testAddr,_ = types.HexToAddress("vite_ab24ef68b84e642c0ddca06beec81c9acb1977bbd7da27a87a")
	// testContractAddr,_ = types.HexToAddress("vite_a3ab3f8ce81936636af4c6f4da41612f11136d71f53bf8fa86")
)

const (
	genesisTimestamp int64 = 1546272000
	csInterval       int64 = 24 * 3600
)

func TestVM_RunV2(t *testing.T) {
	testDir := "./test/run_test/"
	testFiles, ok := ioutil.ReadDir(testDir)
	if ok != nil {
		t.Fatalf("read dir failed, %v", ok)
	}
	for _, testFile := range testFiles {
		if testFile.IsDir() {
			continue
		}
		file, ok := os.Open(testDir + testFile.Name())
		if ok != nil {
			t.Fatalf("open test file failed, %v", ok)
		}
		testCaseMap := new(map[string]VMRunTestCase)
		if ok := json.NewDecoder(file).Decode(testCaseMap); ok != nil {
			t.Fatalf("decode test file %v failed, %v", testFile.Name(), ok)
		}
		for k, testCase := range *testCaseMap {
			fmt.Println(testFile.Name() + ":" + k)
			var currentTime time.Time
			if testCase.SbTime > 0 {
				currentTime = time.Unix(testCase.SbTime, 0)
			} else {
				currentTime = time.Now()
			}
			latestSnapshotBlock := &ledger.SnapshotBlock{
				Height:    testCase.SbHeight,
				Timestamp: &currentTime,
			}
			if len(testCase.SbHash) > 0 {
				sbHash, parseErr := types.HexToHash(testCase.SbHash)
				if parseErr != nil {
					t.Fatal("invalid test case sbHash", "filename", testFile.Name(), "caseName", k, "sbHash", testCase.SbHash)
				}
				latestSnapshotBlock.Hash = sbHash
			}
			var ok bool
			pledgeBeneficialAmount := big.NewInt(0)
			if len(testCase.PledgeBeneficialAmount) > 0 {
				pledgeBeneficialAmount, ok = new(big.Int).SetString(testCase.PledgeBeneficialAmount, 16)
				if !ok {
					t.Fatal("invalid test case data", "filename", testFile.Name(), "caseName", k, "pledgeBeneficialAmount", testCase.PledgeBeneficialAmount)
				}
			}
			code, parseErr := hex.DecodeString(testCase.Code)
			if parseErr != nil {
				t.Fatal("invalid test case code", "filename", testFile.Name(), "caseName", k, "code", testCase.Code, "err", parseErr)
			}

			var db *mockDB
			var vmBlock *vm_db.VmAccountBlock
			var isRetry bool
			var err error

			sendBlock := &ledger.AccountBlock{
				Amount:  big.NewInt(0),
				TokenId: testCase.TokenId,
				Fee:     big.NewInt(0),
			}
			if len(testCase.Fee) > 0 {
				sendBlock.Fee, ok = new(big.Int).SetString(testCase.Fee, 16)
				if !ok {
					t.Fatal("invalid test case data", "filename", testFile.Name(), "caseName", k, "fee", testCase.Fee)
				}
			}
			if len(testCase.Amount) > 0 {
				sendBlock.Amount, ok = new(big.Int).SetString(testCase.Amount, 16)
				if !ok {
					t.Fatal("invalid test case data", "filename", testFile.Name(), "caseName", k, "amount", testCase.Amount)
				}
			}
			if len(testCase.Data) > 0 {
				sendBlock.Data, parseErr = hex.DecodeString(testCase.Data)
				if parseErr != nil {
					t.Fatal("invalid test case data", "filename", testFile.Name(), "caseName", k, "data", testCase.Data)
				}
			}

			if ledger.IsSendBlock(testCase.BlockType) {
				prevBlock := &ledger.AccountBlock{
					BlockType:      ledger.BlockTypeReceive,
					Height:         1,
					Hash:           prevHash,
					PrevHash:       types.ZERO_HASH,
					AccountAddress: testCase.FromAddress,
				}
				sendBlock.PrevHash = prevBlock.Hash
				sendBlock.Height = prevBlock.Height + 1
				sendBlock.BlockType = testCase.BlockType
				sendBlock.AccountAddress = testCase.FromAddress
				sendBlock.ToAddress = testCase.ToAddress
				var newDbErr error
				db, newDbErr = NewMockDB(&testCase.FromAddress, latestSnapshotBlock, prevBlock, quotaInfoList, pledgeBeneficialAmount, testCase.PreBalanceMap, testCase.PreStorage, testCase.PreContractMetaMap, code, genesisTimestamp, forkSnapshotBlockMap)
				if newDbErr != nil {
					t.Fatal("new mock db failed", "filename", testFile.Name(), "caseName", k, "err", newDbErr)
				}
				vm := NewVM(nil)
				vmBlock, isRetry, err = vm.RunV2(db, sendBlock, nil, nil)
			} else if ledger.IsReceiveBlock(testCase.BlockType) {
				sendBlock.BlockType = testCase.SendBlockType
				sendBlock.AccountAddress = testCase.FromAddress
				sendBlock.ToAddress = testCase.ToAddress
				if len(testCase.SendBlockHash) > 0 {
					sendBlock.Hash, parseErr = types.HexToHash(testCase.SendBlockHash)
					if parseErr != nil {
						t.Fatal("invalid test case send block hash", "filename", testFile.Name(), "caseName", k, "hash", testCase.SendBlockHash)
					}
				}
				var prevBlock, receiveBlock *ledger.AccountBlock
				if testCase.SendBlockType == ledger.BlockTypeSendCreate {
					receiveBlock = &ledger.AccountBlock{
						BlockType:      testCase.BlockType,
						PrevHash:       types.Hash{},
						Height:         1,
						AccountAddress: testCase.ToAddress,
					}
				} else {
					prevBlock = &ledger.AccountBlock{
						BlockType:      ledger.BlockTypeReceive,
						Height:         1,
						Hash:           prevHash,
						PrevHash:       types.ZERO_HASH,
						AccountAddress: testCase.ToAddress,
					}
					receiveBlock = &ledger.AccountBlock{
						BlockType:      testCase.BlockType,
						PrevHash:       prevBlock.Hash,
						Height:         prevBlock.Height + 1,
						AccountAddress: testCase.ToAddress,
					}
					if testCase.BlockHeight > 1 {
						receiveBlock.Height = testCase.BlockHeight
						prevBlock.Height = testCase.BlockHeight - 1
					}
				}
				var newDbErr error
				db, newDbErr = NewMockDB(&testCase.ToAddress, latestSnapshotBlock, prevBlock, quotaInfoList, pledgeBeneficialAmount, testCase.PreBalanceMap, testCase.PreStorage, testCase.PreContractMetaMap, code, genesisTimestamp, forkSnapshotBlockMap)
				if newDbErr != nil {
					t.Fatal("new mock db failed", "filename", testFile.Name(), "caseName", k, "err", newDbErr)
				}
				cs := util.NewVMConsensusReader(newConsensusReaderTest(genesisTimestamp, csInterval, testCase.CsDetail))
				vm := NewVM(cs)
				var status util.GlobalStatus
				if testCase.NeedGlobalStatus {
					status = NewTestGlobalStatus(0, latestSnapshotBlock)
				}
				vmBlock, isRetry, err = vm.RunV2(db, receiveBlock, sendBlock, status)
			} else {
				t.Fatal("invalid test case block type", "filename", testFile.Name(), "caseName", k, "blockType", testCase.BlockType)
			}
			if !errorEquals(testCase.Err, err) {
				t.Fatal("invalid test case run result, err", "filename", testFile.Name(), "caseName", k, "expected", testCase.Err, "got", err)
			} else if testCase.IsRetry != isRetry {
				t.Fatal("invalid test case run result, isRetry", "filename", testFile.Name(), "caseName", k, "expected", testCase.IsRetry, "got", isRetry)
			}
			if testCase.Success {
				balanceMapGot, _ := db.GetBalanceMap()
				if vmBlock == nil {
					t.Fatal("invalid test case run result, vmBlock", "filename", testFile.Name(), "caseName", k, "expected", "exist", "got", "nil")
				} else if testCase.BlockType != vmBlock.AccountBlock.BlockType {
					t.Fatal("invalid test case run result, blockType", "filename", testFile.Name(), "caseName", k, "expected", testCase.BlockType, "got", vmBlock.AccountBlock.BlockType)
				} else if testCase.Quota != vmBlock.AccountBlock.Quota {
					t.Fatal("invalid test case run result, quota", "filename", testFile.Name(), "caseName", k, "expected", testCase.Quota, "got", vmBlock.AccountBlock.Quota)
				} else if testCase.QuotaUsed != vmBlock.AccountBlock.QuotaUsed {
					t.Fatal("invalid test case run result, quotaUsed", "filename", testFile.Name(), "caseName", k, "expected", testCase.QuotaUsed, "got", vmBlock.AccountBlock.QuotaUsed)
				} else if checkBalanceResult := checkBalanceMap(testCase.BalanceMap, balanceMapGot); len(checkBalanceResult) > 0 {
					t.Fatal("invalid test case run result, balanceMap", "filename", testFile.Name(), "caseName", k, checkBalanceResult)
				} else if checkStorageResult := checkStorageMap(testCase.Storage, db.getStorageMap()); len(checkStorageResult) > 0 {
					t.Fatal("invalid test case run result, storageMap", "filename", testFile.Name(), "caseName", k, checkStorageResult)
				} else if checkSendBlockListResult := checkSendBlockList(testCase.SendBlockList, vmBlock.AccountBlock.SendBlockList); len(checkSendBlockListResult) > 0 {
					t.Fatal("invalid test case run result, sendBlockList", "filename", testFile.Name(), "caseName", k, checkSendBlockListResult)
				} else if checkLogListResult := checkLogList(testCase.LogList, db.logList); len(checkLogListResult) > 0 {
					t.Fatal("invalid test case run result, logList", "filename", testFile.Name(), "caseName", k, checkLogListResult)
				} else if expected := db.GetLogListHash(); expected != vmBlock.AccountBlock.LogHash {
					t.Fatal("invalid test case run result, logHash", "filename", testFile.Name(), "caseName", k, "expected", expected, "got", vmBlock.AccountBlock.LogHash)
				} else if checkContractMetaMapResult := checkContractMetaMap(testCase.ContractMetaMap, db.getContractMetaMap()); len(checkContractMetaMapResult) > 0 {
					t.Fatal("invalid test case run result, contractMetaMap", "filename", testFile.Name(), "caseName", k, checkContractMetaMapResult)
				} else if expected := db.GetLogListHash(); (vmBlock.AccountBlock.LogHash == nil && expected != nil) ||
					(vmBlock.AccountBlock.LogHash != nil && expected == nil) ||
					(vmBlock.AccountBlock.LogHash != nil && expected != nil && vmBlock.AccountBlock.LogHash != expected) {
					t.Fatal("invalid test case run result, log hash", "filename", testFile.Name(), "caseName", k, "expected", expected, "got", vmBlock.AccountBlock.LogHash)
				}
				if types.IsContractAddr(vmBlock.AccountBlock.AccountAddress) {
					if expected := append(db.GetReceiptHash().Bytes(), 0); err == nil && testCase.SendBlockType != ledger.BlockTypeSendRefund && !bytes.Equal(vmBlock.AccountBlock.Data, expected) {
						t.Fatal("invalid test case run result, data", "filename", testFile.Name(), "caseName", k, "expected", bytesToString(expected), "got", bytesToString(vmBlock.AccountBlock.Data))
					} else if err == nil && testCase.SendBlockType == ledger.BlockTypeSendRefund && len(vmBlock.AccountBlock.Data) > 0 {
						t.Fatal("invalid test case run result, data", "filename", testFile.Name(), "caseName", k, "expected", "nil", "got", bytesToString(vmBlock.AccountBlock.Data))
					} else if expected := append([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, byte(1)); err != nil && err.Error() != util.ErrDepth.Error() && !bytes.Equal(vmBlock.AccountBlock.Data, expected) {
						t.Fatal("invalid test case run result, data", "filename", testFile.Name(), "caseName", k, "expected", bytesToString(expected), "got", bytesToString(vmBlock.AccountBlock.Data))
					} else if expected := append([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, byte(2)); err != nil && err.Error() == util.ErrDepth.Error() && !bytes.Equal(vmBlock.AccountBlock.Data, expected) {
						t.Fatal("invalid test case run result, data", "filename", testFile.Name(), "caseName", k, "expected", bytesToString(expected), "got", bytesToString(vmBlock.AccountBlock.Data))
					}
					if testCase.SendBlockType == ledger.BlockTypeSendCreate {
						if got := hex.EncodeToString(db.code); got != testCase.Code {
							t.Fatal("invalid test case run result, result code", "filename", testFile.Name(), "caseName", k, "expected", testCase.Code, "got", got)
						}
					}
				} else if vmBlock.AccountBlock.IsReceiveBlock() {
					if len(vmBlock.AccountBlock.Data) > 0 {
						t.Fatal("invalid test case run result, receive block data", "filename", testFile.Name(), "caseName", k, "expected", "nil", "got", bytesToString(vmBlock.AccountBlock.Data))
					}
				} else {
					if testCase.BlockData == nil && !bytes.Equal(vmBlock.AccountBlock.Data, stringToBytes(testCase.Data)) {
						t.Fatal("invalid test case run result, send block data", "filename", testFile.Name(), "caseName", k, "expected", testCase.Data, "got", bytesToString(vmBlock.AccountBlock.Data))
					} else if testCase.BlockData != nil && !bytes.Equal(vmBlock.AccountBlock.Data, stringToBytes(*testCase.BlockData)) {
						t.Fatal("invalid test case run result, send block data", "filename", testFile.Name(), "caseName", k, "expected", testCase.BlockData, "got", bytesToString(vmBlock.AccountBlock.Data))
					}
				}
			} else if vmBlock != nil {
				t.Fatal("invalid test case run result, vmBlock", "filename", testFile.Name(), "caseName", k, "expected", "nil", "got", vmBlock.AccountBlock)
			}
		}
	}
}

func commonQuotaInfoList() []types.QuotaInfo {
	quotaInfoList := make([]types.QuotaInfo, 0, 75)
	for i := 0; i < 75; i++ {
		quotaInfoList = append(quotaInfoList, types.QuotaInfo{BlockCount: 0, QuotaTotal: 0, QuotaUsedTotal: 0})
	}
	return quotaInfoList
}

func errorEquals(expected string, got error) bool {
	if (len(expected) == 0 && got == nil) || (len(expected) > 0 && got != nil && expected == got.Error()) {
		return true
	}
	return false
}

func checkBalanceMap(expected map[types.TokenTypeId]string, got map[types.TokenTypeId]*big.Int) string {
	gotCount := 0
	for _, v := range got {
		if v.Sign() > 0 {
			gotCount = gotCount + 1
		}
	}
	expectedCount := len(expected)
	if expectedCount != gotCount {
		return "balanceMap len, expected " + strconv.Itoa(expectedCount) + ", got " + strconv.Itoa(gotCount)
	}
	for k, v := range got {
		if v.Sign() == 0 {
			continue
		}
		expectedV, ok := new(big.Int).SetString(expected[k], 16)
		if !ok {
			return k.String() + " token balance, expected" + expected[k] + ", got " + v.String()
		}
		if v.Cmp(expectedV) != 0 {
			return k.String() + " token balance, expect " + expectedV.String() + ", got " + v.String()
		}
	}
	return ""
}

func checkStorageMap(expected, got map[string]string) string {
	gotCount := 0
	for _, v := range got {
		if len(v) > 0 {
			gotCount = gotCount + 1
		}
	}
	expectedCount := len(expected)
	if expectedCount != gotCount {
		return "storageMap len, expected " + strconv.Itoa(expectedCount) + ", got " + strconv.Itoa(gotCount)
	}
	for k, v := range got {
		if len(v) == 0 {
			continue
		}
		if expectedV, ok := expected[k]; !ok || expectedV != v {
			return k + " storage, expect " + expectedV + ", got " + v
		}
	}
	return ""
}

func checkContractMetaMap(expected, got map[types.Address]*ledger.ContractMeta) string {
	gotCount := len(got)
	expectedCount := len(expected)
	if expectedCount != gotCount {
		return "contract meta map len, expected " + strconv.Itoa(expectedCount) + ", got " + strconv.Itoa(gotCount)
	}
	for k, v := range got {
		expectedV, ok := expected[k]
		if !ok {
			return "contract meta not exists, " + k.String()
		}
		if v.QuotaRatio != expectedV.QuotaRatio ||
			v.Gid != expectedV.Gid ||
			v.SendConfirmedTimes != expectedV.SendConfirmedTimes ||
			v.SeedConfirmedTimes != expectedV.SeedConfirmedTimes {
			return fmt.Sprintf("%v contract meta, expect [%v,%v,%v,%v] , got [%v,%v,%v,%v]", k.String(),
				expectedV.Gid, expectedV.SendConfirmedTimes, expectedV.SeedConfirmedTimes, expectedV.QuotaRatio,
				v.Gid, v.SendConfirmedTimes, v.SeedConfirmedTimes, v.QuotaRatio)
		}
	}
	return ""
}
