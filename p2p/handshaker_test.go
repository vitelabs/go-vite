package p2p

import (
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/vitelabs/go-vite/p2p/protos"
)

func TestHandshakeMsg_Serialize(t *testing.T) {
	var msg HandshakeMsg
	data, err := msg.Serialize()
	if err != nil {
		t.Error("serialize error")
	}

	var msg2 HandshakeMsg
	err = msg2.Deserialize(data)
	if err != nil {
		t.Error("deserialize error")
	}
}

func TestProtoHandshakeMsg_Serialize(t *testing.T) {
	pb := protos.Handshake{
		Version:   0,
		NetId:     0,
		Name:      "",
		ID:        nil,
		Timestamp: 0,
		Protocols: nil,
	}

	data, err := proto.Marshal(&pb)
	if err != nil {
		t.Error("serialize error")
	}

	buffer := proto.NewBuffer(data)
	var pb2 protos.Handshake
	err = buffer.Unmarshal(&pb2)
	if err != nil {
		t.Error("unmarshal error")
	}
}
