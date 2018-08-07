package apis

import (
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/config"
	"os"
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

func (CommonApisImpl) LogDir(noop interface{}, reply *string) error {
	info, e := os.Stat(config.GlobalConfig.RunLogDir())
	if e != nil {
		*reply = ""
		return nil
	}
	if !info.IsDir() {
		*reply = ""
		return nil
	}
	*reply = config.GlobalConfig.RunLogDir()
	return nil
}
