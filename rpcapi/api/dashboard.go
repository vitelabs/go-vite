package api

import (
	"os"
	"runtime"
	"time"

	"github.com/vitelabs/go-vite/common/hexutil"

	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/mem"
	"github.com/vitelabs/go-vite"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/vite"
)

type DashboardApi struct {
	v *vite.Vite
}

func NewDashboardApi(v *vite.Vite) *DashboardApi {
	return &DashboardApi{
		v: v,
	}
}

func (api DashboardApi) OsInfo(id *string) map[string]interface{} {
	result := make(map[string]interface{})
	if id != nil {
		result["reqId"] = id
	}
	stat, e := host.Info()
	if e == nil {
		result["os"] = stat.OS                           // ex: freebsd, linux
		result["platform"] = stat.Platform               // ex: ubuntu, linuxmint
		result["platformFamily"] = stat.PlatformFamily   // ex: debian, rhel
		result["platformVersion"] = stat.PlatformVersion // version of the complete OS
		result["kernelVersion"] = stat.KernelVersion     // version of the OS kernel (if available)
	} else {
		result["err"] = e
	}
	memS, e := mem.VirtualMemory()
	if e != nil {
		result["memTotal"] = memS.Total
		result["memFree"] = memS.Free
	}
	result["cpuNum"] = runtime.NumCPU()
	result["gorountine"] = runtime.NumGoroutine()
	return result
}

func (api DashboardApi) ProcessInfo(id *string) map[string]interface{} {
	result := make(map[string]interface{})
	if id != nil {
		result["reqId"] = id
	}
	result["build_version"] = govite.VITE_BUILD_VERSION
	result["commit_version"] = govite.VITE_VERSION
	if api.v.Config().Reward != nil {
		result["nodeName"] = api.v.Config().Name
		result["rewardAddress"] = api.v.Config().RewardAddr
	}
	result["pid"] = os.Getpid()

	return result
}

type hashHeightTime struct {
	Hash   string
	Height uint64
	Time   int64
}

func (api DashboardApi) RuntimeInfo(id *string) map[string]interface{} {
	result := make(map[string]interface{})
	if id != nil {
		result["reqId"] = id
	}
	result["peersNum"] = len(api.v.Net().Info().Peers)
	result["snapshotPendingNum"] = api.v.Pool().SnapshotPendingNum()
	result["accountPendingNum"] = api.v.Pool().AccountPendingNum().String()
	head := api.v.Chain().GetLatestSnapshotBlock()
	result["latestSnapshot"] = hashHeightTime{head.Hash.String(), head.Height, head.Timestamp.UnixNano() / 1e6}
	result["updateTime"] = time.Now().UnixNano() / 1e6
	result["delayTime"] = api.v.Net().Info().Latency
	if api.v.Producer() != nil {
		result["producer"] = api.v.Producer().GetCoinBase().String()
	}
	priKey := api.v.P2P().Config().PeerKey
	sign := ed25519.Sign(priKey, head.Hash.Bytes())
	result["signData"] = hexutil.Encode(sign)
	return result
}

func (api DashboardApi) NetId() uint {
	return netId
}
