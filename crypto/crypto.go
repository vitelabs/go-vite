package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	crand "crypto/rand"
	"errors"
	"io"
	"strconv"

	"github.com/vitelabs/go-vite/crypto/ed25519"
	"golang.org/x/crypto/curve25519"
)

const (
	gcmAdditionData = "vite"
)

/**
func (ecdh25519) ComputeSecret(private crypto.PrivateKey, peersPublic crypto.PublicKey) (secret []byte) {
	var sec, pri, pub [32]byte
	if ok := checkType(&pri, private); !ok {
		panic("ecdh: unexpected type of private key")
	}
	if ok := checkType(&pub, peersPublic); !ok {
		panic("ecdh: unexpected type of peers public key")
	}

	curve25519.ScalarMult(&sec, &pri, &pub)

	secret = sec[:]
	return
}

func checkType(key *[32]byte, typeToCheck interface{}) (ok bool) {
	switch t := typeToCheck.(type) {
	case [32]byte:
		copy(key[:], t[:])
		ok = true
	case *[32]byte:
		copy(key[:], t[:])
		ok = true
	case []byte:
		if len(t) == 32 {
			copy(key[:], t)
			ok = true
		}
	case *[]byte:
		if len(*t) == 32 {
			copy(key[:], *t)
			ok = true
		}
	}
	return
}

*/

func checkType(key *[32]byte, typeToCheck interface{}) (ok bool) {
	switch t := typeToCheck.(type) {
	case [32]byte:
		copy(key[:], t[:])
		ok = true
	case *[32]byte:
		copy(key[:], t[:])
		ok = true
	case []byte:
		if len(t) == 32 {
			copy(key[:], t)
			ok = true
		}
	case *[]byte:
		if len(*t) == 32 {
			copy(key[:], *t)
			ok = true
		}
	}
	return
}

func X25519ComputeSecret(private []byte, peersPublic []byte) ([]byte, error) {
	var sec, pri, pub [32]byte
	if ok := checkType(&pri, private); !ok {
		return nil, errors.New("unexpected type of private key")
	}
	if ok := checkType(&pub, peersPublic); !ok {
		return nil, errors.New("unexpected type of peers public key")
	}

	curve25519.ScalarMult(&sec, &pri, &pub)

	return sec[:], nil
}

// AesCTRXOR(plainText) = cipherText AesCTRXOR(cipherText) = plainText
func AesCTRXOR(key, inText, iv []byte) ([]byte, error) {

	aesBlock, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	stream := cipher.NewCTR(aesBlock, iv)
	outText := make([]byte, len(inText))
	stream.XORKeyStream(outText, inText)
	return outText, err
}

func AesGCMEncrypt(key, inText []byte) (outText, nonce []byte, err error) {

	aesBlock, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, err
	}
	stream, err := cipher.NewGCM(aesBlock)
	if err != nil {
		return nil, nil, err
	}

	nonce = GetEntropyCSPRNG(12)

	outText = stream.Seal(nil, nonce, inText, []byte(gcmAdditionData))
	return outText, nonce, err
}

func AesGCMDecrypt(key, cipherText, nonce []byte) ([]byte, error) {

	aesBlock, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	stream, err := cipher.NewGCM(aesBlock)
	if err != nil {
		return nil, err
	}

	outText, err := stream.Open(nil, nonce, cipherText, []byte(gcmAdditionData))
	if err != nil {
		return nil, err
	}

	return outText, err
}

func GetEntropyCSPRNG(n int) []byte {
	mainBuff := make([]byte, n)
	_, err := io.ReadFull(crand.Reader, mainBuff)
	if err != nil {
		panic("reading from crypto/rand failed: " + err.Error())
	}
	return mainBuff
}

func VerifySig(pubkey ed25519.PublicKey, message, signdata []byte) (bool, error) {
	if l := len(pubkey); l != ed25519.PublicKeySize {
		return false, errors.New("ed25519: bad public key length: " + strconv.Itoa(l))
	}
	return ed25519.Verify(pubkey, message, signdata), nil
}
