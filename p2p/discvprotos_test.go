package p2p

import (
	"testing"
	"github.com/vitelabs/go-vite/crypto/ed25519"
)

func TestUnpack(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}

	var id NodeID
	copy(id[:], pub)
	p := &Ping{
		ID: id,
	}

	data, hash, err := p.Pack(priv)
	if err != nil {
		t.Fatal(err)
	}

	_, hash2, err := unPacket(data)
	if err != nil {
		t.Fatal(err)
	}

	if hash != hash2 {
		t.Fail()
	}
}
