package p2p

import (
	"encoding/json"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/seiflotfy/cuckoofilter"
	"github.com/vitelabs/go-vite/crypto"
	"github.com/vitelabs/go-vite/p2p/discovery"
	"github.com/vitelabs/go-vite/p2p/protos"
	"gopkg.in/Shopify/sarama.v1"
	"io/ioutil"
	"time"
)

// @section kafka
type producer struct {
	prod sarama.AsyncProducer
}

func newProducer(addrs []string) (*producer, error) {
	config := sarama.NewConfig()
	prod, err := sarama.NewAsyncProducer(addrs, config)

	if err != nil {
		p2pServerLog.Error("can`t create sarama.AsyncProducer", "error", err)
		return nil, err
	}

	return &producer{
		prod: prod,
	}, nil
}

func (p *producer) write(topic string, data []byte) {
	p.prod.Input() <- &sarama.ProducerMessage{
		Topic:     topic,
		Value:     sarama.ByteEncoder(data),
		Timestamp: time.Now(),
	}
}

// @section topo
type Topo struct {
	Pivot string   `json:"pivot"`
	Peers []string `json:"peers"`
}

// add Hash(32bit) to Front, use for determine if it has been received
func (t *Topo) Serialize() ([]byte, error) {
	data, err := proto.Marshal(&protos.Topo{
		Pivot: t.Pivot,
		Peers: t.Peers,
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

	return nil
}

func (t *Topo) Json() []byte {
	buf, _ := json.Marshal(t)
	return buf
}

type topoEvent struct {
	msg    *Msg
	sender *Peer
}

type topoHandler struct {
	record *cuckoofilter.CuckooFilter
	rec    chan *topoEvent
}

func newTopoHandler() *topoHandler {
	return &topoHandler{
		record: cuckoofilter.NewCuckooFilter(1000),
		rec:    make(chan *topoEvent, 10),
	}
}

func (th *topoHandler) Handle(event *topoEvent, svr *Server) {
	defer event.msg.Discard()

	hash := make([]byte, 32)
	n, err := event.msg.Payload.Read(hash)
	if n != 32 || err != nil {
		p2pServerLog.Info(fmt.Sprintf("receive invalid topoMsg from %s@%s", event.sender.ID(), event.sender.RemoteAddr()))
		return
	}

	if th.record.Lookup(hash) {
		p2pServerLog.Info("has received the same topoMsg")
		return
	}

	th.record.Insert(hash)
	go svr.peers.Traverse(func(id discovery.NodeID, p *Peer) {
		if p.ID() != event.sender.ID() {
			SendMsg(p.rw, *event.msg)
		}
	})

	if svr.producer != nil {
		data, err := ioutil.ReadAll(event.msg.Payload)
		if err != nil {
			p2pServerLog.Info("can`t read topoMsg payload, so can`t send to Kafka")
		} else {
			svr.producer.write("p2p_status_event", data)
		}
	}
}
