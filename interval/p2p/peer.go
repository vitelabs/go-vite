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

func (p *peer) SetState(s interface{}) {
	p.state = s
}
func (p *peer) GetState() interface{} {
	if p.state == nil {
		return nil
	}
	return p.state
}

func (p *peer) Write(msg *Msg) error {
	byt, err := json.Marshal(msg)
	if err != nil {
		log.Error("serialize msg fail. err:%v, msg:%v", err, msg)
		return err
	}

	select {
	case p.writeCh <- byt:
	default:
		log.Warn("write channel is full and message will be discarded.")
		return errors.New("write channel is full.")
	}
	return nil
}

func (p *peer) Id() string {
	return string(p.peerId)
}

func (p *peer) RemoteAddr() string {
	return p.remoteAddr.String()
}

func (p *peer) close() {
	p.once.Do(p.realClose)
}
func (p *peer) realClose() {
	close(p.closed)
	close(p.writeCh)
	p.conn.Close()
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
func (p *peer) stop() {
	p.close()
	p.loopWg.Wait()
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

func (p *peer) info() string {
	return "[" + p.selfId + "]-[" + p.peerId + "]"
}

func (p *peer) loopWrite() {
	p.loopWg.Add(1)
	defer p.loopWg.Done()
	defer p.close()

	for {
		select {
		case m, ok := <-p.writeCh:
			if ok {
				p.conn.WriteMessage(websocket.BinaryMessage, m)
			}
		case <-p.closed:
			log.Warn("peer[%s] write closed.", p.peerId)
			return
		}
	}
}
