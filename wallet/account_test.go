package wallet

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAccount(t *testing.T) {
	acc, err := RandomAccount()
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	msg := []byte("hello world")
	sig, pub, err := acc.Sign(msg)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	err = acc.Verify(pub, msg, sig)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	acc2, err := NewAccountFromHexKey(acc.priv.Hex())
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	assert.Equal(t, acc.priv.Hex(), acc2.priv.Hex())
	assert.Equal(t, acc.address.Hex(), acc2.address.Hex())
}
