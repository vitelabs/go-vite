package discovery

import "testing"

func TestNodeID_IsZero(t *testing.T) {
	if !ZERO_NODE_ID.IsZero() {
		t.Fail()
	}
}
