package syncer

import (
	"encoding/json"

	"github.com/vitelabs/go-vite/interval/common"
	"github.com/vitelabs/go-vite/interval/common/log"
	"github.com/vitelabs/go-vite/interval/p2p"
	"github.com/vitelabs/go-vite/interval/verifier"
)

type receiver struct {
	fetcher       *fetcher
	innerHandlers map[common.NetMsgType][]MsgHandler
	handlers      map[common.NetMsgType]map[string]MsgHandler
}

func (r *receiver) Types() []common.NetMsgType {
	return nil
}

func (r *receiver) Id() string {
	return "default-handler"
}

func newReceiver(fetcher *fetcher, rw *chainRw, sender Sender, s *state) *receiver {
	self := &receiver{}
	self.fetcher = fetcher
	tmpInnerHandlers := make(map[common.NetMsgType][]MsgHandler)
	var innerhandlers []MsgHandler

	innerhandlers = append(innerhandlers, &accountHashHandler{fetcher: fetcher})
	innerhandlers = append(innerhandlers, &snapshotHashHandler{fetcher: fetcher})
	innerhandlers = append(innerhandlers, &snapshotBlocksHandler{sWriter: rw, fetcher: fetcher})
	innerhandlers = append(innerhandlers, &accountBlocksHandler{aWriter: rw, fetcher: fetcher})
	innerhandlers = append(innerhandlers, &stateHandler{state: s})
	innerhandlers = append(innerhandlers, &reqAccountHashHandler{aReader: rw, sender: sender})
	innerhandlers = append(innerhandlers, &reqSnapshotHashHandler{sReader: rw, sender: sender})
	innerhandlers = append(innerhandlers, &reqAccountBlocksHandler{aReader: rw, sender: sender})
	innerhandlers = append(innerhandlers, &reqSnapshotBlocksHandler{sReader: rw, sender: sender})

	for _, h := range innerhandlers {
		for _, t := range h.Types() {
			hs := tmpInnerHandlers[t]
			hs = append(hs, h)
			tmpInnerHandlers[t] = hs
		}
	}

	self.innerHandlers = tmpInnerHandlers
	self.handlers = make(map[common.NetMsgType]map[string]MsgHandler)
	return self
}

type stateHandler struct {
	MsgHandler
	state *state
}

func (handler *stateHandler) Types() []common.NetMsgType {
	return []common.NetMsgType{common.State, common.PeerConnected, common.PeerClosed}
}

func (handler *stateHandler) Id() string {
	return "default-state-handler"
}

func (handler *stateHandler) Handle(t common.NetMsgType, msg []byte, peer p2p.Peer) {
	switch t {
	case common.PeerClosed:
		handler.state.peerClosed(peer)
	case common.PeerConnected:
		handler.state.peerConnected(peer)
	case common.State:
		stateMsg := &stateMsg{}

		err := json.Unmarshal(msg, stateMsg)
		if err != nil {
			log.Error("stateHandler.Handle unmarshal fail.")
			return
		}

		handler.state.update(stateMsg, peer)
	}

	//prevState := peer.GetState()
	//if prevState == nil {
	//	peer.SetState(&peerState{Height: stateMsg.Height})
	//} else {
	//	state := prevState.(*peerState)
	//	state.Height = stateMsg.Height
	//}
}

type snapshotHashHandler struct {
	MsgHandler
	fetcher *fetcher
}

func (handler *snapshotHashHandler) Types() []common.NetMsgType {
	return []common.NetMsgType{common.SnapshotHashes}
}

func (handler *snapshotHashHandler) Id() string {
	return "default-snapshotHashHandler"
}

func (handler *snapshotHashHandler) Handle(t common.NetMsgType, msg []byte, peer p2p.Peer) {
	hashesMsg := &snapshotHashesMsg{}

	err := json.Unmarshal(msg, hashesMsg)
	if err != nil {
		log.Error("snapshotHashHandler.Handle unmarshal fail.")
	}
	handler.fetcher.fetchSnapshotBlockByHash(hashesMsg.Hashes)
}

type accountHashHandler struct {
	MsgHandler
	fetcher *fetcher
}

func (handler *accountHashHandler) Types() []common.NetMsgType {
	return []common.NetMsgType{common.AccountHashes}
}

