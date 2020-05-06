package p2p

import (
	"context"
	"net/http"
	"net/url"

	"github.com/gorilla/websocket"
	"github.com/vitelabs/go-vite/interval/common/log"
)

type server struct {
	id   string
	addr string
	p2p  *p2p
	hs   *handShaker

	bootAddr string
	srv      *http.Server
}

var upgrader = websocket.Upgrader{} // use default options

func (self *server) ws(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err == nil {
		peer, err := self.hs.handshake(c)
		if err == nil {
			log.Info("client connect success, add new peer. %v", peer.peerId)
			self.p2p.addPeer(peer)
		} else {
			log.Error("client connect success, but handshake fail. err:%v", err)
		}
	} else {
		log.Error("upgrade error.", err)
	}
}

func (self *server) start() {
	//http.HandleFunc("/ws", self.ws)
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", self.ws)
	srv := &http.Server{Addr: self.addr, Handler: mux}
	self.srv = srv
	go srv.ListenAndServe()
}
func (self *server) loop() {
	self.srv.ListenAndServe()
}

func (self *server) stop() {
	self.srv.Shutdown(context.Background())
}

type dial struct {
	p2p *p2p
	hs  *handShaker
}

func (self *dial) connect(addr string) bool {
	u := url.URL{Scheme: "ws", Host: addr, Path: "/ws"}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err == nil {
		peer, err := self.hs.handshake(c)
		if err == nil {
			log.Info("client connect success, add new peer. %v", peer.peerId)
			self.p2p.addPeer(peer)
		} else {
			log.Error("client connect success, but handshake fail. err:%v", err)
		}
		return true
	} else {
		log.Error("dial error.", err, addr)
		return false
	}
}
