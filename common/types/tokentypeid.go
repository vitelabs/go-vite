package types

import (
	"bytes"
	"encoding/hex"
	"fmt"
	vcrypto "github.com/vitelabs/go-vite/crypto"
	"strings"
)

const (
	TokenTypeIdPrefix       = "tti_"
	tokenTypeIdSize         = 10
	tokenTypeIdChecksumSize = 2
	tokenTypeIdPrefixLen    = len(TokenTypeIdPrefix)
	hexTokenTypeIdLength    = tokenTypeIdPrefixLen + 2*tokenTypeIdSize + 2*tokenTypeIdChecksumSize
)

type TokenTypeId [tokenTypeIdSize]byte

func (tid *TokenTypeId) SetBytes(b []byte) error {
	if length := len(b); length != tokenTypeIdSize {
		return fmt.Errorf("error tokentypeid size error %v", length)
	}
	copy(tid[:], b)
	return nil
}

func (tid TokenTypeId) Hex() string {
	return TokenTypeIdPrefix + hex.EncodeToString(tid[:]) + hex.EncodeToString(vcrypto.Hash(tokenTypeIdChecksumSize, tid[:]))
}
func (t TokenTypeId) Bytes() []byte { return t[:] }
func (t TokenTypeId) String() string {
	return t.Hex()
}

func BytesToTokenTypeId(b []byte) (TokenTypeId, error) {
	var tid TokenTypeId
	err := tid.SetBytes(b)
	return tid, err
}

func HexToTokenTypeId(hexStr string) (TokenTypeId, error) {
	if IsValidHexTokenTypeId(hexStr) {
		tti, _ := getTokenTypeIdFromHex(hexStr)
		return tti, nil
	} else {
		return TokenTypeId{}, fmt.Errorf("Not valid hex TokenTypeId")
	}
}

func IsValidHexTokenTypeId(hexStr string) bool {
	if len(hexStr) != hexTokenTypeIdLength || !strings.HasPrefix(hexStr, TokenTypeIdPrefix) {
		return false
	}

	tti, err := getTokenTypeIdFromHex(hexStr)
	if err != nil {
		return false
	}

	ttiChecksum, err := getTtiChecksumFromHex(hexStr)
	if err != nil {
		return false
	}

	if !bytes.Equal(vcrypto.Hash(tokenTypeIdChecksumSize, tti[:]), ttiChecksum[:]) {
		return false

	}

	return true
}

func getTokenTypeIdFromHex(hexStr string) ([tokenTypeIdSize]byte, error) {
	var b [tokenTypeIdSize]byte
	_, err := hex.Decode(b[:], []byte(hexStr[tokenTypeIdPrefixLen:tokenTypeIdPrefixLen+2*tokenTypeIdSize]))
	return b, err
}

func CreateTokenTypeId(data ...[]byte) TokenTypeId {
	tti, _ := BytesToTokenTypeId(vcrypto.Hash(tokenTypeIdSize, data...))
	return tti
}

func getTtiChecksumFromHex(hexStr string) ([tokenTypeIdChecksumSize]byte, error) {
	var b [tokenTypeIdChecksumSize]byte
	_, err := hex.Decode(b[:], []byte(hexStr[tokenTypeIdPrefixLen+2*tokenTypeIdSize:]))
	return b, err
}
