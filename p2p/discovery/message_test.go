package discovery

import (
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"testing"
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

	data, _, err := p.pack(priv)
	if err != nil {
		t.Fatal(err)
	}

	_, err = unPacket(data)
	if err != nil {
		t.Fatal(err)
	}
}
