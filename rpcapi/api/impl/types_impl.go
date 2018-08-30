package impl

import (
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"strconv"
)

type TypesApisImpl struct {
}

func (TypesApisImpl) String() string {
	return "TypesApisImpl"
}

func (TypesApisImpl) IsValidHexAddress(addrs []string, reply *string) error {
	log.Info("IsValidHexAddress")
	if len(addrs) != 1 {
		return fmt.Errorf("error length addrs %v", len(addrs))
	}
	*reply = strconv.FormatBool(types.IsValidHexAddress(addrs[0]))
	return nil
}

func (TypesApisImpl) IsValidHexTokenTypeId(ttis []string, reply *string) error {
	log.Info("IsValidHexTokenTypeId")
	if len(ttis) != 1 {
		return fmt.Errorf("error length ttis %v", len(ttis))
	}
	*reply = strconv.FormatBool(types.IsValidHexTokenTypeId(ttis[0]))
	return nil
}
