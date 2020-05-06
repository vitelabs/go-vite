package syncer

import (
	"encoding/json"

	"github.com/vitelabs/go-vite/interval/common"
	"github.com/vitelabs/go-vite/interval/common/log"
	"github.com/vitelabs/go-vite/interval/p2p"
)

type reqAccountBlocksHandler struct {
	MsgHandler
	aReader *chainRw
	sender  Sender
}

func (self *reqAccountBlocksHandler) Types() []common.NetMsgType {
	return []common.NetMsgType{common.RequestAccountBlocks}
}

func (self *reqAccountBlocksHandler) Handle(t common.NetMsgType, d []byte, p p2p.Peer) {
	msg := &requestAccountBlockMsg{}
	err := json.Unmarshal(d, msg)
	if err != nil {
		log.Error("[reqAccountBlocksHandler]Unmarshal fail.")
	}

	hashes := msg.Hashes
	if len(hashes) <= 0 {
		return
	}
	var blocks []*common.AccountStateBlock
	for _, v := range hashes {
		block := self.aReader.GetAccountByHashH(msg.Address, v)
		if block == nil {
			continue
		}
		blocks = append(blocks, block)
	}
	if len(blocks) > 0 {
		log.Info("send account blocks, address:%s, blockSize:%d, PId:%s", msg.Address, len(blocks), p.Id())
		self.sender.SendAccountBlocks(msg.Address, blocks, p)
	}
}

func (*reqAccountBlocksHandler) Id() string {
	return "default-request-account-blocks-handler"
}
