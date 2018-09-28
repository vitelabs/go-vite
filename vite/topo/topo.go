package topo

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/seiflotfy/cuckoofilter"
	"github.com/vitelabs/go-vite/crypto"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/monitor"
	"github.com/vitelabs/go-vite/p2p"
	"github.com/vitelabs/go-vite/vite/topo/protos"
	"gopkg.in/Shopify/sarama.v1"
	"sync"
	"time"
)

const Name = "Topo"
const CmdSet = 7
const topoCmd = 1

type TopoHandler struct {
	peers  *sync.Map
	prod   sarama.AsyncProducer
	log    log15.Logger
	term   chan struct{}
	record *cuckoofilter.CuckooFilter
	p2p    *p2p.Server
}

func New(addrs []string) (p *TopoHandler, err error) {
	p = &TopoHandler{
		peers:  new(sync.Map),
		log:    log15.New("module", "Topo"),
		term:   make(chan struct{}),
		record: cuckoofilter.NewCuckooFilter(1000),
	}

	if len(addrs) != 0 {
		var i, j int
		for i = 0; i < len(addrs); i++ {
			if addrs[i] != "" {
				addrs[j] = addrs[i]
			}
		}
		addrs = addrs[:j]
		if len(addrs) != 0 {
			config := sarama.NewConfig()
			prod, err := sarama.NewAsyncProducer(addrs, config)

			if err != nil {
				p.log.Error("can`t create sarama.AsyncProducer", "error", err)
				return p, err
			}

			p.prod = prod
		}
	}

	return p, nil
}

func (t *TopoHandler) Start(svr *p2p.Server) {
	t.p2p = svr
}

func (t *TopoHandler) Stop() {
	select {
	case <-t.term:
	default:
		close(t.term)
	}
}

type Peer struct {
	*p2p.Peer
	rw p2p.MsgReadWriter
}

func (t *TopoHandler) Handle(p *p2p.Peer, rw p2p.MsgReadWriter) error {
	peer := &Peer{p, rw}
	t.peers.Store(p.String(), peer)
	defer t.peers.Delete(p.String())

	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-t.term:
			return nil

		case <-ticker.C:
			monitor.LogEvent("topo", "send")
			topo := t.Topology()

			data, err := topo.Serialize()
			if err != nil {
				t.log.Error(fmt.Sprintf("serialize topo error: %v", err))
			} else {
				rw.WriteMsg(&p2p.Msg{
					CmdSetID: CmdSet,
					Cmd:      topoCmd,
					Id:       0,
					Size:     uint64(len(data)),
					Payload:  data,
				})
			}
		default:
			msg, err := rw.ReadMsg()
			if err != nil {
				t.log.Error(fmt.Sprintf("read msg error: %v", err))
				return err
			}

			if msg.Cmd != topoCmd {
				t.log.Error(fmt.Sprintf("not topoMsg cmd: %d", msg.Cmd))
				return nil
			}

			t.Receive(msg, peer)
		}
	}
}

// the first item is self url
func (t *TopoHandler) Topology() *Topo {
	topo := &Topo{
		Pivot: t.p2p.URL(),
		Peers: make([]string, 0, 10),
	}

	t.peers.Range(func(key, value interface{}) bool {
		str := key.(string)
		topo.Peers = append(topo.Peers, str)
		return true
	})

	return topo
}

func (t *TopoHandler) Receive(msg *p2p.Msg, sender *Peer) {
	defer msg.Discard()

	length := len(msg.Payload)

	if length < 32 {
		t.log.Info(fmt.Sprintf("receive invalid topoMsg from %s@%s", sender.ID(), sender.RemoteAddr()))
		return
	}

	hash := msg.Payload[:32]
	if t.record.Lookup(hash) {
		t.log.Info(fmt.Sprintf("has received the same topoMsg: %s", hex.EncodeToString(hash)))
		return
	}

	topo := new(Topo)
	err := topo.Deserialize(msg.Payload[32:])
	if err != nil {
		t.log.Error(fmt.Sprintf("deserialize topoMsg error: %v", err))
		return
	}

	monitor.LogEvent("topo", "receive")

	t.record.InsertUnique(hash)
	t.peers.Range(func(key, value interface{}) bool {
		id := key.(string)
		if id != sender.String() {
			sender.rw.WriteMsg(msg)
		}
		return true
	})

	if t.prod != nil {
		monitor.LogEvent("topo", "report")
		t.write("p2p_status_event", topo.Json())
		t.log.Info("report topoMsg to kafka")
	}
}

func (t *TopoHandler) write(topic string, data []byte) {
	t.prod.Input() <- &sarama.ProducerMessage{
		Topic:     topic,
		Value:     sarama.ByteEncoder(data),
		Timestamp: time.Now(),
	}
}

func (t *TopoHandler) Protocol() *p2p.Protocol {
	return &p2p.Protocol{
		Name:   Name,
		ID:     CmdSet,
		Handle: t.Handle,
	}
}

// @section topo
type Topo struct {
	Pivot string    `json:"pivot"`
	Peers []string  `json:"peers"`
	Time  time.Time `json:"time"`
}

// add Hash(32bit) to Front, use for determine if it has been received
func (t *Topo) Serialize() ([]byte, error) {
	data, err := proto.Marshal(&protos.Topo{
		Pivot: t.Pivot,
		Peers: t.Peers,
		Time:  t.Time.Unix(),
	})

	if err != nil {
		return nil, err
	}

	hash := crypto.Hash(32, data)

	return append(hash, data...), nil
}

func (t *Topo) Deserialize(buf []byte) error {
	pb := new(protos.Topo)
	err := proto.Unmarshal(buf, pb)
	if err != nil {
		return err
	}

	t.Pivot = pb.Pivot
	t.Peers = pb.Peers
	t.Time = time.Unix(pb.Time, 0)

	return nil
}

func (t *Topo) Json() []byte {
	buf, _ := json.Marshal(t)
	return buf
}
