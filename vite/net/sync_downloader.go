/*
 * Copyright 2019 The go-vite Authors
 * This file is part of the go-vite library.
 *
 * The go-vite library is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Lesser General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * The go-vite library is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Lesser General Public License for more details.
 *
 * You should have received a copy of the GNU Lesser General Public License
 * along with the go-vite library. If not, see <http://www.gnu.org/licenses/>.
 */

package net

import (
	"fmt"
	net2 "net"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/vitelabs/go-vite/common/types"

	"github.com/vitelabs/go-vite/interfaces"

	"github.com/vitelabs/go-vite/log15"
)

type reqState = int32

const (
	reqWaiting reqState = iota
	reqPending
	reqDone
	reqError
	reqCancel
)

var reqStatus = map[reqState]string{
	reqWaiting: "waiting",
	reqPending: "pending",
	reqDone:    "done",
	reqError:   "error",
	reqCancel:  "canceled",
}

type syncTask struct {
	from, to          uint64
	prevHash, endHash types.Hash
	st                reqState
	doneAt            time.Time
}

func (t *syncTask) String() string {
	return strconv.FormatUint(t.from, 10) + "-" + strconv.FormatUint(t.to, 10) + " " + t.prevHash.String() + "-" + t.endHash.String()
}

func (t *syncTask) status() string {
	return t.String() + " " + reqStatus[t.st]
}

func (t *syncTask) wait() {
	t.st = reqWaiting
}

func (t *syncTask) cancel() {
	t.st = reqCancel
}

func (t *syncTask) pending() {
	t.st = reqPending
}

func (t *syncTask) done() {
	t.st = reqDone
	t.doneAt = time.Now()
}

func (t *syncTask) error() {
	if t.st == reqPending {
		t.st = reqError
	}
}

type syncTasks []*syncTask

func (s syncTasks) Len() int {
	return len(s)
}

func (s syncTasks) Less(i, j int) bool {
	return s[i].from < s[j].from
}

func (s syncTasks) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

type chunks [][2]uint64

func (cs chunks) Len() int {
	return len(cs)
}

func (cs chunks) Less(i, j int) bool {
	return cs[i][0] < cs[j][0]
}

func (cs chunks) Swap(i, j int) {
	cs[i], cs[j] = cs[j], cs[i]
}

func (cs chunks) overlap(from, to uint64) (conflict [2]uint64, ok bool) {
	ok = true

	if len(cs) == 0 {
		return
	}

	n := sort.Search(len(cs), func(i int) bool {
		return cs[i][0] > from
	})

	if n == 0 {
		if cs[0][0] <= to {
			return cs[0], false
		} else {
			return
		}
	}

	if cs[n-1][1] >= from {
		return cs[n-1], false
	}

	if n == len(cs) {
		return
	}

	if cs[n][0] <= to {
		return cs[n], false
	}

	return
}

func missingSegments(sortedList interfaces.SegmentList, from, to uint64) (mis [][2]uint64) {
	for _, segment := range sortedList {
		// useless
		if segment.Bound[1] < from {
			continue
		}

		if segment.Bound[0] > to {
			break
		}

		// missing front piece
		if segment.Bound[0] > from {
			mis = append(mis, [2]uint64{
				from,
				segment.Bound[0] - 1,
			})
		}

		// next response
		from = segment.Bound[1] + 1
	}

	// from should equal (cr.to + 1)
	if from-1 < to {
		mis = append(mis, [2]uint64{
			from,
			to,
		})
	}

	return
}

// chunks should be continuous [from, to]
func missingChunks(chunks [][2]uint64, from, to uint64) (mis [][2]uint64) {
	for _, chunk := range chunks {
		// useless
		if chunk[1] < from {
			continue
		}

		if chunk[0] > to {
			break
		}

		// missing front piece
		if chunk[0] > from {
			mis = append(mis, [2]uint64{
				from,
				chunk[0] - 1,
			})
		}

		// next response
		from = chunk[1] + 1
	}

	// from should equal (cr.to + 1)
	if from-1 < to {
		mis = append(mis, [2]uint64{
			from,
			to,
		})
	}

	return
}

