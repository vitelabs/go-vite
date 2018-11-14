package entropystore_test

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/tyler-smith/go-bip39"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/wallet/entropystore"
	"github.com/vitelabs/go-vite/wallet/hd-bip/derivation"
	"github.com/vitelabs/go-vite/wallet/walleterrors"
	"path/filepath"
	"testing"
)

var (
	oldjson = "{\"primaryAddress\":\"vite_398710d31eebf921561a3db5b404c8cbed5d5fe4133ddeaf03\",\"crypto\":{\"ciphername\":\"aes-256-gcm\",\"ciphertext\":\"32cd74dcaf4d3e24217ca801c890a710ca56ebad6497001b0d6ed9499363f80fa7597679aec77f01b105fee0ef5cec20\",\"nonce\":\"7db3f7919ee9c78bf2dd55d9\",\"kdf\":\"scrypt\",\"scryptparams\":{\"n\":4096,\"r\":8,\"p\":6,\"keylen\":32,\"salt\":\"c08901933bceb2de2ebfaf33acd2e033eca3f2f268c6d9fda4259a252e9af2e0\"}},\"seedstoreversion\":1,\"timestamp\":1542180917}"
	newjson = "{\"primaryAddress\":\"vite_398710d31eebf921561a3db5b404c8cbed5d5fe4133ddeaf03\",\"crypto\":{\"ciphername\":\"aes-256-gcm\",\"ciphertext\":\"32cd74dcaf4d3e24217ca801c890a710ca56ebad6497001b0d6ed9499363f80fa7597679aec77f01b105fee0ef5cec20\",\"nonce\":\"7db3f7919ee9c78bf2dd55d9\",\"kdf\":\"scrypt\",\"scryptparams\":{\"n\":4096,\"r\":8,\"p\":6,\"keylen\":32,\"salt\":\"c08901933bceb2de2ebfaf33acd2e033eca3f2f268c6d9fda4259a252e9af2e0\"}},\"version\":1,\"timestamp\":1542180917}"
)

type versionAware struct {
	OldVersion *int `json:"seedstoreversion"`
	Version    *int `json:"version"`
}

func TestVersionAware(t *testing.T) {
	{
		v := new(versionAware)
		json.Unmarshal([]byte(oldjson), v)
		if v.OldVersion == nil {
			t.Fail()
		}
	}

	{
		v := new(versionAware)
		json.Unmarshal([]byte(newjson), v)
		if v.Version == nil {
			t.Fail()
		}
	}

}

func TestCryptoStore_StoreEntropy(t *testing.T) {
	entropy, _ := bip39.NewEntropy(256)
	m, _ := bip39.NewMnemonic(entropy)
	fmt.Println(m)
	seed := bip39.NewSeed(m, "")
	fmt.Println("hexSeed:", hex.EncodeToString(seed))

	addresses, _ := derivation.GetPrimaryAddress(seed)

	filename := filepath.Join(common.DefaultDataDir(), "UTSeed", addresses.String())
	store := entropystore.NewCryptoStore(filename, true)

	err := store.StoreEntropy(entropy, addresses.String(), "123456")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(filename)

}

func TestCryptoStore_ExtractSeed(t *testing.T) {

	seed, _ := hex.DecodeString("pass your seed")
	addresses, _ := derivation.GetPrimaryAddress(seed)

	filename := filepath.Join(common.DefaultDataDir(), "UTSeed", addresses.String())
	store := entropystore.NewCryptoStore(filename, true)

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

	json, e := entropystore.EncryptEntropy(entropy, *addresses, "123456", true)
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
