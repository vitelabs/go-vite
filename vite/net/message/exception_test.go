package message

import "testing"

func TestException_Deserialize(t *testing.T) {
	buf, err := Missing.Serialize()
	if err != nil {
		t.Error(err)
	}

	exp, err := DeserializeException(buf)
	if err != nil {
		t.Error(err)
	}

	if exp != Missing {
		t.Fail()
	}
}