func missingTasks(tasks syncTasks, from, to uint64) (mis syncTasks) {
	for _, t := range tasks {
		// useless
		if t.to < from {
			continue
		}

		if t.from > to {
			break
		}

		// missing front piece
		if t.from > from {
			mis = append(mis, &syncTask{
				from: from,
				to:   t.from - 1,
			})
		}

		// next response
		from = t.to + 1
	}

	// from should equal (cr.to + 1)
	if from-1 < to {
		mis = append(mis, &syncTask{
			from: from,
			to:   to,
		})
	}

	return
}

type DownloaderStatus struct {
	Tasks       []string               `json:"tasks"`
	Connections []SyncConnectionStatus `json:"connections"`
}

type syncDownloader interface {
	start()
	stop()
	status() DownloaderStatus
	// will be block, if cannot download (eg. no peers) or task queue is full
	// must will download the task regardless of task repeat
	download(t *syncTask, must bool) bool
	// cancel tasks between from and to
	cancel(from uint64) (end uint64)
	addListener(listener taskListener)
}

type taskListener = func(t syncTask, err error)

type executor struct {
	mu         sync.Mutex
	tasks      syncTasks
	cond       *sync.Cond
	max, batch int

	pool    *connPoolImpl
	factory syncConnInitiator
	dialing map[string]struct{}
	dialer  *net2.Dialer

	listeners []taskListener
	running   bool
	wg        sync.WaitGroup

	log log15.Logger
}

func newExecutor(max, batch int, peers *peerSet, factory syncConnInitiator) *executor {
	e := &executor{
		max:     max,
		batch:   batch,
		tasks:   make(syncTasks, 0, max),
		pool:    newPool(peers),
		factory: factory,
		dialing: make(map[string]struct{}),
		dialer: &net2.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 5 * time.Second,
		},
		log: netLog.New("module", "downloader"),
	}

	e.cond = sync.NewCond(&e.mu)
	return e
}

func (e *executor) start() {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.running {
		return
	}

	e.running = true

	e.tasks = e.tasks[:0]

	e.wg.Add(1)
	go e.loop()
}

func (e *executor) stop() {
	e.mu.Lock()
	if false == e.running {
		e.mu.Unlock()
		return
	}

	e.running = false
	e.mu.Unlock()

	e.cond.Broadcast()
	e.wg.Wait()
}

func (e *executor) addListener(listener taskListener) {
	e.listeners = append(e.listeners, listener)
}

func (e *executor) status() DownloaderStatus {
	e.mu.Lock()
	tasks := make([]string, len(e.tasks))
	for i, t := range e.tasks {
		tasks[i] = t.status()
	}
	e.mu.Unlock()

	st := DownloaderStatus{
		tasks,
		e.pool.connections(),
	}

	return st
}

// from must be larger than 0
func addTasks(tasks syncTasks, t2 *syncTask, must bool) syncTasks {
	if must {
		var i, j int
		for i = 0; i < len(tasks); i++ {
			t := tasks[i]
			if t.st == reqDone {
				continue
			}

			tasks[j] = tasks[i]
			j++
		}

		tasks = tasks[:j]
	}

	tasks = append(tasks, t2)

	sort.Sort(tasks)
	return tasks
}

// will be blocked when task queue is full
func (e *executor) download(t *syncTask, must bool) bool {
	e.mu.Lock()
	defer e.mu.Unlock()

	if false == must {
		for {
			if len(e.tasks) == cap(e.tasks) && e.running {
				e.cond.Wait()
			} else {
				break
			}
		}
	}

	if false == e.running {
		return false
	}

	e.tasks = addTasks(e.tasks, t, must)

	e.cond.Signal()

	return true
}

func (e *executor) cancel(from uint64) (end uint64) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.tasks, end = cancelTasks(e.tasks, from)
	return
}

// cancel tasks if `t.to > from`
func cancelTasks(tasks syncTasks, from uint64) (tasks2 syncTasks, end uint64) {
	var total = len(tasks)

	if total == 0 {
		return tasks, 0
	}

	var j int
	var t *syncTask
	for i := total - 1; i > -1; i-- {
		t = tasks[i]
		if t.to > from {
			t.cancel()
			j++
			continue
		}

		break
	}

	total = total - j
	tasks = tasks[:total]
	if total > 0 {
		end = tasks[total-1].to
	}

	return tasks, end
}

