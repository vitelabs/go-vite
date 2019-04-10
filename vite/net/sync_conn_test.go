package net

import (
	"crypto/rand"
	"fmt"
	"testing"
	"time"
)

func TestSyncHandshakeMsg_Serialize(t *testing.T) {
	var s = syncHandshakeMsg{
		key:  make([]byte, 32),
		time: time.Now(),
		sign: make([]byte, 64),
	}

	for i := 0; i < 100; i++ {
		_, _ = rand.Read(s.key)
		_, _ = rand.Read(s.sign)

		buf, err := s.Serialize()
		if err != nil {
			panic(err)
		}
		fmt.Println(len(buf))
	}
}
