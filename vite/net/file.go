package net

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/p2p"
	"github.com/vitelabs/go-vite/vite/net/message"
	"io"
	"net"
	"strconv"
	"sync"
	"time"
)

type connContext struct {
	net.Conn
	idle chan struct{}
	file *message.File
}

type fileReq struct {
	file *message.File
	nonce uint64
	peer *Peer
}

type FileServer struct {
	ln     net.Listener
	conns map[string]*connContext	// all connections
	record map[uint64]struct{}	// use to record nonce bind peerId
	request chan *fileReq
	term   chan struct{}
	log    log15.Logger
	wg     sync.WaitGroup
	chain Chain
	peers *peerSet
	receiver blockReceiver
}

func (f *FileServer) ID() string {
	return "default file handler"
}

func (f *FileServer) Cmds() []cmd {
	return []cmd{FileListCode}
}

func (f *FileServer) Handle(msg *p2p.Msg, sender *Peer) error {
	fs := new(message.FileList)
	err := fs.Deserialize(msg.Payload)
	if err != nil {
		return err
	}

	for _, file := range fs.Files {
		f.request <- &fileReq{
			file:  file,
			nonce: fs.Nonce,
			peer:  sender,
		}
	}

	for _, chunk := range fs.Chunk {
		if chunk[1] - chunk[0] != 0 {
			sender.Send(GetChunkCode, msg.Id, &message.Chunk{
				Start:chunk[0],
				End: chunk[1],
			})
		}
	}

	return nil
}

func newFileServer(port uint16, chain Chain, set *peerSet, receiver blockReceiver) (*FileServer, error) {
	ln, err := net.Listen("tcp", "0.0.0.0:"+strconv.FormatUint(uint64(port), 10))

	if err != nil {
		return nil, err
	}

	return &FileServer{
		ln:       ln,
		conns:    make(map[string]*connContext),
		record:   make(map[uint64]struct{}),
		request:  make(chan *fileReq, 10),
		term:     make(chan struct{}),
		log:      log15.New("module", "net/fileServer"),
		chain:    chain,
		peers:    set,
		receiver: receiver,
	}, nil
}

func (s *FileServer) start() {
	defer s.ln.Close()

	s.wg.Add(1)
	go s.sendLoop()

	s.wg.Add(1)
	go s.readLoop()
}

func (s *FileServer) stop() {
	select {
	case <-s.term:
	default:
		close(s.term)
	}

	s.wg.Wait()
}

func (s *FileServer) readLoop() {
	defer s.wg.Done()

	for {
		select {
		case <-s.term:
			return
		default:
			conn, err := s.ln.Accept()
			if err != nil {
				continue
			}

			s.wg.Add(1)
			go s.handleConn(conn)
		}
	}
}

func (f *FileServer) handleConn(conn net.Conn) {
	defer conn.Close()
	defer f.wg.Done()

	timer := time.NewTimer(10 * time.Second)
	defer timer.Stop()

	for {
		select {
		case <-f.term:
			return
		case <- timer.C:
			return
		default:
			msg, err := p2p.ReadMsg(conn, true)
			if err != nil {
				return
			}

			if cmd(msg.Cmd) == GetFileCode {
				timer.Stop()

				req := new(message.RequestFile)
				err = req.Deserialize(msg.Payload)

				if err != nil {
					f.log.Error(fmt.Sprintf("can`t parse message %s", GetFileCode), "error", err)
					return
				}

				nonce := req.Nonce
				if _, ok := f.record[nonce]; ok {
					io.Copy(conn, f.chain.Compressor().FileReader(req.Name))
				} else {
					f.log.Error("wrong nonce")
				}

				timer.Reset(10 * time.Second)
			}
		}
	}
}

func (f *FileServer) sendLoop() {
	defer f.wg.Done()

	for {
		select {
		case <-f.term:
			return
		case req := <- f.request:
			addr := req.peer.FileAddress().String()
			var ctx *connContext
			var ok bool
			if ctx, ok = f.conns[addr]; !ok {
				conn, err := net.Dial("tcp", addr)
				if err != nil {
					return
				}
				ctx = &connContext{
					Conn:        conn,
					idle:    make(chan struct{}, 1),
				}
				ctx.idle <- struct{}{}
				f.conns[addr] = ctx
			}

			select {
			case <- ctx.idle:
				ctx.file = req.file
				f.wg.Add(1)
				go f.Request(req, ctx)
			default:
				// put back
				f.request <- req
			}
		}
	}
}

func (f *FileServer) Request(req *fileReq, ctx *connContext) (err error) {
	defer f.wg.Done()

	msg := message.RequestFile{
		Name:  req.file.Name,
		Nonce: req.nonce,
	}

	data, err := msg.Serialize()
	if err != nil {
		return
	}

	n, err := ctx.Write(data)
	if err != nil {
		return
	}
	if n != len(data) {
		return errors.New("send incompele fileRequest")
	}

	f.readBlocks(ctx)

	ctx.idle <- struct{}{}

	return nil
}

func (f *FileServer) readBlocks(ctx *connContext) {
	defer ctx.Close()

	timer := time.NewTimer(10 * time.Second)
	defer timer.Stop()

	select {
	case <- f.term:
		return
	case <- timer.C:
		return
	default:
		sblocks := make([]*ledger.SnapshotBlock, 0, ctx.file.End - ctx.file.Start + 1)
		mblocks := make(map[types.Address][]*ledger.AccountBlock)

		f.chain.Compressor().BlockParser(ctx, func(block ledger.Block) {
			switch block.(type) {
			case *ledger.SnapshotBlock:
				sblocks = append(sblocks, block.(*ledger.SnapshotBlock))
			case *ledger.AccountBlock:
				ablock := block.(*ledger.AccountBlock)
				addr := ablock.AccountAddress
				mblocks[addr] = append(mblocks[addr], ablock)
			default:
				// nothing
			}
		})
		f.receiver.receiveSnapshotBlocks(sblocks)
		f.receiver.receiveAccountBlocks(mblocks)

		if !timer.Stop() {
			<- timer.C
		}
		timer.Reset(10 * time.Second)
	}
}
