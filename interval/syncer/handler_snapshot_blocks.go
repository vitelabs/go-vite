package syncer

import (
	"encoding/json"

	"github.com/vitelabs/go-vite/interval/common"
	"github.com/vitelabs/go-vite/interval/common/log"
	"github.com/vitelabs/go-vite/interval/p2p"
)

type reqSnapshotBlocksHandler struct {
	MsgHandler
	sReader *chainRw
	sender  Sender
}

func (handler *reqSnapshotBlocksHandler) Types() []common.NetMsgType {
	return []common.NetMsgType{common.RequestSnapshotBlocks}
}

func (handler *reqSnapshotBlocksHandler) Handle(t common.NetMsgType, d []byte, p p2p.Peer) {
	msg := &requestSnapshotBlockMsg{}
	err := json.Unmarshal(d, msg)
	if err != nil {
		log.Error("[reqSnapshotBlocksHandler]Unmarshal fail.")
	}

	hashes := msg.Hashes
	if len(hashes) <= 0 {
		return
	}
	var blocks []*common.SnapshotBlock
	for _, v := range hashes {
		block := handler.sReader.GetSnapshotByHashH(v)
		if block == nil {
			continue
		}
		blocks = append(blocks, block)
	}
	if len(blocks) > 0 {
		handler.sender.SendSnapshotBlocks(blocks, p)
	}
}

func (*reqSnapshotBlocksHandler) Id() string {
	return "default-request-snapshot-blocks-handler"
}
