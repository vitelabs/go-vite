package compress

import (
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"io"
	"path"
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

	chain Chain

	indexer *Indexer
	ticker  *time.Ticker

	tickerDuration time.Duration
}

func NewCompressor(chain Chain, dataDir string) *Compressor {
	c := &Compressor{
		stopSignal: make(chan int),
		status:     STOPPED,
		chain:      chain,
		dir:        filepath.Join(dataDir, "ledger_files"),

		tickerDuration: time.Minute * 10,
	}

	c.indexer = NewIndexer(c.dir)
	return c
}

func (c *Compressor) Indexer() *Indexer {
	return c.indexer
}

func (c *Compressor) FileReader(filename string) io.ReadCloser {
	return NewFileReader(path.Join(c.dir, filename))
}

func (c *Compressor) BlockParser(reader io.Reader, processFunc func(block ledger.Block)) {
	BlockParser(reader, processFunc)
}

func (c *Compressor) Start() bool {
	c.statusLock.Lock()
	defer c.statusLock.Unlock()

	if c.status == RUNNING {
		compressorLog.Error("Compressor is running, Don't start again.")
		return false
	}

	c.status = RUNNING
	c.ticker = time.NewTicker(c.tickerDuration)

	c.wg.Add(1)
	go func() {
		defer c.wg.Done()

		for {
			select {
			case <-c.stopSignal:
				return

			case <-c.ticker.C:
				tmpFileName := filepath.Join(c.dir, "subgraph_tmp")
				task := NewCompressorTask(c.chain, tmpFileName, c.indexer.LatestHeight())
				if result := task.Run(); result.IsSuccess {
					c.indexer.Add(result.Ti, tmpFileName, result.BlockNumbers)
				}
				task.Clear()

			}
		}
	}()

	return true
}

func (c *Compressor) Stop() {
	c.statusLock.Lock()
	defer c.statusLock.Unlock()

	c.ticker.Stop()
	c.stopSignal <- 1
	c.wg.Wait()

	c.stopSignal = make(chan int)
	c.status = STOPPED
}
