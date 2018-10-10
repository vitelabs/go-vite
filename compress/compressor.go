package compress

import (
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"io"
	"os"
	"path"
	"path/filepath"
	"sync"
	"time"
)

const (
	STOPPED      = 0
	RUNNING      = 1
	TASK_RUNNING = 2
)

var compressorLog = log15.New("module", "compressor")

type Compressor struct {
	stopSignal chan int
	wg         sync.WaitGroup

	status     int // 0 is stop, 1 is start
	statusLock sync.Mutex

	dir string

	chain Chain

	indexer *Indexer
	ticker  *time.Ticker

	tickerDuration time.Duration
	log            log15.Logger
}

func NewCompressor(chain Chain, dataDir string) *Compressor {
	c := &Compressor{
		stopSignal: make(chan int),
		status:     STOPPED,
		chain:      chain,
		dir:        filepath.Join(dataDir, "ledger_files"),

		tickerDuration: time.Minute * 10,
		log:            log15.New("module", "compressor"),
	}

	if err := c.createDataDir(); err != nil {
		c.log.Crit("Create data directory failed, error is "+err.Error(), "method", "NewCompressor")
	}
	c.indexer = NewIndexer(c.dir)
	return c
}

func (c *Compressor) createDataDir() error {
	_, sErr := os.Stat(c.dir)
	if os.IsNotExist(sErr) {
		cErr := os.Mkdir(c.dir, 0744)

		if cErr != nil {
			return cErr

		}
	}
	return nil
}

func (c *Compressor) ClearData() error {
	if c.Stop() {
		defer c.Start()
	}

	c.indexer.Clear()
	c.indexer = nil

	if err := os.RemoveAll(c.dir); err != nil && err != os.ErrNotExist {
		return errors.New("Remove " + c.dir + " failed, error is " + err.Error())
	}

	if err := c.createDataDir(); err != nil {
		return errors.New("Create data directory failed, error is " + err.Error())
	}

	c.indexer = NewIndexer(c.dir)
	return nil
}

func (c *Compressor) Indexer() *Indexer {
	return c.indexer
}

func (c *Compressor) FileReader(filename string) io.ReadCloser {
	return NewFileReader(path.Join(c.dir, filename))
}

func (c *Compressor) BlockParser(reader io.Reader, processFunc func(block ledger.Block, err error)) {
	BlockParser(reader, 0, processFunc)
}

func (c *Compressor) Start() bool {
	c.statusLock.Lock()
	defer c.statusLock.Unlock()

	if c.status == RUNNING {
		compressorLog.Error("Compressor is running, Don't start again.")
		return false
	}

	c.log.Info("Compressor started.", "method", "Start")
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
				c.RunTask()
			}
		}
	}()

	return true
}

func (c *Compressor) RunTask() {
	c.statusLock.Lock()
	defer c.statusLock.Unlock()
	if c.status != RUNNING {
		compressorLog.Error("Compressor is not running, Can't run task.")
		return
	}

	c.status = TASK_RUNNING

	tmpFileName := filepath.Join(c.dir, "subgraph_tmp")
	task := NewCompressorTask(c.chain, tmpFileName, c.indexer.LatestHeight())
	if result := task.Run(); result.IsSuccess {
		c.indexer.Add(result.Ti, tmpFileName, result.BlockNumbers)
	}
	task.Clear()

	c.status = RUNNING
}

func (c *Compressor) Stop() bool {
	c.statusLock.Lock()
	defer c.statusLock.Unlock()

	if c.status == STOPPED {
		return false
	}

	c.log.Info("Compressor stopped.", "method", "Stop")

	// stop ticker
	c.ticker.Stop()

	// stop worker
	c.stopSignal <- 1

	// wait worker stop
	c.wg.Wait()

	// reset status
	c.stopSignal = make(chan int)
	c.status = STOPPED
	return true
}
