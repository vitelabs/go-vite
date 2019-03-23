package p2p

import (
	crand "crypto/rand"
	"encoding/hex"
	"github.com/vitelabs/go-vite/p2p/discovery"
	mrand "math/rand"
	"testing"
)

func mockHandshake() *Handshake {
	name := make([]byte, 10)
	crand.Read(name)

	ip := make([]byte, 128)
	for i := range ip {
		ip[i] = byte(mrand.Intn(255))
	}

	cmdSet := make([]CmdSet, 101)
	for i := range cmdSet {
		cmdSet[i] = mrand.Uint32()
	}

	hand := &Handshake{
		Name:       hex.EncodeToString(name),
		ID:         discovery.ZERO_NODE_ID,
		CmdSets:    cmdSet,
		RemoteIP:   ip,
		RemotePort: uint16(mrand.Intn(65535)),
		Port:       uint16(mrand.Intn(65535)),
	}

	return hand
}

func compare(hand, hand2 *Handshake) bool {
	if hand2.Name != hand.Name {
		return false
	}

	if hand2.ID != hand.ID {
		return false
	}

	if hand2.RemotePort != hand.RemotePort {
		return false
	}

	if hand2.Port != hand.Port {
		return false
	}

	for i, cmd := range hand.CmdSets {
		if hand2.CmdSets[i] != cmd {
			return false
		}
	}

	for i, b := range hand.RemoteIP {
		if hand2.RemoteIP[i] != b {
			return false
		}
	}

	return true
}

func TestHandshake_Deserialize(t *testing.T) {
	hand := mockHandshake()

	data, err := hand.Serialize()
	if err != nil {
		t.Error(err)
	}

	hand2 := new(Handshake)
	err = hand2.Deserialize(data)
	if err != nil {
		t.Error(err)
	}

	if !compare(hand, hand2) {
		t.Fail()
	}
}
