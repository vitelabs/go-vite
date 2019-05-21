package netool

import "testing"

func TestCheckRelayIP(t *testing.T) {
	sender := []byte{192, 168, 0, 0}
	ip := []byte{0, 0, 0, 0}

	if CheckRelayIP(sender, ip) == nil {
		t.Error("should error")
	}
}
