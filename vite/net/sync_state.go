package net

import "errors"

var errTooShort = errors.New("too short")
var errUnknownSyncState = errors.New("unknown sync state")

/* +----------------+----------------+--------------------------------------------------------------+
 * |     From       |       To       |                           Comment                         	|
 * +----------------+----------------+--------------------------------------------------------------+
 * |     Syncing    |    SyncDone    | success														|
 * |     Syncing    |   SyncError    | chain get stuck, download error, no peers					|
 * |     Syncing    |   SyncCancel   | stop							 							 	|
 * +----------------+----------------+--------------------------------------------------------------+
 * |   SyncError    |    Syncing     | new peer													 	|
 * |   SyncError    |    SyncDone    | in sync flow, taller peers disconnected, no need to sync	 	|
 * |   SyncError    |   SyncCancel   | in sync flow, stop											|
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

func (s *SyncState) UnmarshalText(text []byte) error {
	str := string(text)
	for k, v := range syncStatus {
		if v == str {
			*s = k
			return nil
		}
	}

	return errUnknownSyncState
}

func (s SyncState) MarshalText() (text []byte, err error) {
	return []byte(s.String()), nil
}

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

	return errUnknownSyncState.Error()
}

type syncErrorCode byte

var syncError = map[syncErrorCode]string{
	syncErrorNoPeers:  "no peers",
	syncErrorStuck:    "stuck",
	syncErrorDownload: "download error",
}

const (
	syncErrorNoPeers syncErrorCode = iota
	syncErrorStuck
	syncErrorDownload
)

func (e syncErrorCode) Error() string {
	text, ok := syncError[e]
	if ok {
		return text
	}

	return "unknown sync error"
}

type syncState interface {
	state() SyncState
	enter()
	sync()
	done()
	error(reason syncErrorCode)
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

func (s syncStateInit) error(reason syncErrorCode) {
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

func (s syncStateSyncing) error(reason syncErrorCode) {
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

func (s syncStateDone) error(reason syncErrorCode) {
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

func (s syncStateError) error(reason syncErrorCode) {
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

func (s syncStateCancel) error(reason syncErrorCode) {
	// cannot happen
}

func (s syncStateCancel) cancel() {
	// self
}
