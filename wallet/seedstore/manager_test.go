package seedstore_test

import (
	"encoding/hex"
	"fmt"
	"github.com/tyler-smith/go-bip39"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/wallet/seedstore"
	"path/filepath"
	"testing"
	"github.com/vitelabs/go-vite/common/types"
)

type testBipTuple struct {
	path    string
	seed    string
	address string
}

const (
	TestSeed = "7a12b142ebc229e1159606d4df676640a8e8945d9aa9e144e0f057981de20159a39903b766cc7f1075a0e236f80050814773f741b3a611710d7bf3113bfe5947"
)

var (
	testTuples = []testBipTuple{
		{"m/44'/666666'/0'", "f62a06ca91715380b8cb08bcf314b4fbd5dcd980e8c9dbe7f0a03ed5c48feb55", "vite_2e7e60fbc82d2e0dae85819051d1803ef9d55b02193d1649dc"},
		{"m/44'/666666'/1'", "34039594a0a05f8d0c50380db2b0eee039241e981e8821449476e1db8b345e7c", "vite_68eb6a19340127801a1914158ba5d25b42ed643846bccb518e"},
		{"m/44'/666666'/2'", "bc85b00ac6f0d98fb49c1c7df706a229d5bac23643155a83ed6a274d598f877c", "vite_474eeed991618615b2b0c802a2081a0872c48084a09517f349"},
		{"m/44'/666666'/3'", "5255b6797ba9913b798708618823ff34258bd77891fb0d7b4d406937abea031b", "vite_4fb028c5b6f145a14a5348dfdce44a2104cdb0440d48256730"},
		{"m/44'/666666'/4'", "2d21bb78c0b3d94ec51ba9801fcb948445dc50dfed3388dd59b3b4cb3f14bc9b", "vite_92e88c4fcde663bc579e6c48a80de91c95b2cae015f9b35f50"},
		{"m/44'/666666'/5'", "b9e50a1ae5f1aba78853d4c902517121dabb2dd26a8adc258d8d9356a9337abc", "vite_d232fd336d317e88bb5e70e96131ae664f7c3135f086ddad1b"},
		{"m/44'/666666'/6'", "ab214c7f2240ab2d0221903f521c330258ae51a57ce6040cdcf55a21520720db", "vite_a717e57a1be279efdaeb7ebf8d558786d525020aacf59c9ba9"},
		{"m/44'/666666'/7'", "a82b7154791a5e1dfd3dd99e9b92f223fef52995a83db75f992ce542a111bef3", "vite_c1ca31665fb193432efa6d20a4747ce5f4a3b7b7bb164420a1"},
		{"m/44'/666666'/8'", "4e95fd2876831c0a5af1b6e05368257c1b09dbb9e373af2ef7f6e9cf7eb5585b", "vite_a7b855f6a5c4e47d60d4633b538145a187849c12bc7b40f238"},
	}

	seedToChild map[string][]testBipTuple

	testSeedStoreManager *seedstore.Manager

	utFilePath = filepath.Join(common.DefaultDataDir(), "UTSeed")
)

func init() {
	seedToChild[TestSeed] = testTuples

	bytes, _ := hex.DecodeString(TEST_Seed)
	testSeedStoreManager, _ = seedstore.StoreNewSeed(utFilePath, bytes, "123456")

}

func GetManagerFromStoreNewSeed() *seedstore.Manager {
	entropy, _ := bip39.NewEntropy(256)
	mnemonic, _ := bip39.NewMnemonic(entropy)
	fmt.Println(mnemonic)
	seed := bip39.NewSeed(mnemonic, "")
	fmt.Println("seed:", hex.EncodeToString(seed))

	manager, e := seedstore.StoreNewSeed(utFilePath, seed, "123456")
	if e != nil {
		panic(e)
		return nil
	}

	return manager
}

func TestStoreNewSeed(t *testing.T) {
	manager := GetManagerFromStoreNewSeed()
	fmt.Println(manager.SeedStoreFile())
	for i := uint32(0); i < 10; i++ {
		path, key, e := manager.DeriveForIndexPath(i, "123456")
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
	for seed, value := range seedToChild {
		testSeedStoreManager.FindAddr("123456")
	}

	addresses, _ := types.HexToAddress("vite_a7b855f6a5c4e47d60d4633b538145a187849c12bc7b40f238")
	testSeedStoreManager.FindAddr("123456", addresses)
}
