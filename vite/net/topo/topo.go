package topo

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/seiflotfy/cuckoofilter"
	"github.com/vitelabs/go-vite/crypto"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/monitor"
	"github.com/vitelabs/go-vite/p2p"
	"github.com/vitelabs/go-vite/p2p/protos"
	"gopkg.in/Shopify/sarama.v1"
	"strconv"
	"sync"
	"time"
)

const Name = "Topo"
const CmdSet = 3
const topoCmd = 1

type Config struct {
	Addrs    []string
	Interval int64 // second
	Topic    string
}

type Topology struct {
	*Config
	p2p    *p2p.Server
	peers  *sync.Map
	prod   sarama.AsyncProducer
	log    log15.Logger
	term   chan struct{}
	rec    chan *Event
	record *cuckoofilter.CuckooFilter
	wg     sync.WaitGroup
}

type Event struct {
	msg    *p2p.Msg
	sender *Peer
}

func New(cfg *Config) *Topology {
	if cfg.Topic == "" {
		cfg.Topic = "p2p_status_event"
	}
	if cfg.Interval == 0 {
		cfg.Interval = 5
	}

	return &Topology{
		Config: cfg,
		peers:  new(sync.Map),
		log:    log15.New("module", "Topo"),
		rec:    make(chan *Event, 10),
		record: cuckoofilter.NewCuckooFilter(1000),
	}
}

var errMissingP2P = errors.New("missing p2p server")

func (t *Topology) Start(p2p *p2p.Server) error {
	t.term = make(chan struct{})

	if p2p == nil {
		return errMissingP2P
	}
	t.p2p = p2p

	if len(t.Config.Addrs) > 0 {
		config := sarama.NewConfig()
		prod, err := sarama.NewAsyncProducer(t.Config.Addrs, config)

		if err != nil {
			t.log.Error(fmt.Sprintf("create topo producer error: %v", err))
			return err
		}

		t.log.Info("topo producer created")
		t.prod = prod
	}

	t.wg.Add(1)
	go t.sendLoop()

	t.wg.Add(1)
	go t.handleLoop()

	return nil
}

func (t *Topology) Stop() {
	select {
	case <-t.term:
	default:
		t.log.Info("topo stop")

		close(t.term)

		if t.prod != nil {
			t.prod.Close()
		}

		t.wg.Wait()

		t.log.Info("topo stopped")
	}
}

type Peer struct {
	*p2p.Peer
	rw    p2p.MsgReadWriter
	errch chan error // async handle msg, error report to this channel
}

func (t *Topology) Handle(p *p2p.Peer, rw p2p.MsgReadWriter) error {
	peer := &Peer{p, rw, make(chan error, 1)}
	t.peers.Store(p.String(), peer)
	defer t.peers.Delete(p.String())

	for {
		select {
		case <-t.term:
			return nil
		case err := <-peer.errch:
			return err
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

			length := len(msg.Payload)

			if length < 32 {
				return fmt.Errorf("receive invalid topoMsg from %s", p)
			}

			t.rec <- &Event{
				msg:    msg,
				sender: peer,
			}
		}
	}
}

func (t *Topology) handleLoop() {
	defer t.wg.Done()

	for {
		select {
		case <-t.term:
			return
		case e := <-t.rec:
			t.Receive(e.msg, e.sender)
		}
	}
}

func (t *Topology) sendLoop() {
	defer t.wg.Done()

	ticker := time.NewTicker(time.Duration(t.Config.Interval * int64(time.Second)))
	defer ticker.Stop()

	for {
		select {
		case <-t.term:
			return

		case <-ticker.C:
			monitor.LogEvent("topo", "send")
			topo := t.Topology()

			data, err := topo.Serialize()
			if err != nil {
				t.log.Error(fmt.Sprintf("serialize topo error: %v", err))
			} else {
				t.peers.Range(func(key, value interface{}) bool {
					peer := value.(*Peer)
					peer.rw.WriteMsg(&p2p.Msg{
						CmdSetID: CmdSet,
						Cmd:      topoCmd,
						Size:     uint64(len(data)),
						Payload:  data,
					})
					return true
				})

				t.write(t.Topic, topo.Json())
			}
		}
	}
}

