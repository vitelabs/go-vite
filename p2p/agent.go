package p2p

import (
	"container/list"
	"crypto/rand"
	"errors"
	"fmt"
	"github.com/vitelabs/go-vite/monitor"
	"github.com/vitelabs/go-vite/p2p/discovery"
	"net"
	"time"
)

// @section Task
type Task interface {
	Perform(a *agent)
}

// @section discovery
type discoverTask struct {
	target  discovery.NodeID
	results []*discovery.Node
}

func (t *discoverTask) Perform(a *agent) {
	t.results = a.svr.discv.Lookup(t.target)
	a.TaskDone(t)

	monitor.LogDuration("p2p/server", "lookup", int64(a.svr.PeersCount()))

	p2pServerLog.Info(fmt.Sprintf("discv tab lookup %s %d nodes", t.target, len(t.results)))
}

// @section dial, connect to node
type dialTask struct {
	target *discovery.Node
}

func (t *dialTask) Perform(a *agent) {
	conn, err := a.Dialer.Dial("tcp", t.target.TCPAddr().String())
	if err != nil {
		p2pServerLog.Error(fmt.Sprintf("tcp dial node %s error: %v", t.target, err))
		return
	}

	a.svr.setupConn(conn, outbound)
	a.TaskDone(t)
}

// @section sleep
type waitTask struct {
	Duration time.Duration
}

func (t *waitTask) Perform(_ *agent) {
	time.Sleep(t.Duration)
}

// @section DialManager
type agent struct {
	*net.Dialer
	maxDials    uint
	dialing     map[discovery.NodeID]connFlag
	aTime       time.Time
	bootNodes   []*discovery.Node
	looking     bool
	wating      bool
	lookResults []*discovery.Node
	svr         *Server
	term        chan struct{}
}

func newAgent(svr *Server) *agent {
	return &agent{
		Dialer: &net.Dialer{
			Timeout: 3 * time.Second,
		},
		maxDials:  svr.maxOutboundPeers(),
		dialing:   make(map[discovery.NodeID]connFlag),
		bootNodes: copyNodes(svr.BootNodes), // agent will modify bootNodes, so use copies
		svr:       svr,
	}
}

func (a *agent) start() {
	a.aTime = time.Now()
	go a.taskLoop()
}

func (a *agent) stop() {
	select {
	case <-a.term:
	default:
		close(a.term)
	}
}

func (a *agent) taskLoop() {
	tasks := list.New()
	a.createTasks(tasks)

	for {
		select {
		case <-a.term:
			return
		default:
			if e := tasks.Front(); e != nil {
				monitor.LogEvent("p2p/dial", "task")

				task, _ := e.Value.(Task)
				task.Perform(a)
				tasks.Remove(e)
			} else {
				a.createTasks(tasks)
			}
		}
	}
}

func (a *agent) createDialTask(n *discovery.Node) (d *dialTask, err error) {
	if err = a.checkDial(n); err != nil {
		return
	}

	a.dialing[n.ID] = outbound
	return &dialTask{
		target: n,
	}, nil
}

func (a *agent) createTasks(tasks *list.List) {
	canDials := a.maxDials - uint(a.svr.peers.outbound) - uint(len(a.dialing))
	restDials := canDials

	// dial one bootNodes
	if a.svr.peers.Size() == 0 && restDials > 0 {
		boot := a.bootNodes[0]
		copy(a.bootNodes[:], a.bootNodes[1:])
		a.bootNodes[len(a.bootNodes)-1] = boot

		task, err := a.createDialTask(boot)
		if err == nil {
			tasks.PushBack(task)
			restDials--
		}
	}

	// randomNodes from table
	randomCandidates := restDials / 2
	if randomCandidates > 0 {
		randomNodes := make([]*discovery.Node, randomCandidates)
		n := a.svr.discv.RandomNodes(randomNodes)
		for i := 0; i < n; i++ {
			task, err := a.createDialTask(randomNodes[i])
			if err == nil {
				tasks.PushBack(task)
				restDials--
			}
		}
	}

	resultIndex := 0
	for ; resultIndex < len(a.lookResults) && restDials > 0; resultIndex++ {
		task, err := a.createDialTask(a.lookResults[resultIndex])
		if err == nil {
			tasks.PushBack(task)
			restDials--
		}
	}
	a.lookResults = a.lookResults[resultIndex:]

	if len(a.lookResults) == 0 && !a.looking {
		var id discovery.NodeID
		rand.Read(id[:])
		tasks.PushBack(&discoverTask{
			target: id,
		})

		a.looking = true
	}

	if restDials == 0 && !a.wating {
		tasks.PushBack(&waitTask{
			Duration: 3 * time.Minute,
		})
		a.wating = true
	}

	p2pServerLog.Info(fmt.Sprintf("p2p server create %d tasks", canDials-restDials))
}

func (a *agent) TaskDone(t Task) {
	switch t2 := t.(type) {
	case *dialTask:
		delete(a.dialing, t2.target.ID)
	case *discoverTask:
		a.looking = false

		a.lookResults = append(a.lookResults, t2.results...)
	case *waitTask:
		a.wating = false
	}
}

func (a *agent) checkDial(n *discovery.Node) error {
	if _, ok := a.dialing[n.ID]; ok {
		return fmt.Errorf("%s is dialing", n)
	}

	if a.svr.peers.Has(n.ID) {
		return fmt.Errorf("%s has connected", n)
	}

	if n.ID == a.svr.self.ID {
		return errors.New("cannot dail self")
	}

	return nil
}
