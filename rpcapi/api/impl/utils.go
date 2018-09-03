package impl

import (
	"encoding/json"
	"github.com/vitelabs/go-vite/log15"
)

var log = log15.New("module", "rpc/api_impl")

func easyJsonReturn(v interface{}, reply *string) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	*reply = string(b)
	return nil
}
