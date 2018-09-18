package compress

import (
	"github.com/vitelabs/go-vite/log15"
	"path/filepath"
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
	wg         sync.WaitGroup

	dir string

	checkInterval time.Duration
	checkCount    int

	chain Chain

	indexer *Indexer
}

func NewCompressor(chain Chain, dataDir string) *Compressor {
	c := &Compressor{
		stopSignal: make(chan int),
		status:     STOPPED,
		chain:      chain,
		dir:        filepath.Join(dataDir, "ledger_files"),

		checkInterval: time.Second * 10,
		checkCount:    60,
	}

	c.indexer = NewIndexer(c.dir)
	return c
}

func (c *Compressor) Start() bool {
	c.statusLock.Lock()
	defer c.statusLock.Unlock()

	if c.status == RUNNING {
		compressorLog.Error("Compressor is running, Don't start again.")
		return false
	}
	c.status = RUNNING

	c.wg.Add(1)
	go func() {
		defer c.wg.Done()

		currentCount := 0
		for {
			select {
			case <-c.stopSignal:
				return
			default:
				if currentCount >= c.checkCount {
					currentCount = 0
					tmpFileName := filepath.Join(c.dir, "subgraph_tmp")
					task := NewCompressorTask(c.chain, tmpFileName, c.indexer.LatestHeight())
					if result := task.Run(); result.IsSuccess {
						c.indexer.Add(result.Ti, tmpFileName)
					}
				}
				time.Sleep(c.checkInterval)
				currentCount++
			}
		}
	}()

	return true
}

func (c *Compressor) Stop() {
	c.statusLock.Lock()
	defer c.statusLock.Unlock()

	c.stopSignal <- 1
	c.wg.Wait()

	c.stopSignal = make(chan int)
	c.status = STOPPED
}
