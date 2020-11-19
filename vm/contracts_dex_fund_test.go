package vm

import (
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/contracts/abi"
	cabi "github.com/vitelabs/go-vite/vm/contracts/abi"
	"github.com/vitelabs/go-vite/vm/contracts/dex"
	dexproto "github.com/vitelabs/go-vite/vm/contracts/dex/proto"
	"github.com/vitelabs/go-vite/vm_db"
	"io/ioutil"
	"math/big"
	"os"
	"testing"
	"time"
)

type DexFundCase struct {
	Name       string
	GlobalEnv  GlobalEnv
	DexStorage *DexStorage
	// environment
	PreStorage    map[string]string
	PreBalanceMap map[types.TokenTypeId]string
	AssetActions  []*AssetAction

	AccountsCheck []*AccountsCheck
	// result
	//
	//SendBlockList []*TestCaseSendBlock
	//LogList       []TestLog
	//Storage       map[string]string
	//BalanceMap    map[types.TokenTypeId]string
}

type GlobalEnv struct {
	// global status
	SbHeight uint64
	SbTime   int64
	SbHash   string
	Quotas   []*StakeQuota
	Balances []*BalanceStorage
	//CsDetail map[uint64]map[string]*ConsensusDetail
}

type StakeQuota struct {
	Address types.Address
	Quota   *big.Int
}

type DexStorage struct {
	Funds   []*FundStorage
	Tokens  []*TokenStorage
	Markets []*MarketStorage
}

type FundStorage struct {
	Address  types.Address
	Accounts []AccountStorage
}

type AccountStorage struct {
	Token           types.TokenTypeId
	Available       *big.Int
	Locked          *big.Int
	VxUnlocking     *big.Int
	CancellingStake *big.Int
}

type BalanceStorage struct {
	Address  types.Address
	Balances []*Balance
}

type Balance struct {
	TokenId types.TokenTypeId
	Amount  *big.Int
}

type TokenStorage struct {
	TokenId        types.TokenTypeId
	Decimals       int32
	Symbol         string
	Index          int32
	Owner          types.Address
	QuoteTokenType int32
}

type MarketStorage struct {
	MarketId             int32
	TradeToken           types.TokenTypeId
	QuoteToken           types.TokenTypeId
	Symbol               string
	QuoteTokenType       int32
	TradeTokenDecimals   int32
	QuoteTokenDecimals   int32
	TakerOperatorFeeRate int32
	MakerOperatorFeeRate int32
	AllowMining          bool
	Valid                bool
	Owner                types.Address
	Creator              types.Address
	Stopped              bool
	Timestamp            int64
	StableMarket         bool
}

type NewMarketAction struct {
	Address    types.Address
	TradeToken types.TokenTypeId
	QuoteToken types.TokenTypeId
}

type AssetAction struct {
	IsDeposit bool
	Address types.Address
	Token     types.TokenTypeId
	Amount    *big.Int
}

type AccountsCheck struct {
	Address  types.Address
	Accounts []AccountStorage
}

func TestContractsFund(t *testing.T) {
	testDir := "./contracts/dex/test/"
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
		testCaseMap := new(map[string]*DexFundCase)
		if ok := json.NewDecoder(file).Decode(testCaseMap); ok != nil {
			t.Fatalf("decode test file %v failed, %v", testFile.Name(), ok)
		}
		for k, testCase := range *testCaseMap {
			fmt.Println(testFile.Name() + ":" + k)
			db := initFundDb(testCase, t)
			vm := NewVM(nil)
			executeActions(testCase, vm, db, t)
			executeChecks(testCase, db, t)
		}
	}
}

