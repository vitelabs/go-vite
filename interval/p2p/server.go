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

func (s *server) ws(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err == nil {
		peer, err := s.hs.handshake(c)
		if err == nil {
			log.Info("client connect success, add new peer. %v", peer.peerId)
			s.p2p.addPeer(peer)
		} else {
			log.Error("client connect success, but handshake fail. err:%v", err)
		}
	} else {
		log.Error("upgrade error.", err)
	}
}

func (s *server) start() {
	//http.HandleFunc("/ws", s.ws)
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", s.ws)
	srv := &http.Server{Addr: s.addr, Handler: mux}
	s.srv = srv
	go srv.ListenAndServe()
}
func (s *server) loop() {
	s.srv.ListenAndServe()
}

func (s *server) stop() {
	s.srv.Shutdown(context.Background())
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
