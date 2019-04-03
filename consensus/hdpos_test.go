package consensus

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/gavv/monotime"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/consensus/core"
	"github.com/vitelabs/go-vite/wallet"
)

func genAddress(n int) []types.Address {
	dir := common.GoViteTestDataDir()
	wallet := wallet.New(&wallet.Config{
		DataDir: dir,
	})
	_, em, err := wallet.NewMnemonicAndEntropyStore("123")
	if err != nil {
		return nil
	}
	em.Unlock("123")

	addressArr := make([]types.Address, n)
	for i := 0; i < n; i++ {
		_, key, e := em.DeriveForIndexPath(uint32(i))
		if e != nil {
			fmt.Println(e)
			return nil

		}
		address, _ := key.Address()
		addressArr[i] = *address
	}
	return addressArr
}

func TestGenPlan(t *testing.T) {
	now := time.Now()
	println("now:\t" + now.Format(time.RFC3339))
	info := core.NewGroupInfo(now, types.ConsensusGroupInfo{NodeCount: 2, Interval: 6, Gid: types.SNAPSHOT_GID})
	var n = uint64(10)
	for i := uint64(0); i < n; i++ {
		plans := info.GenPlanByAddress(i, genAddress(int(n)))
		for i, p := range plans {
			println(strconv.Itoa(i) + ":\t" + p.STime.Format(time.StampMilli) + "\t" + p.Member.String() + "\t")
		}
	}
}

func TestTime2Index(t *testing.T) {
	now := time.Now()
	println("now:\t" + now.Format(time.RFC3339))
	info := core.NewGroupInfo(now, types.ConsensusGroupInfo{NodeCount: 25, Interval: 1, Gid: types.SNAPSHOT_GID, PerCount: 3})

	index := info.Time2Index(time.Now().Add(6 * time.Second))
	println("" + strconv.FormatInt(int64(index), 10))

	index = info.Time2Index(time.Now().Add(74 * time.Second))
	println("" + strconv.FormatInt(int64(index), 10))

	index = info.Time2Index(time.Now().Add(75 * time.Second))
	println("" + strconv.FormatInt(int64(index), 10))

	index = info.Time2Index(time.Now().Add(77 * time.Second))
	println("" + strconv.FormatInt(int64(index), 10))

	index = info.Time2Index(time.Now().Add(150 * time.Second))
	println("" + strconv.FormatInt(int64(index), 10))

	voteTime := info.GenVoteTime(2)
	voteIndex := info.Time2Index(voteTime.Add(time.Second))

	t.Log(fmt.Sprintf("viteTime:%s, voteIndex:%d", voteTime.String(), voteIndex))

	for i := uint64(90); i < 100; i++ {
		t.Log(fmt.Sprintf("startTime:%s, endTime:%s", info.GenSTime(i), info.GenETime(i)))
	}

	time2Index := info.Time2Index(now)
	t.Log("genesis index", time2Index)
	t.Log("genesis next index", info.Time2Index(now.Add(time.Nanosecond*1000)))
}

func TestUpdate(tt *testing.T) {
	////address := genAddress(1)
	////mem := SubscribeMem{Mem: address[0], Notify: times}
	////committee := NewCommittee(time.Now(), 1, int32(len(DefaultMembers)), 3, &chainRw{})
	//var committee Consensus
	////committee.Subscribe(&mem)
	//
	//println("nano sec:" + strconv.FormatInt(time.Millisecond.Nanoseconds(), 10))
	//committee.Subscribe(types.SNAPSHOT_GID, "test", nil, func(e Event) {
	//	addr := e.Address
	//	t := e.Stime
	//	println("addr: " + addr.Hex() +
	//		"\tdiff:" + strconv.FormatInt(time.Now().Sub(t).Nanoseconds(), 10) +
	//		"\ttime:" + t.String())
	//
	//	in := false
	//	electionResult, _ := committee.snapshot.electionTime(t)
	//	for _, plan := range electionResult.Plans {
	//		if plan.Member == addr {
	//			if plan.STime.Unix() == t.Unix() {
	//				in = true
	//				break
	//			}
	//		}
	//	}
	//	if !in {
	//		bytes, _ := json.Marshal(electionResult)
	//		tt.Error("can't find timeIndex, time:"+t.String()+", address:"+addr.String(), string(bytes))
	//	}
	//
	//})
	//
	//committee.Init()
	//go func() {
	//	committee.Start()
	//}()
	//
	//time.Sleep(150 * time.Second)
	//committee.Stop()
}
func TestGen(t *testing.T) {
	address := genAddress(4)
	for _, v := range address {
		println(v.String())
	}
}

func TestMonotime(t *testing.T) {
	for i := 0; i < 100; i++ {
		var start, elapsed time.Duration

		start = monotime.Now()
		time.Sleep(time.Millisecond * 500)
		elapsed = monotime.Since(start)

		fmt.Println(elapsed)
	}

	// Prints: 1.062759ms
}

func TestArray(t *testing.T) {
	data := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13}

	ints := data[0:3]
	fmt.Println(ints)

	ints = data[0:13]
	fmt.Println(ints)
}
