package wallet_test

import (
	"fmt"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/wallet"
	"path/filepath"
	"testing"
)

func TestManager_NewMnemonicAndSeedStore(t *testing.T) {
	manager := wallet.New(&wallet.Config{
		DataDir: filepath.Join(common.DefaultDataDir(), "wallet"),
	})
	mnemonic, em, err := manager.NewMnemonicAndEntropyStore("123456", true)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(mnemonic)
	fmt.Println(em.GetPrimaryAddr())
	fmt.Println(em.GetEntropyStoreFile())

	em, e := manager.RecoverEntropyStoreFromMnemonic(mnemonic, "123456", true)
	if e != nil {
		t.Fatal()
	}

	fmt.Println(em.GetPrimaryAddr())
	fmt.Println(em.GetEntropyStoreFile())
}
