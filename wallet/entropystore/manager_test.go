package entropystore_test

import (
	"encoding/hex"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/tyler-smith/go-bip39"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/wallet/entropystore"
	"github.com/vitelabs/go-vite/wallet/walleterrors"
	"path/filepath"
	"testing"
)

type testBipTuple struct {
	path    string
	seed    string
	address string
}

const (
	TestEntropy  = "d6456de91526eb086ce7f5ad968a64690d171def52cfc9a887e682acbef44e20"
	TestMnemonic = "stone clock kid clean huge loud receive wrong pulse reform october spirit sphere moment run fly situate during whale aim slogan kick decade alpha"
	TestSeed     = "f2b7e88dbd85bc954725fe7d2204618b7a009141c9c6ee664d497865df2f88ade65cb983e88ff8e9644d615d5c324f7bbd9928083e042bcc4b3e55fe297f3bf3"
)

var (
	testTuples = []testBipTuple{
		{"m/44'/666666'/0'", "678d04f47ae63a09ca745a7dcaa75ba6481ad329ba0c617e849bf854f2b9c55a", "vite_80d446de0b3267b03cba7d9b49afa5e71341c7cd0693651ad1"},
		{"m/44'/666666'/1'", "fa42e7cbf7f163f1121cde06d50c1f577e21c51dcd3581ce503df0dbde553ec6", "vite_c5947d16a449ee17e14eb0dc37e702c43b9e8d8b2553b8a801"},
		{"m/44'/666666'/2'", "daee6d8bfec72f3f7b662d2a1ef854a5b1080596dc51508b6972a179ed75df32", "vite_0fc0e0a4d1d761f96ea636f1ff93283a13dfd67e4998449ce8"},
		{"m/44'/666666'/3'", "e11e885d6d73ee04d270fae0bafccfcfa49a3b3021f19d91198d06fb16ab22c6", "vite_7258aefb6850b77cd1a63d862f7b6743a376e5737c54685249"},
		{"m/44'/666666'/4'", "a521d6d0a336e9e26ee8e89caebb6b0c84b92f897f03bcb798575c415d4ab681", "vite_c4f25d2172c9e42d6571a91f2c4f0031ebfd56f30b702684b2"},
		{"m/44'/666666'/5'", "35ab31d40f9e1908b01dcddf9b1f85c403a5a86d2317aabcc3b5a0bcceb8e257", "vite_434846b1293c0544be1bf9f2d62cb7ff1693fb917e495322b0"},
		{"m/44'/666666'/6'", "74d9081480b93278bb297c12dc1056e8828aa4236562553b810be36cd0c03a6b", "vite_2db685bf546ec32361d135874cccd86690b88f0aa4ec261efc"},
		{"m/44'/666666'/7'", "546e5992bdb5be1f00f11d3d578c665d152fa451cc71bccb4eec427c3509722d", "vite_755ffae48b56ef80b265e694faa9fab0f0b16e95594c8aecb9"},
		{"m/44'/666666'/8'", "ca1f60901d08fb2950a19167bcb386b22d991087eb7e934df4d6f862ed6f8d0a", "vite_c00414ba378cd5905d7752dae32b3d2cf675f2718a0f8798f0"},
	}

	seedToChild map[string][]testBipTuple

	testSeedStoreManager *entropystore.Manager

	utFilePath = filepath.Join(common.DefaultDataDir(), "UTSeed")
)

func init() {
	seedToChild := make(map[string][]testBipTuple)
	seedToChild[TestSeed] = testTuples

	testSeedStoreManager, _ = entropystore.StoreNewEntropy(utFilePath, TestMnemonic, "123456", entropystore.DefaultMaxIndex)

}

func GetManagerFromStoreNewSeed() *entropystore.Manager {
	entropy, _ := bip39.NewEntropy(256)
	fmt.Println("entropy:", hex.EncodeToString(entropy))
	mnemonic, _ := bip39.NewMnemonic(entropy)
	fmt.Println(mnemonic)
	seed := bip39.NewSeed(mnemonic, "")
	fmt.Println("seed   :", hex.EncodeToString(seed))

	manager, e := entropystore.StoreNewEntropy(utFilePath, mnemonic, "123456", entropystore.DefaultMaxIndex)
	if e != nil {
		panic(e)
		return nil
	}

	return manager
}

func TestStoreNewSeed(t *testing.T) {
	manager := GetManagerFromStoreNewSeed()
	fmt.Println(manager.EntropyStoreFile())
	for i := uint32(0); i < 10; i++ {
		path, key, e := manager.DeriveForIndexPathWithPassphrase(i, "123456")
		if e != nil {
			t.Fatal(e)
		}
		seed, address, err := key.StringPair()
		if err != nil {
			t.Fatal(err)
		}
		fmt.Println(path, seed, address)
	}
}

func TestManager_FindAddr(t *testing.T) {
	for _, tuples := range seedToChild {
		for k, tuple := range tuples {
			addr, _ := types.HexToAddress(tuple.address)
			key, u, e := testSeedStoreManager.FindAddrWithPassword("123456", addr)
			if e != nil {
				t.Fatal(e)
			}
			rs := key.RawSeed()
			assert.Equal(t, hex.EncodeToString(rs[:]), tuple.seed)
			assert.Equal(t, k, u)
			gAddr, e := key.Address()
			if e != nil {
				t.Fatal(e)
			}
			assert.Equal(t, *gAddr, addr)
		}
	}

	addresses, _, _ := types.CreateAddress()
	_, _, e := testSeedStoreManager.FindAddrWithPassword("123456", addresses)
	if e != walleterrors.ErrNotFind {
		t.Fatal(e)
	}

}

func TestManager_LockAndUnlock(t *testing.T) {
	sm := testSeedStoreManager

	sm.AddLockEventListener(func(event entropystore.UnlockEvent) {
		fmt.Println("receive an event:", event.String())
	})

	_, e := sm.ListAddress(10)
	if e == nil {
		t.Fatal("need error")
	}
	fmt.Println(e)
	e = sm.Unlock("123456")
	if e != nil {
		t.Fatal(e)
	}
	addr, e := sm.ListAddress(10)
	if e != nil {
		t.Fatal(e)
	}
	for i, v := range addr {
		if !sm.IsAddrUnlocked(*v) {
			t.Fatal("expect unlock")
		}
		fmt.Println(i, v.String())
	}
	{
		path, key, _ := sm.DeriveForIndexPath(101)
		seed, addrStr, _ := key.StringPair()
		addr, _ := key.Address()
		fmt.Println(path, seed, addrStr)
		_, _, e := sm.FindAddr(*addr)
		fmt.Println(e)
		if e != walleterrors.ErrNotFind {
			t.Fatal("expect not found error")
		}

		dsm, _ := entropystore.StoreNewEntropy(utFilePath, TestMnemonic, "123456", 200)
		dsm.Unlock("123456")
		k, i, e := dsm.FindAddr(*addr)
		if e != nil {
			t.Fatal(e)
		}
		na, _ := k.Address()
		assert.Equal(t, *na, *addr)
		assert.Equal(t, uint32(101), i)
	}

	sm.Lock()

}
