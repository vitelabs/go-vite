package compress

import (
	"github.com/vitelabs/go-vite/log15"
	"sync"
	"time"
)

const (
	STOPPED = 0
	RUNNING = 1
)

var compressorLog = log15.New("module", "compressor")

type Compressor struct {
	stopSignal chan int
	status     int // 0 is stop, 1 is start
	statusLock sync.Mutex
}

func NewCompressor() *Compressor {
	return &Compressor{
		stopSignal: make(chan int),
		status:     STOPPED,
	}
}

func (c *Compressor) Start() bool {
	c.statusLock.Lock()
	defer c.statusLock.Unlock()

	if c.status == RUNNING {
		compressorLog.Error("Compressor is running, Don't start again.")
		return false
	}
	c.status = RUNNING

	go func() {
		for {
			select {
			case <-c.stopSignal:
				c.statusLock.Lock()
				defer c.statusLock.Unlock()

				c.stopSignal = make(chan int)
				c.status = STOPPED
				return
			default:
				time.Sleep(time.Microsecond * 500)
			}
		}
	}()

	return true
}

func (c *Compressor) Stop() {
	c.stopSignal <- 1
}
