package p2p

import (
	"crypto/rand"
	"errors"
	"fmt"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/monitor"
	"github.com/vitelabs/go-vite/p2p/discovery"
	"github.com/vitelabs/go-vite/p2p/list"
	"net"
	"time"
)

const maxRunTasks = 10

var errNilNode = errors.New("node is nil")

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
	a.log.Info(fmt.Sprintf("run discoveryTask, find %s", t.target))

	t.results = a.svr.discv.Lookup(t.target)

	monitor.LogDuration("p2p/server", "lookup", int64(a.svr.PeersCount()))

	a.log.Info(fmt.Sprintf("run discoveryTask done, find %s, got %d neighbors", t.target, len(t.results)))
}

// @section dial, connect to node
type dialTask struct {
	target *discovery.Node
}

func (t *dialTask) Perform(a *agent) {
	a.log.Info(fmt.Sprintf("run dialTask to %s", t.target))

	conn, err := a.Dialer.Dial("tcp", t.target.TCPAddr().String())
	if err != nil {
		a.log.Error(fmt.Sprintf("run dialTask to %s error: %v", t.target, err))
		return
	}

	a.svr.pending <- struct{}{}
	a.svr.setupConn(conn, outbound)
}

// @section sleep
type waitTask struct {
	Duration time.Duration
}

func (t *waitTask) Perform(a *agent) {
	a.log.Info("run waitTask")

	time.Sleep(t.Duration)
}

// @section DialManager
type agent struct {
	*net.Dialer
	maxDials    uint
	dialing     map[discovery.NodeID]connFlag
	bootNodes   []*discovery.Node
	looking     bool
	wating      bool
	lookResults []*discovery.Node
	svr         *Server
	log         log15.Logger
	tasks       *list.List
	running     chan struct{}
}

func newAgent(svr *Server) *agent {
	a := &agent{
		Dialer: &net.Dialer{
			Timeout: 3 * time.Second,
		},
		maxDials:  svr.maxOutboundPeers(),
		dialing:   make(map[discovery.NodeID]connFlag),
		bootNodes: copyNodes(svr.BootNodes), // agent will modify bootNodes, so use copies
		svr:       svr,
		log:       log15.New("module", "p2p/agent"),
		tasks:     list.New(),
		running:   make(chan struct{}, maxRunTasks),
	}

	return a
}

func (a *agent) scheduleTasks(stop <-chan struct{}, taskDone chan<- struct{}) {
	for {
		select {
		case <-stop:
			return
		case a.running <- struct{}{}:
			if t := a.tasks.Shift(); t != nil {
				t, _ := t.(Task)
				go func(t Task) {
					t.Perform(a)
					<-a.running
				}(t)
			} else {
				a.running <- struct{}{}
				taskDone <- struct{}{}
			}
		}
	}
}

func (a *agent) createDialTask(n *discovery.Node) (d *dialTask, err error) {
	if n == nil {
		return nil, errNilNode
	}

	if err = a.checkDial(n); err != nil {
		return
	}

	a.dialing[n.ID] = outbound
	return &dialTask{
		target: n,
	}, nil
}

func (a *agent) createTasks() uint {
	canDials := a.maxDials - uint(a.svr.peers.outbound) - uint(len(a.dialing))
	restDials := canDials

	// dial one bootNodes
	if a.svr.peers.Size() == 0 && restDials > 0 && len(a.bootNodes) > 0 {
		boot := a.bootNodes[0]
		copy(a.bootNodes[:], a.bootNodes[1:])
		a.bootNodes[len(a.bootNodes)-1] = boot

		task, err := a.createDialTask(boot)
		if err == nil {
			a.tasks.Append(task)
			restDials--
			a.log.Info(fmt.Sprintf("create dialTask: bootnode<%s>", boot))
		}
	}

	// randomNodes from table
	randomCandidates := restDials / 2
	if randomCandidates > 0 {
		randomNodes := make([]*discovery.Node, randomCandidates)
		n := a.svr.discv.RandomNodes(randomNodes)

		a.log.Info(fmt.Sprintf("get %d random nodes from table", n))

		for i := 0; i < n; i++ {
			task, err := a.createDialTask(randomNodes[i])
			if err == nil {
				a.tasks.Append(task)
				restDials--

				a.log.Info(fmt.Sprintf("create dialTask: random<%s>", randomNodes[i]))
			}
		}
	}

	resultIndex := 0
	for ; resultIndex < len(a.lookResults) && restDials > 0; resultIndex++ {
		task, err := a.createDialTask(a.lookResults[resultIndex])

		if err == nil {
			a.tasks.Append(task)
			restDials--

			a.log.Info(fmt.Sprintf("create dialTask: lookResult<%s>", a.lookResults[resultIndex]))
		}
	}
	a.lookResults = a.lookResults[resultIndex:]

	if len(a.lookResults) == 0 && !a.looking {
		var id discovery.NodeID
		rand.Read(id[:])
		a.tasks.Append(&discoverTask{
			target: id,
		})

		a.looking = true
		restDials--

		a.log.Info(fmt.Sprintf("create discoveryTask: target<%s>", id))
	}

	if canDials-restDials == 0 && !a.wating {
		a.tasks.Append(&waitTask{
			Duration: 3 * time.Minute,
		})
		a.wating = true

		a.log.Info("create waitTask")
	}

	a.log.Info(fmt.Sprintf("create %d tasks", canDials-restDials))

	return canDials - restDials
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
	default:
		// do nothing
	}
}

var errHasDialing = errors.New("node is dialing")
var errHasConnected = errors.New("node has connected")
var errDialSelf = errors.New("can`t dial self")

func (a *agent) checkDial(n *discovery.Node) error {
	if _, ok := a.dialing[n.ID]; ok {
		return errHasDialing
	}

	if a.svr.peers.Has(n.ID) {
		return errHasConnected
	}

	if n.ID == a.svr.self.ID {
		return errDialSelf
	}

	return nil
}
