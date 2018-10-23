package entropystore

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/tyler-smith/go-bip39"
	"github.com/vitelabs/go-vite/common/types"
	vcrypto "github.com/vitelabs/go-vite/crypto"
	"github.com/vitelabs/go-vite/wallet/hd-bip/derivation"
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

	scryptR      = 8
	scryptKeyLen = 32

	aesMode    = "aes-256-gcm"
	scryptName = "scrypt"
)

type CryptoStore struct {
	EntropyStoreFilename string
}

func (ks CryptoStore) ExtractSeed(password string) (seed, entropy []byte, err error) {
	entropy, err = ks.ExtractEntropy(password)
	if err != nil {
		return nil, nil, err
	}

	s, e := bip39.NewMnemonic(entropy)
	if e != nil {
		return nil, nil, e
	}

	return bip39.NewSeed(s, ""), entropy, nil
}

func (ks CryptoStore) ExtractEntropy(password string) ([]byte, error) {
	keyjson, err := ioutil.ReadFile(ks.EntropyStoreFilename)
	if err != nil {
		return nil, err
	}

	key, err := DecryptEntropy(keyjson, password)
	if err != nil {
		return nil, err
	}
	return key, nil
}

func (ks CryptoStore) StoreEntropy(entropy []byte, primaryAddr types.Address, password string) error {

	keyjson, e := EncryptEntropy(entropy, primaryAddr, password)
	if e != nil {
		return e
	}

	e = writeKeyFile(ks.EntropyStoreFilename, keyjson)
	if e != nil {
		return e
	}
	return nil
}

func parseJson(keyjson []byte) (k *entropyJSON, kAddress *types.Address, cipherData, nonce, salt []byte, err error) {
	k = new(entropyJSON)
	// parse and check entropyJSON params
	if err := json.Unmarshal(keyjson, k); err != nil {
		return nil, nil, nil, nil, nil, err
	}
	if k.Version != cryptoStoreVersion {
		return nil, nil, nil, nil, nil, fmt.Errorf("version number error : %v", k.Version)
	}

	if !types.IsValidHexAddress(k.PrimaryAddress) {
		return nil, nil, nil, nil, nil, fmt.Errorf("address invalid ： %v", k.PrimaryAddress)
	}
	addr, err := types.HexToAddress(k.PrimaryAddress)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	// parse and check  cryptoJSON params
	if k.Crypto.CipherName != aesMode {
		return nil, nil, nil, nil, nil, fmt.Errorf("cipherName  error : %v", k.Crypto.CipherName)
	}
	if k.Crypto.KDF != scryptName {
		return nil, nil, nil, nil, nil, fmt.Errorf("scryptName  error : %v", k.Crypto.KDF)
	}
	cipherData, err = hex.DecodeString(k.Crypto.CipherText)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	nonce, err = hex.DecodeString(k.Crypto.Nonce)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	// parse and check  scryptParams params
	scryptParams := k.Crypto.ScryptParams
	salt, err = hex.DecodeString(scryptParams.Salt)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	return k, &addr, cipherData, nonce, salt, nil
}

func DecryptEntropy(entropyJson []byte, password string) ([]byte, error) {
	k, kAddress, cipherData, nonce, salt, err := parseJson(entropyJson)
	if err != nil {
		return nil, err
	}
	scryptParams := k.Crypto.ScryptParams

	// begin decrypt
	derivedKey, err := scrypt.Key([]byte(password), salt, scryptParams.N, scryptParams.R, scryptParams.P, scryptParams.KeyLen)
	if err != nil {
		return nil, err
	}

	entropy, err := vcrypto.AesGCMDecrypt(derivedKey[:32], cipherData, []byte(nonce))
	if err != nil {
		return nil, walleterrors.ErrDecryptSeed
	}

	mnemonic, e := bip39.NewMnemonic(entropy)
	if e != nil {
		return nil, e
	}
	seed := bip39.NewSeed(mnemonic, "")

	generateAddr, e := derivation.GetPrimaryAddress(seed)
	if e != nil {
		return nil, e
	}
	if !bytes.Equal(generateAddr[:], kAddress[:]) {
		return nil,
			fmt.Errorf("address content not equal. In file it is : %s  but generated is : %s",
				k.PrimaryAddress, generateAddr.Hex())
	}

	return entropy, nil
}

func EncryptEntropy(seed []byte, addr types.Address, password string) ([]byte, error) {
	n := StandardScryptN
	p := StandardScryptP
	pwdArray := []byte(password)
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

	ScryptParams := scryptParams{
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
		ScryptParams: ScryptParams,
	}

	encryptedKeyJSON := entropyJSON{

		PrimaryAddress: addr.String(),
		Crypto:         cryptoJSON,
		Version:        cryptoStoreVersion,
		Timestamp:      time.Now().UTC().Unix(),
	}

	return json.Marshal(encryptedKeyJSON)
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
