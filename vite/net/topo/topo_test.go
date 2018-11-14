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

func TestCalcDuration(t *testing.T) {
	d := calcDuration(50)
	if d.Seconds() != 50 {
		t.Fail()
	}
}

func TestRmEmptyString(t *testing.T) {
	s0 := []string{}
	s1 := []string{""}
	s2 := []string{"", "hello", "", "", "world"}
	s3 := []string{"", "hello", ""}
	s4 := []string{"", ""}
	s5 := []string{"hello", "world"}
	s6 := []string{"hello", ""}

	if len(rmEmptyString(s0)) != 0 {
		t.Fail()
	}
	if len(rmEmptyString(s1)) != 0 {
		t.Fail()
	}
	if len(rmEmptyString(s2)) != 2 {
		t.Fail()
	}
	if len(rmEmptyString(s3)) != 1 {
		t.Fail()
	}
	if len(rmEmptyString(s4)) != 0 {
		t.Fail()
	}
	if len(rmEmptyString(s5)) != 2 {
		t.Fail()
	}
	if len(rmEmptyString(s6)) != 1 {
		t.Fail()
	}
}