func runTasks(tasks syncTasks, maxBatch int, exec func(t *syncTask)) syncTasks {
	var total = len(tasks)
	var now = time.Now()

	var clean = 0 // clean continuous done tasks
	var continuous = true

	var index = 0
	var batch = 0
	var t *syncTask

	for index = 0; index < total; index += batch {
		t = tasks[index]

		if t.st == reqDone {
			index++
			if continuous && now.Sub(t.doneAt) > 5*time.Second {
				clean++
			}
			continue
		} else if t.st == reqCancel {
			index++
			if continuous {
				clean++
			}
			continue
		} else {
			continuous = false
		}

		if t.st == reqPending {
			batch++
		} else if batch < maxBatch {
			batch++
			exec(t)
		} else {
			break
		}
	}

	if clean > total/5 {
		total = copy(tasks, tasks[clean:])
		tasks = tasks[:total]
	}

	return tasks
}

func (e *executor) loop() {
	defer e.wg.Done()

	var total int
	var batch int
	var peerCount int

Loop:
	for {
		e.mu.Lock()
		for total = len(e.tasks); total == 0 && e.running; total = len(e.tasks) {
			e.cond.Wait()
		}
		if false == e.running {
			e.mu.Unlock()
			break Loop
		}

		batch = e.batch
		peerCount = e.pool.peers.count()
		if batch > peerCount {
			batch = peerCount
		}
		e.tasks = runTasks(e.tasks, batch, e.run)
		e.mu.Unlock()

		if len(e.tasks) < total {
			e.cond.Broadcast()
		}

		// have tasks, but running tasks is batch, so wait for task done
		time.Sleep(100 * time.Millisecond)
	}
}

func (e *executor) run(t *syncTask) {
	t.pending()

	go e.do(t)
}

func (e *executor) doJob(c *syncConn, t *syncTask) error {
	start := time.Now()

	e.log.Info(fmt.Sprintf("download chunk %s from %s", t.String(), c.address()))

	if fatal, err := c.download(*t); err != nil {
		e.log.Warn(fmt.Sprintf("failed to download chunk %s from %s: %v", t, c.address(), err))

		if fatal {
			e.pool.delConn(c)
			e.log.Warn(fmt.Sprintf("delete sync connection %s: %v", c.address(), err))
		}

		return err
	}

	e.log.Info(fmt.Sprintf("download chunk %s from %s elapse %s", t, c.address(), time.Now().Sub(start)))

	return nil
}

func (e *executor) createConn(p Peer) (c *syncConn, err error) {
	addr := p.FileAddress()

	e.mu.Lock()
	if _, ok := e.dialing[addr]; ok {
		e.mu.Unlock()
		err = errPeerDialing
		return
	}
	e.dialing[addr] = struct{}{}
	e.mu.Unlock()

	tcp, err := e.dialer.Dial("tcp", addr)

	e.mu.Lock()
	delete(e.dialing, addr)
	e.mu.Unlock()

	// dial error
	if err != nil {
		e.log.Error(fmt.Sprintf("failed to create sync connection %s: %v", addr, err))
		return
	}

	// handshake error
	c, err = e.factory.initiate(tcp, p)
	if err != nil {
		_ = tcp.Close()
		e.log.Error(fmt.Sprintf("failed to create sync connection %s: %v", addr, err))
		return
	}

	// add error
	if err = e.pool.addConn(c); err != nil {
		e.pool.delConn(c)
		e.log.Warn(fmt.Sprintf("failed to add sync connection: %s: %v", addr, err))
	}

	return
}

func (e *executor) do(t *syncTask) {
	var p Peer
	var c *syncConn
	var err error

	if p, c, err = e.pool.chooseSource(t); err != nil {
		// no tall enough peers
	} else if c != nil {
		if err = e.doJob(c, t); err == nil {
			// downloaded
			t.done()
		}
	} else if p != nil {
		if c, err = e.createConn(p); err == nil {
			if err = e.doJob(c, t); err == nil {
				// downloaded
				t.done()
			}
		}
	} else {
		// no idle peers
		t.wait()
	}

	// only t.st == reqPending
	t.error()

	if t.st == reqDone {
		e.notify(t, err)
	} else {
		// maybe syncConn is busy, should wait
		time.Sleep(100 * time.Millisecond)
	}
}

func (e *executor) notify(t *syncTask, err error) {
	for _, listener := range e.listeners {
		listener(*t, err)
	}
}
