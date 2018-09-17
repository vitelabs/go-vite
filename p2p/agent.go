package p2p

import (
	"crypto/rand"
	"errors"
	"fmt"
	"github.com/seiflotfy/cuckoofilter"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/monitor"
	"github.com/vitelabs/go-vite/p2p/discovery"
	"sync"
	"time"
)

// @section Task
type Task interface {
	Perform(svr *Server)
}

type discoverTask struct {
	results []*discovery.Node
}

func (t *discoverTask) Perform(svr *Server) {
	var target discovery.NodeID
	rand.Read(target[:])
	t.results = svr.discv.Lookup(target)

	monitor.LogDuration("p2p/server", "lookup", int64(svr.PeersCount()))

	p2pServerLog.Info(fmt.Sprintf("discv tab lookup %s %d nodes\n", target, len(t.results)))
}

type dialTask struct {
	flag         connFlag
	target       *discovery.Node
	lastResolved time.Time
	duration     time.Duration
}

func (t *dialTask) Perform(svr *Server) {
	if t.target.ID == svr.discv.ID() {
		return
	}

	conn, err := svr.Dialer.DailNode(t.target)
	if err != nil {
		p2pServerLog.Error(fmt.Sprintf("tcp dial node %s error: %v\n", t.target, err))
		return
	}

	svr.SetupConn(conn, outbound)
}

type waitTask struct {
	Duration time.Duration
}

func (t *waitTask) Perform(svr *Server) {
	time.Sleep(t.Duration)
}

// @section DialManager
type DialManager struct {
	maxDials    uint
	dialing     map[discovery.NodeID]connFlag
	start       time.Time
	bootNodes   []*discovery.Node
	looking     bool
	wating      bool
	lookResults []*discovery.Node
	discv       Discovery
}

func (dm *DialManager) CreateTasks(peers map[discovery.NodeID]*Peer, blockList map[discovery.NodeID]*blockNode) []Task {
	if dm.start.IsZero() {
		dm.start = time.Now()
	}

	var tasks []Task
	addDailTask := func(flag connFlag, n *discovery.Node) bool {
		if _, ok := blockList[n.ID]; ok {
			return false
		}

		if err := dm.checkDial(n, peers); err != nil {
			return false
		}

		dm.dialing[n.ID] = flag
		tasks = append(tasks, &dialTask{
			flag:   flag,
			target: n,
		})
		return true
	}

	dials := dm.maxDials
	for _, p := range peers {
		if p.rw.is(outbound) {
			dials--
		}
	}
	for _, f := range dm.dialing {
		if f.is(outbound) {
			dials--
		}
	}

	// dial one bootNodes
	if len(peers) == 0 && dials > 0 {
		boot := dm.bootNodes[0]
		copy(dm.bootNodes[:], dm.bootNodes[1:])
		dm.bootNodes[len(dm.bootNodes)-1] = boot

		if addDailTask(outbound, boot) {
			dials--
		}
	}

	// randomNodes from table
	randomCandidates := dials / 2
	if randomCandidates > 0 {
		randomNodes := make([]*discovery.Node, randomCandidates)
		n := dm.discv.RandomNodes(randomNodes)
		for i := 0; i < n; i++ {
			if addDailTask(outbound, randomNodes[i]) {
				dials--
			}
		}
	}

	resultIndex := 0
	for ; resultIndex < len(dm.lookResults) && dials > 0; resultIndex++ {
		if addDailTask(outbound, dm.lookResults[resultIndex]) {
			dials--
		}
	}
	dm.lookResults = dm.lookResults[resultIndex:]

	if len(dm.lookResults) == 0 && !dm.looking {
		tasks = append(tasks, &discoverTask{})
		dm.looking = true
	}

	if len(tasks) == 0 && !dm.wating {
		tasks = append(tasks, &waitTask{
			Duration: 3 * time.Minute,
		})
		dm.wating = true
	}

	p2pServerLog.Info("p2p server create tasks", "tasks", len(tasks))

	return tasks
}

func (dm *DialManager) TaskDone(t Task) {
	switch t2 := t.(type) {
	case *dialTask:
		delete(dm.dialing, t2.target.ID)
	case *discoverTask:
		dm.looking = false

		self := dm.discv.ID()
		for _, node := range t2.results {
			if self != node.ID {
				dm.lookResults = append(dm.lookResults, node)
			}
		}
	case *waitTask:
		dm.wating = false
	}
}

func (dm *DialManager) checkDial(n *discovery.Node, peers map[discovery.NodeID]*Peer) error {
	_, exist := dm.dialing[n.ID]
	if exist {
		return fmt.Errorf("%s is dialing", n)
	}
	if peers[n.ID] != nil {
		return fmt.Errorf("%s has connected", n)
	}
	if n.ID == dm.discv.ID() {
		return errors.New("self node")
	}

	return nil
}

func NewDialManager(discv Discovery, maxDials uint, bootNodes []*discovery.Node) *DialManager {
	return &DialManager{
		maxDials:  maxDials,
		bootNodes: copyNodes(bootNodes), // dm will modify bootNodes
		dialing:   make(map[discovery.NodeID]connFlag),
		discv:     discv,
	}
}

type topoHandler struct {
	record cuckoofilter.CuckooFilter
	lock   sync.RWMutex
}

func (th *topoHandler) Add(hash types.Hash) {
	th.lock.Lock()
	defer th.lock.Unlock()
	th.record.Insert(hash[:])
}
func (th *topoHandler) Has(hash types.Hash) bool {
	th.lock.RLock()
	defer th.lock.RUnlock()

	return th.record.Lookup(hash[:])
}

func (th *topoHandler) Broadcast(topo *Topo, peers []*Peer) {
	th.lock.RLock()
	defer th.lock.RUnlock()
	for _, peer := range peers {
		go Send(peer.rw, baseProtocolCmdSet, topoCmd, topo)
	}
}
