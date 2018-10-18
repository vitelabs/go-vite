package chain

import (
	"fmt"
	"github.com/vitelabs/go-vite/crypto"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/pow"
	"testing"
)

func TestGetSubLedgerByHeight(t *testing.T) {
	chainInstance := getChainInstance()
	makeBlocks(chainInstance, 20000)
	chainInstance.Compressor().RunTask()
	files, dbRange := chainInstance.GetSubLedgerByHeight(100, 10000, true)
	for _, fileInfo := range files {
		fmt.Printf("%+v\n", fileInfo)
	}

	fmt.Printf("%+v\n", dbRange)
}

func TestGetSubLedgerByHash(t *testing.T) {
	chainInstance := getChainInstance()
	makeBlocks(chainInstance, 20000)
	chainInstance.Compressor().RunTask()
	files, dbRange, err := chainInstance.GetSubLedgerByHash(&GenesisSnapshotBlock.Hash, 1000, true)
	if err != nil {
		t.Fatal(err)
	}
	for _, fileInfo := range files {
		fmt.Printf("%+v\n", fileInfo)
	}

	fmt.Printf("%+v\n", dbRange)
}

func TestGetConfirmSubLedger(t *testing.T) {
	chainInstance := getChainInstance()
	makeBlocks(chainInstance, 1000)
	_, subLedger, err := chainInstance.GetConfirmSubLedger(0, 1000)
	if err != nil {
		t.Fatal(err)
	}

	//for index, block := range snapshotBlocks {
	//	fmt.Printf("%d: %+v\n", index, block)
	//}

	fmt.Println()

	for addr, blocks := range subLedger {
		fmt.Printf("%s\n", addr.String())
		for _, block := range blocks {
			accountType, err := chainInstance.AccountType(&block.AccountAddress)
			if err != nil {
				t.Error(err)
			}
			b := block.ComputeHash() != block.Hash
			if b {
				t.Error("hash err")
			}
			if len(block.Nonce) != 0 {
				if accountType == ledger.AccountTypeContract {
					t.Error("AccountTypeContract Nonce must be nil")
				}
				t.Log(block.Nonce)
				var nonce [8]byte
				copy(nonce[:], block.Nonce[:8])
				hash256Data := crypto.Hash256(block.AccountAddress.Bytes(), block.PrevHash.Bytes())
				if !pow.CheckPowNonce(nil, nonce, hash256Data) {
					t.Error("CheckPowNonce failed")
				}
			}

		}
	}
}
