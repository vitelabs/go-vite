package wallet_test

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
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

func TestManager_NewMnemonicAndSeedStore2(t *testing.T) {

	for i := 1; i <= 25; i++ {
		manager := wallet.New(&wallet.Config{
			DataDir: fmt.Sprintf("/Users/jie/Documents/vite/src/github.com/vitelabs/cluster1/ledger_datas/ledger_%d/devdata/wallet", i),
		})
		mnemonic, em, err := manager.NewMnemonicAndEntropyStore("123456")
		if err != nil {
			t.Fatal(err)
		}
		fmt.Println(mnemonic, ",", em.GetEntropyStoreFile())
	}
}

func TestManager_NewMnemonicAndSeedStore21(t *testing.T) {
	for i := 1; i <= 5; i++ {
		manager := wallet.New(&wallet.Config{
			DataDir: fmt.Sprintf("/Users/jie/Documents/vite/src/github.com/vitelabs/cluster1/ledger_datas/ledger_%d/devdata/wallet", i),
		})
		manager.Start()
		files := manager.ListAllEntropyFiles()
		for _, v := range files {
			storeManager, err := manager.GetEntropyStoreManager(v)
			if err != nil {
				panic(err)
			}
			storeManager.Unlock("123456")
			//_, key, err := storeManager.DeriveForIndexPath(0)
			//addr, err := key.Address()
			//if err != nil {
			//	panic(err)
			//}
			//
			//fmt.Printf("%s,\n", addr)

			N := uint32(10)
			fmt.Printf("%s\n", v)
			for i := uint32(0); i < N; i++ {
				_, key, err := storeManager.DeriveForIndexPath(i)
				if err != nil {
					panic(err)
				}
				addr, err := key.Address()
				if err != nil {
					panic(err)
				}
				fmt.Printf("\t\"%s\":{\"tti_5649544520544f4b454e6e40\":100000000000000000000000000},\n", addr)
				//fmt.Printf("\t\"%s\":1000000000000000000000,\n", addr)
			}
			fmt.Printf("\n")
		}
	}
}

func TestManager_NewMnemonicAndSeedStore3(t *testing.T) {
	manager := wallet.New(&wallet.Config{
		DataDir: fmt.Sprintf("/Users/jie/Documents/vite/src/github.com/vitelabs/cluster1/tmpWallet"),
	})
	for i := 1; i <= 5; i++ {
		mnemonic, em, err := manager.NewMnemonicAndEntropyStore("123456")
		if err != nil {
			t.Fatal(err)
		}
		fmt.Println(mnemonic, ",", em.GetEntropyStoreFile())
	}
}

func TestManager_NewMnemonicAndSeedStore4(t *testing.T) {
	manager := wallet.New(&wallet.Config{
		DataDir: fmt.Sprintf("/Users/jie/Documents/vite/src/github.com/vitelabs/cluster1/tmpWallet"),
	})
	manager.Start()
	files := manager.ListAllEntropyFiles()
	for _, v := range files {
		storeManager, err := manager.GetEntropyStoreManager(v)
		if err != nil {
			panic(err)
		}
		storeManager.Unlock("123456")
		_, key, err := storeManager.DeriveForIndexPath(0)
		if err != nil {
			panic(err)
		}
		keys, err := key.PrivateKey()
		if err != nil {
			panic(err)
		}
		addr, err := key.Address()
		if err != nil {
			panic(err)
		}
		fmt.Printf("%s,0:%s,%s\n", addr, addr, keys.Hex())
	}
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

func TestRead(t *testing.T) {
	hexPath := "020000000d000000000000000000000000"

	byt, err := hex.DecodeString(hexPath)

	assert.NoError(t, err)

	t.Log(len(byt))
	length := uint8(byt[0])
	t.Log(length)
	var nums []uint32
	nums = append(nums, binary.BigEndian.Uint32(byt[1:5]))
	nums = append(nums, binary.BigEndian.Uint32(byt[6:10]))
	nums = append(nums, binary.BigEndian.Uint32(byt[11:15]))
	nums = append(nums, binary.BigEndian.Uint32(byt[16:20]))
	t.Log(nums)

	path := "m"

	for i := 0; i < int(length); i++ {
		path += fmt.Sprintf("/%d", nums[i])
	}

	t.Log(path)

}
