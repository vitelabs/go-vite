package apis

import "github.com/vitelabs/go-vite/vite"

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
