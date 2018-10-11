package message

import "testing"

var ga GetAccountBlocks

func TestGetAccountBlocks_Serialize(t *testing.T) {
	ga.From.Height = 1999

	buf, err := ga.Serialize()
	if err != nil {
		t.Error(err)
	}

	g := new(GetAccountBlocks)
	err = g.Deserialize(buf)
	if err != nil {
		t.Error(err)
	}

	if g.From.Height != ga.From.Height {
		t.Fail()
	}
}

func TestGetAccountBlocks_DeSerialize(t *testing.T) {
	ga.From.Height = 1999

	buf, err := ga.Serialize()
	if err != nil {
		t.Error(err)
	}

	g := new(GetAccountBlocks)
	err = g.Deserialize(buf)
	if err != nil {
		t.Error(err)
	}

	if g.From.Height != ga.From.Height {
		t.Fail()
	}
}

var gs GetSnapshotBlocks

func TestGetSnapshotBlocks_Deserialize(t *testing.T) {
	gs.From.Height = 1999

	buf, err := gs.Serialize()
	if err != nil {
		t.Error(err)
	}

	g := new(GetSnapshotBlocks)
	err = g.Deserialize(buf)
	if err != nil {
		t.Error(err)
	}

	if g.From.Height != ga.From.Height {
		t.Fail()
	}
}
