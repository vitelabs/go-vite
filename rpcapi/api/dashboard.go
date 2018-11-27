package api

import (
	"runtime"
	"time"

	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/mem"
	"github.com/vitelabs/go-vite"
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

func (api DashboardApi) OsInfo() map[string]interface{} {
	result := make(map[string]interface{})
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

func (api DashboardApi) ProcessInfo() map[string]interface{} {
	result := make(map[string]interface{})
	result["build_version"] = govite.VITE_BUILD_VERSION
	result["commit_version"] = govite.VITE_VERSION
	return result
}

type hashHeightTime struct {
	Hash   string
	Height uint64
	Time   int64
}

func (api DashboardApi) RuntimeInfo() map[string]interface{} {
	result := make(map[string]interface{})
	result["peersNum"] = len(api.v.Net().Info().Peers)
	result["snapshotPendingNum"] = api.v.Pool().SnapshotPendingNum()
	result["accountPendingNum"] = api.v.Pool().AccountPendingNum().String()
	head := api.v.Chain().GetLatestSnapshotBlock()
	result["latestSnapshot"] = hashHeightTime{head.Hash.String(), head.Height, head.Timestamp.UnixNano() / 1e6}
	result["updateTime"] = time.Now().UnixNano() / 1e6
	return result
}
