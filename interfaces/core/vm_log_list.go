package core

import (
	"bytes"

	"github.com/golang/protobuf/proto"

	"github.com/vitelabs/go-vite/v2/common/types"
	"github.com/vitelabs/go-vite/v2/common/upgrade"
	"github.com/vitelabs/go-vite/v2/common/vitepb"
	"github.com/vitelabs/go-vite/v2/crypto"
)

type VmLog struct {
	Topics []types.Hash `json:"topics"` // the abstract information about the log
	Data   []byte       `json:"data"`   // the detail information about the log
}

type VmLogList []*VmLog

func (a VmLogList) Len() int      { return len(a) }
func (a VmLogList) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a VmLogList) Less(i, j int) bool {
	if len(a[i].Topics) != len(a[j].Topics) {
		return len(a[i].Topics) < len(a[j].Topics)
	}
	for ii, ai := range a[i].Topics {
		aj := a[j].Topics[ii]
		r := ai.Cmp(aj)
		if r != 0 {
			return r < 0
		}
	}
	return bytes.Compare(a[i].Data, a[j].Data) < 0
}

func (vll VmLogList) Hash(snapshotHeight uint64, address types.Address, prevHash types.Hash) *types.Hash {
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

	if upgrade.IsSeedUpgrade(snapshotHeight) {
		// append address bytes
		source = append(source, address.Bytes()...)
		source = append(source, prevHash.Bytes()...)
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

func (vll VmLogList) Serialize() ([]byte, error) {
	return proto.Marshal(vll.Proto())
}

func (vll *VmLogList) Deserialize(buf []byte) error {
	pb := &vitepb.VmLogList{}
	if err := proto.Unmarshal(buf, pb); err != nil {
		return err
	}
	return vll.DeProto(pb)
}

func (vll *VmLogList) DeProto(pb *vitepb.VmLogList) error {
	var tmp VmLogList
	for _, vmLogPb := range pb.List {

		var topics []types.Hash
		for _, topicPb := range vmLogPb.Topics {
			topic, _ := types.BytesToHash(topicPb)
			topics = append(topics, topic)
		}

		tmp = append(tmp, &VmLog{
			Topics: topics,
			Data:   vmLogPb.Data,
		})
	}
	*vll = tmp
	return nil
}
