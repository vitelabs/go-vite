package test

import (
	"strconv"
	"time"
)

type TestBlock struct {
	Thash      string
	Theight    int
	TpreHash   string
	Tsigner    string
	Ttimestamp time.Time
}

func (tb *TestBlock) Height() int {
	return tb.Theight
}

func (tb *TestBlock) Hash() string {
	return tb.Thash
}

func (tb *TestBlock) PreHash() string {
	return tb.TpreHash
}

func (tb *TestBlock) Signer() string {
	return tb.Tsigner
}
func (tb *TestBlock) Timestamp() time.Time {
	return tb.Ttimestamp
}
func (tb *TestBlock) String() string {
	return "Theight:[" + strconv.Itoa(tb.Theight) + "]\tThash:[" + tb.Thash + "]\tTpreHash:[" + tb.TpreHash + "]\tTsigner:[" + tb.Tsigner + "]"
}
