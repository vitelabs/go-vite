package net

/* +----------------+----------------+-----------------------------------------------------------+
 * |     From       |       To       |                           Comment                         |
 * +----------------+----------------+-----------------------------------------------------------+
 * |    SyncWait    |     Syncing    | find taller peers										 |
 * |    SyncWait    |    SyncDone    | find peers, but not tall enough							 |
 * |    SyncWait    |   SyncCancel   | cancel before wait timeout								 |
 * |    SyncWait    |    SyncError   | cannot find peers after wait timeout						 |
 * +----------------+----------------+-----------------------------------------------------------+
 * |     Syncing    | SyncDownloaded | all sync tasks done										 |
 * |     Syncing    |    SyncDone    | taller peers disconnected, cancel remaining tasks		 |
 * |     Syncing    |   SyncError    | chain stop growing long time before all sync tasks done	 |
 * |     Syncing    |   SyncCancel   | cancel before all tasks done 							 |
 * +----------------+----------------+-----------------------------------------------------------+
 * | SyncDownloaded |    SyncDone    | all tasks done, and chain grow to target height			 |
 * | SyncDownloaded |    SyncError   | chain stop growing long time after all sync tasks done	 |
 * | SyncDownloaded |   SyncCancel   | cancel after all tasks done, before chain grow to target	 |
 * | SyncDownloaded |     Syncing    | find taller peers, syncPeer is more taller				 |
 * +----------------+----------------+-----------------------------------------------------------+
 * |   SyncError    |    SyncWait    | last sync flow exited, eg. no peers, find new taller peers|
 * |   SyncError    |    Syncing     | in sync flow, find new taller peers						 |
 * |   SyncError    |    SyncDone    | in sync flow, tall peers disconnected, no need to sync	 |
 * |   SyncError    |   SyncCancel   | in sync flow, cancel										 |
 * +----------------+----------------+-----------------------------------------------------------+
 * |   SyncCancel   |    SyncWait    | new sync flow											 |
 * +----------------+----------------+-----------------------------------------------------------+
 * |    SyncDone    |    SyncWait    | new sync flow											 |
 * +----------------+----------------+-----------------------------------------------------------+
 * |    SyncInit    |    SyncWait    | find new peer											 |
 * +----------------+----------------+-----------------------------------------------------------+
 */

type SyncState byte

const (
	SyncInit SyncState = iota
	SyncWait
	Syncing
	SyncDownloaded
	SyncDone
	SyncError
	SyncCancel
)

func (s SyncState) syncExited() bool {
	return s == SyncDone || s == SyncError || s == SyncCancel
}

var syncStatus = map[SyncState]string{
	SyncInit:       "Sync Not Start",
	SyncWait:       "Sync wait",
	Syncing:        "Synchronising",
	SyncDownloaded: "Sync downloaded",
	SyncDone:       "Sync done",
	SyncError:      "Sync error",
	SyncCancel:     "Sync canceled",
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
	wait()
	start()
	downloaded()
	done()
	error()
	cancel()
}

type syncStateHost interface {
	setState(state syncState)
}

type syncStateInit struct {
	host syncStateHost
}

func (s syncStateInit) state() SyncState {
	return SyncInit
}

func (s syncStateInit) wait() {
	s.host.setState(syncStateWait{
		host: s.host,
	})
}

func (s syncStateInit) start() {
	// cannot happen
}

func (s syncStateInit) downloaded() {
	// cannot happen
}

func (s syncStateInit) done() {
	// cannot happen
}

func (s syncStateInit) error() {
	// cannot happen
}

func (s syncStateInit) cancel() {
	// cannot happen
}

type syncStateWait struct {
	host syncStateHost
}

func (s syncStateWait) state() SyncState {
	return SyncWait
}
func (s syncStateWait) wait() {
	// nothing
}

func (s syncStateWait) start() {
	s.host.setState(syncStatePending{
		host: s.host,
	})
}

func (s syncStateWait) downloaded() {
	// cannot happen
}

func (s syncStateWait) done() {
	s.host.setState(syncStateDone{
		host: s.host,
	})
}

func (s syncStateWait) error() {
	s.host.setState(syncStateError{
		host: s.host,
	})
}

func (s syncStateWait) cancel() {
	s.host.setState(syncStateCancel{
		host: s.host,
	})
}

type syncStatePending struct {
	host syncStateHost
}

func (s syncStatePending) wait() {
	// cannot happen
}

func (s syncStatePending) state() SyncState {
	return Syncing
}

func (s syncStatePending) start() {
	// self
}

func (s syncStatePending) downloaded() {
	s.host.setState(syncStateDownloaded{
		host: s.host,
	})
}

func (s syncStatePending) done() {
	s.host.setState(syncStateDone{
		host: s.host,
	})
}

func (s syncStatePending) error() {
	s.host.setState(syncStateError{
		host: s.host,
	})
}

func (s syncStatePending) cancel() {
	s.host.setState(syncStateCancel{
		host: s.host,
	})
}

type syncStateDownloaded struct {
	host syncStateHost
}

func (s syncStateDownloaded) state() SyncState {
	return SyncDownloaded
}

func (s syncStateDownloaded) wait() {
	// cannot happen
}

func (s syncStateDownloaded) start() {
	s.host.setState(syncStatePending{
		host: s.host,
	})
}

func (s syncStateDownloaded) downloaded() {
	// self
}

func (s syncStateDownloaded) done() {
	s.host.setState(syncStateDone{
		host: s.host,
	})
}

func (s syncStateDownloaded) error() {
	s.host.setState(syncStateError{
		host: s.host,
	})
}

func (s syncStateDownloaded) cancel() {
	s.host.setState(syncStateCancel{
		host: s.host,
	})
}

type syncStateDone struct {
	host syncStateHost
}

func (s syncStateDone) state() SyncState {
	return SyncDone
}

func (s syncStateDone) wait() {
	s.host.setState(syncStateWait{
		host: s.host,
	})
}

func (s syncStateDone) start() {
	// cannot happen
}

func (s syncStateDone) downloaded() {
	// cannot happen
}

func (s syncStateDone) done() {
	// self
}

func (s syncStateDone) error() {
	// cannot happen
}

func (s syncStateDone) cancel() {
	// cannot happen
}

type syncStateError struct {
	host syncStateHost
}

func (s syncStateError) state() SyncState {
	return SyncError
}

func (s syncStateError) wait() {
	s.host.setState(syncStateWait{
		host: s.host,
	})
}

func (s syncStateError) start() {
	s.host.setState(syncStatePending{
		host: s.host,
	})
}

func (s syncStateError) downloaded() {
	// cannot happen
}

func (s syncStateError) done() {
	s.host.setState(syncStateDone{
		host: s.host,
	})
}

func (s syncStateError) error() {
	s.host.setState(syncStateError{
		host: s.host,
	})
}

func (s syncStateError) cancel() {
	s.host.setState(syncStateCancel{
		host: s.host,
	})
}

type syncStateCancel struct {
	host syncStateHost
}

func (s syncStateCancel) wait() {
	s.host.setState(syncStateWait{
		host: s.host,
	})
}

func (s syncStateCancel) state() SyncState {
	return SyncCancel
}

func (s syncStateCancel) start() {
	// cannot happen
}

func (s syncStateCancel) downloaded() {
	// cannot happen
}

func (s syncStateCancel) done() {
	// cannot happen
}

func (s syncStateCancel) error() {
	// cannot happen
}

func (s syncStateCancel) cancel() {
	// cannot happen
}
