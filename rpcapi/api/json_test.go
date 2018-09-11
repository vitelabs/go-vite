package api

import (
	"encoding/json"
	"github.com/vitelabs/go-vite/common/types"
	"math/big"
	"testing"
)

type DummyBlock struct {
	AccountAddress *types.Address `json:",omitempty"` // Self account
	ConfirmedTimes *string        `json:",omitempty"` // bigint block`s confirmed times
	T              *big.Int       `json:",omitempty"`
}

func TestJson(t *testing.T) {

	addresses, _, _ := types.CreateAddress()
	s := "12"
	d0 := DummyBlock{
		AccountAddress: nil,
		ConfirmedTimes: &s,
		T:              big.NewInt(1),
	}
	d1 := DummyBlock{
		AccountAddress: &addresses,
		ConfirmedTimes: nil,
	}
	bytes, e := json.Marshal(d0)
	if e != nil {
		t.Fatal(e)
	}
	println(string(bytes))

	println("d1")
	bytes, e = json.Marshal(d1)
	if e != nil {
		t.Fatal(e)
	}
	println(string(bytes))

	j := `{"ConfirmedTimes":"12","T":1}`
	var jd DummyBlock
	err := json.Unmarshal([]byte(j), &jd)
	if err != nil {
		t.Fatal(err)
	}
	if *jd.ConfirmedTimes != "12" {
		t.Fatal("ConfirmedTimes err")
	}

	if jd.AccountAddress != nil {
		t.Fatal("AccountAddress err")
	}
	println(jd.AccountAddress)

}
