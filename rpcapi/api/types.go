package api

import "github.com/vitelabs/go-vite/common/types"

type TypesApi struct {
}

func (TypesApi) String() string {
	return "TypesApi"
}

func (TypesApi) IsValidHexAddress(addr string) bool {
	log.Info("IsValidHexAddress")
	return types.IsValidHexAddress(addr)
}

func (TypesApi) IsValidHexTokenTypeId(tti string) bool {
	log.Info("IsValidHexTokenTypeId")
	return types.IsValidHexTokenTypeId(tti)
}
