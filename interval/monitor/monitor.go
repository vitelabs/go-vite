package monitor

import (
	"encoding/json"
	"os"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vitelabs/go-vite/log15"
)

func init() {
	m = &monitor{r: newRing(60)}
	//设置日志文件地址
	dir := "/Users/jie/log/test.log"
	log15.Info(dir)
	log15.Root().SetHandler(
		log15.LvlFilterHandler(log15.LvlInfo, log15.Must.FileHandler(dir, log15.JsonFormat())),
	)
	logger = log15.New("logtype", "1", "appkey", "govite", "PID", strconv.Itoa(os.Getpid()))
	go loop()
}

var m *monitor

var logger log15.Logger

type monitor struct {
	ms sync.Map
	r  *ring
}

type msg struct {
	Group string
	Type  int
	Name  string
	Cnt   int64
	Sum   int64
}

func (self *msg) add(i int64) *msg {
	atomic.AddInt64(&self.Cnt, 1)
	atomic.AddInt64(&self.Sum, i)
	return self
}

func (self *msg) String() string {
	return "{\"Cnt\":" + strconv.FormatInt(self.Cnt, 10) + ",\"Sum\":" + strconv.FormatInt(self.Sum, 10) + "}"
}

func (self *msg) reset() *msg {
	atomic.StoreInt64(&self.Sum, 0)
	atomic.StoreInt64(&self.Cnt, 0)
	return self
}
func (self *msg) snapshot() msg {
	return *self
}

func newMsg(g string, name string, t int) *msg {
	return &msg{Group: g, Name: name, Type: t}
}

func key(t string, name string) string {
	return t + "-" + name
}
func LogEvent(t string, name string) {
	log(t, name, 1, 1)
}

func LogTime(t string, name string, tm time.Time) {
	log(t, name, 2, time.Now().Sub(tm).Nanoseconds())
}

func LogDuration(t string, name string, duration int64) {
	log(t, name, 2, duration)
}

func log(g string, name string, t int, i int64) {
	k := key(g, name)
	value, ok := m.ms.Load(k)
	if ok {
		value.(*msg).add(i)
	} else {
		m.ms.Store(k, newMsg(g, name, t).add(i))
	}
}

type Stat struct {
	Name    string
	Type    string
	Cnt     int64
	Sum     int64
	Avg     float64 // Sum / Cnt
	CntMean float64 // Cnt / Cap
	Cap     int32
}

func (self *Stat) merge(ms *msg) *Stat {
	self.Cnt += ms.Cnt
	self.Sum += ms.Sum
	self.Cap++
	return self
}
func (self *Stat) avg() *Stat {
	if self.Cnt > 0 {
		self.Avg = float64(self.Sum) / float64(self.Cnt)

	}

	if self.Cap > 0 {
		self.CntMean = float64(self.Cnt) / float64(self.Cap)
	}
	return self
}

func Stats() []*Stat {
	msgs := stats()

	var r []*Stat
	for _, v := range msgs {
		r = append(r, v)
	}

	sort.Sort(byStr(r))
	return r
}

func StatsJson() string {
	r := stats()
	b, _ := json.Marshal(r)
	return string(b)
}

func stats() map[string]*Stat {
	all := m.r.all()
	msgs := make(map[string]*Stat)
	for _, v := range all {
		msgM := v.(map[string]*msg)
		for k2, v2 := range msgM {
			if v2.Cnt <= 0 {
				continue
			}
			tmpM, ok := msgs[k2]
			if ok {
				tmpM.merge(v2)
			} else {
				s := &Stat{Name: v2.Name, Type: v2.Group}
				s.merge(v2)
				msgs[k2] = s
			}
		}
	}
	for _, v := range msgs {
		v.avg()
	}
	return msgs
}

type byStr []*Stat

func (a byStr) Len() int      { return len(a) }
func (a byStr) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a byStr) Less(i, j int) bool {
	if a[i].Type == a[j].Type {
		return a[i].Name < a[j].Name
	} else {
		return a[i].Type < a[j].Type
	}
}

func loop() {
	t := time.NewTicker(time.Second * 1)
	for {

		select {
		case <-t.C:
			snapshot := make(map[string]*msg)

			m.ms.Range(func(k, v interface{}) bool {
				tmpM := v.(*msg)
				c := tmpM.Cnt
				s := tmpM.Sum
				t := tmpM.Type
				key := k.(string)
				logger.Info("",
					"group", "monitor",
					"name", key,
					"interval", 1,
					"metric-cnt", c,
					"metric-sum", s,
					"metric-type", t,
				)
				sm := tmpM.snapshot()
				snapshot[key] = &sm
				tmpM.reset()
				return true
			})
			m.r.add(snapshot)
		}
	}

}
