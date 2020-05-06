package p2p

import (
	"sort"
	"strconv"
	"testing"
	"time"

	"github.com/vitelabs/go-vite/interval/common/log"
)

func TestServerStart(t *testing.T) {
	bootAddr := "localhost:8000"

	startBootNode(bootAddr)

	N := 2
	var list []*p2p
	for i := 0; i < N; i++ {
		addr := "localhost:808" + strconv.Itoa(i)
		log.Info(addr)
		p2p := p2p{id: strconv.Itoa(i), addr: addr, closed: make(chan struct{}), linkBootAddr: bootAddr}
		p2p.Start()
		list = append(list, &p2p)
	}

	time.Sleep(time.Second * time.Duration(20))
	for _, v := range list {
		allPeers := v.allPeers()
		if !full(v.id, allPeers, N) {
			t.Errorf("error for p2p conn. id:%d, peers:%s.", v.id, allPeers)
		}
	}
	for _, v := range list {
		v.Stop()
	}

}
func startBootNode(s string) {
	b := bootnode{peers: make(map[string]*peer)}
	b.start(s)
	time.Sleep(time.Second * time.Duration(2))
}
func full(self string, peers map[string]*peer, N int) bool {
	var keys []string
	for k := range peers {
		keys = append(keys, k)
	}
	keys = append(keys, self)

	if len(keys) != N {
		return false
	}
	sort.Strings(keys)

	for k, v := range keys {
		if strconv.Itoa(k) != v {
			return false
		}
	}
	return true
}

func TestFull(t *testing.T) {
	N := 10
	var m = make(map[string]*peer)
	for i := 0; i < N; i++ {
		addr := "localhost:808" + strconv.Itoa(i)
		log.Info(addr)
		peer := peer{peerId: strconv.Itoa(i), selfId: strconv.Itoa(-1)}
		m[strconv.Itoa(i)] = &peer
	}
	if full("2", m, N) {
		t.Errorf("error not full.")
	}
	delete(m, "2")

	if !full("2", m, N) {
		t.Errorf("error full.")
	}
	if full("3", m, N) {
		t.Errorf("error not full")
	}
}