func executeActions(testCase *DexFundCase, vm *VM, db *testDatabase, t *testing.T) {
	if testCase.AssetActions != nil {
		for _, action := range testCase.AssetActions {
			sendBlock := &ledger.AccountBlock{}
			sendBlock.AccountAddress = action.Address
			sendBlock.BlockType = ledger.BlockTypeSendCall
			sendBlock.ToAddress = types.AddressDexFund
			db.addr = action.Address
			var (
				vmSendBlock *vm_db.VmAccountBlock
				err error
			)
			if action.IsDeposit {
				sendBlock.TokenId = action.Token
				sendBlock.Amount = action.Amount
				sendBlock.Data, _ = cabi.ABIDexFund.PackMethod(cabi.MethodNameDexFundDeposit)
				vmSendBlock, _, err = vm.RunV2(db, sendBlock, nil, nil)
			} else {
				sendBlock.TokenId = ledger.ViteTokenId
				sendBlock.Data, err = cabi.ABIDexFund.PackMethod(cabi.MethodNameDexFundWithdraw, action.Token, action.Amount)
				vmSendBlock, _, err = vm.RunV2(db, sendBlock, nil, nil)
			}
			//fmt.Printf("executeActions send runVm err %v\n", err)
			assert.True(t, err == nil, "vm.RunV2 handle send result err not nil")
			db.addr = types.AddressDexFund
			rcBlock := &ledger.AccountBlock{}
			rcBlock.AccountAddress = types.AddressDexFund
			rcBlock.BlockType = ledger.BlockTypeReceive
			vmSendBlock, _, err = vm.RunV2(db, rcBlock, vmSendBlock.AccountBlock, nil)
			//fmt.Printf("handle receive runVm err %v\n", err)
			assert.True(t, err == nil, "vm.RunV2 handle receive result err not nil")
		}
	}
}

func executeChecks(testCase *DexFundCase, db vm_db.VmDb, t *testing.T) {
	if testCase.AccountsCheck != nil {
		for idx, bc := range testCase.AccountsCheck {
			fund, ok := dex.GetFund(db, bc.Address)
			assert.True(t, ok, fmt.Sprintf("fund not exist for %s, %s, %d", testCase.Name, bc.Address.String(), idx))
			for idx1, bl := range bc.Accounts {
				acc, ok1 := dex.GetAccountByToken(fund, bl.Token)
				assert.True(t, ok1, fmt.Sprintf("account not exist for %s, %s, %s, %d", testCase.Name, bc.Address.String(), bl.Token.String(), idx1))
				assert.Equal(t, bl.Available.String(), new(big.Int).SetBytes(acc.Available).String(), fmt.Sprintf("account amt equals for %s, %s, %s, %d", testCase.Name, bc.Address.String(), bl.Token.String(), idx1))
			}
		}
	}
}

