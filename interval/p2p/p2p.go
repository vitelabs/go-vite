package p2p

import (
	"encoding/json"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/interval/common"
	"github.com/vitelabs/go-vite/interval/common/config"
	"github.com/vitelabs/go-vite/interval/common/log"
)

// 0:origin 1: initing 2:inited 3:starting 4:started 5:stopping 6:stopped
type P2PLifecycle struct {
	common.LifecycleStatus
}

type Peer interface {
	Write(msg *Msg) error
	Id() string
	RemoteAddr() string
	SetState(interface{})
	GetState() interface{}
}

type BootLinkPeer struct {
	Id   string
	Addr string
}

type HandShaker interface {
	GetState() (interface{}, error)
	Handshake(peerId string, state []byte) error
	DecodeState(state []byte) interface{}
	EncodeState(state interface{}) []byte
}
type defaultHandShaker struct {
}

func (self *defaultHandShaker) GetState() (interface{}, error) {
	return nil, nil
}

func (self *defaultHandShaker) Handshake(peerId string, state []byte) error {
	return nil
}

func (self *defaultHandShaker) DecodeState(state []byte) interface{} {
	return nil
}

func (self *defaultHandShaker) EncodeState(state interface{}) []byte {
	return nil
}

type Msg struct {
	T    common.NetMsgType // type: 2~100 basic msg  101~200:biz msg
	Data []byte
}

func NewMsg(t common.NetMsgType, data []byte) *Msg {
	return &Msg{T: t, Data: data}
}

type MsgHandle func(common.NetMsgType, []byte, Peer)

type P2P interface {
	BestPeer() (Peer, error)
	AllPeer() ([]Peer, error)
	SetHandlerFn(MsgHandle)
	SetHandShaker(HandShaker)
	Init()
	Start()
	Stop()
	Id() string
}

type Boot interface {
	Start()
	Stop()
	All() []*BootLinkPeer
}

type p2p struct {
	P2PLifecycle
	//peers  []*peer
	mu     sync.Mutex
	server *server
	dial   *dial
	linker *linker
	boot   *bootnode
	hs     *handShaker
	bizHs  HandShaker

	peers        map[string]*peer
	pendingDials map[string]string
	id           string
	netId        int
	addr         string
	linkBootAddr string
	closed       chan struct{}
	loopWg       sync.WaitGroup
	msgHandleFn  MsgHandle
}

func NewP2P(config *config.P2P) P2P {
	p2p := &p2p{id: config.NodeId, netId: config.NetId, addr: "localhost:" + strconv.Itoa(config.Port), closed: make(chan struct{}), linkBootAddr: config.LinkBootAddr}
	return p2p
}

func (self *p2p) Id() string {
	return self.id
}
func (self *p2p) BestPeer() (Peer, error) {
	if len(self.peers) > 0 {
		for _, v := range self.peers {
			return v, nil
		}
	}
	return nil, errors.New("can't find best peer.")
}

func (self *p2p) AllPeer() ([]Peer, error) {
	var result []Peer
	for _, v := range self.peers {
		result = append(result, v)
	}
	if len(result) > 0 {
		return result, nil
	}
	return nil, nil
}

func (self *p2p) SetHandlerFn(handler MsgHandle) {
	if self.Status() >= common.PreStart {
		panic("p2p has started, could not set handleFn.")
	}
	self.msgHandleFn = handler
}
func (self *p2p) SetHandShaker(hs HandShaker) {
	if self.Status() >= common.PreInit {
		panic("p2p has started, could not set HandShaker.")
	}
	self.bizHs = hs
}

func (self *p2p) addPeer(peer *peer) {
	self.mu.Lock()
	defer self.mu.Unlock()
	old, ok := self.peers[peer.peerId]
	if ok && old != peer {
		log.Warn("peer exist, close new peer: %v", peer.info())
		peer.close()
		return
	}
	self.peers[peer.peerId] = peer
	go self.loopRead(peer)
	go peer.loopWrite()
}
func (self *p2p) loopRead(peer *peer) {
	self.loopWg.Add(1)
	defer self.loopWg.Done()
	conn := peer.conn
	defer peer.close()
	defer delete(self.peers, peer.peerId)
	if self.msgHandleFn != nil {
		self.msgHandleFn(common.PeerConnected, nil, peer)
		defer self.msgHandleFn(common.PeerClosed, nil, peer)
	}
	for {
		select {
		case <-self.closed:
			log.Info("peer[%s] closed.", peer.info())
			return
		default:
			messageType, p, err := conn.ReadMessage()
			if messageType == websocket.CloseMessage {
				log.Warn("read closed message, peer: %s", peer.info())
				return
			}
			if err != nil {
				log.Error("read message error, peer: %s, err:%v", peer.info(), err)
				return
			}
			if messageType == websocket.BinaryMessage {
				msg := &Msg{}
				err := json.Unmarshal(p, msg)
				if err != nil {
					log.Error("serialize msg fail. messageType:%d, msg:%v", messageType, p)
					continue
				}
				if self.msgHandleFn != nil {
					self.msgHandleFn(msg.T, msg.Data, peer)
				}
			} else {
				log.Info("read message: %s", string(p))
			}
		}
	}
}

func (self *p2p) allPeers() map[string]*peer {
	self.mu.Lock()
	defer self.mu.Unlock()
	result := make(map[string]*peer, len(self.peers))
	for k, v := range self.peers {
		result[k] = v
	}
	return result
}

func (self *p2p) Init() {
	defer self.PreInit().PostInit()
	self.pendingDials = make(map[string]string)
	self.peers = make(map[string]*peer)
	if self.bizHs == nil {
		self.bizHs = &defaultHandShaker{}
	}
	self.hs = &handShaker{self, self.bizHs}
	self.dial = &dial{p2p: self, hs: self.hs}
	self.server = &server{id: self.id, addr: self.addr, bootAddr: self.linkBootAddr, p2p: self, hs: self.hs}
	self.linker = newLinker(self, url.URL{Scheme: "ws", Host: self.linkBootAddr, Path: "/ws"})
}
func (self *p2p) Start() {
	defer self.PreStart().PostStart()
	self.server.start()
	self.linker.start()
	go self.loop()
}

func (self *p2p) loop() {
	self.loopWg.Add(1)
	defer self.loopWg.Done()
	for {
		ticker := time.NewTicker(3 * time.Second)

		select {
		case <-ticker.C:
			for i, v := range self.pendingDials {
				_, ok := self.peers[i]
				if !ok {
					log.Info("node " + self.server.id + " try to connect to " + i)
					connectted := self.dial.connect(v)
					if connectted {
						log.Info("connect success." + self.server.id + ":" + i)
						delete(self.pendingDials, i)
					} else {
						delete(self.pendingDials, i)
					}
				} else {
					log.Info("has connected for " + self.server.id + ":" + i)
				}
			}
		case <-self.closed:
			log.Info("p2p[%s] closed.", self.id)
			return
		}
	}
}

func (self *p2p) Stop() {
	defer self.PreStop().PostStop()
	self.linker.stop()
	for _, v := range self.peers {
		v.stop()
	}
	self.server.stop()
	close(self.closed)
	self.loopWg.Wait()
}

func (self *p2p) addDial(id string, addr string) bool {
	self.mu.Lock()
	defer self.mu.Unlock()
	if id == self.id {
		return false
	}
	_, pok := self.peers[id]
	_, dok := self.pendingDials[id]
	if !pok && !dok {
		self.pendingDials[id] = addr
		return true
	}
	return false
}
