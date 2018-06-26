package common

import (
	"testing"
)

const (
	CorrectTTI       = "tti_2445f6e5cde8c2c70e446c83"
	WrongTTICheckSum = "tti_2445f6e5cde8c2c70e446c84"
	WrongTTIMain     = "tti_2445f6e5cae8c2c70e446c83"
	WrongTTILen      = "tti_2445f6e5cae8c2c70e446c8"
	WrongTTIPre      = "1tti_2445f6e5cae8c2c70e446c"
)

func BenchmarkCreateTokenTypeId(b *testing.B) {
	CreateTokenTypeId()
}

func TestHexToTokenTypeId(t *testing.T) {
	tti, err := HexToTokenTypeId(CorrectTTI)
	if err != nil || tti.Hex() != CorrectTTI {
		t.Fatal("expect correct but wrong")
	}

	_, err = HexToTokenTypeId(WrongTTICheckSum)
	if err == nil {
		t.Fatal("WrongTTICheckSum expect wrong but correct")
	}

	_, err = HexToTokenTypeId(WrongTTIMain)
	if err == nil {
		t.Fatal("WrongTTIMain expect wrong but correct")
	}

	_, err = HexToTokenTypeId(WrongTTILen)
	if err == nil {
		t.Fatal("WrongTTILen expect wrong but correct")
	}

	_, err = HexToTokenTypeId(WrongTTIPre)
	if err == nil {
		t.Fatal("WrongTTIPre expect wrong but correct")
	}
}
