package derivation

import (
	"testing"
)


func TestExampleMnemonic12(t *testing.T) {
	if err := RandomMnemonic12(""); err != nil {
		t.Fatal(err)
	}

	if err := RandomMnemonic12("123456"); err != nil {
		t.Fatal(err)
	}

	var b [16]byte
	if err := Menmonic(b[:],""); err != nil {
		t.Fatal(err)
	}
}

func TestExampleMnemonic24(t *testing.T) {
	if err := RandomMnemonic24(""); err != nil {
		t.Fatal(err)
	}

	if err := RandomMnemonic24("123456"); err != nil {
		t.Fatal(err)
	}
}
