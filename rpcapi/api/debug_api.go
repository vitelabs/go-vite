package api

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/vite"
)

type DebugApi struct {
	v *vite.Vite
}

func (api DebugApi) Info(addr *types.Address) string {
	return api.v.Pool().Info(addr)
}

func (api DebugApi) Snapshot() map[string]interface{} {
	return api.v.Pool().Snapshot()
}

func (api DebugApi) Account(addr types.Address) map[string]interface{} {
	return api.v.Pool().Account(addr)
}

func (api DebugApi) SnapshotChainDetail(chainId string) map[string]interface{} {
	return api.v.Pool().SnapshotChainDetail(chainId)
}

func (api DebugApi) AccountChainDetail(addr types.Address, chainId string) map[string]interface{} {
	return api.v.Pool().AccountChainDetail(addr, chainId)
}

func (api DebugApi) Nodes() []string {
	if p2p := api.v.P2P(); p2p != nil {
		return p2p.Nodes()
	}

	return nil
}

func NewDebugApi(v *vite.Vite) *DebugApi {
	return &DebugApi{
		v: v,
	}
}
