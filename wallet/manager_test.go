package wallet_test

import (
	"fmt"
	"path/filepath"
	"strconv"
	"testing"

	"encoding/hex"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/crypto"
	"github.com/vitelabs/go-vite/wallet"
)

func deskTopDir() string {
	home := common.HomeDir()
	if home != "" {
		return filepath.Join(home, "Desktop", "wallet18")
	}
	return ""
}

func TestManager_Gen18(t *testing.T) {
	manager := wallet.New(&wallet.Config{
		DataDir: deskTopDir(),
	})

	password := "66e44688e02a334d"
	var addrs []string
	for i := 0; i < 18; i++ {
		mnemonic, em, err := manager.NewMnemonicAndEntropyStore(password)
		if err != nil {
			t.Fatal(err)
		}
		fmt.Println(mnemonic)
		fmt.Println(em.GetPrimaryAddr())
		addrs = append(addrs, em.GetPrimaryAddr().String())
		fmt.Println()
		fmt.Println()
	}

	for i := 0; i < 18; i++ {
		fmt.Println(addrs[i])
	}

}

func TestManager_Gen26(t *testing.T) {
	manager := wallet.New(&wallet.Config{
		DataDir: deskTopDir(),
	})

	csprng := crypto.GetEntropyCSPRNG(8)
	password := hex.EncodeToString(csprng)
	fmt.Println(password)
	fmt.Println()
	fmt.Println()
	var addrs []string
	for i := 0; i < 26; i++ {
		mnemonic, em, err := manager.NewMnemonicAndEntropyStore(password)
		if err != nil {
			t.Fatal(err)
		}
		fmt.Println(mnemonic)
		fmt.Println(em.GetPrimaryAddr())
		em.Unlock(password)
		path, key, _ := em.DeriveForIndexPath(20181108)

		address, _ := key.Address()
		privateKey, _ := key.PrivateKey()
		fmt.Println(path, address, hex.EncodeToString(privateKey))
		addrs = append(addrs, address.String())
		fmt.Println()
		fmt.Println()
		fmt.Println()
	}

	for i := 0; i < 26; i++ {
		fmt.Println(addrs[i])
	}

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

	em.Unlock("123456")

	for i := 0; i < 10; i++ {
		_, key2, e := em.DeriveForIndexPath(uint32(i))
		if e != nil {
			t.Fatal(e)
		}
		address, _ := key2.Address()
		privateKeys, _ := key2.PrivateKey()
		fmt.Println(address.String() + "@" + privateKeys.Hex())

	}
}

func TestManager_GetEntropyStoreManager(t *testing.T) {
	manager := wallet.New(&wallet.Config{
		DataDir: "/Users/xxx/Library/GVite/testdata/wallet/",
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
