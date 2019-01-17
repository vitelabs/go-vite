package net

import (
	"errors"
	"fmt"
	"io"
	"math/rand"
	net2 "net"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vitelabs/go-vite/p2p/list"

	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/p2p"
	"github.com/vitelabs/go-vite/vite/net/message"
)

var errNoSuitablePeers = errors.New("no suitable peers")

type fileConnState byte

const (
	fileConnStateNew fileConnState = iota
	fileConnStateIdle
	fileConnStateBusy
	fileConnStateClosed
)

type fileParser interface {
	BlockParser(reader io.Reader, blockNum uint64, processFunc func(block ledger.Block, err error))
}

type fileConn struct {
	net2.Conn
	id     peerId
	st     uint64 // time + state
	failed int    // failed times
	speed  int64  // download speed
	parser fileParser
}

func (f *fileConn) state() (st fileConnState, t int64) {
	stt := atomic.LoadUint64(&f.st)
	return fileConnState(stt & 0xff), int64(stt >> 8)
}

func (f *fileConn) setState(st fileConnState) {
	stt := uint64(time.Now().Unix()<<8) | uint64(st)
	atomic.StoreUint64(&f.st, stt)
}

func (f *fileConn) download(file File, rec blockReceiver) (height uint64, fatal bool, outerr error) {
	f.setState(fileConnStateBusy)
	defer f.setState(fileConnStateIdle)

	getFiles := &message.GetFiles{
		Names: []string{file.Filename},
	}

	msg, outerr := p2p.PackMsg(CmdSet, p2p.Cmd(GetFilesCode), 0, getFiles)

	if outerr != nil {
		return
	}

	f.Conn.SetWriteDeadline(time.Now().Add(fWriteTimeout))
	if outerr = p2p.WriteMsg(f.Conn, msg); outerr != nil {
		return
	}

	var sCount, aCount uint64
	f.Conn.SetReadDeadline(time.Now().Add(fileTimeout))
	f.parser.BlockParser(f.Conn, file.BlockNumbers, func(block ledger.Block, err error) {
		// Fatal error, then close the connection, and disconnect the peer
		if outerr != nil {
			fatal = true
			f.Conn.Close()
		}

		switch block.(type) {
		case *ledger.SnapshotBlock:
			block := block.(*ledger.SnapshotBlock)
			sCount++
			outerr = rec.receiveSnapshotBlock(block)
			height = block.Height

		case *ledger.AccountBlock:
			block := block.(*ledger.AccountBlock)
			aCount++
			outerr = rec.receiveAccountBlock(block)
		}
	})

	sTotal := file.EndHeight - file.StartHeight + 1
	aTotal := file.BlockNumbers - sTotal
	if sCount < sTotal || aCount < aTotal {
		outerr = fmt.Errorf("incomplete file %s %d/%d, %d/%d", file.Filename, sCount, sTotal, aCount, aTotal)
		return
	}

	return
}

func (f *fileConn) reset() {

}

func (f *fileConn) close() error {
	f.setState(fileConnStateClosed)
	return f.Conn.Close()
}

type fileConns []*fileConn

func (fl fileConns) Len() int {
	return len(fl)
}

func (fl fileConns) Less(i, j int) bool {
	return fl[i].speed > fl[j].speed
}

func (fl fileConns) Swap(i, j int) {
	fl[i], fl[j] = fl[j], fl[i]
}

type fileConnections struct {
	sources []*fileSource
	conns   fileConns
}

type filename = string
type peerId = string

type fileSource struct {
	id    string
	addr  string
	files []File
	fail  int32
}

type fileClient struct {
	fqueue list.List // wait to download

	parser fileParser

	rec blockReceiver

	peers *peerSet

	mu       sync.Mutex
	filePool map[filename]fileConnections
	connPool map[peerId]*fileConn
	srcPool  map[peerId]*fileSource

	dialer *net2.Dialer

	log log15.Logger
}

func newFileClient(parser fileParser, rec blockReceiver, peers *peerSet) *fileClient {
	return &fileClient{
		fqueue:   list.New(),
		parser:   parser,
		peers:    peers,
		dialer:   &net2.Dialer{Timeout: 5 * time.Second},
		rec:      rec,
		filePool: make(map[filename]fileConnections),
		connPool: make(map[peerId]*fileConn),
		srcPool:  make(map[peerId]*fileSource),
		log:      log15.New("module", "net/fileClient"),
	}
}

