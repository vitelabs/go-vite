package apis

import (
	"github.com/vitelabs/go-vite/vite"
	"encoding/json"
)

type GlobalApis interface {
	GetGlobalDataDir(noop interface{}, reply *string) error
}

type GlobalApisImpl struct {
	v *vite.Vite
}

func (g GlobalApisImpl) GetGlobalDataDir(noop interface{}, reply *string) error {
	*reply = g.v.Config().DataDir
	return nil
}

func NewGlobalApis(v *vite.Vite) GlobalApis {
	return &GlobalApisImpl{v: v}
}

func easyJsonReturn(v interface{}, reply *string) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	*reply = string(b)
	return nil
}