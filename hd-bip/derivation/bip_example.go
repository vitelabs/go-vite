package derivation

import (
	"fmt"
	"github.com/tyler-smith/go-bip39"
	"encoding/hex"
)

func RandomMnemonic12(passphrase string) error {
	fmt.Println("RandomMnemonic12:")
	entropy, _ := bip39.NewEntropy(128)
	return Menmonic(entropy, passphrase)
}

func RandomMnemonic24(passphrase string) error {
	fmt.Println("RandomMnemonic24:")
	entropy, _ := bip39.NewEntropy(256)
	return Menmonic(entropy, passphrase)
}

func Menmonic(entropy []byte, passphrase string) error {

	mnemonic, _ := bip39.NewMnemonic(entropy)
	fmt.Println(mnemonic)

	fmt.Println("entropy")
	fmt.Println(hex.EncodeToString(entropy))

	if passphrase != "" {
		fmt.Println("passphrase:")
		fmt.Println(passphrase)
	}
	seed := bip39.NewSeed(mnemonic, passphrase)
	fmt.Println("Seed hex:")
	fmt.Println(hex.EncodeToString(seed))

	key, e := NewMasterKey(seed)
	if e != nil {
		return e
	}
	fmt.Println("primary key hex:")
	fmt.Println(hex.EncodeToString(key.Key))

	fmt.Println("Accounts:")
	for i := 0; i < 10; i++ {
		path := fmt.Sprintf(ViteAccountPathFormat, i)
		k, e := DeriveForPath(path, seed)
		//k, e := key.Derive(uint32(i))
		if e != nil {
			return e
		}
		seed, address, err := k.StringPair()
		if err != nil {
			return e
		}
		fmt.Println(path, seed, " "+address)

	}
	return nil
}
