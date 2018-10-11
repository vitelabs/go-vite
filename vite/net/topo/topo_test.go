package topo

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

var topo = Topo{
	Pivot: "whatever",
	Peers: nil,
	Time:  UnixTime(time.Now()),
}

func TestTopoJson(t *testing.T) {
	buf, err := json.Marshal(topo)
	if err != nil {
		t.Error(err)
	}

	fmt.Println(string(buf))

	topo2 := new(Topo)
	err = json.Unmarshal(buf, topo2)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(topo2)

	if topo2.Time.Unix() != topo.Time.Unix() {
		t.Fail()
	}
}
