package message

import (
	crand "crypto/rand"
	mrand "math/rand"
	"testing"
)

func MockGetAccountBlocks() GetAccountBlocks {
	var ga GetAccountBlocks

	ga.From.Height = mrand.Uint64()
	crand.Read(ga.From.Hash[:])

	ga.Count = mrand.Uint64()
	ga.Forward = mrand.Intn(10) > 5

	crand.Read(ga.Address[:])

	return ga
}

func equalGetAccountBlocks(g, g2 GetAccountBlocks) bool {
	if g.From.Height != g2.From.Height {
		return false
	}

	if g.From.Hash != g2.From.Hash {
		return false
	}

	if g.Count != g2.Count {
		return false
	}

	if g.Forward != g2.Forward {
		return false
	}

	if g.Address != g2.Address {
		return false
	}

	return true
}

func TestGetAccountBlocks_Serialize(t *testing.T) {
	ga := MockGetAccountBlocks()

	buf, err := ga.Serialize()
	if err != nil {
		t.Error(err)
	}

	var g GetAccountBlocks
	err = g.Deserialize(buf)
	if err != nil {
		t.Error(err)
	}

	if !equalGetAccountBlocks(ga, g) {
		t.Fail()
	}
}

func MockGetSnapshotBlocks() GetSnapshotBlocks {
	var ga GetSnapshotBlocks

	ga.From.Height = mrand.Uint64()
	crand.Read(ga.From.Hash[:])

	ga.Count = mrand.Uint64()
	ga.Forward = mrand.Intn(10) > 5

	return ga
}

func equalGetSnapshotBlocks(g, g2 GetSnapshotBlocks) bool {
	if g.From.Height != g2.From.Height {
		return false
	}

	if g.From.Hash != g2.From.Hash {
		return false
	}

	if g.Count != g2.Count {
		return false
	}

	if g.Forward != g2.Forward {
		return false
	}

	return true
}

func TestGetSnapshotBlocks_Deserialize(t *testing.T) {
	gs := MockGetSnapshotBlocks()

	buf, err := gs.Serialize()
	if err != nil {
		t.Error(err)
	}

	var g GetSnapshotBlocks
	err = g.Deserialize(buf)
	if err != nil {
		t.Error(err)
	}

	if !equalGetSnapshotBlocks(gs, g) {
		t.Fail()
	}
}
