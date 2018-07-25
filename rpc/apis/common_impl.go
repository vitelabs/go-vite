package apis

import (
	"fmt"
	"encoding/json"
	"github.com/vitelabs/go-vite/common/types"
	"strconv"
)

type CommonApisImpl struct {
}

func (CommonApisImpl) String() string {
	return "CommonApisImpl"
}

func (CommonApisImpl) IsValidHexAddress(addrs []string, reply *string) error {

	if len(addrs) != 1 {
		return fmt.Errorf("error length addrs %v", len(addrs))
	}
	*reply = strconv.FormatBool(types.IsValidHexAddress(addrs[0]))
	return nil
}

func (CommonApisImpl) IsValidHexTokenTypeId(ttis []string, reply *string) error {
	if len(ttis) != 1 {
		return fmt.Errorf("error length ttis %v", len(ttis))
	}
	*reply = strconv.FormatBool(types.IsValidHexTokenTypeId(ttis[0]))
	return nil
}

func easyJsonReturn(v interface{}, reply *string) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	*reply = string(b)
	return nil
}