func initFundDb(dexFundCase *DexFundCase, t *testing.T) *testDatabase {
	var currentTime time.Time
	if dexFundCase.GlobalEnv.SbTime > 0 {
		currentTime = time.Unix(dexFundCase.GlobalEnv.SbTime, 0)
	} else {
		currentTime = time.Now()
	}
	latestSnapshotBlock := &ledger.SnapshotBlock{
		Height:    dexFundCase.GlobalEnv.SbHeight,
		Timestamp: &currentTime,
	}
	if len(dexFundCase.GlobalEnv.SbHash) > 0 {
		sbHash, parseErr := types.HexToHash(dexFundCase.GlobalEnv.SbHash)
		if parseErr != nil {
			t.Fatal("invalid test case sbHash", "sbHash", dexFundCase.GlobalEnv.SbHash)
		}
		latestSnapshotBlock.Hash = sbHash
	}
	var db *testDatabase
	var newDbErr error
	//db, newDbErr = NewMockDB(&types.AddressDexFund, latestSnapshotBlock, nil, quotaInfoList, big.NewInt(0), nil, nil, nil, []byte{}, genesisTimestamp, forkSnapshotBlockMap)
	viteTotalSupply := new(big.Int).Mul(big.NewInt(1e9), big.NewInt(1e18))
	db, _, _, _, _, _ = prepareDb(viteTotalSupply)
	t2 := time.Unix(1600663514, 0)
	snapshot20 := &ledger.SnapshotBlock{Height: 2000, Timestamp: &t2, Hash: types.DataHash([]byte{10, 2})}
	db.snapshotBlockList = append(db.snapshotBlockList, snapshot20)
	if newDbErr != nil {
		t.Fatal("new mock db failed", "name", dexFundCase.Name)
	}

	if len(dexFundCase.GlobalEnv.Quotas) > 0 {
		db.storageMap[types.AddressQuota] = make(map[string][]byte, 0)
		for _, qt := range dexFundCase.GlobalEnv.Quotas {
			data, packErr := abi.ABIQuota.PackVariable(abi.VariableNameStakeBeneficial, new(big.Int).Mul(big.NewInt(1e9), big.NewInt(1e18)))
			assert.True(t, packErr == nil)
			key := ToKey(abi.GetStakeBeneficialKey(qt.Address))
			db.storageMap[types.AddressQuota][key] = data
		}
	}

	if len(dexFundCase.GlobalEnv.Balances) > 0 {
		for _, balance := range dexFundCase.GlobalEnv.Balances {
			db.balanceMap[balance.Address] = make(map[types.TokenTypeId]*big.Int)
			for _, bl := range balance.Balances {
				db.balanceMap[balance.Address][bl.TokenId] = bl.Amount
			}
		}
	}

	if dexFundCase.DexStorage != nil {
		if dexFundCase.DexStorage.Funds != nil {
			for _, fd := range dexFundCase.DexStorage.Funds {
				fund := &dex.Fund{}
				fund.Address = fd.Address.Bytes()
				for _, acc := range fd.Accounts {
					account := &dexproto.Account{}
					account.Token = acc.Token.Bytes()
					account.Available = acc.Available.Bytes()
					account.Locked = acc.Locked.Bytes()
					account.VxUnlocking = acc.VxUnlocking.Bytes()
					account.CancellingStake = acc.CancellingStake.Bytes()
					fund.Accounts = append(fund.Accounts, account)
				}
				dex.SaveFund(db, fd.Address, fund)
			}
		}
		if dexFundCase.DexStorage.Tokens != nil {
			for _, tk := range dexFundCase.DexStorage.Tokens {
				tokenInfo := &dex.TokenInfo{}
				tokenInfo.TokenId = tk.TokenId.Bytes()
				tokenInfo.Decimals = tk.Decimals
				tokenInfo.Symbol = tk.Symbol
				tokenInfo.Index = tk.Index
				tokenInfo.Owner = tk.Owner.Bytes()
				tokenInfo.QuoteTokenType = tk.QuoteTokenType
				dex.SaveTokenInfo(db, tk.TokenId, tokenInfo)
			}
		}
		if dexFundCase.DexStorage.Markets != nil {
			for _, mk := range dexFundCase.DexStorage.Markets {
				mkInfo := &dex.MarketInfo{}
				mkInfo.MarketId = mk.MarketId
				mkInfo.MarketSymbol = mk.Symbol
				mkInfo.TradeToken = mk.TradeToken.Bytes()
				mkInfo.QuoteToken = mk.QuoteToken.Bytes()
				mkInfo.QuoteTokenType = mk.QuoteTokenType
				mkInfo.TradeTokenDecimals = mk.TradeTokenDecimals
				mkInfo.QuoteTokenType = mk.QuoteTokenType
				mkInfo.TakerOperatorFeeRate = mk.TakerOperatorFeeRate
				mkInfo.MakerOperatorFeeRate = mk.MakerOperatorFeeRate
				mkInfo.AllowMining = mk.AllowMining
				mkInfo.Valid = mk.Valid
				mkInfo.Owner = mk.Owner.Bytes()
				mkInfo.Creator = mk.Creator.Bytes()
				mkInfo.Stopped = mk.Stopped
				mkInfo.Timestamp = mk.Timestamp
				mkInfo.StableMarket = mk.StableMarket
				dex.SaveMarketInfo(db, mkInfo, mk.TradeToken, mk.QuoteToken)
			}
		}
	}
	return db
}
