package chain

import (
	"net"
	"os"
	"testing"
)

func TestChain_DeleteSnapshotBlocks(t *testing.T) {
	addr := os.Args[1]
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		panic(err)
	}

	_, err = conn.Write([]byte("hello"))
	if err != nil {
		panic(err)
	}
}
