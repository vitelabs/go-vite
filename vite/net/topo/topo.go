package topo

import (
	"encoding/json"
	"fmt"
	"github.com/vitelabs/go-vite/p2p/list"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/seiflotfy/cuckoofilter"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/crypto"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/monitor"
	"github.com/vitelabs/go-vite/p2p"
	"github.com/vitelabs/go-vite/p2p/protos"
	"gopkg.in/Shopify/sarama.v1"
)

const Name = "Topo"
const CmdSet = 3
const topoCmd = 1

var defaultTopic = "p2p_status_event"
var defaultInterval int64 = 60

type Config struct {
	Addrs    []string
	Interval int64 // second
	Topic    string
}

func safeConfig(cfg *Config) *Config {
	if cfg == nil {
		return &Config{
			Interval: defaultInterval,
			Topic:    defaultTopic,
		}
	}

	if cfg.Topic == "" {
		cfg.Topic = defaultTopic
	}
	if cfg.Interval == 0 {
		cfg.Interval = defaultInterval
	}

	return cfg
}

type Topology struct {
	*Config
	p2p       *p2p.Server
	peers     *sync.Map
	peerCount int32 // atomic
	prod      sarama.AsyncProducer
	log       log15.Logger
	term      chan struct{}
	rec       chan *Event
	record    *cuckoofilter.CuckooFilter
	wg        sync.WaitGroup
}

type Event struct {
	msg    *p2p.Msg
	sender *Peer
}

func New(cfg *Config) *Topology {
	cfg = safeConfig(cfg)

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

	if addrs := rmEmptyString(t.Config.Addrs); len(addrs) > 0 {
		config := sarama.NewConfig()
		if prod, err := sarama.NewAsyncProducer(addrs, config); err != nil {
			t.log.Error(fmt.Sprintf("create topo producer error: %v", err))
			return err
		} else {
			t.log.Info("topo producer created")
			t.prod = prod
		}
	}

	t.wg.Add(1)
	common.Go(t.sendLoop)

	t.wg.Add(1)
	common.Go(t.handleLoop)

	return nil
}

func (t *Topology) Stop() {
	if t.term == nil {
		return
	}

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
	rw      p2p.MsgReadWriter
	errChan chan error // async handle msg, error report to this channel
}

func (t *Topology) Handle(p *p2p.Peer, rw *p2p.ProtoFrame) error {
	peer := &Peer{p, rw, make(chan error)}
	t.peers.Store(p.String(), peer)
	defer t.peers.Delete(p.String())

	atomic.AddInt32(&t.peerCount, 1)
	defer atomic.AddInt32(&t.peerCount, ^int32(0))

	for {
		select {
		case <-t.term:
			return nil
		case err := <-peer.errChan:
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

			if len(msg.Payload) < 32 {
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

	var hash []byte
	queue := list.New()

	for {
		select {
		case <-t.term:
			return
		case e := <-t.rec:
			if queue.Size() > 30 {
				queue.Clear()
			}

			hash = e.msg.Payload[:32]
			if t.record.InsertUnique(hash) {
				queue.Append(e)
			}

		default:
			if e := queue.Shift(); e != nil {
				e := e.(*Event)
				t.Receive(e.msg, e.sender)
			} else {
				time.Sleep(20 * time.Millisecond)
			}
		}
	}
}

func (t *Topology) sendLoop() {
	defer t.wg.Done()

	ticker := time.NewTicker(time.Duration(t.Config.Interval * int64(time.Second)))
	defer ticker.Stop()

	var msg *Topo

	for {
		select {
		case <-t.term:
			return

		case <-ticker.C:
			monitor.LogEvent("topo", "send")

			msg = t.Topology()
			if data, err := msg.Serialize(); err != nil {
				t.log.Error(fmt.Sprintf("serialize topo error: %v", err))
			} else {
				t.broadcast(&p2p.Msg{
					CmdSet:  CmdSet,
					Cmd:     topoCmd,
					Payload: data,
				}, nil)

				t.write(t.Topic, msg.Json())
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
	topoMsg := new(Topo)

	if err := topoMsg.Deserialize(msg.Payload[32:]); err != nil {
		t.log.Error(fmt.Sprintf("deserialize topoMsg error: %v", err))
		sender.errChan <- err
		return
	}

	monitor.LogEvent("topo", "receive")

	// broadcast to other peer
	t.broadcast(msg, sender)

	// report to kafka
	t.write("p2p_status_event", topoMsg.Json())
}

func (t *Topology) write(topic string, data []byte) {
	if t.prod == nil {
		return
	}

	defer monitor.LogTime("topo", "kafka", time.Now())

	t.prod.Input() <- &sarama.ProducerMessage{
		Topic:     topic,
		Value:     sarama.ByteEncoder(data),
		Timestamp: time.Now(),
	}

	monitor.LogEvent("topo", "report")
}

func (t *Topology) broadcast(msg *p2p.Msg, except *Peer) {
	var count int32 = 0
	var id string
	var p *Peer
	t.peers.Range(func(key, value interface{}) bool {
		id, p = key.(string), value.(*Peer)

		if except != nil && id == except.String() {
			// do nothing
		} else {
			p.rw.WriteMsg(msg)
			count++
		}

		// just broadcast to 1/3 peers
		if count > t.peerCount/3 {
			return false
		}

		return true
	})
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

func calcDuration(t int64) time.Duration {
	return time.Duration(t * int64(time.Second))
}

func rmEmptyString(strs []string) (ret []string) {
	if len(strs) == 0 {
		return
	}

	var i, j int
	for i = 0; i < len(strs); i++ {
		if strs[i] != "" {
			strs[j] = strs[i]
			j++
		}
	}

	return strs[:j]
}
