package consensus

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/log15"
)

func TestContractDposCs_ElectionIndex(t *testing.T) {
	dir := "/Users/jie/Documents/vite/src/github.com/vitelabs/cluster1/ledger_datas/ledger_1/devdata"
	genesisJson := GenesisJson
	c, err := NewChainInstanceFromDir(dir, false, genesisJson)

	assert.NoError(t, err)

	rw := newChainRw(c, log15.New())
	groupInfo, err := rw.GetMemberInfo(types.DELEGATE_GID)
	assert.NoError(t, err)

	cs := newContractDposCs(groupInfo, rw, log15.New())

	proof := newRollbackProof(rw.rw)
	index := cs.Time2Index(time.Now())
	for i := index; i > 0; i-- {
		_, etime := cs.Index2Time(i)
		hashes, err := proof.ProofHash(etime)
		if err != nil {
			t.Error(err)
			assert.FailNow(t, err.Error())
		}
		if rw.rw.IsGenesisSnapshotBlock(hashes) {
			break
		}

		result, err := cs.ElectionIndex(i)
		assert.NoError(t, err)

		vs := "\n"
		for _, v := range result.Plans {
			vs += fmt.Sprintf("\t\t[%s][%s][%s]\n", v.STime, v.ETime, v.Member)
		}
		t.Log(i, len(result.Plans), hashes, result.STime, result.ETime, vs)

	}

}
