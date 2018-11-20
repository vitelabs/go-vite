package p2p

import (
	"math/rand"
)

func mockPayload() ([]byte, int) {
	var size int

	for {
		size = rand.Intn(10000)
		if size > 0 {
			break
		}
	}

	buf := make([]byte, size)
	for i := 0; i < size; i++ {
		buf[i] = byte(rand.Intn(254))
	}

	return buf, size
}

func mockMsg() (*Msg, error) {
	msg := NewMsg()

	msg.CmdSet = rand.Uint32()
	msg.Cmd = uint16(rand.Uint32())
	msg.Id = rand.Uint64()

	payload, _ := mockPayload()
	msg.Payload = payload

	return msg, nil
}
