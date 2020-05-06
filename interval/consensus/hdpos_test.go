package consensus

import (
	"strconv"
	"testing"
	"time"

	"github.com/vitelabs/go-vite/interval/common"
)

func genAddress(n int) []common.Address {
	addressArr := make([]common.Address, n)
	for i := 0; i < n; i++ {
		addressArr[i] = common.HexToAddress("viteshan" + strconv.Itoa(i))
	}
	return addressArr
}

func TestGenPlan(t *testing.T) {
	now := time.Now()
	println("now:\t" + now.Format(time.RFC3339))
	info := membersInfo{genesisTime: now, memberCnt: 2, interval: 6}
	var n = 10
	for i := 0; i < n; i++ {
		result := info.genPlan(int32(i), genAddress(n))
		plans := result.plans
		for i, p := range plans {
			println(strconv.Itoa(i) + ":\t" + p.sTime.Format(time.StampMilli) + "\t" + p.member.String() + "\t" +
				result.sTime.Format(time.StampMilli) + "\t" + result.eTime.Format(time.StampMilli))
		}
	}
}

func TestTime2Index(t *testing.T) {
	now := time.Now()
	println("now:\t" + now.Format(time.RFC3339))
	info := membersInfo{genesisTime: now, memberCnt: 2, interval: 6}

	index := info.time2Index(time.Now().Add(6 * time.Second))
	println("" + strconv.FormatInt(int64(index), 10))

	index = info.time2Index(time.Now().Add(13 * time.Second))
	println("" + strconv.FormatInt(int64(index), 10))

	var i int
	i = 1000000000000000
	println(strconv.Itoa(i))

}

func TestUpdate(t *testing.T) {
	address := genAddress(1)
	mem := SubscribeMem{Mem: address[0], Notify: make(chan time.Time)}
	committee := NewCommittee(time.Now(), 1, 4)
	committee.Subscribe(&mem)

	committee.update()
}
func TestGen(t *testing.T) {
	address := genAddress(4)
	for _, v := range address {
		println(v.String())
	}
}

func TestRemovePrevious(t *testing.T) {
	teller := newTeller(time.Now(), 1, 4)
	for i := 0; i < 10; i++ {
		teller.electionIndex(int32(i))
	}
	time.Sleep(10 * time.Second)
	cnt := teller.removePrevious(time.Now())
	println("个数:\t" + strconv.Itoa(int(cnt)))
}
