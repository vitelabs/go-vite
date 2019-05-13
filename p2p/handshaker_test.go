package p2p

import (
	"bytes"
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

	if pb.Version != pb2.Version {
		t.Errorf("different version: %d %d", pb.Version, pb2.Version)
	}
	if pb.NetId != pb2.NetId {
		t.Errorf("different net: %d %d", pb.NetId, pb2.NetId)
	}
	if pb.Name != pb2.Name {
		t.Errorf("different name: %s %s", pb.Name, pb2.Name)
	}
	if bytes.Equal(pb.ID, pb2.ID) == false {
		t.Errorf("different id")
	}
	if pb.Timestamp != pb2.Timestamp {
		t.Errorf("different time: %d %d", pb.Timestamp, pb2.Timestamp)
	}
}
