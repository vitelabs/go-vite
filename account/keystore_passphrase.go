package account

import (
	"encoding/hex"
	"fmt"
	"time"
	"go-vite/common"
	"io/ioutil"
	"github.com/pborman/uuid"
	"go-vite/crypto/ed25519"
	"path/filepath"
	"os"
	"golang.org/x/crypto/scrypt"
	vcrypto "go-vite/crypto"
	"encoding/json"
)

const (
	keyHeaderKDF = "scrypt"

	// StandardScryptN is the N parameter of Scrypt encryption algorithm, using 256MB
	// memory and taking approximately 1s CPU time on a modern processor.
	StandardScryptN = 1 << 18

	// StandardScryptP is the P parameter of Scrypt encryption algorithm, using 256MB
	// memory and taking approximately 1s CPU time on a modern processor.
	StandardScryptP = 1

	// LightScryptN is the N parameter of Scrypt encryption algorithm, using 4MB
	// memory and taking approximately 100ms CPU time on a modern processor.
	LightScryptN = 1 << 12

	// LightScryptP is the P parameter of Scrypt encryption algorithm, using 4MB
	// memory and taking approximately 100ms CPU time on a modern processor.
	LightScryptP = 6

	scryptR     = 8
	scryptDKLen = 32
)

type keyStorePassphrase struct {
	keysDirPath string
}

func (ks keyStorePassphrase) ExtractKey(addr common.Address, auth string) (*Key, error) {
	keyjson, err := ioutil.ReadFile(ks.fullKeyFileName(addr))
	if err != nil {
		return nil, err
	}
	key, err := DecryptKey(keyjson, auth)
	if err != nil {
		return nil, err
	}
	// Make sure we're really operating on the requested key (no swap attacks)
	if key.Address != addr {
		return nil, fmt.Errorf("key content mismatch: have account %x, want %x", key.Address, addr)
	}
	return key, nil
}

func (ks keyStorePassphrase) StoreKey(key *Key, password string) error {
	keyjson, err := EncryptKey(key, password)
	if err != nil {
		return err
	}
	return writeKeyFile(ks.fullKeyFileName(key.Address), keyjson)
}

func DecryptKey(keyjson []byte, password string) (*Key, error) {
	k := new(encryptedKeyJSON)
	priv := new(ed25519.PrivateKey)

	return &Key{
		Id:         uuid.UUID(k.Id),
		Address:    common.PubkeyToAddress([]byte(k.Crypto.CipherText)),
		PrivateKey: priv,
	}, nil
}

func EncryptKey(key *Key, password string) ([]byte, error) {
	n := StandardScryptN
	p := StandardScryptP
	pwdArray := []byte(password)
	salt := vcrypto.GetEntropyCSPRNG(32)
	derivedKey, err := scrypt.Key(pwdArray, salt, n, scryptR, p, scryptDKLen)
	if err != nil {
		return nil, err
	}
	encryptKey := derivedKey[:32]

	ciphertext, nonce, err := vcrypto.AesGCMEncrypt(encryptKey, *key.PrivateKey)
	if err != nil {
		return nil, err
	}

	scryptParamsJSON := make(map[string]interface{}, 5)
	scryptParamsJSON["n"] = n
	scryptParamsJSON["r"] = scryptR
	scryptParamsJSON["p"] = p
	scryptParamsJSON["dklen"] = scryptDKLen
	scryptParamsJSON["salt"] = hex.EncodeToString(salt)

	cryptoJSON := cryptoJSON{
		Cipher:     "aes-128-gcm",
		CipherText: hex.EncodeToString(ciphertext),
		Nonce:      hex.EncodeToString(nonce),
		KDF:        "scrypt",
		KDFParams:  scryptParamsJSON,
	}

	encryptedKeyJSON := encryptedKeyJSON{

		Address: hex.EncodeToString(key.Address[:]),
		Crypto:  cryptoJSON,
		Id:      key.Id.String(),
		Version: version,
	}
	return json.Marshal(encryptedKeyJSON)
}

func writeKeyFile(file string, content []byte) error {
	// Create the keystore directory with appropriate permissions
	// in case it is not present yet.
	const dirPerm = 0700
	if err := os.MkdirAll(filepath.Dir(file), dirPerm); err != nil {
		return err
	}
	// Atomic write: create a temporary hidden file first
	// then move it into place. TempFile assigns mode 0600.
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

// todo NOW we use Eth file name rule to generate filename, In Late we will use our custom rule
func (ks keyStorePassphrase) fullKeyFileName(keyAddr common.Address) string {
	ts := time.Now().UTC()
	return ks.keysDirPath + "/" + fmt.Sprintf("UTC--%s--%s", toISO8601(ts), hex.EncodeToString(keyAddr[:]))
}

func toISO8601(t time.Time) string {
	var tz string
	name, offset := t.Zone()
	if name == "UTC" {
		tz = "Z"
	} else {
		tz = fmt.Sprintf("%03d00", offset/3600)
	}
	return fmt.Sprintf("%04d-%02d-%02dT%02d-%02d-%02d.%09d%s", t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), tz)
}
