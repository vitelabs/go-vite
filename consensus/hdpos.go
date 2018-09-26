package consensus

import (
	"math/big"
	"time"

	"github.com/vitelabs/go-vite/common/types"
)

var DefaultMembers = []string{
	"vite_2ad661b3b5fa90af7703936ba36f8093ef4260aaaeb5f15cf8",
	"vite_1cb2ab2738cd913654658e879bef8115eb1aa61a9be9d15c3a",
	"vite_2ad1b8f936f015fc80a2a5857dffb84b39f7675ab69ae31fc8",
	"vite_85e8adb768aed85f2445eb1d71b933370d2980916baa3c1f3c",
	"vite_93dd41694edd756512da7c4af429f3e875c374a53bfd217e00",
	"vite_f8dfcad17c08f9748271cce96eddc2b3732b399f6367597708",

	//"vite_e77a71d44c65155e1474d708134c53c9dfb7af08b0299dc10d",
	//"vite_1e7d413c276725c6a9e2f5fa8fadfa291e16ea3695c27760b1",
	//"vite_946e7f9554cc5efc6e404a6510c3c6a098ac4296c049cc348b",
	//"vite_d6bf8cf4b590c8e7bfc846463e4b67522d5dda3040d4ff83e8",
	//"vite_ed143aa15a2bfb039b903e7368a297dc0a11288b4af7d7ceb5",
	//"vite_559ddc6918f287316a93ed063ca64621fe2e33a45457c90c11",
	//"vite_bdcb0d7ab9b3db2b229b70613f6a3d18c083457fae3289e926",
	//"vite_080d19e9edf8365e79a7a3e57d253f072648c843d1f7084818",
	//"vite_771e5d643432cd3a50249ed42d1e09c198f1ebd0425f8090e4",
	//"vite_0575741ef5b9fe23a107cc8e2affbd6799088f2debab8264af",
	//"vite_bae6cb6e8dcaa3cd7475fdf5520f14c0556bf9b0950aed0fd4",
	//"vite_cd585066ed35c1e27c9ac32a8fe13c6a1addbbe21e9badc84c",
	//"vite_98fe74d859935cbf833b876599abd9fc4334719cd62d039b5e",
	//"vite_d1f21f486b4ae94e2f3ce237e620ad524fbd6c1313275ef64c",
	//"vite_c873add4fada0029da924ec3557bd838a8de2220917da10569",
	//"vite_323033f4d510ee4ff9f0d4be1d71d63372c1c15d0c9001f041",
	//"vite_7a10c045903421df0915dd8d5807d37d2dff343df961365ef7",
	//"vite_c84d89b7adabaac7d86bd318f988ea39568a90eeafd3eece18",
	//"vite_8c52581f7498350be103b0f0ea20a4f66fde1193bd6668d04e",
}

func conv(mems []string) []types.Address {
	addressArr := make([]types.Address, len(mems))
	for i, v := range mems {
		addressArr[i], _ = types.HexToAddress(v)
	}
	return addressArr
}

type SubscribeMem struct {
	Mem    types.Address
	Notify chan time.Time
}

// SignerFn is a signer callback function to request a hash to be signed by a
// backing account.
type SignerFn func(a types.Address, data []byte) (signedData, pubkey []byte, err error)

type membersInfo struct {
	genesisTime time.Time
	memberCnt   int32 // Number of producer
	interval    int32 // unit: second, time interval at which the block is generated
	perCnt      int32 // Number of blocks generated per node
	randCnt     int32
	LowestLimit *big.Int
}

type memberPlan struct {
	STime  time.Time
	ETime  time.Time
	Member types.Address
}

type electionResult struct {
	Plans  []*memberPlan
	STime  time.Time
	ETime  time.Time
	Index  int32
	Hash   types.Hash
	Height uint64
}

func (self *membersInfo) genPlan(index int32, members []types.Address) *electionResult {
	result := electionResult{}
	sTime := self.genSTime(index)
	result.STime = sTime

	var plans []*memberPlan
	for _, member := range members {
		for i := int32(0); i < self.perCnt; i++ {
			etime := sTime.Add(time.Duration(self.interval) * time.Second)
			plan := memberPlan{STime: sTime, ETime: etime, Member: member}
			plans = append(plans, &plan)
			sTime = etime
		}
	}
	result.ETime = self.genETime(index)
	result.Plans = plans
	result.Index = index
	return &result
}

func (self *membersInfo) time2Index(t time.Time) int32 {
	subSec := int64(t.Sub(self.genesisTime).Seconds())
	i := subSec / int64((self.interval * self.memberCnt * self.perCnt))
	return int32(i)
}
func (self *membersInfo) genSTime(index int32) time.Time {
	planInterval := self.interval * self.memberCnt * self.perCnt
	return self.genesisTime.Add(time.Duration(planInterval*index) * time.Second)
}
func (self *membersInfo) genETime(index int32) time.Time {
	planInterval := self.interval * self.memberCnt * self.perCnt
	return self.genesisTime.Add(time.Duration(planInterval*index+1) * time.Second)
}
