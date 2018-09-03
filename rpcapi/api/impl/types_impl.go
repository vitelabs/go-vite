package impl

import (
	"github.com/vitelabs/go-vite/common/types"
)

type TypesApisImpl struct {
}

func (TypesApisImpl) String() string {
	return "TypesApisImpl"
}

func (TypesApisImpl) IsValidHexAddress(addr string) bool {
	log.Info("IsValidHexAddress")
	return types.IsValidHexAddress(addr)
}

func (TypesApisImpl) IsValidHexTokenTypeId(tti string) bool {
	log.Info("IsValidHexTokenTypeId")
	return types.IsValidHexTokenTypeId(tti)
}
