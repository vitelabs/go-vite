package p2p

import (
	"encoding/json"
	"net/url"
	"strconv"
	"testing"
	"time"

	"fmt"

	"github.com/gorilla/websocket"
	"github.com/vitelabs/go-vite/interval/common/log"
)

func TestBootNode(t *testing.T) {
	b := bootnode{peers: make(map[string]*peer), closed: make(chan struct{})}
	s := "localhost:8000"
	b.start(s)
	//c := make(chan int)
	var last *websocket.Conn
	for i := 0; i < 10; i++ {
		conn := cliBootNode(s, strconv.Itoa(i))
		if last != nil {
			for {
				contain, e := contain(conn, s, strconv.Itoa(i-1))
				if e != nil {
					t.Error("error contain %v", strconv.Itoa(i), e)
					break
				}
				if !contain {
					break
				} else {
					time.Sleep(time.Second * time.Duration(2))
				}
			}
		}
		for {
			contain, e := contain(conn, s, strconv.Itoa(i))
			if e != nil {
				t.Error("error contain %v", strconv.Itoa(i), e)
				break
			}
			if contain {
				break
			} else {
				time.Sleep(time.Second * time.Duration(2))
			}
		}
		last = conn
		conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(time.Second))
	}
	b.Stop()
	//c <- 90
}

func cliBootNode(addr string, id string) *websocket.Conn {
	u := url.URL{Scheme: "ws", Host: addr, Path: "/ws"}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Error("dial:", err)
	}
	c.WriteJSON(&bootReq{Id: id, Addr: addr})
	return c
}

func contain(conn *websocket.Conn, targetAddr string, targetId string) (bool, error) {
	conn.WriteJSON(&bootReq{Tp: 1})
	log.Info("send request.")
	_, message, err := conn.ReadMessage()
	if err != nil {
		log.Error("read fail.", err)
		return false, err
	}
	log.Info("recv: %s", string(message))
	res := []bootReq{}
	json.Unmarshal(message, &res)
	result := false
	for _, r := range res {
		id := r.Id
		addr := r.Addr
		if id == targetId && addr == targetAddr {
			result = true
		}
	}
	return result, nil
}

func TestChannel(t *testing.T) {
	c := make(chan int)
	_, ok := <-c
	fmt.Println(ok)
}
