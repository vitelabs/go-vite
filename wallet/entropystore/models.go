package entropystore

import (
	"errors"
	"github.com/tyler-smith/go-bip39"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/wallet/hd-bip"
)

const (
	storeVersion = 2
)

type EntropyProfile struct {
	Entropy             []byte         `json:"-"`
	seed                []byte         `json:"-"`
	MnemonicLang        string         `json:"mnemonicLang"`
	UseTwoFactorPhrases bool           `json:"useTwoFactorPhrases"`
	PrimaryAddress      *types.Address `json:"primaryAddress"`
}

func (ep *EntropyProfile) ExtractMnemonic() (mnemonic string, err error) {
	wordList := hd_bip.GetWordList(ep.MnemonicLang)
	currentWl := bip39.GetWordList()
	if wordList[0] != currentWl[0] {
		bip39.SetWordList(wordList)
		defer bip39.SetWordList(currentWl)
	}

	m, e := bip39.NewMnemonic(ep.Entropy)
	if e != nil {
		return "", e
	}
	return m, nil
}

func (ep *EntropyProfile) GetSeed(extensionWord *string) (seed []byte, err error) {
	if ep.seed != nil {
		return ep.seed, nil
	}
	wordList := hd_bip.GetWordList(ep.MnemonicLang)
	currentWl := bip39.GetWordList()
	if wordList[0] != currentWl[0] {
		bip39.SetWordList(wordList)
		defer bip39.SetWordList(currentWl)
	}

	m, e := bip39.NewMnemonic(ep.Entropy)
	if e != nil {
		return nil, e
	}

	if ep.UseTwoFactorPhrases {
		if extensionWord != nil {
			seed = bip39.NewSeed(m, *extensionWord)
		} else {
			return nil, errors.New("empty extension word")
		}
	} else {
		seed = bip39.NewSeed(m, "")
	}

	ep.seed = seed

	return seed, nil
}

type versionAware struct {
	OldVersion *int `json:"seedstoreversion"`
	Version    *int `json:"version"`
}

type entropyJSON struct {
	PrimaryAddress string     `json:"primaryAddress"`
	Crypto         cryptoJSON `json:"crypto"`
	Version        int        `json:"seedstoreversion"`
	Timestamp      int64      `json:"timestamp"`
}

type EntropyJSONV1 struct {
	EntropyProfile
	Id        string     `json:"uuid"`
	Crypto    cryptoJSON `json:"crypto"`
	Version   int        `json:"version"`
	Timestamp int64      `json:"timestamp"`
}

type cryptoJSON struct {
	CipherName   string       `json:"ciphername"`
	CipherText   string       `json:"ciphertext"`
	Nonce        string       `json:"nonce"`
	KDF          string       `json:"kdf"`
	ScryptParams scryptParams `json:"scryptparams"`
}

type scryptParams struct {
	N      int    `json:"n"`
	R      int    `json:"r"`
	P      int    `json:"p"`
	KeyLen int    `json:"keylen"`
	Salt   string `json:"salt"`
}