func (fc *fileClient) addResource(files Files, sender Peer) {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	id := sender.ID()

	src := fc.srcPool[id]
	if src == nil {
		src = &fileSource{
			id:   id,
			addr: sender.RemoteAddr().String(),
		}
	}

	src.files = files
	for _, file := range files {
		if fcons, ok := fc.filePool[file.Filename]; ok {

		} else {
			fcons = fileConnections{
				sources: []*fileSource{src},
				conns:   nil,
			}
		}
	}

	fc.srcPool[id] = src
}

func (fc *fileClient) delResource(id peerId) {

}

func (fc *fileClient) download(file File) (height uint64, err error) {
	var c *fileConn
	var src *fileSource

	for {
		if c, err = fc.chooseConn(file.Filename); err != nil {
			// no peers
			return 0, err
		} else if c != nil {
			height, err = fc.doJob(c, file)
			if err != nil {
				// retry
				continue
			} else {
				// done
				return
			}
		} else if src, err = fc.chooseFreshSrc(file.Filename); err != nil {
			// no peers
			return
		} else if src != nil {
			c, err = fc.createConn(src)
			// dial error, will retry
			if err != nil {
				continue
			} else {
				height, err = fc.doJob(c, file)
				if err != nil {
					continue
				} else {
					return
				}
			}
		} else {
			fc.wait(file)
		}
	}
}

func (fc *fileClient) doJob(c *fileConn, file File) (height uint64, err error) {
	var fatal bool
	height, fatal, err = c.download(file, fc.rec)
	if err != nil && fatal {
		fc.fatalPeer(c.id)
	}
	return
}

func (fc *fileClient) wait(file File) {
	fc.mu.Lock()
	fc.fqueue.Append(file)
	fc.mu.Unlock()

	time.Sleep(500 * time.Millisecond)
}

func (fc *fileClient) fatalPeer(id string) {

}

func (fc *fileClient) stop() {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	for id, c := range fc.connPool {
		c.close()
		delete(fc.connPool, id)
	}

	fc.filePool = make(map[filename]fileConnections)
}

// tcp dial error
func (fc *fileClient) createConn(src *fileSource) (c *fileConn, err error) {
	tcp, err := fc.dialer.Dial("tcp", src.addr)

	if err != nil {
		atomic.AddInt32(&src.fail, 1)
		return
	}

	fc.mu.Lock()
	defer fc.mu.Unlock()

	// has a file connection, maybe create concurrently
	var exist bool
	if c, exist = fc.connPool[src.id]; exist {
		tcp.Close()
		return
	}

	c = &fileConn{
		Conn:   tcp,
		id:     src.id,
		parser: fc.parser,
	}
	c.setState(fileConnStateNew)

	// add to conn pool
	fc.connPool[src.id] = c

	// add to file pool
	for _, file := range src.files {
		name := file.Filename
		if fpool, ok := fc.filePool[name]; ok {
			fpool.conns = append(fpool.conns, c)
		}
	}
	return
}

// choose a src, haven`t created file connection
// will return error if have no peers
func (fc *fileClient) chooseFreshSrc(name filename) (src *fileSource, err error) {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	if p, ok := fc.filePool[name]; ok {
		n := len(p.sources)
		if n == 0 {
			return nil, errNoSuitablePeers
		}

		ids := rand.Perm(n)
		for i := 0; i < n; i++ {
			j := ids[i]
			src = p.sources[j]
			if _, ok = fc.connPool[src.id]; ok {
				continue
			}

			return
		}

		return nil, nil
	}
	return nil, errNoSuitablePeers
}

// choose a fast file connection, or random create new conns
func (fc *fileClient) chooseConn(name filename) (c *fileConn, err error) {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	if p, ok := fc.filePool[name]; ok {
		m := len(p.sources)
		if m == 0 {
			return nil, errNoSuitablePeers
		}

		n := len(p.conns)
		hasIdlePeers := len(p.sources) > n

		if n > 0 {
			// sort by speed
			sort.Sort(p.conns)

			// create new conn, don`t choose slow file conn
			create := rand.Intn(10) < 5

			for i := 0; i < n; i++ {
				c = p.conns[i]
				st, _ := c.state()
				if st == fileConnStateBusy {
					continue
				}

				if i <= n/3 {
					return
				}

				if hasIdlePeers && create {
					return nil, nil
				}

				return
			}
		}
	}
	return nil, errNoSuitablePeers
}

func (fc *fileClient) remove(id peerId) {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	//id := sender.ID()
	//
	//if conn, ok := conns[id]; ok {
	//	delete(conns, id)
	//	conn.Close()
	//}
	//
	//delete(pFiles, id)
	//
	//for _, r := range fRecord {
	//	r.peers = r.peers.delete(id)
	//}
}
