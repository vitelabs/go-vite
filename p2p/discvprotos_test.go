package p2p

import (
	"testing"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"fmt"
)

func TestUnpack(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println("pubKey ", pub)

	var id NodeID
	copy(id[:], pub)
	p := &Ping{
		ID: id,
	}

	data, hash, err := p.Pack(priv)
	if err != nil {
		t.Fatal(err)
	}

	m, hash2, err := unPacket(data)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println("unpack ", m)

	if hash != hash2 {
		t.Fail()
	}
}
