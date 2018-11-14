package entropystore

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"github.com/tyler-smith/go-bip39"
	"github.com/vitelabs/go-vite/common/types"
	vcrypto "github.com/vitelabs/go-vite/crypto"
	"github.com/vitelabs/go-vite/wallet/walleterrors"
	"golang.org/x/crypto/scrypt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"
)

const (
	// StandardScryptN is the N parameter of Scrypt encryption algorithm, using 256MB
	// memory and taking approximately 1s CPU time on a modern processor.
	StandardScryptN = 1 << 18

	// StandardScryptP is the P parameter of Scrypt encryption algorithm, using 256MB
	// memory and taking approximately 1s CPU time on a modern processor.
	StandardScryptP = 1

	// memory and taking approximately 100ms CPU time on a modern processor.
	LightScryptN = 1 << 12

	// LightScryptP is the P parameter of Scrypt encryption algorithm, using 4MB
	// memory and taking approximately 100ms CPU time on a modern processor.
	LightScryptP = 6

	scryptR      = 8
	scryptKeyLen = 32

	aesMode    = "aes-256-gcm"
	scryptName = "scrypt"
)

type CryptoStore struct {
	EntropyStoreFilename string
	UseLightScrypt       bool
}

func NewCryptoStore(entropyStoreFilename string, useLightScrypt bool) *CryptoStore {
	return &CryptoStore{
		EntropyStoreFilename: entropyStoreFilename,
		UseLightScrypt:       useLightScrypt,
	}
}

func (ks CryptoStore) ExtractSeed(passphrase string) (seed, entropy []byte, err error) {
	entropy, err = ks.ExtractEntropy(passphrase)
	if err != nil {
		return nil, nil, err
	}

	s, e := bip39.NewMnemonic(entropy)
	if e != nil {
		return nil, nil, e
	}

	return bip39.NewSeed(s, ""), entropy, nil
}

func (ks CryptoStore) ExtractEntropy(passphrase string) ([]byte, error) {
	keyjson, err := ioutil.ReadFile(ks.EntropyStoreFilename)
	if err != nil {
		return nil, err
	}

	key, err := DecryptEntropy(keyjson, passphrase)
	if err != nil {
		return nil, err
	}
	return key, nil
}

func (ks CryptoStore) StoreEntropy(entropy []byte, flagData, passphrase string) error {

	keyjson, e := EncryptEntropy(entropy, flagData, passphrase, ks.UseLightScrypt)
	if e != nil {
		return e
	}

	e = writeKeyFile(ks.EntropyStoreFilename, keyjson)
	if e != nil {
		return e
	}
	return nil
}

func DecryptEntropy(entropyJson []byte, passphrase string) ([]byte, error) {
	k, cipherData, nonce, salt, err := parseJson(entropyJson)
	if err != nil {
		return nil, err
	}
	scryptParams := k.Crypto.ScryptParams

	// begin decrypt
	derivedKey, err := scrypt.Key([]byte(passphrase), salt, scryptParams.N, scryptParams.R, scryptParams.P, scryptParams.KeyLen)
	if err != nil {
		return nil, err
	}

	entropy, err := vcrypto.AesGCMDecrypt(derivedKey[:32], cipherData, []byte(nonce))
	if err != nil {
		return nil, walleterrors.ErrDecryptEntropy
	}

	return entropy, nil
}

func EncryptEntropy(seed []byte, flagData, passphrase string, useLightScrypt bool) ([]byte, error) {
	n := StandardScryptN
	p := StandardScryptP
	if useLightScrypt {
		n = LightScryptN
		p = LightScryptP
	}
	pwdArray := []byte(passphrase)
	salt := vcrypto.GetEntropyCSPRNG(32)
	derivedKey, err := scrypt.Key(pwdArray, salt, n, scryptR, p, scryptKeyLen)
	if err != nil {
		return nil, err
	}
	encryptKey := derivedKey[:32]

	ciphertext, nonce, err := vcrypto.AesGCMEncrypt(encryptKey, seed)
	if err != nil {
		return nil, err
	}

	scryptParams := scryptParams{
		N:      n,
		R:      scryptR,
		P:      p,
		KeyLen: scryptKeyLen,
		Salt:   hex.EncodeToString(salt),
	}

	cryptoJSON := cryptoJSON{
		CipherName:   aesMode,
		CipherText:   hex.EncodeToString(ciphertext),
		Nonce:        hex.EncodeToString(nonce),
		KDF:          scryptName,
		ScryptParams: scryptParams,
	}

	encryptedKeyJSONV1 := entropyJSONV1{

		FlagData:  flagData,
		Crypto:    cryptoJSON,
		Version:   storeVersion,
		Timestamp: time.Now().UTC().Unix(),
	}

	return json.Marshal(encryptedKeyJSONV1)
}

func parseJson(keyjson []byte) (k *entropyJSON, cipherData, nonce, salt []byte, err error) {

	v := new(versionAware)
	json.Unmarshal(keyjson, v)
	if v.OldVersion != nil && *v.OldVersion == 1 || v.Version != nil && *v.Version == storeVersion {
		k = new(entropyJSON)

		// parse and check entropyJSON params
		if err := json.Unmarshal(keyjson, k); err != nil {
			return nil, nil, nil, nil, err
		}
		if k.Version != storeVersion {
			return nil, nil, nil, nil, fmt.Errorf("version number error : %v", k.Version)
		}

		if !types.IsValidHexAddress(k.PrimaryAddress) {
			return nil, nil, nil, nil, fmt.Errorf("address invalid ： %v", k.PrimaryAddress)
		}

		// parse and check  cryptoJSON params
		if k.Crypto.CipherName != aesMode {
			return nil, nil, nil, nil, fmt.Errorf("cipherName  error : %v", k.Crypto.CipherName)
		}
		if k.Crypto.KDF != scryptName {
			return nil, nil, nil, nil, fmt.Errorf("scryptName  error : %v", k.Crypto.KDF)
		}
		cipherData, err = hex.DecodeString(k.Crypto.CipherText)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		nonce, err = hex.DecodeString(k.Crypto.Nonce)
		if err != nil {
			return nil, nil, nil, nil, err
		}

		// parse and check  scryptParams params
		scryptParams := k.Crypto.ScryptParams
		salt, err = hex.DecodeString(scryptParams.Salt)
		if err != nil {
			return nil, nil, nil, nil, err
		}

		return k, cipherData, nonce, salt, nil
	}

	return nil, nil, nil, nil, errors.New("unknown version")
}

func writeKeyFile(file string, content []byte) error {

	if err := os.MkdirAll(filepath.Dir(file), 0700); err != nil {
		return err
	}

	f, err := ioutil.TempFile(filepath.Dir(file), "."+filepath.Base(file)+".tmp")
	if err != nil {
		return err
	}

	if _, err := f.Write(content); err != nil {
		f.Close()
		os.Remove(f.Name())
		return err
	}
	f.Close()
	return os.Rename(f.Name(), file)
}
