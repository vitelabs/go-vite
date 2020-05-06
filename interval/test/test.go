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

func (self *TestBlock) Height() int {
	return self.Theight
}

func (self *TestBlock) Hash() string {
	return self.Thash
}

func (self *TestBlock) PreHash() string {
	return self.TpreHash
}

func (self *TestBlock) Signer() string {
	return self.Tsigner
}
func (self *TestBlock) Timestamp() time.Time {
	return self.Ttimestamp
}
func (self *TestBlock) String() string {
	return "Theight:[" + strconv.Itoa(self.Theight) + "]\tThash:[" + self.Thash + "]\tTpreHash:[" + self.TpreHash + "]\tTsigner:[" + self.Tsigner + "]"
}