// the first item is self url
func (t *Topology) Topology() *Topo {
	topo := &Topo{
		Pivot: t.p2p.URL(),
		Peers: make([]*p2p.ConnProperty, 0, 10),
		Time:  UnixTime(time.Now()),
	}

	t.peers.Range(func(key, value interface{}) bool {
		p := value.(*Peer)
		topo.Peers = append(topo.Peers, p.GetConnProperty())
		return true
	})

	return topo
}

func (t *Topology) Receive(msg *p2p.Msg, sender *Peer) {
	defer msg.Discard()

	hash := msg.Payload[:32]
	if t.record.Lookup(hash) {
		t.log.Warn(fmt.Sprintf("has received the same topoMsg: %s", hex.EncodeToString(hash)))
		return
	}

	topo := new(Topo)
	err := topo.Deserialize(msg.Payload[32:])
	if err != nil {
		t.log.Error(fmt.Sprintf("deserialize topoMsg error: %v", err))

		select {
		case sender.errch <- err:
		default:
		}

		return
	}

	monitor.LogEvent("topo", "receive")

	t.record.InsertUnique(hash)
	// broadcast to other peer
	t.peers.Range(func(key, value interface{}) bool {
		id := key.(string)
		p := value.(*Peer)
		if id != sender.String() {
			p.rw.WriteMsg(msg)
		}
		return true
	})

	t.write("p2p_status_event", topo.Json())
}

func (t *Topology) write(topic string, data []byte) {
	if t.prod == nil {
		return
	}

	t.prod.Input() <- &sarama.ProducerMessage{
		Topic:     topic,
		Value:     sarama.ByteEncoder(data),
		Timestamp: time.Now(),
	}

	monitor.LogEvent("topo", "report")
	t.log.Info("report topoMsg to kafka")
}

func (t *Topology) Protocol() *p2p.Protocol {
	return &p2p.Protocol{
		Name:   Name,
		ID:     CmdSet,
		Handle: t.Handle,
	}
}

// @section topo
type UnixTime time.Time

func (t UnixTime) MarshalJSON() ([]byte, error) {
	stamp := fmt.Sprintf("%d", t.Unix())
	return []byte(stamp), nil
}

func (t *UnixTime) UnmarshalJSON(data []byte) (err error) {
	i64, err := strconv.ParseInt(string(data), 10, 64)
	if err != nil {
		return
	}

	*t = UnixTime(time.Unix(i64, 0))
	return nil
}

func (t UnixTime) Unix() int64 {
	return time.Time(t).Unix()
}

type Topo struct {
	Pivot string              `json:"pivot,omitempty"`
	Peers []*p2p.ConnProperty `json:"peers,omitempty"`
	Time  UnixTime            `json:"time,omitempty"`
}

// add Hash(32bit) to Front, use for determine if it has been received
func (t *Topo) Serialize() ([]byte, error) {
	pbs := make([]*protos.ConnProperty, len(t.Peers))

	for i, cp := range t.Peers {
		pbs[i] = cp.Proto()
	}

	data, err := proto.Marshal(&protos.Topo{
		Pivot: t.Pivot,
		Peers: pbs,
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

	for _, cpb := range pb.Peers {
		cp := new(p2p.ConnProperty)
		cp.Deproto(cpb)
		t.Peers = append(t.Peers, cp)
	}

	t.Pivot = pb.Pivot
	t.Time = UnixTime(time.Unix(pb.Time, 0))

	return nil
}

// report to kafka
func (t *Topo) Json() []byte {
	buf, _ := json.Marshal(t)
	return buf
}
