package impl

import (
	"encoding/json"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/rpcapi/api"
)

var log = log15.New("module", "rpc/api_impl")

func tryMakeConcernedError(err error, reply *string) error {
	errjson, concerned := api.MakeConcernedError(err)
	if concerned {
		*reply = errjson
		return nil
	}
	return err
}

func easyJsonReturn(v interface{}, reply *string) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	*reply = string(b)
	return nil
}
