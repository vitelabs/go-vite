package entropystore_test

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/tyler-smith/go-bip39"

	walleterrors "github.com/vitelabs/go-vite/v2/common/errors"
	"github.com/vitelabs/go-vite/v2/common/fileutils"
	"github.com/vitelabs/go-vite/v2/wallet/entropystore"
	"github.com/vitelabs/go-vite/v2/wallet/hd-bip/derivation"
)

func TestCryptoStore_StoreEntropy(t *testing.T) {
	entropy, _ := bip39.NewEntropy(256)
	m, _ := bip39.NewMnemonic(entropy)
	fmt.Println(m)
	seed := bip39.NewSeed(m, "")
	fmt.Println("hexSeed:", hex.EncodeToString(seed))

	primaryAddr, _ := derivation.GetPrimaryAddress(seed)

	filename := filepath.Join(fileutils.CreateTempDir(), primaryAddr.String())
	store := entropystore.CryptoStore{filename}

	err := store.StoreEntropy(entropy, *primaryAddr, "123456")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(filename)

	testCryptoStore_ExtractSeed(t, filename, seed)
}

func testCryptoStore_ExtractSeed(t *testing.T, filename string, seed []byte) {
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

func TestCryptoStore_ExtractMnemonic(t *testing.T) {
	store := entropystore.CryptoStore{"filename"}

	seed, entropy, _ := store.ExtractSeed("password")
	fmt.Println(bip39.NewMnemonic(entropy))

	key, _ := derivation.DeriveWithIndex(0, seed)
	address, _ := key.Address()
	fmt.Println(address)
}

func TestDecryptEntropy(t *testing.T) {
	entropy, _ := hex.DecodeString(TestEntropy)
	mnemonic, e := bip39.NewMnemonic(entropy)
	if e != nil {
		t.Fatal(e)
	}
	seed := bip39.NewSeed(mnemonic, "")

	primaryAddr, _ := derivation.GetPrimaryAddress(seed)

	json, e := entropystore.EncryptEntropy(entropy, *primaryAddr, "123456")
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
		fmt.Println("passphrase error case:")
		_, e := entropystore.DecryptEntropy(json, "1234576")
		if e != walleterrors.ErrDecryptEntropy {
			t.Fatal("no error")
		}
	}
}
