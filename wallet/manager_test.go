package wallet_test

import (
	"fmt"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/wallet"
)

func deskTopDir() string {
	home := common.HomeDir()
	if home != "" {
		return filepath.Join(home, "Desktop", "wallet")
	}
	return ""
}

func TestManager_NewMnemonicAndSeedStore(t *testing.T) {
	manager := wallet.New(&wallet.Config{
		DataDir: deskTopDir(),
	})
	mnemonic, em, err := manager.NewMnemonicAndEntropyStore("123456")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(mnemonic)
	fmt.Println(em.GetPrimaryAddr())
	fmt.Println(em.GetEntropyStoreFile())
	//
	//em2, e := manager.RecoverEntropyStoreFromMnemonic(mnemonic, "123456")
	//if e != nil {
	//	t.Fatal()
	//}
	//em2.Unlock("123456")
	em.Unlock("123456")

	for i := 0; i < 1000; i++ {
		_, key2, e := em.DeriveForIndexPath(uint32(i))
		if e != nil {
			t.Fatal(e)
		}
		address, _ := key2.Address()
		privateKeys, _ := key2.PrivateKey()
		fmt.Println(address.String() + "@" + privateKeys.Hex())
		//_, key, err2 := em.DeriveForIndexPath(uint32(i))
		//if err2 != nil {
		//	t.Fatal(err2)
		//}
		//assert.True(t, bytes.Equal(key2.Key, key.Key))
		//assert.True(t, bytes.Equal(key2.ChainCode, key.ChainCode))

	}

	//fmt.Println(em.GetPrimaryAddr())
	//fmt.Println(em.GetEntropyStoreFile())
}

func TestManager_GetEntropyStoreManager(t *testing.T) {
	manager := wallet.New(&wallet.Config{
		DataDir: "/Users/zhutiantao/Library/GVite/testdata/wallet/",
	})
	manager.Start()
	storeManager, err := manager.GetEntropyStoreManager("vite_b1c00ae7dfd5b935550a6e2507da38886abad2351ae78d4d9a")
	if err != nil {
		t.Fatal(err)
	}
	storeManager.Unlock("123456")
	for i := 0; i < 25; i++ {
		_, key, _ := storeManager.DeriveForIndexPath(uint32(i))
		address, _ := key.Address()
		fmt.Println(strconv.Itoa(i) + ":" + address.String())
	}
}
