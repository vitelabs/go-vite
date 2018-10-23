package derivation

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/binary"
	"errors"
	"regexp"
	"strconv"
	"strings"

	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto/ed25519"
)

const (
	ViteAccountPrefix      = "m/44'/666666'"
	VitePrimaryAccountPath = "m/44'/666666'/0'"
	ViteAccountPathFormat  = "m/44'/666666'/%d'"
	FirstHardenedIndex     = 1 << 31 // bip 44, hardened child key mast begin with 2^32
	seedModifier           = "ed25519 blake2b seed"
)

var (
	ErrInvalidPath        = errors.New("invalid derivation path")
	ErrNoPublicDerivation = errors.New("no public derivation for ed25519")

	pathRegex = regexp.MustCompile("^m(\\/[0-9]+')+$")
)

type Key struct {
	Key       []byte
	ChainCode []byte
}

func (k Key) Address() (address *types.Address, err error) {
	pubkey, e := k.PublicKey()
	if e != nil {
		return nil, e
	}

	addr := types.PubkeyToAddress(pubkey)
	return &addr, nil
}

func (k Key) StringPair() (seed string, address string, err error) {
	seed = hex.EncodeToString(k.Key)
	pubkey, e := k.PublicKey()
	if e != nil {
		return "", "", e
	}

	address = types.PubkeyToAddress(pubkey).String()
	return seed, address, nil
}

// DeriveForPath derives key for a path in BIP-44 format and a seed.
// Ed25119 derivation operated on hardened keys only.
func DeriveForPath(path string, seed []byte) (*Key, error) {
	if !isValidPath(path) {
		return nil, ErrInvalidPath
	}

	key, err := NewMasterKey(seed)
	if err != nil {
		return nil, err
	}

	segments := strings.Split(path, "/")
	for _, segment := range segments[1:] {
		i64, err := strconv.ParseUint(strings.TrimRight(segment, "'"), 10, 32)
		if err != nil {
			return nil, err
		}

		i := uint32(i64) + FirstHardenedIndex
		key, err = key.Derive(i)
		if err != nil {
			return nil, err
		}
	}

	return key, nil
}

func DeriveWithIndex(i uint32, seed []byte) (*Key, error) {
	path := fmt.Sprintf(ViteAccountPathFormat, i)
	return DeriveForPath(path, seed)
}

func GetPrimaryAddress(seed []byte) (*types.Address, error) {
	key, e := DeriveWithIndex(0, seed)
	if e != nil {
		return nil, e
	}
	return key.Address()
}

func NewMasterKey(seed []byte) (*Key, error) {
	hmac := hmac.New(sha512.New, []byte(seedModifier))
	_, err := hmac.Write(seed)
	if err != nil {
		return nil, err
	}
	sum := hmac.Sum(nil)
	key := &Key{
		Key:       sum[:32],
		ChainCode: sum[32:],
	}
	return key, nil
}

func (k *Key) Derive(i uint32) (*Key, error) {
	// no public derivation for ed25519
	if i < FirstHardenedIndex {
		return nil, ErrNoPublicDerivation
	}

	iBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(iBytes, i)
	key := append([]byte{0x0}, k.Key...)
	data := append(key, iBytes...)

	hmac := hmac.New(sha512.New, k.ChainCode)
	_, err := hmac.Write(data)
	if err != nil {
		return nil, err
	}
	sum := hmac.Sum(nil)
	newKey := &Key{
		Key:       sum[:32],
		ChainCode: sum[32:],
	}
	return newKey, nil
}

func (k Key) PublicKey() (ed25519.PublicKey, error) {
	reader := bytes.NewReader(k.Key)
	pub, _, err := ed25519.GenerateKey(reader)
	if err != nil {
		return nil, err
	}
	return pub[:], nil
}

func (k Key) PrivateKey() (ed25519.PrivateKey, error) {
	reader := bytes.NewReader(k.Key)
	_, priv, err := ed25519.GenerateKey(reader)
	if err != nil {
		return nil, err
	}
	return priv, nil
}

func (k Key) SignData(message []byte) (pub ed25519.PublicKey, signData []byte, err error) {
	priv, e := k.PrivateKey()
	if e != nil {
		return nil, nil, e
	}
	return priv.PubByte(), ed25519.Sign(priv, message), nil
}

func (k *Key) RawSeed() [32]byte {
	var rawSeed [32]byte
	copy(rawSeed[:], k.Key[:])
	return rawSeed
}

func isValidPath(path string) bool {
	if !pathRegex.MatchString(path) {
		return false
	}

	// Check for overflows
	segments := strings.Split(path, "/")
	for _, segment := range segments[1:] {
		_, err := strconv.ParseUint(strings.TrimRight(segment, "'"), 10, 32)
		if err != nil {
			return false
		}
	}

	return true
}
