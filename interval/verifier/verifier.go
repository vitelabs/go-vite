package verifier

import (
	"time"

	"github.com/vitelabs/go-vite/interval/common"
	"github.com/vitelabs/go-vite/interval/common/face"
	"github.com/vitelabs/go-vite/interval/utils"
	"github.com/vitelabs/go-vite/interval/version"
)

type VerifyResult int

const (
	PENDING VerifyResult = iota
	FAIL
	SUCCESS
)

func (result VerifyResult) Done() bool {
	if result == FAIL || result == SUCCESS {
		return true
	} else {
		return false
	}
}

type VerifyType int

const (
	VerifyReferred VerifyType = iota
)

type Callback func(block common.Block, stat BlockVerifyStat)
type Verifier interface {
	VerifyReferred(block common.Block) BlockVerifyStat
}

type BlockVerifyStat interface {
	VerifyResult() VerifyResult
	ErrMsg() string
	Task() Task
}

type Task interface {
	Done() bool
	Requests() []face.FetchRequest
}

func NewFailTask() Task {
	return &failTask{t: time.Now()}
}
func NewSuccessTask() Task {
	return successT
}

var successT = &successTask{}

type successTask struct {
}

func (t *successTask) Done() bool {
	return true
}

func (*successTask) Requests() []face.FetchRequest {
	return nil
}

type failTask struct {
	t time.Time
}

func (self *failTask) Done() bool {
	if time.Now().After(self.t.Add(time.Second * 3)) {
		return true
	}
	return false
}

func (*failTask) Requests() []face.FetchRequest {
	return nil
}

type verifyTask struct {
	tasks   []Task
	version int
	reader  face.ChainReader
	v       *version.Version
	fail    bool
	t       time.Time
}

func (vt *verifyTask) Requests() []face.FetchRequest {
	var reqs []face.FetchRequest
	for _, t := range vt.tasks {
		for _, r := range t.Requests() {
			reqs = append(reqs, r)
		}
	}
	return reqs
}

func (vt *verifyTask) Done() bool {
	// if version increase.  task has done.
	if vt.v.Val() != vt.version {
		return true
	}

	if time.Now().After(vt.t.Add(time.Second * 5)) {
		return true
	}
	// if because fail, must wait for version increase.
	if vt.fail {
		return false
	}
	// pending task check
	for _, t := range vt.tasks {
		if !t.Done() {
			return false
		}
	}
	// all pending done
	return true
}

type accountPendingTask struct {
	reader  face.AccountReader
	result  bool
	request face.FetchRequest
}

func (self *accountPendingTask) Requests() []face.FetchRequest {
	var reqs []face.FetchRequest
	return append(reqs, self.request)
}
func (self *accountPendingTask) Done() bool {
	if self.result {
		return true
	}
	block, e := self.reader.HeadAccount(self.request.Chain)
	if e == nil && block != nil && block.Height() >= self.request.Height {
		self.result = true
		return true
	}
	return false
}

type snapshotPendingTask struct {
	reader  face.SnapshotReader
	result  bool
	request face.FetchRequest
}

func (spt *snapshotPendingTask) Requests() []face.FetchRequest {
	var reqs []face.FetchRequest
	return append(reqs, spt.request)
}

func (spt *snapshotPendingTask) Done() bool {
	if spt.result {
		return true
	}

	block, e := spt.reader.HeadSnapshot()
	if e == nil && block != nil && block.Height() >= spt.request.Height {
		spt.result = true
		return true
	}
	return false
}

func (vt *verifyTask) pendingSnapshot(hash string, height uint64) {
	request := face.FetchRequest{Chain: "", Hash: hash, Height: height, PrevCnt: 1}
	vt.tasks = append(vt.tasks, &snapshotPendingTask{vt.reader, false, request})
}
func (vt *verifyTask) pendingAccount(addr string, height uint64, hash string, prevCnt uint64) {
	request := face.FetchRequest{Chain: addr, Hash: hash, Height: height, PrevCnt: prevCnt}
	vt.tasks = append(vt.tasks, &accountPendingTask{vt.reader, false, request})
}

func VerifyAccount(block *common.AccountStateBlock) bool {
	hash := utils.CalculateAccountHash(block)
	return block.Hash() == hash
}

func VerifySnapshotHash(block *common.SnapshotBlock) bool {
	hash := utils.CalculateSnapshotHash(block)
	return block.Hash() == hash
}
