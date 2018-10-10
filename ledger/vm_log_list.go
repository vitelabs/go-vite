package ledger

import (
	"github.com/golang/protobuf/proto"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto"
	"github.com/vitelabs/go-vite/vitepb"
)

type VmLog struct {
	Topics []types.Hash
	Data   []byte
}

type VmLogList []*VmLog

func (vll VmLogList) Hash() *types.Hash {
	if len(vll) == 0 {
		return nil
	}
	var source []byte

	// Nonce
	for _, vmLog := range vll {
		for _, topic := range vmLog.Topics {
			source = append(source, topic.Bytes()...)
		}
		source = append(source, vmLog.Data...)
	}

	hash, _ := types.BytesToHash(crypto.Hash256(source))
	return &hash
}

func (vll VmLogList) Proto() *vitepb.VmLogList {
	pb := &vitepb.VmLogList{}

	for _, vmLog := range vll {
		var topicsPb [][]byte
		for _, topic := range vmLog.Topics {
			topicsPb = append(topicsPb, topic.Bytes())
		}
		pb.List = append(pb.List, &vitepb.VmLog{
			Topics: topicsPb,
			Data:   vmLog.Data,
		})
	}

	return pb
}

func VmLogListDeProto(pb *vitepb.VmLogList) VmLogList {
	var vll VmLogList
	for _, vmLogPb := range pb.List {

		var topics []types.Hash
		for _, topicPb := range vmLogPb.Topics {
			topic, _ := types.BytesToHash(topicPb)
			topics = append(topics, topic)
		}

		vll = append(vll, &VmLog{
			Topics: topics,
			Data:   vmLogPb.Data,
		})
	}
	return vll
}

func (vll VmLogList) Serialize() ([]byte, error) {
	return proto.Marshal(vll.Proto())
}

func VmLogListDeserialize(buf []byte) (VmLogList, error) {
	pb := &vitepb.VmLogList{}
	if err := proto.Unmarshal(buf, pb); err != nil {
		return nil, err
	}

	return VmLogListDeProto(pb), nil
}
