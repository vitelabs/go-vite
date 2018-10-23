package entropystore_test

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/tyler-smith/go-bip39"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/wallet/entropystore"
	"github.com/vitelabs/go-vite/wallet/hd-bip/derivation"
	"github.com/vitelabs/go-vite/wallet/walleterrors"
	"path/filepath"
	"testing"
)


func TestCryptoStore_StoreEntropy(t *testing.T) {
	entropy, _ := bip39.NewEntropy(256)
	m, _ := bip39.NewMnemonic(entropy)
	fmt.Println(m)
	seed := bip39.NewSeed(m, "")
	fmt.Println("hexSeed:", hex.EncodeToString(seed))

	addresses, _ := derivation.GetPrimaryAddress(seed)

	filename := filepath.Join(common.DefaultDataDir(), "UTSeed", addresses.String())
	store := entropystore.CryptoStore{filename}

	err := store.StoreEntropy(entropy, *addresses, "123456")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(filename)

}

func TestCryptoStore_ExtractSeed(t *testing.T) {

	seed, _ := hex.DecodeString("pass your seed")
	addresses, _ := derivation.GetPrimaryAddress(seed)

	filename := filepath.Join(common.DefaultDataDir(), "UTSeed", addresses.String())
	store := entropystore.CryptoStore{filename}

	seedExtract, entropy, err := store.ExtractSeed("123456")
	fmt.Println(bip39.NewMnemonic(entropy))
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(seed, seedExtract) {
		t.Fatal("not equal")
	}
}

func TestDecryptEntropy(t *testing.T) {
	entropy, _ := hex.DecodeString(TestEntropy)
	mnemonic, e := bip39.NewMnemonic(entropy)
	if e != nil {
		t.Fatal(e)
	}
	seed := bip39.NewSeed(mnemonic, "")

	addresses, _ := derivation.GetPrimaryAddress(seed)

	json, e := entropystore.EncryptEntropy(entropy, *addresses, "123456")
	if e != nil {
		t.Fatal(e)
	}
	fmt.Println(string(json))

	{
		fmt.Println("success case:")
		decryptentropy, e := entropystore.DecryptEntropy(json, "123456")
		if e != nil {
			t.Fatal(e)
		}

		if !bytes.Equal(entropy, decryptentropy) {
			t.Fatal("not equal")
		}

	}

	{
		fmt.Println("password error case:")
		_, e := entropystore.DecryptEntropy(json, "1234576")
		if e != walleterrors.ErrDecryptSeed {
			t.Fatal("no error")
		}
	}

}
