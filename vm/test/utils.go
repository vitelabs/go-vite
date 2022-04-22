package test

import (
	"github.com/vitelabs/go-vite/v2/common/types"
	"github.com/vitelabs/go-vite/v2/common/upgrade"
	"github.com/vitelabs/go-vite/v2/interfaces"
	ledger "github.com/vitelabs/go-vite/v2/interfaces/core"
	"github.com/vitelabs/go-vite/v2/vm"
	"math/big"
	"testing"
	"time"
)

func InitCustomFork(t *testing.T) {
	upgrade.CleanupUpgradeBox(t)
	upgrade.InitUpgradeBox(upgrade.NewCustomUpgradeBox(
		map[string]*upgrade.UpgradePoint{
			"SeedFork":            &upgrade.UpgradePoint{Height: 100, Version: 1},
			"DexFork":             &upgrade.UpgradePoint{Height: 200, Version: 2},
			"DexFeeFork":          &upgrade.UpgradePoint{Height: 250, Version: 3},
			"StemFork":            &upgrade.UpgradePoint{Height: 300, Version: 4},
			"LeafFork":            &upgrade.UpgradePoint{Height: 400, Version: 5},
			"EarthFork":           &upgrade.UpgradePoint{Height: 500, Version: 6},
			"DexMiningFork":       &upgrade.UpgradePoint{Height: 600, Version: 7},
			"DexRobotFork":        &upgrade.UpgradePoint{Height: 600, Version: 8},
			"DexStableMarketFork": &upgrade.UpgradePoint{Height: 600, Version: 9},
			"Version10Fork": 	   &upgrade.UpgradePoint{Height: 1000, Version: 10},
			"VEP19Fork": 		   &upgrade.UpgradePoint{Height: 1100, Version: 11},
		},
	))
}

var (
	testDB *vm.TestDatabase
	testAddress types.Address
	prevHash types.Hash
	prevHeight uint64
	testSnapshot *ledger.SnapshotBlock
	globalStatus *vm.TestGlobalStatus
)

type CallParams struct {
	Address types.Address
	Calldata []byte
	TokenId *types.TokenTypeId
	Amount *big.Int
	BlockType byte
	Referrer *types.Hash
	Callback *big.Int
}

func InitTestContext(t *testing.T) (*vm.TestDatabase, types.Address) {
	vm.InitVMConfig(true, true, true, false, "")
	InitCustomFork(t)
	viteTotalSupply := new(big.Int).Mul(big.NewInt(1e9), big.NewInt(1e18))
	testDB, testAddress, _, prevHash, testSnapshot, _ = vm.PrepareDb(viteTotalSupply)
	t2 := time.Unix(1600663514, 0)
	snapshot20 := &ledger.SnapshotBlock{Height: 2000, Timestamp: &t2, Hash: types.DataHash([]byte{10, 2})}
	testDB.SnapshotBlockList = append(testDB.SnapshotBlockList, snapshot20)
	prevHeight = 2
	globalStatus = vm.NewTestGlobalStatus(0, testSnapshot)

	return testDB, testAddress
}

func callContract(params CallParams) (*interfaces.VmAccountBlock, error) {
	return callContractBy(&testAddress, params)
}

func callContractBy(caller *types.Address, params CallParams) (*interfaces.VmAccountBlock, error) {
	testDB.Addr = params.Address
	prevHeight += 1
	blockType := ledger.BlockTypeSendCall
	if params.BlockType == 0 && (params.Referrer != nil || params.Callback != nil) {
		blockType = ledger.BlockTypeSendSyncCall
	} else if params.BlockType > 0 {
		blockType = params.BlockType
	}
	from := testAddress
	if caller != nil {
		from = *caller
	}

	sendBlock := &ledger.AccountBlock{
		Height:         prevHeight,
		ToAddress:      params.Address,
		AccountAddress: from,
		Amount:         big.NewInt(0),
		TokenId:        ledger.ViteTokenId,
		BlockType:      blockType,
		Fee:            big.NewInt(0),
		PrevHash:       prevHash,
		Data:           params.Calldata,
	}
	if params.TokenId != nil {
		sendBlock.TokenId = *params.TokenId
	}
	if params.Amount != nil {
		sendBlock.Amount = params.Amount
	}
	sendBlock.Hash = sendBlock.ComputeHash()
	if sendBlock.BlockType == ledger.BlockTypeSendSyncCall{
		context := &ledger.ExecutionContext{}
		if params.Referrer != nil {
			context.ReferrerSendHash = *params.Referrer
		}
		if params.Callback != nil {
			context.CallbackId = *params.Callback
		}
		testDB.SetExecutionContext(&sendBlock.Hash, context)
	}

	receiveBlock := &ledger.AccountBlock{
		Height:         1,
		AccountAddress: params.Address,
		BlockType:      ledger.BlockTypeReceive,
		FromBlockHash:  sendBlock.Hash,
	}
	receiveBlock.Hash = receiveBlock.ComputeHash()

	vm := vm.NewVM(nil, nil)
	vmBlock, _, err := vm.RunV2(testDB, receiveBlock, sendBlock, globalStatus)

	return vmBlock, err
}