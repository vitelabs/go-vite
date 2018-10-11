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
