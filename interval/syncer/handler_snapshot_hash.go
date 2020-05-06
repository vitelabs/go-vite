package syncer

import (
	"encoding/json"

	"github.com/vitelabs/go-vite/interval/common"
	"github.com/vitelabs/go-vite/interval/common/log"
	"github.com/vitelabs/go-vite/interval/p2p"
)

type reqSnapshotHashHandler struct {
	MsgHandler
	sReader *chainRw
	sender  Sender
}

func (self *reqSnapshotHashHandler) Types() []common.NetMsgType {
	return []common.NetMsgType{common.RequestSnapshotHash}
}

func (self *reqSnapshotHashHandler) Handle(t common.NetMsgType, d []byte, p p2p.Peer) {
	msg := &requestSnapshotHashMsg{}
	err := json.Unmarshal(d, msg)
	if err != nil {
		log.Error("[reqSnapshotHashHandler]Unmarshal fail.")
		return
	}

	var hashes []common.HashHeight
	hashH := common.HashHeight{Hash: msg.Hash, Height: msg.Height}

	for i := msg.PrevCnt; i > 0; i-- {
		if i < 0 {
			break
		}
		block := self.sReader.GetSnapshotByHashH(hashH)
		if block == nil {
			break
		}
		hashes = append(hashes, hashH)
		hashH = common.HashHeight{Hash: block.PreHash(), Height: block.Height() - 1}
	}

	if len(hashes) == 0 {
		return
	}
	self.sender.SendSnapshotHashes(hashes, p)
}

func (self *reqSnapshotHashHandler) Id() string {
	return "default-request-snapshot-hash-handler"
}
