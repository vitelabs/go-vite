package rpc

import "github.com/julienschmidt/httprouter"

var _router *httprouter.Router

func GetRouter () *httprouter.Router {
	if _router == nil {
		_router = httprouter.New()

		_router.GET("/", HelloWorld)
	}

	return _router
}

