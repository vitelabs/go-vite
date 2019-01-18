package entropystore

import (
	"errors"
	"github.com/tyler-smith/go-bip39"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/wallet/hd-bip"
	"github.com/vitelabs/go-vite/wallet/hd-bip/derivation"
	"github.com/vitelabs/go-vite/wallet/walleterrors"
)

func NewMnemonic(language string, mnemonicSize *int) (string, error) {
	size := 24
	if mnemonicSize != nil {
		size = *mnemonicSize
		if size != 12 && size != 15 && size != 18 && size != 21 && size != 24 {
			return "", errors.New("wrong mnemonic size")
		}
	}

	entropySize := 32 * size / 3

	entropy, err := bip39.NewEntropy(entropySize)
	if err != nil {
		return "", err
	}

	wordList := hd_bip.GetWordList(language)
	currentWl := bip39.GetWordList()
	if wordList[0] != currentWl[0] {
		bip39.SetWordList(wordList)
		defer bip39.SetWordList(currentWl)
	}

	return bip39.NewMnemonic(entropy)

}

func MnemonicToEntropy(mnemonic, language string, useTwoFactorPhrases bool, extensionWord *string) (entropyprofile *EntropyProfile, e error) {
	wordList := hd_bip.GetWordList(language)
	currentWl := bip39.GetWordList()
	if wordList[0] != currentWl[0] {
		bip39.SetWordList(wordList)
		defer bip39.SetWordList(currentWl)
	}

	entropy, e := bip39.EntropyFromMnemonic(mnemonic)
	if e != nil {
		return nil, e
	}
	var primaryAddress *types.Address
	if useTwoFactorPhrases {
		if extensionWord != nil {
			primaryAddress, e = derivation.GetPrimaryAddress(bip39.NewSeed(mnemonic, *extensionWord))
			if e != nil {
				return nil, e
			}
		} else {
			return nil, walleterrors.ErrEmptyExtensionWord
		}
	} else {
		primaryAddress, e = derivation.GetPrimaryAddress(bip39.NewSeed(mnemonic, ""))
		if e != nil {
			return nil, e
		}
	}

	return &EntropyProfile{
		Entropy:             entropy,
		MnemonicLang:        language,
		UseTwoFactorPhrases: useTwoFactorPhrases,
		PrimaryAddress:      primaryAddress,
	}, nil
}

func FindAddrFromEntropy(entropy EntropyProfile, addr types.Address, extensionWord *string, maxSearchIndex uint32) (key *derivation.Key, index uint32, e error) {

	seed, e := entropy.GetSeed(extensionWord)
	if e != nil {
		return nil, 0, e
	}

	for i := uint32(0); i < maxSearchIndex; i++ {
		key, e := derivation.DeriveWithIndex(i, seed)
		if e != nil {
			return nil, 0, e
		}
		genAddr, e := key.Address()
		if e != nil {
			return nil, 0, e
		}
		if addr == *genAddr {
			return key, i, nil
		}
	}
	return nil, 0, walleterrors.ErrAddressNotFound
}
