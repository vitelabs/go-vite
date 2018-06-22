package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	crand "crypto/rand"
	"io"
)

const (
	gcmAdditionData = "vite"
)

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
