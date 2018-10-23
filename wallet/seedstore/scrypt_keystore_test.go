package seedstore_test

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/tyler-smith/go-bip39"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/wallet/hd-bip/derivation"
	"github.com/vitelabs/go-vite/wallet/seedstore"
	"github.com/vitelabs/go-vite/wallet/walleterrors"
	"path/filepath"
	"testing"
)

const (
	TEST_Seed_JSON = `{"primaryAddress":"vite_ca5a606b192bc689ba8baa425553b95e3e60a5c65ef7c834b3","crypto":{"ciphername":"aes-256-gcm","ciphertext":"f47cfe193ec5c2b59b0184e22e20c5cd42926e4a5e7dc0767092cb892634160778132de02fc93606ec8b0de80ff23281b9228dc6228f1c105ba3effc0757f896354e9928c324a5a358075990e1cfeb0a","nonce":"7d8a40f1a7376ef564610160","kdf":"scrypt","scryptparams":{"n":262144,"r":8,"p":1,"keylen":32,"salt":"91acf21616a2b14268b50d509bca1c1a3722fe8ec6f2c00336c1a53699222d83"}},"seedstoreversion":1,"timestamp":1540268010}`
	TEST_Seed      = "d2755a7b11dfed9cab27d4c2db832c7bb2be755f0d4ef53c86adefc5420164c727631765efe270680d2a0870602c95b09a2c547e107bf7f7aa5de9a060fe261b"
)

func TestSeedStorePassphrase_StoreSeed(t *testing.T) {
	entropy, _ := bip39.NewEntropy(256)
	m, _ := bip39.NewMnemonic(entropy)
	fmt.Println(m)
	seed := bip39.NewSeed(m, "")
	fmt.Println("hexSeed:", hex.EncodeToString(seed))

	addresses, _ := derivation.GetPrimaryAddress(seed)

	filename := filepath.Join(common.DefaultDataDir(), "Seed", addresses.String())
	store := seedstore.SeedStorePassphrase{filename}

	err := store.StoreSeed(seed, "123456")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(filename)

}

func TestSeedStorePassphrase_ExtractSeed(t *testing.T) {

	seed, _ := hex.DecodeString("pass your seed")
	addresses, _ := derivation.GetPrimaryAddress(seed)

	filename := filepath.Join(common.DefaultDataDir(), "Seed", addresses.String())
	store := seedstore.SeedStorePassphrase{filename}

	seedExtract, err := store.ExtractSeed("123456")
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(seed, seedExtract) {
		t.Fatal("not equal")
	}

}

func TestDecryptSeed(t *testing.T) {
	seed, _ := hex.DecodeString(TEST_Seed)
	addresses, _ := derivation.GetPrimaryAddress(seed)

	json, e := seedstore.EncryptSeed(seed, *addresses, "123456")
	if e != nil {
		t.Fatal(e)
	}
	fmt.Println(string(json))

	{
		fmt.Println("success case:")
		decryptSeed, e := seedstore.DecryptSeed(json, "123456")
		if e != nil {
			t.Fatal(e)
		}

		if !bytes.Equal(seed, decryptSeed) {
			t.Fatal("not equal")
		}

	}

	{
		fmt.Println("password error case:")
		_, e := seedstore.DecryptSeed(json, "1234576")
		if e != walleterrors.ErrDecryptSeed {
			t.Fatal("no error")
		}
	}

}
