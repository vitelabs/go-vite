package net

/* +----------------+----------------+--------------------------------------------------------------+
 * |     From       |       To       |                           Comment                         	|
 * +----------------+----------------+--------------------------------------------------------------+
 * |     Syncing    |    SyncDone    | success														|
 * |     Syncing    |   SyncError    | chain get stuck, download error, no peers					|
 * |     Syncing    |   SyncCancel   | stop							 							 	|
 * +----------------+----------------+--------------------------------------------------------------+
 * |   SyncError    |    Syncing     | new peer													 	|
 * |   SyncError    |    SyncDone    | in sync flow, taller peers disconnected, no need to sync	 	|
 * |   SyncError    |   SyncCancel   | stop														 	|
 * +----------------+----------------+--------------------------------------------------------------+
 * |    SyncDone    |    Syncing     | new peer													 	|
 * +----------------+----------------+--------------------------------------------------------------+
 * |    SyncInit    |    Syncing     | new peer											 			|
 * |    SyncInit    |    SyncDone    | no need to sync												|
 * |    SyncInit    |    SyncError   | no peers														|
 * |    SyncInit	|	SyncCancel	 | stop															|
 * +----------------+----------------+--------------------------------------------------------------+
 */

type SyncState byte

const (
	SyncInit SyncState = iota
	Syncing
	SyncDone
	SyncError
	SyncCancel
)

func (s SyncState) syncExited() bool {
	return s == SyncDone || s == SyncError || s == SyncCancel
}

var syncStatus = map[SyncState]string{
	SyncInit:   "Sync Not Start",
	Syncing:    "Synchronising",
	SyncDone:   "Sync done",
	SyncError:  "Sync error",
	SyncCancel: "Sync canceled",
}

func (s SyncState) String() string {
	status, ok := syncStatus[s]
	if ok {
		return status
	}

	return "unknown sync state"
}

type syncState interface {
	state() SyncState
	enter()
	sync()
	done()
	error()
	cancel()
}

type syncStateHost interface {
	setState(syncState)
}

// state init
type syncStateInit struct {
	host syncStateHost
}

func (s syncStateInit) state() SyncState {
	return SyncInit
}

func (s syncStateInit) enter() {
	// do nothing
}

func (s syncStateInit) sync() {
	s.host.setState(syncStateSyncing{
		host: s.host,
	})
}

func (s syncStateInit) done() {
	s.host.setState(syncStateDone{
		host: s.host,
	})
}

func (s syncStateInit) error() {
	s.host.setState(syncStateError{
		host: s.host,
	})
}

func (s syncStateInit) cancel() {
	s.host.setState(syncStateCancel{
		host: s.host,
	})
}

// state syncing
type syncStateSyncing struct {
	host syncStateHost
}

func (s syncStateSyncing) state() SyncState {
	return Syncing
}

func (s syncStateSyncing) enter() {

}

func (s syncStateSyncing) sync() {
	// self
	// maybe get taller peers
	s.host.setState(syncStateSyncing{
		host: s.host,
	})
}

func (s syncStateSyncing) done() {
	s.host.setState(syncStateDone{
		host: s.host,
	})
}

func (s syncStateSyncing) error() {
	s.host.setState(syncStateError{
		host: s.host,
	})
}

func (s syncStateSyncing) cancel() {
	s.host.setState(syncStateCancel{
		host: s.host,
	})
}

// state done
type syncStateDone struct {
	host syncStateHost
}

func (s syncStateDone) state() SyncState {
	return SyncDone
}

func (s syncStateDone) enter() {

}

func (s syncStateDone) sync() {
	s.host.setState(syncStateSyncing{
		host: s.host,
	})
}

func (s syncStateDone) done() {
	// self
}

func (s syncStateDone) error() {
	// cannot happen
}

func (s syncStateDone) cancel() {
	// do nothing
}

// state error
type syncStateError struct {
	host syncStateHost
}

func (s syncStateError) state() SyncState {
	return SyncError
}

func (s syncStateError) enter() {

}

func (s syncStateError) sync() {
	s.host.setState(syncStateSyncing{
		host: s.host,
	})
}

func (s syncStateError) done() {
	s.host.setState(syncStateDone{
		host: s.host,
	})
}

func (s syncStateError) error() {
	// self
}

func (s syncStateError) cancel() {
	s.host.setState(syncStateCancel{
		host: s.host,
	})
}

// state cancel
type syncStateCancel struct {
	host syncStateHost
}

func (s syncStateCancel) state() SyncState {
	return SyncCancel
}

func (s syncStateCancel) enter() {

}

func (s syncStateCancel) sync() {
	// cannot happen
}

func (s syncStateCancel) done() {
	// cannot happen
}

func (s syncStateCancel) error() {
	// cannot happen
}

func (s syncStateCancel) cancel() {
	// self
}