func (handler *accountHashHandler) Handle(t common.NetMsgType, msg []byte, peer p2p.Peer) {
	hashesMsg := &accountHashesMsg{}
	err := json.Unmarshal(msg, hashesMsg)
	if err != nil {
		log.Error("accountHashHandler.Handle unmarshal fail.")
	}
	handler.fetcher.fetchAccountBlockByHash(hashesMsg.Address, hashesMsg.Hashes)
}
func (handler *accountHashHandler) Id() string {
	return "default-accountHashHandler"
}

type snapshotBlocksHandler struct {
	MsgHandler
	fetcher *fetcher
	sWriter *chainRw
}

func (handler *snapshotBlocksHandler) Types() []common.NetMsgType {
	return []common.NetMsgType{common.SnapshotBlocks}
}

func (handler *snapshotBlocksHandler) Handle(t common.NetMsgType, msg []byte, peer p2p.Peer) {
	hashesMsg := &snapshotBlocksMsg{}
	err := json.Unmarshal(msg, hashesMsg)
	if err != nil {
		log.Error("snapshotBlocksHandler.Handle unmarshal fail.")
	}
	for _, v := range hashesMsg.Blocks {
		r := verifier.VerifySnapshotHash(v)
		if !r {
			log.Warn("error hash for snapshot block. %v", v)
			continue
		}
		handler.fetcher.done(v.Hash(), v.Height())
		handler.sWriter.AddSnapshotBlock(v)
	}
}

func (handler *snapshotBlocksHandler) Id() string {
	return "default-snapshotBlocksHandler"
}

type accountBlocksHandler struct {
	MsgHandler
	fetcher *fetcher
	aWriter *chainRw
}

func (handler *accountBlocksHandler) Types() []common.NetMsgType {
	return []common.NetMsgType{common.AccountBlocks}
}

func (handler *accountBlocksHandler) Handle(t common.NetMsgType, msg []byte, peer p2p.Peer) {
	hashesMsg := &accountBlocksMsg{}
	err := json.Unmarshal(msg, hashesMsg)
	if err != nil {
		log.Error("accountBlocksHandler.Handle unmarshal fail.")
	}
	for _, v := range hashesMsg.Blocks {
		r := verifier.VerifyAccount(v)
		if !r {
			log.Warn("error hash for account block. %v", v)
			continue
		}
		handler.fetcher.done(v.Hash(), v.Height())
		handler.aWriter.AddAccountBlock(v.Signer(), v)
	}
}

func (handler *accountBlocksHandler) Id() string {
	return "default-accountBlocksHandler"
}

func (r *receiver) Handle(t common.NetMsgType, msg []byte, peer p2p.Peer) {
	r.innerHandle(t, msg, peer, r.innerHandlers)
	r.handle(t, msg, peer, r.handlers)
}
func (r *receiver) innerHandle(t common.NetMsgType, msg []byte, peer p2p.Peer, handlers map[common.NetMsgType][]MsgHandler) {
	hs := handlers[t]

	if hs != nil {
		for _, h := range hs {
			h.Handle(t, msg, peer)
		}
	}
}

func (r *receiver) handle(t common.NetMsgType, msg []byte, peer p2p.Peer, handlers map[common.NetMsgType]map[string]MsgHandler) {
	hs := handlers[t]
	if hs != nil {
		for _, h := range hs {
			h.Handle(t, msg, peer)
		}
	}
}

func (r *receiver) RegisterHandler(handler MsgHandler) {
	r.append(r.handlers, handler)
	log.Info("register msg handler, type:%v, handler:%s", handler.Types(), handler.Id())
}

func (r *receiver) UnRegisterHandler(handler MsgHandler) {
	r.delete(r.handlers, handler)
	log.Info("unregister msg handler, type:%v, handler:%s", handler.Types(), handler.Id())
}
func (r *receiver) append(hmap map[common.NetMsgType]map[string]MsgHandler, h MsgHandler) {
	for _, t := range h.Types() {
		hs := hmap[t]
		if hs == nil {
			hs = make(map[string]MsgHandler)
			hmap[t] = hs
		}
		hs[h.Id()] = h
	}
}

func (r *receiver) delete(hmap map[common.NetMsgType]map[string]MsgHandler, h MsgHandler) {
	for _, t := range h.Types() {
		hs := hmap[t]
		delete(hs, h.Id())
	}
}
