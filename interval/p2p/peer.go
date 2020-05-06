package p2p

import (
	"net"
	"sync"

	"encoding/json"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/interval/common/log"
)

type closeOnce struct {
	closed     chan struct{}
	once       sync.Once
	writeCh    chan []byte
	writeChCap int
}

type peer struct {
	closeOnce
	peerId      string
	selfId      string
	peerSrvAddr string
	conn        *websocket.Conn
	remoteAddr  net.Addr
	loopWg      sync.WaitGroup
	state       interface{}
}

func (self *peer) SetState(s interface{}) {
	self.state = s
}
func (self *peer) GetState() interface{} {
	if self.state == nil {
		return nil
	}
	return self.state
}

func (self *peer) Write(msg *Msg) error {
	byt, err := json.Marshal(msg)
	if err != nil {
		log.Error("serialize msg fail. err:%v, msg:%v", err, msg)
		return err
	}

	select {
	case self.writeCh <- byt:
	default:
		log.Warn("write channel is full and message will be discarded.")
		return errors.New("write channel is full.")
	}
	return nil
}

func (self *peer) Id() string {
	return string(self.peerId)
}

func (self *peer) RemoteAddr() string {
	return self.remoteAddr.String()
}

func (self *peer) close() {
	self.once.Do(self.realClose)
}
func (self *peer) realClose() {
	close(self.closed)
	close(self.writeCh)
	self.conn.Close()
}

//func (self *peer) loop() {
//	conn := self.conn
//	defer self.close()
//	self.loopWg.Add(1)
//	defer self.loopWg.Done()
//	for {
//		select {
//		case <-self.closed:
//			log.Info("peer[%s] closed.", self.info())
//			return
//		default:
//			messageType, p, err := conn.ReadMessage()
//			if messageType == websocket.CloseMessage {
//				log.Warn("read closed message, peer: %s", self.info())
//				return
//			}
//			if err != nil {
//				log.Error("read message error, peer: %s, err:%v", self.info(), err)
//				return
//			}
//			log.Info("read message: %s", string(p))
//		}
//	}
//}
func (self *peer) stop() {
	self.close()
	self.loopWg.Wait()
}

func newPeer(fromId string, toId string, peerSrvAddr string, conn *websocket.Conn, s interface{}) *peer {
	c := conn.CloseHandler()
	remoteAddr := conn.RemoteAddr()
	peer := &peer{peerId: fromId, selfId: toId, peerSrvAddr: peerSrvAddr, conn: conn, remoteAddr: remoteAddr, state: s}
	peer.closed = make(chan struct{})
	peer.writeChCap = 1000
	peer.writeCh = make(chan []byte, peer.writeChCap)
	conn.SetCloseHandler(func(code int, text string) error {
		log.Info("peer received closed msg. %s, %v", peer.info(), remoteAddr)
		return c(code, text)
	})
	return peer
}

func (self *peer) info() string {
	return "[" + self.selfId + "]-[" + self.peerId + "]"
}

func (self *peer) loopWrite() {
	self.loopWg.Add(1)
	defer self.loopWg.Done()
	defer self.close()

	for {
		select {
		case m, ok := <-self.writeCh:
			if ok {
				self.conn.WriteMessage(websocket.BinaryMessage, m)
			}
		case <-self.closed:
			log.Warn("peer[%s] write closed.", self.peerId)
			return
		}
	}
}
