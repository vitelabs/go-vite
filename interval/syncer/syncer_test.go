package syncer

import (
	"encoding/json"
	"strconv"
	"testing"
	"time"

	"github.com/asaskevich/EventBus"
	"github.com/vitelabs/go-vite/interval/common"
	"github.com/vitelabs/go-vite/interval/common/face"
	"github.com/vitelabs/go-vite/interval/common/log"
	"github.com/vitelabs/go-vite/interval/p2p"
)

type TestPeer struct {
	fn    p2p.MsgHandle
	state interface{}
}

func (peer *TestPeer) SetState(state interface{}) {
	peer.state = state
}

func (peer *TestPeer) GetState() interface{} {
	return peer.state
}
func (peer *TestPeer) RemoteAddr() string {
	return ""
}

func (peer *TestPeer) Write(msg *p2p.Msg) error {
	log.Info("write msg, msgType:%s", msg.T)
	peer.fn(msg.T, msg.Data, peer)
	return nil
}

func (peer *TestPeer) Id() string {
	return "testPeer"
}

type TestP2P struct {
	bestPeer *TestPeer
}

func (test *TestP2P) Start() {
	panic("implement me")
}

func (test *TestP2P) Stop() {
	panic("implement me")
}

func (test *TestP2P) Id() string {
	panic("implement me")
}

func (test *TestP2P) SetHandlerFn(fn p2p.MsgHandle) {
	test.bestPeer.fn = fn
}

func (test *TestP2P) BestPeer() (p2p.Peer, error) {
	return test.bestPeer, nil
}

func (test *TestP2P) AllPeer() ([]p2p.Peer, error) {
	return []p2p.Peer{test.bestPeer}, nil
}

type TestAccountReader struct {
}

func (reader *TestAccountReader) GenesisSnapshost() (*common.SnapshotBlock, error) {
	panic("implement me")
}

func (reader *TestAccountReader) HeadSnapshost() (*common.SnapshotBlock, error) {
	panic("implement me")
}

func (reader *TestAccountReader) AddAccountBlock(account string, block *common.AccountStateBlock) error {
	panic("implement me")
}

func (reader *TestAccountReader) AddSnapshotBlock(block *common.SnapshotBlock) {
	panic("implement me")
}

func (reader *TestAccountReader) GetSnapshotBlocksByHashH(hashH common.HashHeight) *common.SnapshotBlock {
	log.Info("TestSnapshotReader#GetSnapshotBlocksByHashH, hash:%s, height:%d", hashH.Hash, hashH.Height)
	return genSnapshotBlock(hashH)
}

func (reader *TestAccountReader) GetAccountBlocksByHashH(address string, hashH common.HashHeight) *common.AccountStateBlock {
	log.Info("TestAccountReader#GetAccountBlocksByHashH, address:%s, hash:%s, height:%d", address, hashH.Hash, hashH.Height)
	return genAccountBlock(address, hashH)
}

func TestSyncer(t *testing.T) {
	peer := &TestPeer{}
	p := &TestP2P{}
	accountReader := &TestAccountReader{}
	p.bestPeer = peer
	syncer := NewSyncer(p, EventBus.New())
	syncer.Init(accountReader)
	fetcher := syncer.Fetcher()
	address := "viteshan"
	testHandler := &TestHandler{}
	syncer.Handlers().RegisterHandler(testHandler)

	peer.fn = syncer.DefaultHandler().Handle

	var hashHeight common.HashHeight
	hashHeight = genHashHeight(5)
	fetcher.Fetch(face.FetchRequest{Hash: hashHeight.Hash, Height: hashHeight.Height, PrevCnt: 5, Chain: address})
	hashHeight = genHashHeight(6)
	fetcher.Fetch(face.FetchRequest{Hash: hashHeight.Hash, Height: hashHeight.Height, PrevCnt: 5, Chain: address})

	time.Sleep(2 * time.Second)
	if testHandler.cnt != 6 {
		t.Error("error number.", testHandler.cnt)
	}
}

type TestHandler struct {
	cnt int
}

func (handler *TestHandler) Handle(t common.NetMsgType, msg []byte, peer p2p.Peer) {
	if t == common.SnapshotBlocks {
		hashesMsg := &snapshotBlocksMsg{}
		err := json.Unmarshal(msg, hashesMsg)
		if err != nil {
			log.Error("TestHandler.Handle unmarshal fail.")
		}
		handler.cnt = handler.cnt + len(hashesMsg.Blocks)
	} else if t == common.AccountBlocks {
		hashesMsg := &accountBlocksMsg{}
		err := json.Unmarshal(msg, hashesMsg)
		if err != nil {
			log.Error("TestHandler.Handle unmarshal fail.")
		}
		handler.cnt = handler.cnt + len(hashesMsg.Blocks)
	}
}

func (handler *TestHandler) Types() []common.NetMsgType {
	return []common.NetMsgType{common.SnapshotBlocks, common.AccountBlocks}
}

func (handler *TestHandler) Id() string {
	return "testHandler"
}

func genHashHeight(height int) common.HashHeight {
	return common.HashHeight{Hash: strconv.Itoa(height), Height: height}
}

func genSnapshotBlock(hashH common.HashHeight) *common.SnapshotBlock {
	preHashH := genHashHeight(hashH.Height - 1)
	return common.NewSnapshotBlock(hashH.Height, hashH.Hash, preHashH.Hash, "viteshan", time.Now(), nil)
}
func genAccountBlock(address string, hashH common.HashHeight) *common.AccountStateBlock {
	preHashH := genHashHeight(hashH.Height - 1)
	return common.NewAccountBlock(hashH.Height, hashH.Hash, preHashH.Hash, address, time.Now(), 0, 0, common.SEND, address, "viteshan2", "", -1)
}
