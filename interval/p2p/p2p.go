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

func (handler *defaultHandShaker) GetState() (interface{}, error) {
	return nil, nil
}

func (handler *defaultHandShaker) Handshake(peerId string, state []byte) error {
	return nil
}

func (handler *defaultHandShaker) DecodeState(state []byte) interface{} {
	return nil
}

func (handler *defaultHandShaker) EncodeState(state interface{}) []byte {
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

func (pp *p2p) Id() string {
	return pp.id
}
func (pp *p2p) BestPeer() (Peer, error) {
	if len(pp.peers) > 0 {
		for _, v := range pp.peers {
			return v, nil
		}
	}
	return nil, errors.New("can't find best peer.")
}

func (pp *p2p) AllPeer() ([]Peer, error) {
	var result []Peer
	for _, v := range pp.peers {
		result = append(result, v)
	}
	if len(result) > 0 {
		return result, nil
	}
	return nil, nil
}

func (pp *p2p) SetHandlerFn(handler MsgHandle) {
	if pp.Status() >= common.PreStart {
		panic("p2p has started, could not set handleFn.")
	}
	pp.msgHandleFn = handler
}
func (pp *p2p) SetHandShaker(hs HandShaker) {
	if pp.Status() >= common.PreInit {
		panic("p2p has started, could not set HandShaker.")
	}
	pp.bizHs = hs
}

func (pp *p2p) addPeer(peer *peer) {
	pp.mu.Lock()
	defer pp.mu.Unlock()
	old, ok := pp.peers[peer.peerId]
	if ok && old != peer {
		log.Warn("peer exist, close new peer: %v", peer.info())
		peer.close()
		return
	}
	pp.peers[peer.peerId] = peer
	go pp.loopRead(peer)
	go peer.loopWrite()
}
func (pp *p2p) loopRead(peer *peer) {
	pp.loopWg.Add(1)
	defer pp.loopWg.Done()
	conn := peer.conn
	defer peer.close()
	defer delete(pp.peers, peer.peerId)
	if pp.msgHandleFn != nil {
		pp.msgHandleFn(common.PeerConnected, nil, peer)
		defer pp.msgHandleFn(common.PeerClosed, nil, peer)
	}
	for {
		select {
		case <-pp.closed:
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
				if pp.msgHandleFn != nil {
					pp.msgHandleFn(msg.T, msg.Data, peer)
				}
			} else {
				log.Info("read message: %s", string(p))
			}
		}
	}
}

func (pp *p2p) allPeers() map[string]*peer {
	pp.mu.Lock()
	defer pp.mu.Unlock()
	result := make(map[string]*peer, len(pp.peers))
	for k, v := range pp.peers {
		result[k] = v
	}
	return result
}

func (pp *p2p) Init() {
	defer pp.PreInit().PostInit()
	pp.pendingDials = make(map[string]string)
	pp.peers = make(map[string]*peer)
	if pp.bizHs == nil {
		pp.bizHs = &defaultHandShaker{}
	}
	pp.hs = &handShaker{pp, pp.bizHs}
	pp.dial = &dial{p2p: pp, hs: pp.hs}
	pp.server = &server{id: pp.id, addr: pp.addr, bootAddr: pp.linkBootAddr, p2p: pp, hs: pp.hs}
	pp.linker = newLinker(pp, url.URL{Scheme: "ws", Host: pp.linkBootAddr, Path: "/ws"})
}
func (pp *p2p) Start() {
	defer pp.PreStart().PostStart()
	pp.server.start()
	pp.linker.start()
	go pp.loop()
}

func (pp *p2p) loop() {
	pp.loopWg.Add(1)
	defer pp.loopWg.Done()
	for {
		ticker := time.NewTicker(3 * time.Second)

		select {
		case <-ticker.C:
			for i, v := range pp.pendingDials {
				_, ok := pp.peers[i]
				if !ok {
					log.Info("node " + pp.server.id + " try to connect to " + i)
					connectted := pp.dial.connect(v)
					if connectted {
						log.Info("connect success." + pp.server.id + ":" + i)
						delete(pp.pendingDials, i)
					} else {
						delete(pp.pendingDials, i)
					}
				} else {
					log.Info("has connected for " + pp.server.id + ":" + i)
				}
			}
		case <-pp.closed:
			log.Info("p2p[%s] closed.", pp.id)
			return
		}
	}
}

func (pp *p2p) Stop() {
	defer pp.PreStop().PostStop()
	pp.linker.stop()
	for _, v := range pp.peers {
		v.stop()
	}
	pp.server.stop()
	close(pp.closed)
	pp.loopWg.Wait()
}

func (pp *p2p) addDial(id string, addr string) bool {
	pp.mu.Lock()
	defer pp.mu.Unlock()
	if id == pp.id {
		return false
	}
	_, pok := pp.peers[id]
	_, dok := pp.pendingDials[id]
	if !pok && !dok {
		pp.pendingDials[id] = addr
		return true
	}
	return false
}
