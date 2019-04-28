package message

import "testing"

func TestException_Deserialize(t *testing.T) {
	buf, err := Missing.Serialize()
	if err != nil {
		t.Error(err)
	}

	e := new(Exception)
	err = e.Deserialize(buf)
	if err != nil {
		t.Error(err)
	}

	if *e != Missing {
		t.Fail()
	}
}
